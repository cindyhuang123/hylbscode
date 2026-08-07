# HyLbsCode 构建 Makefile
# 根据目标系统自动选择编译参数（Fyne 官方做法）：
#   - Windows: 加 -H=windowsgui 隐藏控制台窗口（Fyne 的 fyne package 内部就是这样做的）
#   - Linux/macOS: 普通桌面构建，无需额外参数
#
# 常用命令：
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

.PHONY: build run run-debug test schema clean

build:
	CGO_ENABLED=1 $(GO) build $(if $(LDFLAGS),-ldflags '$(LDFLAGS)') -o $(BINARY) .

run:
	$(GO) run .

run-debug:
	$(GO) run . -d

test:
	$(GO) test ./internal/...

schema:
	$(GO) run ./cmd/schema

clean:
	rm -f $(BINARY)
