package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/app"
	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/db"
	"github.com/cindyhuang123/hylbscode/internal/gui"
	"github.com/cindyhuang123/hylbscode/internal/logging"
	"github.com/cindyhuang123/hylbscode/internal/skills"
	"github.com/gofrs/flock"
)

func fatal(format string, args ...any) {
	logging.ErrorPersist(format, args...)
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

// alertAndExit shows a modal dialog explaining why the app cannot start (a
// second instance in the same working directory) and exits when dismissed.
func alertAndExit(message string) {
	a := fyneapp.NewWithID("com.hylbscode.single-instance-alert")
	w := a.NewWindow("hylbscode")
	w.Resize(fyne.NewSize(420, 160))
	d := dialog.NewInformation("hylbscode", message, w)
	d.SetOnClosed(func() {
		a.Quit()
	})
	w.SetContent(container.NewCenter(widget.NewLabel(message)))
	d.Show()
	w.ShowAndRun()
	os.Exit(1)
}

func main() {
	defer logging.RecoverPanic("main", func() {
		logging.ErrorPersist("Application terminated due to unhandled panic")
	})

	debug := flag.Bool("d", false, "enable debug logging")
	flag.Parse()

	workingDir, err := os.Getwd()
	if err != nil {
		fatal("failed to get working directory: %v", err)
	}
	cfg, err := config.Load(workingDir, *debug)
	if err != nil {
		fatal("failed to load configuration: %v", err)
	}
	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	// 日志同时写入 {data.directory}/hylbscode.log，方便离线排查 GUI 交互问题。
	if err := os.MkdirAll(cfg.Data.Directory, 0o755); err != nil {
		fatal("failed to create data directory for logs: %v", err)
	}
	// 同一工作目录只允许一个实例: 文件锁绑定工作目录, 进程退出时内核
	// 自动释放, 无残留; 第二个实例检测到占用直接退出, 避免多窗口
	// 并发读写同一数据库。
	singleLock := flock.New(filepath.Join(cfg.Data.Directory, "single_instance.lock"))
	held, err := singleLock.TryLock()
	if err != nil {
		logging.WarnPersist(fmt.Sprintf("single-instance lock unavailable: %v", err))
	} else if held {
		defer singleLock.Unlock()
	} else {
		alertAndExit("该工作目录下已有 hylbscode 窗口在运行，请勿重复打开。")
	}
	logFile, err := os.OpenFile(filepath.Join(cfg.Data.Directory, "hylbscode.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		fatal("failed to open log file: %v", err)
	}
	defer logFile.Close()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.MultiWriter(os.Stderr, logFile), &slog.HandlerOptions{Level: level})))
	if !config.HasProviderCredentials() {
		fmt.Fprintln(os.Stderr, "warning: no LLM provider credentials detected in environment variables.")
		fmt.Fprintln(os.Stderr, "  (providers configured in the config file are still honored)")
	}

	// 内置技能库按需加载：工作目录还没有 skills/ 时，用内置模板自动生成一份；
	// 已存在（用户自定义）则原样保留。
	if err := skills.EnsureDefault(filepath.Join(cfg.WorkingDir, "skills")); err != nil {
		logging.WarnPersist(fmt.Sprintf("failed to ensure default skills directory: %v", err))
	}

	conn, err := db.Connect()
	if err != nil {
		fatal("failed to connect to database: %v", err)
	}
	defer conn.Close()

	// 监听 SIGINT/SIGTERM:后台进程被 kill 时也能优雅退出,
	// 不依赖 GUI 窗口(窗口可能被外部销毁而不触发 quit 流程)。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	core, err := app.New(ctx, conn)
	if err != nil {
		fatal("failed to initialize application: %v", err)
	}

	a := fyneapp.NewWithID("com.hylbscode.desktop")
	g := gui.NewMainWindow(a, core, ctx)
	cancel := gui.SetupSubscriptions(g, ctx)

	// 窗口一旦销毁(正常关闭或被外部销毁)就必须退出进程,
	// 否则 a.Run() 一直阻塞,进程无窗口挂在后台。
	var quitOnce sync.Once
	quit := func() {
		quitOnce.Do(func() {
			cancel()
			a.Quit()
		})
	}
	g.Window().SetOnClosed(quit)

	// 后台进程被 kill(SIGINT/SIGTERM)时同样优雅退出。
	go func() {
		<-ctx.Done()
		logging.Info("termination signal received, quitting")
		quit()
	}()

	g.Show()
	if !config.HasProviderCredentials() {
		g.ShowProviderSetup()
	}
	a.Run()

	// 事件循环退出后清理核心资源(LSP/MCP 子进程、watcher goroutine)。
	core.Shutdown()
}
