---
name: dependency-guard
description: >
  升级依赖前评估破坏面：列可升级项、按影响分档、定位受影响调用点时套用。
  触发词: "升级依赖", "依赖检查", "依赖升级", "破坏面", "outdated", "升级前检查", "兼容性"。
---

# 依赖升级守门

**先评估破坏面，再升级；升级写操作等用户确认。**

## Procedure
1. 识别生态：`go.mod`（Go）、`package.json`（Node）、`requirements.txt` / `pyproject.toml`（Python）。
2. 列清单：Go 用 `go list -m -u all`；Node 用 `npm outdated`；Python 用 `pip list --outdated`。
3. 分档：major（可能 breaking）/ minor（向后兼容）/ patch（安全修复），分别列表。
4. 查破坏面：major 项用 Fetch 查 changelog / release notes；用 Grep 找本仓库中该依赖的调用点，标出可能受影响的文件。
5. 输出升级建议顺序（先 patch 后 minor，major 单独列出），**不在用户确认前不执行升级**。

## 输出模板
```markdown
## 依赖升级评估
| 依赖 | 当前 | 最新 | 档位 | 受影响调用点 | 建议 |
### 升级顺序建议
### 需用户确认的 major 项
```

## 禁止
- 不许未经用户确认执行 `go get -u` / `npm install` 等写操作。
- 不许用 wget 等危险命令拉取包信息；确需用时先征得用户确认。
