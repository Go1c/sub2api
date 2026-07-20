---
name: upstream-sync-20260720-v0162
description: main 同步至 v0.1.162 后，将 0.1.160→0.1.162 增量分主题并入 dev 的评估台账与进度
metadata:
  type: record
  date: 2026-07-20
  status: T1 安全/收尾批次完成；T2/T3 按主题另开
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

## 关键新能力（dev 状态）

- ✅ `ingress_reject` + auth abuse + auth cache outbox（#224）
- ⚠️ `image_storage`：已有 S3 实现；**后台 `image_storage_settings` 服务/UI 全量 port 仍缺**；本批只补 env 空默认
- ⚠️ client-ip 自定义 CDN 头 / `ForwardedClientIPHeaders` 全量 port 仍缺（与 fork IP 栈深度交织）
- ⚠️ branding SVG / batchImage i18n 等前端：按 T2 另开
- ⚠️ T2 网关大量 first-parent 落点与 wave2 重叠，**禁止**整包 merge main→dev

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
| T1a 小安全 | `sync/t1-security-v0162` | ✅ 已合 PR #223 |
| T1b step-up 开关 | — | ✅ 已在 `origin/dev`（#4526 等价能力齐全，跳过） |
| T1c ingress reject | `sync/t1c-ingress-reject-v0162` | ✅ PR #224 已合 `dev`；migration 915/916 |
| T1c follow-up | `fix/usage-log-insert-placeholders` | ✅ PR #225 已合：usage_logs `$57/$58`、golangci baseline、integration suite 漂移 |
| T1d 安全小补丁 | `sync/t1d-infra-v0162` | ✅ PR #226 已合：image_storage 空默认 + compose env 透传 |
| T1d GitHub token | `sync/t1d-github-token-v0162` | ✅ PR #227 已合：UPDATE_GITHUB_TOKEN |
| T2 小修复批 | `sync/t2-small-fixes-v0162` | ✅ PR #228 已合 |
| T2 网关修复批 | `sync/t2-gateway-fixes-v0162` | ✅ PR #229 已合：Codex models 401 下线、Anthropic stop_reason null、WS HTTP bridge failover、OpenAI quota 错误形态、Responses SSE/image intent、Grok 手动测试调度门 |
| T2 杂项批 | `sync/t2-misc-v0162` | ✅ PR #230 已合：sticky 同账号重试不计缓存、Codex list envelope、Claude `[1m]` 后缀、channels 滚动、balance modal dark、system update detach（fork 形态）、redis compose 续行、i18n 硬编码 |
| T2 model_not_found | `sync/t2-model-not-found-4508` | ✅ PR #231 已合：#4508 临时账号耗尽保留 503，不误报 model_not_found |
| T2 plan-validity | `sync/t2-plan-validity-4528` | ✅ PR #232 已合：#4528 套餐有效期文案去写死「天」 |
| T2 renew-expired | `sync/t2-renew-expired-4541` | 进行中：#4541 admin assign 过期订阅续期（fork 额度池字段适配） |
| T2/T3 | — | **不整包合**；first-parent 仍有大量与 wave2 重叠的网关/前端 PR，需按主题另开 |

### T2 misc 冲突 / 适配纪要

| 项 | 决策 |
|---|---|
| system update detach | 只 port `systemUpdateContext` + PerformUpdate 解耦 + 前端 update 15min timeout；**不**引入 main 的 `RollbackToVersion` / already-up-to-date / interface 抽象（fork 无该能力） |
| system_handler_test.go | fork 无此测试文件 → 不恢复 |
| redis local volume | 保留 fork `./redis_data:/data`（无 `:Z`）；只合 command 行续 `\` |
| plan currency / Trae / grok video / branding SVG / dark palette 全量 | 仍跳过（深度冲突或 fork 产品差异） |

### CI 基线债（全 PR 共有，不在本台账批次阻塞）
- `backend-security` govulncheck：S3 SDK / crypto/tls 调用图（#222/#223/#224/#225 均红）


## 验收门槛（每批 PR）

- `go build ./cmd/server`
- `go test -tags=unit`（相关包全绿；改 repository 时至少跑触及包）
- 涉及 integration 源码：`go vet -tags integration ./...`
- 涉及 frontend：`cd frontend && pnpm typecheck && pnpm build`
- PR：`gh pr create --repo Go1c/sub2api --base dev`


## T1c 冲突处理纪要

### 已按「并集」处理（非漏合历史）

| 区域 | 决策 |
|---|---|
| API key auth | 保留 fork 中文余额不足文案 + 上游 header 尺寸/ingress Mark |
| API key errors | 保留 fork `ErrFallbackKeyInvalid` + 上游 `ErrAPIKeyAuthOverloaded` |
| Ops port/models/service | 保留 fork `LookupDeletedKeyAudit` / retry / request-body queue 路径；并入上游 runtime settings / queue body bounds / ingress aggregator 依赖 |
| Ops cleanup | 同时清理 `ops_retry_attempts` 与 `ops_ingress_reject_aggregates` |
| wire / wire_gen | 保留 fork 用户请求监控等 cleanup；并入 ingress/outbox/runtime refresh 生命周期 |
| migration | `183/184` → `915/916`（fork 编号空间） |

### 产品确认（已定）

| 项 | 结论 |
|---|---|
| `IgnoreInvalidApiKeyErrors` 强制 true | 接受（跟上游） |
| Ops 详情 `api_key_prefix` | 保留 |
| 恢复 3 个上游测试文件 | 已恢复并合入本 PR |
| `deploy/README.md` | 不恢复；保留 `EDGE_SECURITY.md` |
| 合并顺序 | #222 → #223 → #224（用户授权全部合入） |
