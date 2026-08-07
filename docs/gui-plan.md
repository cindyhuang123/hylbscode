# HyLbsCode GUI 桌面版方案（Fyne）

> 状态：已确认定稿（v3）
> 决策：本分支仅保留 GUI（TUI 已独立拉分支）；按里程碑全做（M0→M3）；方案存于本文件。

## 1. 背景与目标

hylbscode 目前是 TUI 应用（Bubble Tea）。目标：在**当前分支**上改造为纯 GUI 桌面版，
复用现有核心层（`internal/` 下与 UI 无关的包），UI 层改用 **Fyne v2**。

- 本分支只保留一份 GUI 二进制；TUI 代码由已分出的分支保留，本分支移除。
- 主题使用 **Fyne 原生主题系统**，不做 lipgloss 主题移植。

## 2. 架构总览

```
main.go (GUI 入口: 启动装配 + 窗口)
  └─ internal/gui/          ← 新增，纯 Fyne 前端
       ├─ bridge.go         pubsub → fyne.Do 桥接（7 个订阅）
       ├─ layout.go         主布局: 左会话列表 | 中聊天区
       ├─ chat/             消息列表/单条消息/输入区
       ├─ sidebar/          会话/todo/文件面板
       ├─ dialog/           权限确认/设置
       └─ theme.go          品牌色 fyne.Theme（可选）
  └─ internal/core          ← 复用，零 UI 依赖
       app / config / db / llm / message / session / permission
       / history / pubsub / todo / logging / lsp / format / fileutil
```

## 3. 本分支清理清单（M0 前置，移除 TUI）

| 项 | 现状 | 处理 |
|---|---|---|
| `internal/tui/` | 全部 TUI（theme/components/page） | 整个移除（独立分支已保留） |
| `internal/app/app.go` | `initTheme()`（:64-65,:93-105）调用 `theme.SetTheme(cfg.TUI.Theme)` | 移除 initTheme 与 theme 导入（:24）；app 构造不再读 `cfg.TUI` |
| `internal/diff/diff.go` | 深依赖 `theme.CurrentTheme()` 调色板（:327,:553,:619,:682,:742） | 将 `theme.Theme` 类型与调色板收编为中性包（建议 `internal/palette` 或内联进 diff），移除 `internal/tui/theme` 依赖；输出保持 ANSI 着色文本，由 GUI 渲染层处理 |
| `internal/completions/` | `history.go`/`files-folders.go` 依赖 `internal/tui/components/dialog` | 随 CLI 移除；如保留则去掉 dialog 依赖 |
| `cmd/root.go` | cobra 交互入口 + TUI 装配（`setupSubscriptions`） | 由 GUI 启动替代；`cmd/schema` 保留（schema 生成工具，低成本） |
| `main.go` | `cmd.Execute()` | 改为 GUI bootstrap（保留 `logging.RecoverPanic` 包装） |
| `.goreleaser.yml` | `CGO_ENABLED=0` 构建 CLI | 改造为 GUI 构建（CGO=1，见 §8） |
| `config.TUI` | 仅 app.go 使用 | 本分支不再引用；结构字段保留以兼容旧配置，GUI 新增 `config.GUI` 段（theme/窗口尺寸） |

## 4. 原样复用清单（零改动）

`config`、`db`、`app.App`（含 LSP 生命周期，去掉 initTheme 后）、`llm/*`（provider/agent/tools/prompt/models）、
`message`、`session`、`permission`、`history`、`pubsub`、`todo`、`logging`、`format`、`fileutil`、`version`。

## 5. GUI 复刻的 7 个订阅

复用 TUI 分支的 `setupSubscriber[T]` 泛型模式，输出端由 `program.Send` 换成 **`fyne.Do()`**：

| Broker 事件 | 消费方 |
|---|---|
| `agent.AgentEvent` | 聊天区流式增量（工具调用/内容块） |
| `message.Message` | 聊天区消息列表（创建/更新/删除） |
| `session.Session` | 会话列表 + 状态栏 |
| `permission.PermissionRequest` | 权限确认弹窗 |
| `todo.Todo` | todo 面板 |
| `logging.LogMessage` | 日志面板 |
| `history.File` | 文件历史面板 |

桥接要点：后台 goroutine 收事件 → `fyne.Do(func(){ 更新 widget + Refresh() })`；
聊天流式更新按 **80ms 合并** 提交（吸取 4f4e2e8 防抖 bug 教训：合并命令必须真正执行 + 回归测试）。

## 6. 技术选型

| 项 | 选择 | 说明 |
|---|---|---|
| Fyne | **v2.6+（最新稳定 v2.x）** | `fyne.Do()` 自 2.6.0 起可用，后台 goroutine 更新 UI 必需 |
| 消息渲染 | `widget.List` + 每条消息独立自定义 widget（`ExtendBaseWidget` + `CreateRenderer`） | 增量刷新，避免全量重绘 |
| Markdown | `widget.NewRichTextFromMarkdown` / `AppendMarkdown` | 原生内置，契合流式追加 |
| 输入框 | `widget.Entry` 多行 + Ctrl+Enter 发送 / Shift+Enter 换行 | 标准做法 |
| 权限弹窗 | `dialog.NewCustom` 模态 | 对接 `permission.Service`（Grant/Deny/GrantPersistant） |
| UI 测试 | `fyne.io/fyne/v2/test` | `test.NewApp`、widget 断言 |
| 自动滚底 | 消息列表滚动到末尾（流式期间跟随到底部） | — |

