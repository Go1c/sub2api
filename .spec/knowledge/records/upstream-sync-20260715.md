---
name: upstream-sync-20260715
description: upstream v0.1.156 有序同步台账——记录 ordered 分支上 fork 适配、unit 验证与未完成项（2026-07-15 起）。
metadata:
  type: record
  date: 2026-07-15
  status: 进行中
---

# Upstream Sync 台账 — 2026-07-15 (ordered)

## 范围与结论

- 分支：`sync/upstream-20260715-ordered`（相对 `origin/dev` 约 +110 commits，working tree 约 350+ 变更文件，含 staged/unstaged）。
- 目标上游：主线约至 `v0.1.156`；本地 `backend/cmd/server/VERSION` 已 bump 为 **`0.1.156`**。
- 方法：按有序计划选择性移植上游能力；**优先上游 deps**，保留 fork 支付 / 订阅 / 余额语义（不做 multi-currency / refund 整包）；恢复 fork 的上游 balance probe / login helpers，同时保留 OpenAI usage force-probe 参数。
- 结果（截至本台账）：
  - **`go test -tags=unit ./...` 全量通过**（exit 0，~49 ok 包，无 FAIL）。
  - `go build ./cmd/server` 通过；wire 手维护可编译。
  - **`go vet -tags=unit ./...` 通过**（无输出）。
  - 前端 **`pnpm install` + `pnpm typecheck` + `pnpm build` 通过**（产物写入 `backend/internal/web/dist`）。
  - integration：本机无 docker/psql/redis；可编译的 integration 包抽样通过（如 `routes`、`tlsfingerprint`）；全量 DB/Redis 集成未跑。
  - golangci-lint 本机不可用；commit 切分 / push 仍待完成。

## 策略摘要

| 维度 | 决策 |
|------|------|
| 依赖 | prefer upstream |
| 支付 / 订阅 / 余额 | 保留 fork；不整包 multi-currency/refund |
| OpenAI usage probe | 保留 force-probe 参数；恢复 fork probe/login helpers |
| wire | 本仓库 wire CLI 不可靠；`wire_gen.go` 手维护 |
| 提交 | 未齐 deliverable 前不 commit / push |

## 已完成适配（本会话及前序切片）

### 网关 / handler

- usage-record worker pool drop → 同步回退，避免丢计费任务。
- API key rate limit 响应：`403`（非 `429`）。
- `ensureForwardErrorResponse`：已写 body 不覆盖。
- gateway usage 钱包余额加载 / 附加；quota 路径携带 wallet balance。
- LinuxDo OAuth callback：始终建 pending choice session，不直接发 `#access_token`。
- Codex text image intent 识别 + OpenAI Responses image intent 挂钩。
- admin settings：`SitePages` / logo URL 校验 / WeChat `*bool` 等；ops system log `APIKeyID` 过滤校验。
- `ExposeUpstreamModelToUser`：DTO / group / usage log 门控。

### 鉴权 / 订阅额度池（关键恢复）

- 恢复 fork **用户级额度池鉴权**路径（对齐 `origin/dev`）：`loadUsableCreditSubscriptionForAuth` + `ListUsableCreditSubscriptions` / `SubscriptionCoversGroup`。
- 主中间件与 Google 风格中间件均：优先可用额度池订阅；无可用订阅时回退余额；限额类错误在无余额时返回 **403**（非 429）。
- 余额不足文案恢复中文：`账户余额不足，请先充值后再使用。`
- `RequireGroupAssignment`：未分组 Key 拒绝时 `MarkOpsClientBusinessLimited(...APIKeyGroupUnassigned)`。
- scheduler metadata 白名单补齐：`ParentAccountID` / `QuotaDimension`、credentials `compact_model_mapping`、extra 键 `openai_responses_mode` / `openai_responses_supported` / codex auto-pause / `model_rate_limits`。

### repository / usage_logs

- insert/select/scan 列序与 fork 的 `subscription_cost_usd` / `balance_cost_usd` / image metadata / `long_context_billing_applied` 对齐。
- `GetUserModelStats` 按 `requested_model`（空回退 `model`）聚合。
- `GetStatsWithFilters` 汇总列拆分 `total_cache_creation_tokens` / `total_cache_read_tokens`。
- ops error where：`phase=upstream` 跳过 status≥400 守卫；默认路径保留 `cyber_policy` 豁免。
- migration checksum 白名单补齐 109/110/112/118/123 等历史 hash（含 fork 当前文件 hash）。
- HTTP upstream：`ResponseHeaderTimeout=0` 表示默认 5m，不再覆盖为 0。

### DI / wire

- 恢复 fork ProviderSet 条目：站内信 / 抽奖 / 发票 / 模型广场 / 订阅浪费统计 / AccountErrorHistory。
- 手修 `cmd/server/wire_gen.go` 调用 arity，使 `go build ./cmd/server` 通过。

### 测试 build tag / 契约 / stub

- 依赖 `//go:build unit` stub 的测试补 tag（如 antigravity gateway、apicompat custom tools）。
- `api_contract_test.go`：补 `CountExhaustedFallbackReferers` / `SubscriptionCreditExtension` stubs；`GroupID *int64`；`NewRedeemHandler` 双参；`NewUsageHandler` 双参；fixtures 增补 `expose_upstream_model_to_user` / 额度池字段 / `invitation_registration_mode` / `site_pages`。
- `testutil.StubConcurrencyCache`：补 `CleanupExpiredAccountSlotKeys`。
- middleware 测试对齐额度池语义：`getUsable`/`listUsable` 桩；限额 → 403 `SUBSCRIPTION_INVALID`。

