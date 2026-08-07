# HyLbsCode

> 桌面 AI 编程助手 —— 基于 Fyne 的跨平台 GUI 应用,基于golang opencode改造.

HyLbsCode 是一个本地优先的 AI 编程助手桌面应用：三栏布局（会话 / 聊天 / Todo），流式渲染，内置工具调用、LSP 诊断与 MCP 支持，数据全部存储在本地 SQLite。

## 功能特性

- **多模型支持** — Anthropic / OpenAI / Gemini / Copilot / Bedrock / Azure / VertexAI / Groq / OpenRouter / XAI / 本地端点
- **流式响应** — Markdown、思考过程、工具调用块实时渲染（80ms 合并节流）
- **AI 工具调用** — bash / edit / write / glob / grep / ls / view / fetch / patch / sourcegraph / diagnostics，破坏性操作需弹窗批准
- **MCP 支持** — 从配置文件动态加载 MCP 服务器（stdio / sse）
- **LSP 集成** — 诊断、代码导航、文件监听
- **本地存储** — SQLite，会话支持父子层级与全文搜索
- **文件版本历史** — 基于数据库按会话管理
- **自定义字体** — `gui.font` 配置覆盖内置 UI 字体

## 下载安装

### v1.0.0

| 平台 | 文件 | 校验 |
|---|---|---|
| Linux x86_64 | [`hylbscode-linux-x86_64.tar.gz`](git_release/v1.0.0/hylbscode-linux-x86_64.tar.gz) | [checksums.txt](git_release/v1.0.0/checksums.txt) |
| Windows x86_64 | [`hylbscode-windows-x86_64.zip`](git_release/v1.0.0/hylbscode-windows-x86_64.zip) | [checksums.txt](git_release/v1.0.0/checksums.txt) |

```bash
# Linux
tar -xzf hylbscode-linux-x86_64.tar.gz
./hylbscode
```

```powershell
# Windows —— 解压 zip 后运行 hylbscode.exe
Expand-Archive hylbscode-windows-x86_64.zip
.\hylbscode.exe
```

校验文件完整性：

```bash
sha256sum -c checksums.txt
```

> 所有版本产物统一放在 [`git_release/`](git_release/) 目录，每个版本一个子目录。

### 从源码构建

```bash
go mod download        # 安装 Go 依赖
go build -o hylbscode . # 构建（需要 CGO：gcc + X11/Wayland 开发库）
./hylbscode            # 运行
```

需要 Go 1.26+。Linux 构建依赖：`gcc`、`libgl1-mesa-dev`、`xorg-dev`。

## 快速开始

1. 配置模型凭据（环境变量，或在配置文件中设置，见下文）
2. 运行 `hylbscode`
3. 在输入框输入问题，按 `Enter` 发送，`Shift+Enter` 换行

## 配置

配置文件 `.hylbscode.json`（参考 [`sample_config.json`](sample_config.json)），查找顺序：

1. `$HOME/.hylbscode.json`
2. `$XDG_CONFIG_HOME/hylbscode/.hylbscode.json`
3. `./.hylbscode.json`（当前工作目录）

常用字段：

| 字段 | 说明 |
|---|---|
| `provider` / `model` | 使用的模型提供商与模型 |
| `autoCompact` | 接近上下文窗口时自动摘要（默认 true） |
| `shell` | bash 工具的 shell 路径/参数 |
| `mcpServers` | MCP 服务器定义（stdio 或 sse） |
| `contextPaths` | 注入到提示词的项目说明文件列表 |
| `lsp` | LSP 客户端配置（按语言键名，如 `gopls`） |
| `gui.font` | ttf/otf 字体文件路径，覆盖内置 UI 字体 |

完整的配置结构见 [`internal/config/config.go`](internal/config/config.go)。

## 开发

技术栈：Go 1.26+、Fyne GUI、SQLite（go-sqlite3）、LSP 客户端、pubsub 事件驱动。

```bash
go test ./internal/...   # 全部测试
go run ./cmd/schema      # 重新生成 hylbscode-schema.json
```

目录结构：

```
main.go                # GUI 入口（config → db → app → fyne）
cmd/schema/            # hylbscode-schema.json 生成器
internal/
  app/                 # 应用中枢，组件串联，LSP 初始化/关闭
  config/              # 配置加载 (viper)
  db/                  # SQLite（sqlc 查询 + goose 迁移）
  gui/                 # Fyne GUI（MainWindow、ChatArea、ChatInput、bridge）
  llm/                 # agent 编排 / models / prompt / provider / tools
  lsp/                 # LSP 客户端、协议、工作区监听、诊断缓存
  message/             # 消息模型、CRUD、内容部件
  pubsub/              # 通用发布/订阅代理（泛型，基于 channel）
  session/             # 会话 CRUD（对话，支持父子层级）
  todo/                # Todo CRUD
  version/             # 版本号（构建时注入）
```

服务间通过 pubsub 事件桥接：service → pubsub broker → channel → bridge goroutine → `fyne.Do()` → widget 更新。

## 开源协议

MIT
