---
name: changelog-miner
description: >
  从 git 提交/差异里挖掘关键改动，分类整理并补齐遗漏风险时套用。
  触发词: "changelog", "变更日志", "关键改动", "遗漏", "提交分析", "版本差异", "git log 分析"。
---

# 变更挖掘

**按影响面分类提交，标出 breaking 与被遗漏的风险点。**

## Procedure
1. 圈定范围：确定 commit 区间或 tag 范围（如 `v1.2.0..HEAD`），范围不明先向用户确认。
2. 拉取提交：用 Git 工具读 `git log --oneline`，关键提交再 `git show` 看 diff。
3. 分类打标：feat / fix / perf / refactor / docs / chore / **breaking-change**（接口、配置、依赖大版本）。
4. 挖风险：在 diff 中找删除行、依赖版本变化、配置键变化、公共 API 签名变化，逐条标注影响面。
5. 对照已有 CHANGELOG.md 找遗漏，列出未收录项。

## 输出模板
```markdown
## 变更挖掘报告（<范围>）
| 类型 | 提交 | 摘要 | 风险 |
### Breaking Changes
### 遗漏项（CHANGELOG 未收录）
```

## 禁止
- 本技能只读；不许修改 git 历史、不许自动 commit。
- 不许漏掉大版本依赖升级与公共接口改动。
