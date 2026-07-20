---
name: upstream-sync-20260720-v0162
description: main 同步至 v0.1.162 后，将 0.1.160→0.1.162 增量分主题并入 dev 的评估台账与进度
metadata:
  type: record
  date: 2026-07-20
  status: 进行中
---

# Upstream Sync 台账 — 2026-07-20 (v0.1.162)

## 范围

| 项 | 值 |
|---|---|
| fork `main` | 已与 `upstream/main` 对齐 `e625ce3b3`（`0.1.162`） |
| 同步方式 | GitHub `merge-upstream` API（fast-forward） |
| 增量基线 | 上一版 fork main `57914967c`（`0.1.160`）→ `e625ce3b3`（`0.1.162`） |
| first-parent | **65** 个落点（含 VERSION/sponsors） |
| 全量 commit | **176** |
| 触及文件 | **377**（+23k / −2.4k） |
| `dev` tip（评估时） | `1f0fa14e3`（PR #221 wave2-continue-21 已合；PR #222 回归修复进行中） |
| **禁止** | 整包 `git merge main` → `dev`（trial 约 **505** 处 both-modified） |

## 分类（first-parent，关键词粗分）

| 档 | 数量 | 说明 |
|---|---|---|
| T1 安全 | 4 | step-up 默认关、ingress reject 过滤、prompt-audit fail-closed、S3 临时密钥拒绝持久化 |
| T1 基础设施 | 9 | compose/Dockerfile、BindEnv、client-ip、github release token、image storage env 等 |
| T2 网关 | ~32 | Grok / OpenAI / Codex / Anthropic / WS / sticky / scheduler 等（与 wave2 重叠大，逐主题） |
| T2 前端 | 5 | i18n、dark palette、branding SVG、channels scroll 等 |
| T3 计费/订阅展示 | 5 | plan currency symbol、renew expired、validity label、antigravity plan_type、balance modal |
| CHORE / OTHER | 10 | VERSION、sponsors、杂项修复 |

## 关键新能力（dev 上尚缺）

- `ingress_reject` 中间件 + ops 聚合 + cleanup CLI
- `auth_cache_invalidation_outbox`（migration + service + repo）
- `invalid_auth_abuse_limiter`
- `image_storage_settings` / env 可达性
- branding SVG logo helpers、`batchImage` i18n 拆分等

## Migration 编号映射

upstream `183`/`184` 在 fork 编号空间已被占用（fork 用 `900+` 扩展）。并入时：

| upstream | fork 拟编号 | 说明 |
|---|---|---|
| `183_ops_ingress_reject_aggregates.sql` | `915_ops_ingress_reject_aggregates.sql` | 在 `914_prompt_audit_full_prompt` 之后 |
| `184_auth_cache_invalidation_outbox.sql` | `916_auth_cache_invalidation_outbox.sql` | 同上 |

## 执行顺序

1. **T1a** 小安全修复：prompt-audit fail-closed（#4565）、S3 ephemeral key（#4638）
2. **T1b** security switches default-off / step-up 可关（#4526）— 注意 fork settings 深度改造
3. **T1c** ingress reject + auth abuse + auth cache outbox（#4515 主体）— 含 migration 重编号与 wire
4. **T1d** 其余 infra（compose / client-ip / github token / image storage env）按需
5. **T2** 网关/前端：对照 wave2 已合内容，只补 **dev 缺且有价值** 的 PR
6. **T3** 计费/订阅：保留 fork 额度池与支付语义，只 port 不冲突的展示/小修复

## 进度

| 批次 | 分支 | 状态 |
|---|---|---|
| 评估 + main sync | — | ✅ `origin/main` = `upstream/main` = `e625ce3b3` |
| T1a 小安全 | `sync/t1-security-v0162` | ✅ #4565 prompt-audit fail-closed + #4638 S3 ephemeral key（backup only；image_storage 待 T1d） |
| T1b step-up 开关 | — | ✅ 已在 `origin/dev`（#4526 等价能力齐全，跳过） |
| T1c ingress reject | 待开 / 后续 PR | 待（#4515 约 206 文件，含 migration 重编号） |
| T2/T3 | 待开 | 待 |

## 验收门槛（每批 PR）

- `go build ./cmd/server`
- `go test -tags=unit`（相关包全绿；改 repository 时至少跑触及包）
- 涉及 integration 源码：`go vet -tags integration ./...`
- 涉及 frontend：`cd frontend && pnpm typecheck && pnpm build`
- PR：`gh pr create --repo Go1c/sub2api --base dev`
