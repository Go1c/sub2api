---
name: testing
description: 测试与验收——本地必跑命令、CI 流水线、集成测试 build tag、DoD；实现功能 / 修 bug / 提 PR 前查
metadata:
  type: doc
  level: L1
  status: 已交付
---

# 测试与验收(含本地验证政策)

> 什么时候写测试、产出怎么算"验收通过"。"先写失败测试再实现"的方法见技能 [`skills/test-driven-development`](../../skills/test-driven-development/SKILL.md);本文定**政策**(何时用、跑什么),技能讲**方法**。

## 本地必跑(commit / PR 前)

**前端(`frontend/`)** —— 任何代码改动:

```bash
cd frontend && pnpm typecheck && pnpm build
```

涉及 UI 的改动还要:dev server 起来 curl 过,或请求用户浏览器确认。**只凭类型检查不得报"完成"**(硬护栏见 [`rules/system.md`](../../rules/system.md))。

**后端(`backend/`)**:

```bash
cd backend
go test -tags=unit ./...           # 单元测试
go test -tags=integration ./...    # 集成测试(必须带 -tags,否则漏检)
go vet -tags integration ./...     # 集成代码编译校验
golangci-lint run ./...            # v2.9
```

## CI 流水线(GitHub Actions)

| Workflow | 触发 | 检查 |
|----------|------|------|
| `backend-ci.yml` | push / PR | 单元 + 集成测试 + golangci-lint v2.9 |
| `security-scan.yml` | push / PR / 每周一 | govulncheck + gosec + pnpm audit |
| `release.yml` | tag `v*` | 构建发布(PR 不触发) |

要求:Go **1.25.7**;前端 `pnpm install --frozen-lockfile`,必须提交 `pnpm-lock.yaml`。

## 何时走 TDD

- 必须:新功能、修 bug。
- 可不走:纯文档改动、一次性脚本。方法见 `skills/test-driven-development`。

## 验收标准(Definition of Done)

- [ ] 前端 `pnpm typecheck` + `pnpm build` 通过;UI 改动经浏览器 / curl 确认。
- [ ] 后端 unit + integration 测试全绿,`golangci-lint` 无新增问题。
- [ ] 改 interface → test stub 已补全;改 ent schema → 生成代码已提交;改 package.json → lock 已同步。
- [ ] 相关知识文档已更新(见 [`workflow.md`](./workflow.md))。

## 相关

- 工作流:[`workflow.md`](./workflow.md) · 环境与坑点:[`dev-environment.md`](./dev-environment.md)
- 集成测试 build tag 的踩坑背景见 [`dev-environment.md`](./dev-environment.md) 坑 6 / 坑 8。
