---
name: upstream-sync-v0177-codex-compact-group-usage
description: main 快进到 0.1.177 后，将 Codex turn-state、native compaction v2、分组用量日 rollup 按主题带入 dev 的评估与适配台账。
metadata:
  type: record
  date: 2026-08-15
  status: 已实现，待 Review 合入 dev
---

# Upstream Sync 台账 — 2026-08-15 (v0.1.177)

## 范围

| 项 | 值 |
|---|---|
| fork `main` | 已与 `upstream/main` 对齐 `baeac1f3d`（`0.1.177`） |
| 同步方式 | GitHub `repo sync` 快进 `origin/main`；**不**整包 merge 进 `dev` |
| 增量基线 | 上一版 fork main `fbfdcef81`（`0.1.176`）→ `baeac1f3d`（`0.1.177`） |
| first-parent | 4 个落点 |
| 全量 commit | 13 |
| 触及文件 | 68（+4413 / −311） |
| `dev` tip（开工时） | `e79e43720` |
| **禁止** | 整包 `git merge main` → `dev`；直推 `main` / `publish` |

## 分类（first-parent）

| 档 | 上游 | 建议 |
|---|---|---|
| T1 网关协议 | `#5668` Codex turn-state 透传 + 跨账号回声守卫；指纹收敛 ID 暂存/透传 raw 改写 | **移植**；**保留 fork 默认 `session`**，不跟上游改成默认 `off` |
| T2 网关协议 | `#5641` native remote compaction v2：探测改走流式 `/responses`，legacy `/responses/compact` 仅作旧桥 | **移植** |
| T3 管理端用量 | `#5649` 分组用量按日 rollup + 昨日列 + 服务端时区 | **移植**；迁移改号 `924` / `925` |
| CHORE | `VERSION` → `0.1.177` | **跳过**（fork `VERSION` 仍是独立口径 `0.1.160`） |
| INFRA | Go `1.26.6`、CI workflow 版本钉 | **跳过**（不在本主题；避免无关 CI 漂移） |

## 已决策

- 指纹收敛默认保持 fork `session`（未设置 extra 键仍收敛）。上游 `#5610` 改成 opt-in `off` 是为了避免存量账号被静默改写；本 fork 已有管理端开关且线上按 `session` 运行，跟改默认会改变现网行为。
- native v2 判定改为「裸 `/responses` + `stream:true` + `compaction_trigger`」，不再要求客户端必须带 `x-codex-beta-features`；出站由网关补注该头。
- 分组日桶跟服务端配置时区，管理端不再传浏览器 `timezone` query。
- 上游 `222`/`223` 迁移落到 fork `924_group_usage_daily_rollups.sql` / `925_group_usage_rollup_timezone.sql`，避开 `900+` 与重复的 `923_*`。

## 明确排除

- 整包 merge upstream / 直推 `main` / `publish`。
- 把指纹默认改成 `off`，以及账号弹窗「默认关闭」文案。
- `.github/workflows/*` 与 `backend/go.mod` 的 Go 1.26.6 钉版本。
- 把 fork `VERSION` 改成 `0.1.177`。

## 验证

| 门禁 | 结果 |
|---|---|
| turn-state / compact probe / fingerprint 聚焦 `go test -tags=unit ./internal/service` | 通过 |
| compact body-signal `go test -tags=unit ./internal/handler` | 通过 |
| rollup / cleanup / timezone / migrations 聚焦测试 | 通过 |
| `go vet -tags integration`（改动包） | 通过 |
| 前端 `pnpm typecheck` | 通过 |
| `admin.groups.usage-summary` + `GroupsView.columnSettings` | 8/8 通过 |
| 前端 `pnpm build` | 通过 |
| 全量 `go test ./...` | 未跑（既有时长问题，不作为本主题回归） |
| 管理端 Groups 昨日列 | 无浏览器工具；靠 typecheck / 单测与代码审查 |

交付分支：`sync/v0177-codex-compact-group-usage` → `--base dev`。

## 相关

- 工作流：[`../standards/workflow.md`](../standards/workflow.md)
- 上一轮 Grok 主题：[`upstream-sync-grok-v0176-jwt-xsearch-pricing.md`](upstream-sync-grok-v0176-jwt-xsearch-pricing.md)
