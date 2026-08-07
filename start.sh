#!/usr/bin/env bash
# 等价于 ./hylbscode & disown，但 nohup 在脚本环境更稳（& disown 依赖交互式 shell 的 job control）
set -euo pipefail

cd "$(dirname "$0")"

# ---- 中文输入法(XIM)修复 ----

# restart_fcitx5 以正确环境变量重启 fcitx5(XIM 服务端)。
restart_fcitx5() {
    dbus-send --session --dest=org.fcitx.Fcitx5 --type=method_call /controller org.fcitx.Fcitx.Controller1.Exit >/dev/null 2>&1 || true
    sleep 1
    nohup env DISPLAY="${DISPLAY:-:0}" XMODIFIERS=@im=fcitx5 \
        GTK_IM_MODULE=fcitx5 QT_IM_MODULE=fcitx5 \
        /usr/bin/fcitx5 -d >/dev/null 2>&1 &
    sleep 2
}

# KDE Wayland 会话启动 fcitx5 时可能继承 XMODIFIERS=@im=none，导致其 XIM
# 服务未注册到 X 根窗口；根窗口的 XIM_SERVERS 属性也可能残留无效项
# (@server=none/@server=ibus) 干扰 XIM 客户端。这里检测并修复。
if command -v fcitx5 >/dev/null 2>&1; then
    FCITX_PID=$(pgrep -x fcitx5 2>/dev/null | head -1 || true)
    if [ -n "$FCITX_PID" ]; then
        # 检查 fcitx5 进程的 XMODIFIERS 是否正常
        if ! grep -q "XMODIFIERS=@im=fcitx5" "/proc/$FCITX_PID/environ" 2>/dev/null; then
            echo "fcitx5 XIM 服务未正常注册，重启 fcitx5..."
            restart_fcitx5
        fi
    else
        # fcitx5 未运行，直接启动
        restart_fcitx5
    fi
fi

# 清理根窗口残留的 XIM_SERVERS(仅保留 fcitx5 注册项)
if command -v xprop >/dev/null 2>&1; then
    XIM_SERVERS=$(xprop -root XIM_SERVERS 2>/dev/null || true)
    if [ -n "$XIM_SERVERS" ] && printf '%s' "$XIM_SERVERS" | grep -qv "@server=fcitx5"; then
        echo "清理残留的 XIM_SERVERS..."
        xprop -root -remove XIM_SERVERS 2>/dev/null || true
        # 重新注册 XIM 服务
        restart_fcitx5
    fi
fi

# 程序自身使用 X11 后端时，确保环境变量正确（配合 -tags x11 构建）
export XMODIFIERS=@im=fcitx5
export GTK_IM_MODULE=fcitx5
export QT_IM_MODULE=fcitx5

nohup ./hylbscode >/dev/null 2>&1 &
