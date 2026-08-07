#!/usr/bin/env bash
# 安装 hylbscode:
#   默认构建并安装到 ~/.local/bin(免 sudo,用户级)
#   --system  安装到 /usr/local/bin(需要 sudo)
#   --uninstall 卸载
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY="hylbscode"
VERSION_LDFLAG="-X hylbscode/internal/version.Version="

mode="user"
action="install"
while [ "$#" -gt 0 ]; do
  case "$1" in
    --system) mode="system"; shift 1;;
    --uninstall) action="uninstall"; shift 1;;
    *) echo "Unknown parameter: $1"; echo "Usage: $0 [--system] [--uninstall]"; exit 1;;
  esac
done

if [ "$mode" = "system" ]; then
  INSTALL_DIR="/usr/local/bin"
else
  INSTALL_DIR="$HOME/.local/bin"
fi

install_binary() {
  echo ">> Building $BINARY ..."
  VERSION="$VERSION_LDFLAG$(git -C "$PROJECT_DIR" describe --tags --always 2>/dev/null || echo dev)"
  (cd "$PROJECT_DIR" && CGO_ENABLED=1 go build -ldflags "-s -w $VERSION" -o /tmp/hylbscode-install .)

  echo ">> Installing to $INSTALL_DIR ..."
  mkdir -p "$INSTALL_DIR"
  if [ -w "$INSTALL_DIR" ]; then
    install -m 755 /tmp/hylbscode-install "$INSTALL_DIR/$BINARY"
  else
    sudo install -m 755 /tmp/hylbscode-install "$INSTALL_DIR/$BINARY"
  fi
  rm -f /tmp/hylbscode-install
  echo ">> Installed: $INSTALL_DIR/$BINARY"
}

uninstall_binary() {
  if [ -f "$INSTALL_DIR/$BINARY" ]; then
    echo ">> Removing $INSTALL_DIR/$BINARY ..."
    if [ -w "$INSTALL_DIR/$BINARY" ]; then
      rm -f "$INSTALL_DIR/$BINARY"
    else
      sudo rm -f "$INSTALL_DIR/$BINARY"
    fi
    echo ">> Uninstalled."
  else
    echo ">> Not installed: $INSTALL_DIR/$BINARY"
    exit 1
  fi
}

case "$action" in
  install) install_binary;;
  uninstall) uninstall_binary;;
esac

if [ "$action" = "install" ] && ! echo ":$PATH:" | grep -q ":$INSTALL_DIR:"; then
  echo ">> NOTE: $INSTALL_DIR is not in your PATH. Add it with:"
  echo "   echo 'export PATH=\"\$PATH:$INSTALL_DIR\"' >> ~/.bashrc && source ~/.bashrc"
fi
