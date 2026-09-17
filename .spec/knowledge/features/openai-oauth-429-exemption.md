---
name: openai-oauth-429-exemption
description: OpenAI OAuth 429 拉闸豁免：账号默认豁免 Retry-After 冷却（瞬时 429 同账号重试消化），可手动或由号池自动巡检降级时关闭
metadata:
  type: doc
  level: L2
  status: 已实现
---

# OpenAI OAuth 429 拉闸豁免

452e651c0「Honor full Retry-After windows for OpenAI OAuth 429」把带 `Retry-After` 的 429 升级为账号级拉闸（内存熔断 + `handle429` 落库 + 禁止同账号重试），IP 组 Codex 池因瞬时 RPM 429 频繁整池轮转冷却。本特性给这条链路加按账号豁免，并接入号池自动巡检做自动生命周期。

## 语义

- **豁免键**：`accounts.extra.oauth429_cooldown_enforced`。**缺省（无键或 false）= 豁免**；显式 `true` = 恢复 452e651c0 行为。存量账号无需迁移。
- **豁免账号**：`classifyOpenAIOAuth429ForAccount` 把 `RetryAfter` 分类降为瞬时——同账号重试窗口内不熔断、不落库、可重试（重试等待取整窗 Retry-After，受 2 分钟窗口 + 3 次预算约束；放不进预算就换号）。
- **不受豁免影响**：真实配额耗尽（x-codex 5h/7d used ≥ 100%、body `usage_limit_reached`/`resets_at`）始终保留 Quota5h/Quota7d/QuotaReset 长冷却；spark 影子账号整段跳过。
- **手动开关**：EditAccountModal（OpenAI OAuth / SetupToken）「429 限流豁免」，保存时显式落 `oauth429_cooldown_enforced = !enabled`（两种状态都写键，避免依赖缺省语义）。
- **自动联动**：号池自动巡检 `close_429_exemption_on_degrade`（默认开）——判定降级时对 OpenAI OAuth 系账号 `UpdateExtra` 写 `true`（key 级 JSONB 合并，随 scheduler outbox 生效）；与分组/模型降级一致**不自动恢复**，恢复靠编辑账号重新打开。

## 分类管道

`openai_account_runtime_block_fastpath.go`：

- `classifyOpenAIOAuth429Signal`（内部）：产出 disposition + resetAt + `exhausted`（是否真实耗尽）。
- `classifyOpenAIOAuth429`（自由函数，无账号上下文）：仅 spark 模型级限流（`HandleOpenAICodexSparkRateLimit`，只认 Quota5h/7d）使用，语义不变。
- `classifyOpenAIOAuth429ForAccount`：`markOpenAIOAuth429RateLimited` / `shouldRetryOpenAIOAuth429OnSameAccountWithResponse` / `ShouldRetryOpenAIOAuth429` 三个调用点使用；豁免账号把非耗尽的 RetryAfter 归瞬时，连带 `handle429` 的 OAuth 早退门（不落库）自动恢复。

注意：`SetRateLimited` 在 452e651c0 起就是只延长不缩短（`SetRateLimitedIfLater`），豁免不改变这一点——enforced 账号的落库窗口仍只涨不清，需「恢复状态」或自然到期。

## 已决策

- 豁免缺省开：等于全池恢复 452e651c0 之前的行为，直到账号被巡检降级或手动关闭。曾考虑「短 Retry-After 全局归瞬时」的阈值层，因会让 enforced（被降级）账号也不拉闸、与降级语义冲突而放弃。
- 不豁免真实耗尽；不豁免 spark 影子。
- 巡检关闭豁免不自动恢复（与分组/模型降级同哲学）。

## 关键文件

- `backend/internal/service/openai_account_runtime_block_fastpath.go`（分类管道 + 豁免接入）
- `backend/internal/service/account.go`（`OAuth429CooldownEnforcedExtraKey` / `OAuth429CooldownExempt`）
- `backend/internal/service/account_pool_auto_inspect.go` / `_service.go`（`Close429ExemptionOnDegrade` 配置与降级动作）
- `frontend/src/components/account/EditAccountModal.vue`（豁免开关）、`frontend/src/components/admin/account/PoolAutoInspectDialog.vue`（联动配置）

## 相关

- 号池自动巡检（联动入口）：[[account-pool-auto-inspect]]
- IP 组 429 语义：[[openai-oauth-ip-group]]
