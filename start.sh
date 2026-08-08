#!/usr/bin/env bash
# 中文输入法(XIM)判断式修复 + 启动 hylbscode
set -euo pipefail

cd "$(dirname "$0")"

LOG_FILE="hylbscode-start.log"

# ---- 中文输入法(XIM)判断式修复 ----
# 判断标准: 根窗口 XIM_SERVERS 属性(XIM 服务的真实注册状态)。
#   - 属性干净(只含 @server=fcitx5) -> fcitx5 XIM 已注册, 不做任何干预
#   - 属性含残留无效项(@server=none/@server=ibus 等) -> 清理并重启 fcitx5
#   - 属性不存在且 fcitx5 在运行 -> XIM 服务未注册, 重启 fcitx5
#   - fcitx5 未运行 -> 以正确环境变量启动
# 注意: 不依据 fcitx5 进程的 XMODIFIERS 环境变量判断(KDE 会话启动的
# fcitx5 可能不带该变量, 但 XIM 服务已正常注册)。

# start_fcitx5 以正确环境变量启动 fcitx5(XIM 服务端)。
start_fcitx5() {
    dbus-send --session --dest=org.fcitx.Fcitx5 --type=method_call /controller org.fcitx.Fcitx.Controller1.Exit >/dev/null 2>&1 || true
    sleep 1
    nohup env DISPLAY="${DISPLAY:-:0}" XMODIFIERS=@im=fcitx5 \
        GTK_IM_MODULE=fcitx5 QT_IM_MODULE=fcitx5 \
        /usr/bin/fcitx5 -d >/dev/null 2>&1 &
    sleep 2
}

# root_xim_servers 输出根窗口 XIM_SERVERS 属性内容; 不存在时输出空。
root_xim_servers() {
    if ! command -v xprop >/dev/null 2>&1; then
        echo ""
        return
    fi
    xprop -root XIM_SERVERS 2>/dev/null || true
}

# xim_servers_clean 判断 XIM_SERVERS 是否干净(只含 fcitx 服务)。
xim_servers_clean() {
    local line f found=0
    line="$(root_xim_servers)"
    [[ -z "$line" ]] && return 1
    [[ "$line" == *"not found"* ]] && return 1
    # 只检查 @server=xxx 服务项, 跳过 "=" "XIM_SERVERS(ATOM)" 等非服务词
    for f in $(echo "$line" | tr ',' ' '); do
        f="${f//\"}"
        [[ "$f" == @server=* ]] || continue
        found=1
        if [[ "$f" != "@server=fcitx5" && "$f" != "@server=fcitx" ]]; then
            return 1
        fi
    done
    # 有 fcitx 服务项即视为干净, 无任何 @server 项视为未注册
    [[ "$found" == 1 ]]
}

if command -v fcitx5 >/dev/null 2>&1; then
    if pgrep -x fcitx5 >/dev/null 2>&1; then
        # fcitx5 在运行: 检查 XIM 服务是否真实注册
        if ! xim_servers_clean; then
            line="$(root_xim_servers)"
            if [[ -z "$line" || "$line" == *"not found"* ]]; then
                echo "$(date '+%F %T') fcitx5 运行中但 XIM 服务未注册, 重启 fcitx5" >>"$LOG_FILE"
            else
                echo "$(date '+%F %T') 检测到残留 XIM_SERVERS: $line, 清理并重启 fcitx5" >>"$LOG_FILE"
                xprop -root -remove XIM_SERVERS 2>/dev/null || true
            fi
            start_fcitx5
        fi
    else
        # fcitx5 未运行: 直接以正确环境变量启动
        echo "$(date '+%F %T') fcitx5 未运行, 启动 fcitx5" >>"$LOG_FILE"
        start_fcitx5
    fi
fi

# 客户端侧输入法环境变量(GLXW X11 后端通过 XMODIFIERS 连接 XIM)
export XMODIFIERS=@im=fcitx5
export GTK_IM_MODULE=fcitx5
export QT_IM_MODULE=fcitx5

# 程序自带日志(hylbscode.log), 这里仅把启动期错误追加到 start 日志
nohup ./hylbscode >>"$LOG_FILE" 2>&1 &
