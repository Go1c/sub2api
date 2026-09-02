---
status: completed
---

# 任务：国模底座第一刀（常量 + schema + 探测）

从 `origin/main` 把 kimi/zhipu/deepseek 一等支持的**底座**迁到 fork `sync/v0179-cn-providers`（基线 `origin/dev` = `a7d3e8dbb`，已含 fork PR #364+#365）。

不要整包 merge main。不要碰 `publish`。不要改 `docker-compose.yml`。不要合 Composite 路由（第五刀）。不要合网关 Anthropic native 转发（第二刀）。不要合 Create/Edit 账号弹窗 UI（第三刀）。

## 做什么

1. 常量：`PlatformKimi` / `PlatformZhipu` / `PlatformDeepseek`、`AccountModePayG/Coding`、`APIProtocol*`、默认 base_url、`IsCNProvider`、`AllowedQuotaPlatforms` 扩到 8 平台、`AllowedSchedulingThresholdPlatforms`（常量可加；fork 没有 `account_scheduling_threshold_eval.go`，**不要整文件移植该评估器**）。不要加 `PlatformKiro`。`PlatformComposite` 本刀可不加（留给第五刀）。
2. `Account` 方法：`IsKimi/IsZhipu/IsDeepseek/IsCNProvider`、`IsOpenAICompatible` 含 CN、`GetOpenAIBaseURL` CN 分支、`GetAccountMode` / `IsCodingPlan` / `GetAPIProtocol` / `IsAdaptiveAPIProtocol` / `GetCNProtocolBaseURL` / `GetAnthropicProtocolBaseURL` / `GetOpenAIFormatBaseURL` / `GetCNAPIKey` / `GetCodingPlanProvider`。对照 `origin/main` 的 `account.go` **外科移植**，不要整文件替换。
3. schema：`backend/ent/schema/user_platform_quota.go` Validate 加入 kimi/zhipu/deepseek。
4. 迁移 **934**（origin `224_user_platform_quotas_add_cn_providers.sql` remap）：放宽 `user_platform_quotas_platform_check`。930 已被 fingerprint seed 占用。origin `227_composite_routes_add_cn_providers.sql` **本刀不做**（fork 没有 `composite_model_routes` 表）。
5. 探测服务：从 **origin/main HEAD** 拷贝并适配（含后续修复 #5911/#5906/probe_url）：
   - `cn_provider_balance_service.go` + `_test.go`
   - `cn_provider_quota_service.go`
   - `cn_provider_balance_check_service.go` + `_test.go`
   - `cn_provider_probe_url.go` + `_test.go`
   - `cn_providers_test.go` / `cn_provider_foraccount_test.go`（与探测相关的部分）
   - `ratelimit_cn_providers.go`
   - `ratelimit_service.go` 的 402/403/429 CN 钩子（fork 已有 `notifyAccountSchedulingBlocked` / `SetTempUnschedulable` / `handleOpenAI403`）
6. config：`GatewayCNProvidersConfig` + viper defaults + 若 origin 把 CN host 放进 url allowlist，fork 同步（不碰支付配置）。
7. admin：`cn_provider_handler.go`、`registerCNProviderRoutes`、`AdminHandlers.CNProvider`、handler/service/cmd wire + `wire_gen.go` 重新生成、cleanup Stop。
8. `api_contract_test.go` 的 `default_platform_quotas` 加上 kimi/zhipu/deepseek。
9. 前端仅 API：`frontend/src/api/admin/cnProviders.ts` + `index.ts` 导出。不要改 Create/Edit modal、不要加 Quota/Balance Cell。
10. 知识：等本刀测试绿后再改 `.spec/knowledge/records/upstream-sync-v0179-newest-gateway.md`（也可本刀一起改「明确排除 CN」→「已授权合入，第一刀进行中」）。

## 涉及范围

- `backend/internal/domain/constants.go`
- `backend/internal/service/domain_constants.go`
- `backend/internal/service/account.go`（CN 方法 + GetOpenAIBaseURL）
- `backend/ent/schema/user_platform_quota.go`
- `backend/migrations/934_user_platform_quotas_add_cn_providers.sql` + 对应 migration test
- `backend/internal/service/cn_provider_*.go` `ratelimit_cn_providers.go` `ratelimit_service.go`
- `backend/internal/config/config.go`
- `backend/internal/handler/admin/cn_provider_handler.go` `handler.go` `handler/wire.go`
- `backend/internal/server/routes/admin.go` `api_contract_test.go`
- `backend/internal/service/wire.go` `backend/cmd/server/wire.go` `wire_gen.go`
- `frontend/src/api/admin/cnProviders.ts` `index.ts`

## 验收标准

- [x] `origin/dev` 上原先没有的 `PlatformKimi/Zhipu/Deepseek` / `IsCNProvider` 在 fork 可编译引用
- [x] `user_platform_quotas` CHECK 允许 kimi/zhipu/deepseek；迁移号是 931 不是 224
- [x] 未改 `docker-compose.yml`、未改支付/订阅、未加 227/composite 表
- [x] `go test -tags=unit` 覆盖改动包：至少 `./internal/service`（CN 相关 `-run` 或全包）、`./internal/handler`、`./internal/handler/admin`、`./internal/server`、`./migrations`
- [x] `go vet -tags integration ./internal/service ./internal/handler ./internal/handler/admin ./internal/server ./internal/config ./cmd/server`
- [x] 若改了 frontend API：`cd frontend && pnpm typecheck && pnpm build`
- [x] 探测单测包含 origin 已有的 DeepSeek 非法 relay payload 拒绝（#5911）和 fake 加锁（#5906）
- [x] 不提交网关 anthropic_native 转发文件、不提交 CreateAccountModal CN UI

## 依赖

无（底座第一刀）