## 7. 主题方案（Fyne 原生，不碰 lipgloss）

- **基础**：`theme.DefaultTheme()`（自动跟随系统深浅色）起步，零成本。
- **切换**：`a.Settings().SetTheme(theme)` 运行时切换，支持内置 `DarkTheme()` / `LightTheme()`；配置项 `config.GUI.Theme = auto/light/dark`。
- **品牌色（可选，M2 收尾）**：实现 `fyne.Theme` 接口——`Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color`，`Font/Icon/Size` 委托 `theme.DefaultTheme()`。
- `internal/gui/theme.go` 与现有 core 零耦合；**不导入 `internal/tui/theme`，不依赖 lipgloss**。

## 8. 目录结构

```
cmd/gui/main.go          # GUI 入口（若 main.go 直启则可省，默认用 main.go）
internal/gui/
  app.go                 # fyne.App 装配：窗口、布局、订阅桥接、关闭清理（停止订阅/关闭 db/LSP）
  bridge.go              # pubsub → fyne.Do 桥接（复用 setupSubscriber 泛型模式）
  layout.go              # 主布局：左会话列表 | 中聊天区
  chat/message_item.go   # 单条消息 widget（markdown、工具调用块、流式增量）
  chat/message_list.go   # widget.List 封装 + 80ms 合并刷新 + 自动滚底
  chat/input.go          # 输入区（Ctrl+Enter 发送、Shift+Enter 换行）
  sidebar/sessions.go    # 会话列表（新建/切换/删除）
  sidebar/todos.go       # todo 面板
  sidebar/files.go       # 文件历史面板
  dialog/permission.go   # 权限确认（含"本会话记住"→ GrantPersistant）
  dialog/settings.go     # 模型选择、主题切换、API key 配置
  logs.go                # 日志面板（logging.Subscribe）
  theme.go               # 品牌色 fyne.Theme 实现（可选，委托内置主题）
```

## 9. 里程碑（按里程碑全做）

- **M0 骨架 + 清理**：移除 TUI/CLI 交互入口与核心→TUI 耦合（§3）；`main.go` 启动 GUI；核心装配
  （config→db→app.New→MCP 初始化）组织进 GUI 侧；三栏布局；7 订阅桥接；会话列表 + 发消息 +
  纯文本流式显示 + 自动滚底。**验收：能完整对话一轮（含流式输出）。**
- **M1 对等渲染**：Markdown 渲染、工具调用块（bash 实时输出）、权限弹窗、ANSI 处理评估
  （剥离 vs 解析为 RichText 色段）、`config.GUI` 生效。**验收：与 TUI 功能等价。**
- **M2 完整功能**：会话切换/新建/删除、todo 面板、日志面板、主题切换（Fyne 原生）、模型切换、
  附件、设置页（API key）、品牌色主题（可选）。**验收：日常使用无缺口。**
- **M3 交付**：打包（§10）、测试覆盖（桥接/渲染/权限流/80ms 合并回归）、长会话性能调优。
  **验收：发布产物可运行。**

## 10. 构建与打包

- GUI 需要 CGO + OpenGL：
  - Linux 构建依赖：`libgl1-mesa-dev`、`xorg-dev`（Debian/Ubuntu）；`CGO_ENABLED=1`
  - 跨平台：`fyne-cross`（linux/macOS/Windows 交叉产物）
  - macOS：原生 darwin arm64/amd64
- `.goreleaser.yml` 改造：builds 段 `CGO_ENABLED=1` 或改用 fyne-cross 产物；保留
  `-X hylbscode/internal/version.Version` ldflags。

## 11. 测试策略

- 核心层测试全部保留（llm/message/session/db 等，本分支不受影响）。
- GUI 新增：
  - `bridge_test.go`：事件→主线程调度，含 **80ms 合并刷新回归测试**（防止重蹈 4f4e2e8）
  - `message_item_test.go`：单条消息渲染断言（fyne test）
  - `permission_test.go`：权限弹窗 Grant/Deny/GrantPersistant 流程
- headless CI：`fyne.io/fyne/v2/test` 软件渲染；如需真实窗口冒烟用 xvfb。

## 12. 风险与注意

1. **ANSI 着色**：diff 工具结果含 ANSI 色码，RichText 不识别——M1 定方案（剥离或转色段）。
2. **长会话流式性能**：每消息独立 widget + 80ms 合并刷新，避免全量重建。
3. **Fyne 代码块渲染**：`RichTextFromMarkdown` 对围栏代码块支持需实测；备选自绘代码块段。
4. **Linux IME**：Fyne Entry 中文输入法支持需在目标平台验证。
5. **headless 测试**：GUI 测试依赖 fyne test 包软件渲染，注意 CI 无显示环境。
6. **Fyne 版本**：必须 v2.6+（`fyne.Do`）；升级到最新稳定 v2.x。

## 13. 决策记录（已确认）

| # | 决策 | 结论 |
|---|---|---|
| 1 | 入口形态 | **完全独立，仅一份 GUI**；TUI 已独立拉分支，本分支移除 TUI |
| 2 | 启动装配 | 本分支只保留 GUI 版本，装配逻辑归入 GUI 侧（无 TUI 共存顾虑） |
| 3 | 实施范围 | **按里程碑全做**（M0→M3） |
| 4 | 主题 | **Fyne 原生主题系统**（用户明确：不需要 theme.SetTheme 那套） |
| 5 | 方案位置 | 本文件 `docs/gui-plan.md` |
