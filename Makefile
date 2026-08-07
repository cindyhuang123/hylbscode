# HyLbsCode 构建 Makefile
# 根据目标系统自动选择编译参数（Fyne 官方做法）：
#   - Windows: 加 -H=windowsgui 隐藏控制台窗口（Fyne 的 fyne package 内部就是这样做的）
#   - Linux: 默认走 X11(XWayland) 后端，中文输入法(fcitx/ibus)才能工作；
#     GLFW 的 Wayland 后端未实现输入法协议，原生 Wayland 下无法输入中文。
#     需要原生 Wayland 时用: make build WAYLAND=1
#
# 常用命令：
#   make deps         # 安装编译依赖（自动识别 Fedora / Ubuntu）
#   make build        # 构建当前系统可执行文件 (hylbscode / hylbscode.exe)
#   make run          # 直接运行
#   make run-debug    # 开启调试日志运行
#   make test         # 运行全部测试
#   make schema       # 重新生成 hylbscode-schema.json
#   make clean        # 删除构建产物

GO     ?= go
GOOS   ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)
BINARY  = hylbscode

ifeq ($(GOOS),windows)
BINARY  := $(BINARY).exe
LDFLAGS := -H=windowsgui
endif

# ---- Linux 发行版相关 ----
# 自动识别发行版，用于选择依赖包和编译参数。
ifeq ($(GOOS),linux)
DISTRO := $(shell . /etc/os-release 2>/dev/null && echo $$ID)

# 默认使用 X11 后端以支持中文输入法；设置 WAYLAND=1 走原生 Wayland。
ifneq ($(WAYLAND),1)
BUILD_TAGS := x11
endif

# 编译依赖（pkg-config 库 + 系统头文件）
FEDORA_DEPS = libX11-devel libXrandr-devel libXinerama-devel \
              libXcursor-devel libXi-devel libXxf86vm-devel \
              libxkbcommon-devel mesa-libGL-devel
UBUNTU_DEPS = libx11-dev libxrandr-dev libxinerama-dev \
              libxcursor-dev libxi-dev libxxf86vm-dev \
              libxkbcommon-dev libgl1-mesa-dev
endif

.PHONY: build run run-debug test schema clean deps

# 按发行版安装编译依赖（需要 root）。
deps:
	@if [ "$(DISTRO)" = "fedora" ]; then \
		sudo dnf install -y $(FEDORA_DEPS); \
	elif [ "$(DISTRO)" = "ubuntu" ] || [ "$(DISTRO)" = "debian" ]; then \
		sudo apt-get install -y $(UBUNTU_DEPS); \
	else \
		echo "未识别的发行版: $(DISTRO)，请手动安装以下包:"; \
		echo "  Fedora: $(FEDORA_DEPS)"; \
		echo "  Ubuntu: $(UBUNTU_DEPS)"; \
		exit 1; \
	fi

build:
	CGO_ENABLED=1 $(GO) build $(if $(BUILD_TAGS),-tags '$(BUILD_TAGS)') $(if $(LDFLAGS),-ldflags '$(LDFLAGS)') -o $(BINARY) .

run:
	$(GO) run $(if $(BUILD_TAGS),-tags '$(BUILD_TAGS)') .

run-debug:
	$(GO) run $(if $(BUILD_TAGS),-tags '$(BUILD_TAGS)') . -d

test:
	$(GO) test $(if $(BUILD_TAGS),-tags '$(BUILD_TAGS)') ./internal/...

schema:
	$(GO) run ./cmd/schema

clean:
	rm -f $(BINARY)
