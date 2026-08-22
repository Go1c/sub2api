---
name: sync-openai-quota-reset-from-main
description: 从 origin/main 合入 OpenAI Auth 登录 / 查询重置卡 / 使用重置卡，修复当前 404
status: completed
---

# 从 main 合入 OpenAI 重置卡与 Auth 登录

管理端账号行上「查询次数 / 重置」返回 axios `Request failed with status code 404`。根因不是前端写错，而是 `origin/dev` 相对 `origin/main` 没把路由和后半段实现挂上。

## 根因

`frontend` 已调用：

- `POST /admin/openai/create-from-codex-pat`（Codex PAT Auth 登录）
- `GET /admin/openai/accounts/:id/quota`（查询重置卡；main 面板实际改走 POST refresh）
- `POST /admin/openai/accounts/:id/reset-quota`（使用重置卡）

Handler 方法在 `backend/internal/handler/admin/openai_oauth_handler.go` 已存在，但 `registerOpenAIOAuthRoutes` 只挂了 generate-auth-url / exchange-code / refresh-token / create-from-oauth。Gin 无路由 → 404。

`origin/main` 的完整实现还包含：

- `POST /accounts/:id/quota/refresh` + `CacheResetCreditsSnapshot`
- 重置后 `RecoverAccountState`、回写缓存、把账号行交给前端
- Agent identity 重置时 assertion + 无效 task 恢复（main 已删除 fork 的 `ErrAgentIdentityResetNotSupported` 硬拦截）

## 硬性约束

- **不要自己写功能实现。** 用 `git checkout origin/main -- <file>` 取 main 文件；混有 fork 代码的文件只摘 main 对应 hunk。
- **禁止整包 merge main → dev。**
- **禁止在当前 `release/*` 分支上提交。** 从 `origin/dev` 开主题分支。
- **禁止直推 `main` / `publish`。** PR `--repo Go1c/sub2api --base dev`。
- 不得带入 main 的 Ollama / channel-monitor-v2 / 计费倍率同步 / composite group 等无关改动。

## 建议分支名

`sync/openai-quota-reset-from-main`

基线：`git fetch origin && git checkout -B sync/openai-quota-reset-from-main origin/dev`

## 可整文件取自 origin/main

```
backend/internal/handler/admin/openai_oauth_handler.go
backend/internal/handler/admin/openai_oauth_handler_reset_quota_test.go
backend/internal/handler/admin/openai_oauth_handler_spark_shadow_test.go
backend/internal/service/openai_quota_service.go
backend/internal/service/openai_quota_spark_window_test.go
frontend/src/components/account/OpenAIQuotaResetCell.vue
frontend/src/components/account/__tests__/OpenAIQuotaResetCell.spark_shadow.spec.ts
```

不要取：`openai_quota_platform_contract_test.go`（网关 RecordUsage，与本主题无关）。
不要整文件取：`account_repo.go`、`admin.go`、`accounts.ts`、`AccountUsageCell.vue`、`AccountsView.vue`、`types/index.ts`、i18n 整包、`wire_gen.go`（可 generate）。

## 必须外科手术合入

1. `backend/internal/server/routes/admin.go` `registerOpenAIOAuthRoutes` 补 main 这 4 行：
   - `POST /create-from-codex-pat`
   - `GET /accounts/:id/quota`
   - `POST /accounts/:id/quota/refresh`
   - `POST /accounts/:id/reset-quota`
2. `backend/internal/repository/account_repo.go` 的 `schedulerNeutralExtraKeyPrefixes` 只加 `"codex_reset_credit_"`。不要把 main 的 `upstream_billing_rate_sync` / `ollama_cloud_usage` 或 `UpdateWithAccountBillingSettings` 签名带进来。
3. `backend/internal/handler/admin/account_handler_long_context_billing_test.go`：`NewOpenAIOAuthHandler(..., nil, nil)` 第四参。
4. 改完 handler 构造函数后 `cd backend && go generate ./cmd/server`，让 `wire_gen.go` 注入 `rateLimitService`。不要手写 wire_gen。
5. `frontend/src/api/admin/accounts.ts`：按 main 改 `OpenAIQuotaResetResult`、新增 `OpenAIQuotaRefreshResult` / `refreshOpenAIQuota`（POST `/quota/refresh`）、`resetOpenAIQuota` 90s timeout；导出把 `queryOpenAIQuota` 换成 `refreshOpenAIQuota`。保留 fork 其它 API。
6. `frontend/src/types/index.ts` Account.extra 加 `codex_reset_credit_snapshot?: { available_count?: number; credits?: { expires_at?: string }[] }`。
7. i18n 的 `admin.accounts.openaiQuotaReset`：同步 main 的 `resetSuccess` / `resetCacheRefreshFailed` / `resetAccountRecoveryFailed` / `resetAccountRefreshFailed` / `refreshCachePersistFailed`。改 `en/admin/accounts.ts`、`zh/admin/accounts.ts`，并同样补 `en.ts` / `zh.ts` 里同名块。
8. `AccountUsageCell.vue`（不要整文件覆盖）：
   - `<OpenAIQuotaResetCell>` 加 `@account-updated="handleQuotaResetAccountUpdated"`
   - 有 5h/7d 数据的那支按 main 用 `#pre-actions` 放本地 `loadActiveUsage` 按钮
   - `defineEmits<{ 'account-updated': [account: Account] }>()`
   - `handleQuotaResetAccountUpdated` + `suppressOpenAIUsageRefreshUntil`（抄 main，接到现有 `watch(openAIUsageRefreshKey)`）
9. `AccountsView.vue` 的 `<AccountUsageCell>` 加 `@account-updated="handleAccountUpdated"`（该函数已存在）。

## 行为对齐 main（不要保留 fork 硬拦截）

origin/dev 的 `ErrAgentIdentityResetNotSupported` 在 main 已删除，改为重置时走 agent identity assertion + 无效 task 恢复。合入 main 的 `openai_quota_service.go` / spark window test 后，fork 独有的 `TestResetCreditAgentIdentityRejectedBeforeUpstream` 会随文件被替换，这是预期。

## 验收

- [ ] `registerOpenAIOAuthRoutes` 含上述 4 条路由
- [ ] `go test -tags=unit ./internal/handler/admin ./internal/service -run 'OpenAI|ResetCredit|CacheResetCredits|CreateShadow'` 通过
- [ ] `go vet -tags integration ./internal/handler/admin ./internal/service ./internal/server/...`
- [ ] `NewOpenAIOAuthHandler` 四参数，`wire_gen.go` 传入 `rateLimitService`
- [ ] 前端 `OpenAIQuotaResetCell` 调 `refreshOpenAIQuota` 而不是 `queryOpenAIQuota`
- [ ] `cd frontend && pnpm exec vitest run src/components/account/__tests__/OpenAIQuotaResetCell.spark_shadow.spec.ts`
- [ ] `cd frontend && pnpm typecheck && pnpm build`
- [ ] 知识沉淀：`knowledge/features/` 一篇 + `knowledge/README.md` 一行
- [ ] 无任务外文件、无整包 main 内容

## 不要做

- 不在 release 分支提交
- 不改 VERSION
- 不整包 merge main
- 不重写配额服务 / 重置卡 UI