### 版本

- `backend/cmd/server/VERSION`：`0.1.150` → **`0.1.156`**。

### 前端 greening（typecheck + build）

- `pnpm install` 后 `vue-tsc --noEmit` 与 `pnpm build`（`vue-tsc -b && vite build`）全绿。
- 从 git 历史恢复缺失模块：
  - utils/api：`api/url.ts`、`utils/errorBadges.ts`、`utils/proxyExpiry.ts`
  - admin helpers：`views/admin/groupsSupportedModelScopes.ts`、`views/admin/codexFingerprintSignals.ts`
  - settings：`views/admin/settings/EmailTemplateEditor.vue`、`OpenAIFastPolicyUserSelector.vue`（+ 对应 spec）
  - 组件：`components/common/ProxyAdBanner.vue`、`components/user/UserErrorDetailModal.vue`
- 补齐 / 对齐类型：`types/index.ts`（lottery / site-message / CreateApiKey fallback+allowed_models / User invoice fields / AccountUsageInfo.upstream_balance）。
- 重建 `credentialsBuilder` helpers（header override / plan_type / antigravity project_id）；`openaiWsMode` 增加 `HTTP_BRIDGE`。
- admin/user API 形状补齐：`accounts`（importCodexSession / createOpenAICodexPAT / revertProxyFallback / createSparkShadow）、channelMonitor `APIMode`+jitter、riskControl thresholds/keywords/pre_block metrics、ops `api_key_id`、proxies filter `expired`、`usage` 的 `listMyErrorRequests` / `getMyErrorDetail`。
- app store：`service_quota_enabled` 默认 + `sidebarScrollTop`。

## Fork 例外（刻意保留）

- 支付、订阅额度池、余额扣费与 fork 三桶语义。
- 站内信 / 抽奖 / 发票 / 模型广场 / 订阅浪费统计等 fork 功能链。
- 不把上游 multi-currency / refund 整包并入。
- 鉴权：用户级额度池优先（非上游「仅分组 subscription_type active 订阅」）。

## 验证

### 已通过

```text
go test -tags=unit -count=1 -timeout 300s ./...   # 全量，exit 0，~49 ok 包
go vet -tags=unit ./...                           # 无诊断
go build ./cmd/server/
pnpm typecheck / pnpm build                       # frontend，产物 → backend/internal/web/dist
# 抽样 integration（无 DB 依赖）：routes、tlsfingerprint ok
```

### 部分 / 未跑

- [x] 前端 `pnpm typecheck`
- [x] 前端 `pnpm build`
- [x] `go vet -tags=unit ./...`
- [ ] `go test -tags=integration ./...`（本机无 docker/psql/redis；需测试库后重跑）
- [ ] `go vet -tags=integration ./...` 全量（已对无 DB 包抽样）
- [ ] `golangci-lint run ./...`（本机未安装）
- [ ] ent 生成代码与 schema 最终一致性复核（`go generate ./ent` 未强制重跑；当前可编译）
- [ ] PR 合入 `dev` 前的 diff 复审与 **按主题切 commit**
- [ ] commit / push（明确要求前不做）

## Working tree 盘点（commit 切分用）

约 370+ 路径（含 staged/unstaged/untracked）。前端 greening 约 25+ 文件（types、API、恢复的 vue/ts 模块、build dist 若纳入另计）。

| 桶 | 约计 | 说明 |
|----|------|------|
| service | ~190 | 网关/计费/probe/额度池等主体 |
| ent | ~57 | schema + 生成代码 |
| handler | ~55 | admin/user DTO 与路由 handler |
| repository | ~19 | usage_logs 列序 / ops filter |
| frontend | ~25+ | typecheck/build greening + 历史恢复模块 |
| pkg | ~16 | 共享库 |
| server_cmd | ~9 | wire_gen / routes / VERSION |
| migrations / spec / other | 少量 | checksum 白名单 + 台账 |

## 建议 commit 切分（尚未执行）

1. **ent / schema / migrate** — 生成代码与 schema 对齐  
2. **repository / usage_logs / migrations checksum** — 列序、ops filter、checksum 白名单  
3. **service / apicompat / gateway 行为** — probe helpers、image intent、usage worker 等  
4. **middleware 额度池鉴权** — `loadUsableCreditSubscriptionForAuth` + 测试  
5. **handler / dto / routes / wire** — 构造签名、fork ProviderSet、`wire_gen`  
6. **frontend greening** — types / 缺失 vue·ts 模块 / accounts·monitor·risk·usage API / credentialsBuilder / build dist（若需入库）  
7. **api_contract / unit tags / VERSION / knowledge ledger** — 契约与台账  

## 环境阻塞（integration 全量）

- 本机：`no_docker` / `no_psql`；repository 级 integration 依赖 Postgres + Redis。
- 恢复条件：起测试 compose 或配置 `DATABASE_URL` / Redis 后执行 `make test-integration`。

## 已知风险

- working tree 极大（staged + unstaged 数百文件）：需按主题切 commit，避免单包「巨型同步」难审。
- wire 手维护易与 ProviderSet 漂移；后续改构造签名必须同步 `wire_gen.go`。
- integration 未跑：ent / migration / 真实 DB 路径仍可能有 arity 或 checksum 问题。
- 鉴权已切回额度池优先：与上游语义不同，属 **fork 刻意行为**，合入前需在 PR 说明。

## 发布纪律

同步内容必须先经 PR 合入 `dev`；再从远端最新 `dev` 精确快照做 release 分支 promotion 到 `publish`。不得直接推 `publish`，不得在 release 分支补提交。
