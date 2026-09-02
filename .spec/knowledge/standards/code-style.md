---
name: code-style
description: 代码风格与视觉识别——LumioAPI 品牌色 / 字体 / 类约定、前端开发约定；写代码 / 改样式时查
metadata:
  type: doc
  level: L1
  status: 已交付
---

# 代码风格与视觉识别

> 写代码 / 改样式时遵守的约定。能交给工具(formatter / linter / golangci-lint)强制的优先交给工具;本文只写需要人 / Agent 判断的部分。

## 视觉识别(LumioAPI · 改样式的北极星)

- 主色 `#4f8cff`(brand-500)配 navy `#1a2f5a`(brand-900)渐变。
- 标题 / 品牌文案用衬线 `Source Serif 4` + `Noto Serif SC`(Google Fonts)。
- 辅助 UI 文字才用 system sans(`.ui-sans` 类)。
- 数字用 `.ui-mono` 类,便于对齐。

## 前端约定(`frontend/` 是当前开发焦点)

- 包管理用 **pnpm**(不是 npm);改 `package.json` 后必须同步提交 `pnpm-lock.yaml`。
- TypeScript 走严格检查:`pnpm typecheck` 必须过。
- 构建产物输出到 `backend/internal/web/dist/`(`pnpm build`)。
- 改动控制范围,不顺手重构,避免增加 upstream 同步负担。

## 后端约定(`backend/` 来自 upstream,不主动改)

- Go,Ent ORM + Gin。改 `ent/schema/*.go` 后必须 `go generate ./ent` 并提交生成代码。
- 给 interface 加方法后,补全所有 test stub / mock(否则编译失败)。
- 集成测试用 build tag,见 [`testing.md`](./testing.md)。

## 注释 / 命名

- 注释解释"为什么",不复述"做了什么";与周围代码风格保持一致。
- 命名跟随既有模式,不引入与上游冲突的新风格。

## 相关

- 工作流:[`workflow.md`](./workflow.md) · 测试:[`testing.md`](./testing.md) · 环境:[`dev-environment.md`](./dev-environment.md)
