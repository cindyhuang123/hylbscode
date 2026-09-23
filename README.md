# HyLbsCode

> 桌面 AI 编程助手 —— 基于 Fyne 的跨平台 GUI 应用，由 Go 编写，参考 golang opencode 改造。

HyLbsCode 是一个本地优先的 AI 编程助手桌面应用：三栏布局（会话 / 聊天 / Todo），流式渲染，内置工具调用、LSP 诊断与 MCP 支持，数据全部存储在本地 SQLite。

## 功能特性

- **多模型支持** — Anthropic / OpenAI / Gemini / Copilot / Bedrock / Azure / VertexAI / Groq / OpenRouter / XAI / 本地端点
- **DeepSeek 思考模式** — coder/task 默认开启 thinking（`reasoning_effort` 可配推理力度），工具轮中途自动降级防 400；未配置时静默回退，无启动噪音警告
- **流式响应** — Markdown、思考过程、工具调用块实时渲染（80ms 合并节流）
- **AI 工具调用** — bash / edit / write / glob / grep / ls / view / fetch / patch / sourcegraph / diagnostics，破坏性操作需弹窗批准
- **MCP 支持** — 从配置文件动态加载 MCP 服务器（stdio / sse）
- **LSP 集成** — 诊断、代码导航、文件监听
- **本地存储** — SQLite，会话支持父子层级与全文搜索
- **文件版本历史** — 基于数据库按会话管理
- **技能库（Skills）** — 内置多类技能，技能目录常驻、`read_skill` 按需加载完整工作流
- **中英双语界面** — 菜单切换语言，即时生效
- **窗口默认最大化** — 启动即系统级最大化（未配置自定义尺寸时）
- **退出确认** — 文件菜单可勾选关闭窗口是否弹确认
- **消息一键复制** — 对话/回复可整段复制到剪贴板
- **输入框快捷键** — 外层 ChatInput 实现 `fyne.Shortcutable`，把 `Ctrl+V/C/X/A` 委托给内层 `widget.Entry`（其内嵌自身注册的 paste/copy/cut/selectAll），因此焦点常驻外层时快捷键依然生效，不再只剩右键菜单
- **防乱码** — 自定义字体加载前校验必需字形（CJK/箭头/勾叉），缺失自动回退系统字体

## 下载安装

### 从源码构建

```bash
# 国内环境优先使用代理加速依赖下载
go env -w GOPROXY=https://goproxy.cn,direct

go mod download        # 安装 Go 依赖
make build             # 构建（需要 CGO：gcc + X11/Wayland 开发库）
./hylbscode            # 运行（建议通过 ./start.sh 启动以修复中文输入法）
```

需要 Go 1.26+。Linux 构建依赖：`gcc`、`libgl1-mesa-dev`、`xorg-dev`。

> 发布产物统一放在 [`git_release/`](git_release/) 目录，每个版本一个子目录（当前 0.1.0 尚未产出归档包）。

## 快速开始

1. 配置模型凭据：编辑 `~/.hylbscode.json`（见下文配置），或复制 [`sample_config.json`](sample_config.json)
2. 运行 `./start.sh`（自动处理中文输入法 XIM）或 `./hylbscode`
3. 在输入框输入问题，按 `Enter` 发送，`Shift+Enter` 换行

## 配置

配置文件为 `.hylbscode.json`，查找顺序（`viper`，第一个命中的生效）：

1. `$HOME/.hylbscode.json`（主目录，**推荐只维护这一份**）
2. `$XDG_CONFIG_HOME/hylbscode/.hylbscode.json`
3. `$HOME/.config/hylbscode/.hylbscode.json`

常用字段：

