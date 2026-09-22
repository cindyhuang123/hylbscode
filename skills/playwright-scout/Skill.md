---
name: playwright-scout
description: >
  用 Playwright 做网页/E2E 自动化检查，基于 DOM 文本断言（本环境 AI 看不到截图）。
  触发词: "playwright", "网页测试", "页面测试", "E2E", "UI 自动化", "web 检查", "表单测试"。
---

# Playwright 侦察（DOM 断言版）

**本环境 View 工具无法读取图片，一切结论必须来自 DOM 文本与命令的文本输出。**

## Procedure
1. 前置：`npx playwright --version` 确认可用；缺失则 `npm i -D @playwright/test && npx playwright install chromium`（装的是 Playwright 自带 chromium）。系统 `chrome` / `safari` 命令仍被环境禁用；`firefox` 已可用但属危险命令，调用前需用户确认。
2. 写用例：在 `tests/` 下建 `.spec.ts`，用 `getByText` / `getByRole` / `getByLabel` 做 DOM 断言，避免视觉断言。
3. 运行：`npx playwright test --reporter=list`，纯文本输出便于解析。
4. 截图仅交付：确需截图时用 `page.screenshot({ path })` 存盘并告知用户路径，**由用户自行查看**，不基于截图下结论。
5. 失败分析：读文本输出中的 expected / actual 与堆栈，定位到测试文件行号。

## 输出模板
```markdown
## Playwright 检查结果
- 用例: <file>
- 命令: npx playwright test --reporter=list
- 结果: pass N / fail M
### 失败明细
| 用例 | 断言 | 期望 | 实际 | 位置 |
```

## 禁止
- 不许调用 chrome / safari 命令（环境已禁）；firefox 可用但需用户确认。
- 不许基于截图做视觉结论（AI 看不到图片内容）；截图只作交付物。
- 不许用 grep 排除失败用例来制造全绿。
