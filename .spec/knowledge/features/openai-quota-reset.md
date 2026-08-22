---
name: openai-quota-reset
description: 管理端 OpenAI/Codex Auth 登录、查询重置卡、消耗重置卡；路由必须挂上，面板走 POST /quota/refresh 并回写 extra.codex_reset_credit_snapshot
metadata:
  type: doc
  level: L2
  status: 已实现
---

# OpenAI 重置卡查询 / 消耗与 Codex PAT 登录

简介：管理端账号行「查询次数 / 重置」调用 OpenAI Codex 上游的 rate-limit reset credits；Codex PAT Auth 登录走 `POST /admin/openai/create-from-codex-pat`。功能实现来自 `origin/main`，本 fork 曾只落下 handler 未挂路由，面板 axios 404。

## 背景 / 目标

- 管理员需要在账号表查询 Codex 重置卡张数、到期时间，并消耗一张立即恢复 5h/7d 窗口。
- `origin/dev` 已有 handler 方法（`CreateAccountFromCodexPAT` / `QueryQuota` / `ResetQuota`），但 `registerOpenAIOAuthRoutes` 只挂了 generate-auth-url / exchange-code / refresh-token / create-from-oauth。Gin 无路由 → 404。
- 合入时**禁止整包 merge main**，只取重置卡 / Auth 登录主题；不要带 Ollama、channel-monitor-v2、计费倍率同步、composite groups。

## 设计

- **路由（必须四条都挂）**：
  - `POST /admin/openai/create-from-codex-pat`
  - `GET /admin/openai/accounts/:id/quota`（只读查询，给 API 消费者）
  - `POST /admin/openai/accounts/:id/quota/refresh`（查询并持久化重置卡快照；面板走这条）
  - `POST /admin/openai/accounts/:id/reset-quota`（消耗一张重置卡）
- **RefreshQuota**：上游 `QueryUsage` 成功后调用 `CacheResetCreditsSnapshot`，写入 `extra.codex_reset_credit_snapshot`。快照写失败仍返回用量，`cache_persisted=false`，保留旧缓存（有张数无到期明细的快照不能落库，否则无法老化过期卡）。
- **ResetQuota 后处理**（信用已在上游消耗、不可退）：先 `RateLimitService.RecoverAccountState(InvalidateToken: true)`，再回读配额并刷新缓存，最后把账号行交给前端。后处理从客户端取消中拆出，超时 8s。失败用 `warning_code` 报告，不把已成功的消耗改成错误。
- **Agent identity**：不再用 fork 的 `ErrAgentIdentityResetNotSupported` 硬拦截。重置时走 agent identity assertion；task 无效则恢复一次再重试。
- **Spark 影子**：影子不能消耗重置卡（`ErrSparkShadowResetNotSupported`），请在母账号上重置。查询可解析到母账号凭证，但快照写在被查询的那一行。
- **调度中性 extra**：`schedulerNeutralExtraKeyPrefixes` 含 `codex_reset_credit_`，快照更新不触发调度缓存失效。
- **前端**：`refreshOpenAIQuota` → `POST /quota/refresh`；`resetOpenAIQuota` timeout 90s。`OpenAIQuotaResetCell` 从 `extra.codex_reset_credit_snapshot` 水合。重置成功后 `account-updated` 回传账号；`AccountUsageCell` 用短窗抑制随后的 `/usage` 自动刷新。5h/7d 有数据时，本地 `loadActiveUsage` 放在 `#pre-actions`。

## 已决策

- 实现从 `origin/main` 取文件 / hunk，不在 fork 重写配额服务或重置卡 UI。
- 面板不绑 GET `/quota`：查询必须写快照，且审计中间件只记 mutating verb。
- 合入 main 后删除 `ErrAgentIdentityResetNotSupported` 是预期，不是回归。

## 待解决

- 无。

## 相关

- Handler：`backend/internal/handler/admin/openai_oauth_handler.go`
- 路由：`backend/internal/server/routes/admin.go` `registerOpenAIOAuthRoutes`
- 服务：`backend/internal/service/openai_quota_service.go`（`QueryUsage` / `CacheResetCreditsSnapshot` / `ResetCredit`）
- 前端：`frontend/src/components/account/OpenAIQuotaResetCell.vue`、`AccountUsageCell.vue`、`frontend/src/api/admin/accounts.ts`
- extra 类型：`frontend/src/types/index.ts` `codex_reset_credit_snapshot`