| 字段 | 说明 |
|---|---|
| `providers` / `agents` | 模型服务商（apiKey/baseURL）与各智能体模型 |
| `agents.<name>.thinking` | DeepSeek 思考开关：`enabled` / `disabled`；默认 coder/task 开启，summarizer/title 关闭（title 仅 80 tokens，思考会挤占输出） |
| `agents.<name>.reasoningEffort` | 推理力度：`low` / `medium` / `high`，仅 thinking 开启时生效（DeepSeek 将 medium 映射为 high）；未配置时静默用 medium |
| `autoCompact` | 接近上下文窗口时自动摘要（默认 true） |
| `shell` | bash 工具的 shell 路径/参数 |
| `mcpServers` | MCP 服务器定义（stdio 或 sse） |
| `contextPaths` | 注入到提示词的项目说明/指令文件列表；缺省时用内置默认列表（`CLAUDE.md`、`hylbscode.md` 等约定文件名） |
| `lsp` | LSP 客户端配置（按语言键名，如 `gopls`） |
| `gui.theme` | `auto` / `light` / `dark` |
| `gui.width` / `gui.height` | 显式指定窗口尺寸；留空则启动时系统级最大化 |
| `gui.confirmQuit` | 关闭窗口是否弹确认（`true`/`false`，默认确认，可在文件菜单切换） |
| `gui.unrestricted` | 完全放开所有操作权限（`true`/`false`，默认需确认，可在文件菜单切换）：bash 的路径/脚本/危险命令确认、banned 命令（curl/wget 等）及 view/edit/grep 等所有工具的权限弹窗全部跳过，**高风险** |
| `gui.font` | ttf/otf 字体文件路径；加载前校验必需字形，缺失则回退系统字体 |
| `extraModels` | 追加模型列表 |

完整结构见 [`internal/config/config.go`](internal/config/config.go)。

## 技能（Skills）

`skills/<类别>/Skill.md` 是技能定义（**工作目录**下的 `skills/`），每条包含 YAML frontmatter（`name` + `description` 触发词）与中文正文（流程 / 模板 / 禁止项）。在**没有** `skills/` 的新目录打开 hylbscode 时，会自动用内置模板（`internal/skills/data/`，随二进制嵌入）生成完整技能库；已存在则原样保留，用户可自由增删类别。技能采用**按需加载**：启动时只把技能目录（名称 + 一句话说明 + 触发词）注入模型上下文；当用户请求命中某技能触发词时，模型调用 `read_skill` 工具加载对应完整指令再执行，避免一次性塞满上下文。

内置类别：

| 类别 | 定位 |
|---|---|
| `coding` | 编码工作流：先探后动、改完必测、提交规范 |
| `planning` | 计划/任务拆解/里程碑，SMART 目标 |
| `reporting` | 周报月报/总结/方案等正式报告 |
| `research` | 调查/选型/根因，多源交叉验证 |
| `documentation` | README/API/架构文档，读者导向 |
| `presentation` | 汇报演讲，结论先行 + 故事线 |
| `meeting-notes` | 会议纪要，以行动项为核心 |
| `knowledge` | 笔记/费曼/复盘 |
| `decision` | 决策分析，加权打分 + 决策日志 |
| `learning` | 快速上手新领域，最小闭环 |
| `exploration` | 无经验/无参考的未知领域探索 |
| `terminal-sense` | 读日志/报错/堆栈，翻译成排查线索 |
| `ci-fixer` | CI 失败先复现、根因、最小修复 |
| `data-cleaner` | 批量清洗 CSV/Excel/JSON（python3 脚本化） |
| `changelog-miner` | 从 git 提交挖关键改动，标 breaking 风险 |
| `dependency-guard` | 升级依赖前评估破坏面 |
| `release-notes` | 从 diff 提炼用户视角的发布说明 |
| `playwright-scout` | Playwright DOM 文本断言版 E2E（AI 无法看截图） |

新增一个类别只需在 `skills/` 下新建目录并放一个 `Skill.md`（frontmatter 带 `name` 与 `description`），启动扫描后会自动出现在技能目录中，无需改配置。

## 开发

技术栈：Go 1.26+、Fyne GUI、SQLite（go-sqlite3）、LSP 客户端、pubsub 事件驱动。

```bash
go test ./internal/...   # 全部测试（GUI 用 -tags x11）
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
  skills/              # 技能库内置模板（go:embed 嵌入二进制）+ 扫描/按需加载逻辑
  todo/                # Todo CRUD
  version/             # 版本号（构建时注入）
```

`skills/`（工作目录下）是运行时技能库：在**没有** `skills/` 的新目录打开 hylbscode 时会自动用内置模板生成一份；已存在则原样保留（用户可自行增删、扩展类别）。

服务间通过 pubsub 事件桥接：service → pubsub broker → channel → bridge goroutine → `fyne.Do()` → widget 更新。

## 开源协议

MIT
