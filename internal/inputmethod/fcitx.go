// Package inputmethod 处理 Linux 桌面中文输入法(IME)相关问题。
//
// 背景: Fedora 的 KDE Wayland 会话在启动 fcitx5 时可能通过 imsettings 将
// XMODIFIERS 设为 @im=none, 导致 fcitx5 的 XIM 服务未注册到 X 根窗口。
// 本程序使用 GLFW X11(XWayland) 后端, 依赖 XIM 协议接收中文输入,
// 因此这种情况下无法输入中文。这里在启动早期检测并修复。
package inputmethod

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/cindyhuang123/hylbscode/internal/logging"
)

const fcitxProcName = "fcitx5"

// EnsureFcitxXIM 确保 fcitx5 的 XIM 服务可用(仅 Linux)。
// 只在同时满足以下条件时动作, 其他环境完全不干预:
//   - 系统存在 fcitx5 可执行文件
//   - 已运行的 fcitx5 进程带有 XMODIFIERS=@im=none (异常)
//
// 修复方式: 通过 dbus 请求 fcitx5 退出, 再以正确的环境变量重新启动。
// 同时会把正确的 XMODIFIERS/GTK_IM_MODULE/QT_IM_MODULE 设置到当前进程,
// 供 GLFW X11 后端连接输入法使用。
func EnsureFcitxXIM() {
	if runtime.GOOS != "linux" {
		return
	}
	bin := findFcitx5()
	if bin == "" {
		return
	}

	broken := fcitxXIMBroken()
	stale := ximServersStale()
	if !broken && !stale {
		return
	}

	if broken {
		logging.Info("fcitx5 XIM service not registered (XMODIFIERS=@im=none), restarting fcitx5")
	} else {
		logging.Info("stale XIM_SERVERS detected, restarting fcitx5 to re-register")
	}
	// 先退出旧 fcitx5, 删除残留属性, 再以正确环境重启让其重新注册 XIM 服务。
	if err := requestFcitxExit(); err != nil {
		logging.Warn("failed to request fcitx5 exit", "err", err)
	}
	time.Sleep(time.Second)
	if stale {
		logging.Info("clearing stale XIM_SERVERS on root window")
		if err := clearXIMServers(); err != nil {
			logging.Warn("failed to clear XIM_SERVERS", "err", err)
		}
	}
	if err := startFcitx5(bin); err != nil {
		logging.Warn("failed to start fcitx5", "err", err)
	}
	time.Sleep(2 * time.Second)

	// 让当前进程(及其子进程)使用正确的输入法环境变量。
	os.Setenv("XMODIFIERS", "@im=fcitx5")
	os.Setenv("GTK_IM_MODULE", "fcitx5")
	os.Setenv("QT_IM_MODULE", "fcitx5")
}

// ximServersStale 判断根窗口 XIM_SERVERS 是否残留无效服务器项。
// 正常情况下该属性应只含当前运行的 XIM 服务器(如 @server=fcitx5)。
// 含 @server=none/@server=ibus 等异常项即视为残留, 需清理。
func ximServersStale() bool {
	out, err := exec.Command("xprop", "-root", "XIM_SERVERS").CombinedOutput()
	if err != nil || len(out) == 0 {
		return false
	}
	line := strings.TrimSpace(string(out))
	if strings.Contains(line, "not found") {
		return false
	}
	// 只允许纯 fcitx 服务, 其它项(@server=none、@server=ibus 等)都是残留。
	fields := strings.Fields(line)
	for _, f := range fields {
		f = strings.Trim(f, ",")
		if f != "" && f != "@server=fcitx5" && f != "@server=fcitx" && !strings.HasPrefix(f, "XIM_SERVERS") {
			return true
		}
	}
	return false
}

// clearXIMServers 删除根窗口的 XIM_SERVERS 属性。
func clearXIMServers() error {
	if _, err := exec.LookPath("xprop"); err != nil {
		return fmt.Errorf("xprop not found: %w", err)
	}
	return exec.Command("xprop", "-root", "-remove", "XIM_SERVERS").Run()
}

// findFcitx5 返回 fcitx5 可执行文件路径; 不存在时返回空串。
func findFcitx5() string {
	if p, err := exec.LookPath("fcitx5"); err == nil {
		return p
	}
	for _, cand := range []string{
		"/usr/bin/fcitx5",
		"/usr/local/bin/fcitx5",
		"/usr/libexec/fcitx5",
	} {
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	return ""
}

// fcitxXIMBroken 判断已运行的 fcitx5 是否带异常环境变量 XMODIFIERS=@im=none。
// 没有 fcitx5 进程时返回 false(无需修复)。
func fcitxXIMBroken() bool {
	pids, err := fcitxPIDs()
	if err != nil || len(pids) == 0 {
		return false
	}
	// 任一 fcitx5 进程环境正常即视为 OK, 只有全部异常才需要修复。
	for _, pid := range pids {
		if envHasXMODIFIERS(pid) {
			return false
		}
	}
	return true
}

// fcitxPIDs 返回所有 fcitx5 进程的 PID。
func fcitxPIDs() ([]string, error) {
	procs, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var pids []string
	for _, proc := range procs {
		if !proc.IsDir() {
			continue
		}
		pid := proc.Name()
		if !allDigits(pid) {
			continue
		}
		comm, err := os.ReadFile(filepath.Join("/proc", pid, "comm"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(comm)) == fcitxProcName {
			pids = append(pids, pid)
		}
	}
	return pids, nil
}

// envHasXMODIFIERS 返回指定进程的环境变量中是否含 XMODIFIERS=@im=fcitx5。
func envHasXMODIFIERS(pid string) bool {
	data, err := os.ReadFile(filepath.Join("/proc", pid, "environ"))
	if err != nil {
		return false
	}
	for _, kv := range strings.Split(string(data), "\x00") {
		if kv == "XMODIFIERS=@im=fcitx5" || kv == "XMODIFIERS=@im=fcitx" {
			return true
		}
	}
	return false
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// requestFcitxExit 通过 session dbus 请求 fcitx5 退出。
func requestFcitxExit() error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("connect session bus: %w", err)
	}
	defer conn.Close()
	obj := conn.Object("org.fcitx.Fcitx5", dbus.ObjectPath("/controller"))
	call := obj.Call("org.fcitx.Fcitx.Controller1.Exit", 0)
	return call.Err
}

// startFcitx5 以正确环境变量启动 fcitx5(分离进程)。
func startFcitx5(bin string) error {
	display := os.Getenv("DISPLAY")
	if display == "" {
		display = ":0"
	}
	cmd := exec.Command(bin, "-d")
	cmd.Env = append(os.Environ(),
		"DISPLAY="+display,
		"XMODIFIERS=@im=fcitx5",
		"GTK_IM_MODULE=fcitx5",
		"QT_IM_MODULE=fcitx5",
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
