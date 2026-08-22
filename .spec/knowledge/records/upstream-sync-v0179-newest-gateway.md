---
name: upstream-sync-v0179-newest-gateway
description: origin/main 快进到 0.1.179 后，按「最近半个月、从最新窗口」把可落地的网关/鉴权修复带入 dev 的台账；整包 merge 已证实不可行。
metadata:
  type: record
  date: 2026-08-22
  status: 已实现，待 Review 合入 dev
---

# Upstream Sync 台账 — 2026-08-22 (v0.1.179 最新窗口)

## 范围

| 项 | 值 |
|---|---|
| fork `main` | merge-upstream API 快进到 `67380eafd`（`0.1.179`），与 `upstream/main` 一致 |
| 同步方式 | 快进 `origin/main`；**不**整包 merge 进 `dev`；从最新窗口 cherry-pick / 适配移植 |
| 增量基线 | 上一版 fork main `baeac1f3d`（`0.1.177`，2026-08-15）→ `67380eafd`（`0.1.179`） |
| first-parent | 88 个落点（2026-08-17…08-21） |
| 全量 commit | 271 |
| 触及文件 | 582（+48053 / −4033） |
| `dev` tip（开工时） | `1f13ec9f7` |
| **禁止** | 整包 `git merge main` → `dev`；直推 `main` / `publish` |

用户约束：优先合最新提交；历史分叉（merge-base 仍是 2026-05 的 `a466e80ed`）不整包带入；外沿是最近半个月。半个月里 8-07…8-15 已部分经 v0176 / v0177 进过 `dev`，本轮从 **0.1.178/179 最新窗口** 下手。

## 整包 merge 试探（不做）

`git merge-tree origin/dev origin/main`：

- `dev` 独有 1216 提交，`main` 独有 2774
- 冲突 **829**（content 473 / add-add 345 / modify-delete 11）
- `backend/internal/service` 312 个冲突文件

结论与 2026-05-29 评估一致：不能把 `main` 整包合进 `dev`。

## 已纳入（T1 最新窗口）

干净 cherry-pick：

| 上游 | 内容 |
|---|---|
| `#5581` | passthrough 模型发现 |
| `#5759` | Codex usage probe 模型名清洗 |
| `#5801` | Chat Completions 缓冲读错误转 failover（fork 适配：failover 错误带上上游响应头，不跟 main 的变参账号副作用链） |
| `#5625` | Antigravity 官方 daily endpoint |
| `#5612` | Antigravity paid-tier endpoint |
| `#5632` | apicompat 流式 tool name 空串 |
| `#6049` | OpenAI sticky prefix system |
| `#6053` | 前端 token refresh 锁 CPU |
| `#5549` | OpenAI capabilities 空集合 |
| `#5844` | Grok inline image / 去掉多余 view_image |

手工适配：

| 上游 | 内容 |
|---|---|
| `#6016` | Prompt Audit `config_loaded` 只在真正变化 / 恢复失败时打日志 |
| `#5720` | 邀请码占用与建用户原子化（TOCTOU）；`user_repo.Create` 优先加入 `TxFromContext` |
| `#5815` | grok-4.6 保留 `xhigh`；4.5 仍折成 `high`；`minimal`→`low` |
| `#5765` 子集 | Grok 用量条带上 24h/7d 本地 window-stats |
| `#5764` `#5767` `#5822` `#5868` `#5881` | Grok/WS client tools + tool_search_output discoveries；HTTP `forwardGrokResponses` 已接 mapping / 流还原 |
| `#5685` | Anthropic 兼容池模式 429 可同账号重试 |
| `#5661` | OpenAI API-key Responses 把 custom/tool_search 降成 function，回程还原 |
| `#5845` | WS 后手 429：未写出下游时换号重放当前轮，不重放第一轮 |
| `#5954` | DeepSeek/Kimi/GLM 的 `max` 保留为 max，旧 GPT 仍折成 xhigh |
| `#5876` | 组内无此模型视为本地配置问题，不进 SLA / 不记上游账号 |
| `#5810` | 原生 `POST /v1/responses/input_tokens`；Grok 与自定义 relay 走本地估算 |
| `#5567` | Anthropic SSE `overloaded_error` 在未写出下游时语义化为 529，可 failover |
| `#5004` | 延迟加载工具去掉 cache_control，断点打在最后一个非 deferred 工具 |
| `#5725` | Gemini 混用 builtin + function 时打开 `includeServerSideToolInvocations` |
| `#5714` | ops 批量写失败不再回退逐条插入 |
| `#5834` | 可配置代理探测 URL；fork 默认仍是 ip-api/httpbin，并加 ipify/chatgpt-trace |
| `#4057` | ops SLA 卡：窗口请求数为 0 时显示中性 `-` |
| `#5925` 子集 | Grok HTTP upstream profile；`encrypted_content` 422 与 400 一样清洗后同账号重试 |
| `#5888` 第一刀 | Responses 入口：`messages`/`prompt` 收成原生 `input`；工具 schema 修 null type + 去掉 lookaround；上游拒收字段后同请求重试（含跨账号 budget）。fork 无 CN provider，null-type 只对 OpenAI/Anthropic 开 |
| `#5888` 第二刀 | Grok ModelInput 清洗（Chat/OpenAI 回放项收成 xAI 子集）；explicit compact 模型不可用时同账号换 fallback 模型再试一次。全局 `openai_compact_model` 只作失败后回退，首次请求仍只看账号 `compact_model_mapping` |
| `#5888` 第三刀 | WS session preempt（Redis owner + 进程内 registry；同 session 新请求取消旧连接）；OpenAI 池模式 API-key 健康熔断（默认 `Enabled=false`，不改现网行为）；ops 错误详情：SSE 终帧捕获、2xx recovered 上游遥测、逐次 `skip_monitoring`。fork 保留请求体/重试头、INVALID_API_KEY 删 key 归因、recovered `account_auth` 的 `error_source=gateway`。未带 origin 的 `ResolvedTargetPlatformFromContext` / sticky `ErrStickySessionNotFound`；`ReportOpenAIAccountScheduleResult` 仍用 fork 的 `accountID` 签名，健康观察走独立方法 |
| `#5888` 第四刀 | Codex OAuth 把上游保留名 `python` 改写成 `python__sub2api`，回程按 context 映射还原（HTTP SSE / JSON / WS）。接到现有 OAuth transform 与 passthrough/WS 帧路径；未改 fork 的 `codex-auto-review→luna` 映射，也未把 origin 的 `normalizeOpenAIOAuthResponsesCompatibilityFields` 再塞进 transform（prompt 收成 input 仍走第一刀的 HTTP legacy ingress） |
| `#5925` 余下 | Grok 内容策略 403 不 failover / 不冷却；compaction blob 与 nested `encrypted_content` 信封同账号清洗重试；input 递归剥 JSON null；team+model 429 overlay + 调度过滤；stream idle 180s 可同账号再试；独立 xAI reasoning tokens 折算。未整包 origin `xai/models` 目录、`billing_service`、`grok_free_quota_gate`、failover_loop 利润否决 |
| `#5676` | OpenAI 容量降载（overloaded 文案/码）标成请求级瞬时故障：不冷却账号、同账号有界重试；空 reasoning/message 前导不算语义输出；SSE keepalive 注释不挡 failover；WS HTTP 桥 turn-1 先暂存再判定 |
| `#5729` | Chat→Responses fallback 把 `reasoning_content` 按 reasoning item id 写入 Redis（7 天 TTL）；后续历史只剩 `encrypted_content` 时按 id 回填，DeepSeek thinking 400 可过。加密-only 项走 cache；last-turn reasoning 会重放到同轮带 tool_call 的 assistant 消息。fork 只给现有 sticky-session `GatewayCache` 加两个方法，未带 origin 的 Grok-video billed extras / `mediaByCallID` |
| `#5760` 子集 | 凭据面（OAuth 换 Token / 刷新 / PAT whoami）与探针/models/ForceCodexCLI 默认身份改走 `CodexCanonical*`，去掉硬编码 `codex-cli/0.91.0` 和重复的 `openAICodexProbeVersion`。fork 没有面板 UA 解析器，规范身份 = 编译期 CLI 常量；**未**改 `enforceCodexIdentityHeaders` 为 origin 的强制改写（fork 仍是配对 + 降载归一化） |
| `#4005` | 顶栏用户角色走 `admin.users.roles.*` i18n，不再 raw `user.role` + CSS capitalize |
| `#4006` | `html`/`html.dark` 声明 `color-scheme`；日期选择器暗色日历图标不再 invert |
| `#4049` | 用户仪表盘今日/累计 token 卡补 cache 分解 |
| `#4053` | 公告空列表文案改为「还没有公告」而不是加载失败 |
| `#5697` | ops 错误分布图例同时显示 label 与 count |
| `#5715` | 账号页 proxies/groups 改为 `Promise.allSettled`，代理加载失败不挡分组筛选 |
| `#5669` | OpenAI OAuth 402 `deactivated_workspace` 联动熔断同 Team 其余 active 账户（进程内去重 60s）；fastpath 与 `HandleUpstreamError` 都先于 model-not-found / 池模式早退 |
| `#5755` | Gemini `ErrorPolicySkipped` 与 OpenAI 对齐：自定义错误码未命中且不可 failover 时对客户端隐藏上游细节（500 + 固定文案）；池模式 4xx 保真透传；可 failover 的 5xx/429 仍换号。fork 原先 Skipped 一律写 500 且仍会 `handleGeminiUpstreamError` |
| `#5721` | 批量编辑 OpenAI 设置：长上下文计费、端点能力、Responses 路由；影子账号长上下文跟随母账号并回报 inherited count。fork 长上下文 UI 覆盖 oauth/apikey/setup-token（passthrough 开关仍不含 setup-token）；未带 origin 的 ProbeEnabled 批量字段 |
| `#5636` | 账号 `status.expired` 文案：代理/分组列表 `t('admin.accounts.status.' + value)` 缺 key 时会露出 raw path。en/zh 模块 + 单体包 + zh-Hant 都补了 |
| `#5749` | 清掉 Sora 平台删除后的死引用：DTO `sora_client_enabled`、settings/overview i18n、README「暂时不可用」段、`deploy/config.example.yaml` 的 `gateway.sora_*` / 顶层 `sora:` / `sync_linked_sora_accounts`。fork Go 侧已无 Sora 结构体或路由，yaml 不会再生效；fork 另清了单体 `en.ts`/`zh.ts`/`zh-Hant.ts` |
| `#5794` | README star-history 图源改 `star-history.dera.page`（旧 `api.star-history.com` 因 GitHub API 限制挂了）。fork 没有 README_CN/JA |
| gitignore | 忽略 `.codegraph/`（origin `354825674`） |
| `#2148` | ops 错误列表弹窗在 `time_range=custom` 时改传 `start_time`/`end_time`（后端不认 custom）。fork 仪表盘 header 已有自定义时间，漏接到 `OpsErrorDetailsModal` |
| `#5839` 子集 | 创建账号 API key placeholder 抽成 `apiKeyValuePlaceholder`（openai/gemini/grok/anthropic）；未带 origin 的 kimi/zhipu/deepseek case |
| `#5888/#5925` 接线 | Grok `/responses/compact` 转摘要 turn + 回程 compaction item；stream idle 冷却；spending-limit reauth 在 quota/token-refresh 生效。修 CI unused |
| `#5838` | 编辑用户弹窗补角色 Select。fork 后端本来就收 `role`（升 admin 要 step-up），UI 漏了字段 |
| `#5738` 子集 | Codex fingerprint seed 写入 extra + `930_backfill_codex_fingerprint_seed.sql`（origin 225 remap）。开启 device/session/full 时仓储原子补 UUID；**未**改 fork 默认 session 收敛，也未整包 origin 出站 ID 派生 |

交付分支：`sync/v0179-newest-gateway` 已合入 `dev`。续做：`sync/v0179-remaining` → `--base dev`。

## 明确排除（不要当遗漏）

- 整包 merge upstream / 直推 `main` / `publish`。
- 国内部署链 `#5666` 及后续 CN quota / DeepSeek / header-override（fork 无 `PlatformComposite` / CN provider 面）。
- `#5888` 已切完本窗口要的四刀；未整包 origin transform 的 prompt 兼容函数 / namespace 其余差异。
- `#5925` 未抽：`grok_free_quota_gate`、origin `xai/models.go` 全量目录（306 vs 76）、`billing_service` 其余 reasoning 折算（2024 vs 1382）、failover_loop 利润否决整包、`setting_gateway_runtime` extras。
- `#5742` Grok 响应模型别名审计：fork 没有 `upstream_response_model.go`，不能只带 helper。
- `#5738` 已 remap 为 930 并接 UpdateExtra；未带 origin「缺省改 off」与出站 ID 全量派生（fork 仍默认 session）。
- 渠道分时价 / 档位乘数 / channel-monitor quota mode。
- `#5815` 的 Chat 原路径：fork 没有 `normalizeGrokChatReasoningEffort`，只接到了 Responses `patchGrokResponsesBody`。
- `#5708` 首页 model plaza（fork `HomeView` 已有 `/models` 导航，不是 origin 那两套 header）。
- `#5711` Antigravity mixed tool-config 透传（fork 无 `antigravity_gateway_compat.go`）。
- `#5609` 认证快照定价字段（fork 已有 v16 `ModelPricing` / `LongContextPricingEnabled`）。
- `#5716` SMTP 未配置跳过到期提醒（fork 的 `SubscriptionExpiryService` 没有 reminder 路径）。
- `#5875` 平台筛选目录：origin 把 kimi/zhipu/deepseek + composite 收成共享 catalog；fork 无 Composite / CN provider 面，整包会把排除的平台塞进筛选。
- `#5839` 已抽 fork 已有平台的 placeholder computed；未带 CN provider 的 kimi/zhipu/deepseek 占位符。
- `#2148` 已接到 fork 的 `OpsErrorDetailsModal`（列表弹窗）+ 仪表盘 props；单条 `OpsErrorDetailModal` 不拉错误列表，无需 custom 时间。
- `#5662` 只改 `docs/ADMIN_PAYMENT_INTEGRATION_API.md`，fork 无该文档。
- `#5762` usage_log GROUPING SETS + 新 SQL 索引：要 remap 900+ 迁移，且会碰 fork 已有分组用量 rollup，单独立项。
- `VERSION` / sponsors / Dockerfile Go 镜像钉。
- `#5712` Ollama Cloud 用量查询按钮：父功能是窗口外的 `#4776`（`ollama_cloud_usage` 整栈）。fork 没有 `OllamaCloudUsageCell.vue` / `refreshOllamaCloudUsage`，不能只合按钮。
- `#5913` DeepSeek Responses 账号测试、`#5911` DeepSeek relay balance、`#5842` 自适应 API 协议、`#6009`/`#6011`/`#5919`/`#5847`/`#5837`/`#5782`/`#5730` 等 CN provider 面。fork 无 `GetAPIProtocol` / `PlatformDeepseek` / `testCNProvider*`。
- `#6048` Composite messages dispatch、`#5654` Composite 视频端点、`#5817`/`#5816` Composite 新平台。fork `sanitizeGroupMessagesDispatchFields` 仍把非 openai 的 `AllowMessagesDispatch` 打成 false。
- `#5906` CN 额度探测测试加锁：只动 `cn_provider_balance_check_service_test.go`。
- `#5708` 首页 model plaza 入口：origin 往内建 compact/default header 加链接；fork `HomeView.vue` 是独立品牌首页，没有那两套 header，不能原样贴。

## 验证

| 门禁 | 结果 |
|---|---|
| `go test -tags=unit ./internal/securityaudit`（含 config_loaded） | 通过 |
| `go test -tags=unit ./internal/service -run 'TestChatCompletionsBufferedResponsesReadError\|InvitationCode\|Register_Invitation'` | 通过 |
| `go test -tags=unit ./internal/pkg/apicompat ./internal/pkg/antigravity ./internal/pkg/openai` | 通过 |
| `go vet -tags integration`（改动包） | 通过 |
| 前端 `tokenRefresh.spec.ts` | 7/7 通过 |
| `go test -tags=unit ./internal/service ./internal/handler`（#5954/#5876/#5810） | 通过（service 3.9s / handler 4.1s） |
| `go vet -tags integration`（改动包，含 input_tokens / ops SLA） | 通过 |
| `go test -tags=unit`（#5567/#5004/#5725/#5714/#5834/#5925 子集） | 通过（config / repository / antigravity / service） |
| `go vet -tags integration`（config / repository / antigravity / service） | 通过 |
| 前端 `vue-tsc --noEmit`（#4057 SLA 空窗口） | 通过 |
| `go test -tags=unit ./internal/service`（#5888 第一刀：ingress/schema/retry + Forward/Grok） | 通过 |
| `go vet -tags integration ./internal/service`（#5888 第一刀） | 通过 |
| `go test -tags=unit ./internal/service`（#5888 第二刀：Grok ModelInput / compact fallback） | 通过 |
| `go vet -tags integration ./internal/service ./internal/handler`（#5888 第二刀） | 通过 |
| `go test -tags=unit`（#5888 第三刀：WS preempt / health breaker / ops logger） | 通过（service / repository / handler 相关 `-run`） |
| `go vet -tags integration ./internal/handler ./internal/service ./internal/repository`（#5888 第三刀） | 通过 |
| 前端 `vue-tsc --noEmit` + `errorDetailResponse.spec.ts`（ops 详情弹窗） | 通过 |
| `go test -tags=unit ./internal/service`（#5888 第四刀：Codex tool-name rewrite） | 通过 |
| `go vet -tags integration ./internal/service`（#5888 第四刀） | 通过 |
| `go test -tags=unit ./internal/service ./internal/handler ./internal/pkg/xai`（#5925 余下：content-policy / compaction / idle / team 429 / usage） | 通过 |
| `go vet -tags integration ./internal/service ./internal/handler ./internal/pkg/xai`（#5925 余下） | 通过 |
| `go test -tags=unit ./internal/service`（#5676 容量降载 / 前导暂存 / keepalive） | 通过 |
| `go vet -tags integration ./internal/service ./internal/handler`（#5676） | 通过 |
| `go test -tags=unit ./internal/pkg/apicompat ./internal/repository ./internal/service`（#5729 reasoning cache / Chat fallback restore） | 通过 |
| `go vet -tags integration ./internal/pkg/apicompat ./internal/repository ./internal/service`（#5729；`internal/testutil` 全是 `//go:build unit`，integration vet 不扫） | 通过 |
| `go test -tags=unit ./internal/service ./internal/repository`（#5760 identity / OAuth / models） | 通过 |
| `go vet -tags integration ./internal/service ./internal/repository`（#5760） | 通过 |
| `go test -tags=unit ./internal/service -run TestTeamLinkedError_`（#5669） | 通过 |
| `go vet -tags integration ./internal/service`（#5669） | 通过 |
| 前端 `vue-tsc --noEmit` + `pnpm build` + `AccountsView.usageWindowsHint.spec.ts`（#4005/#4006/#4049/#4053/#5697/#5715） | 通过（3/3；UI 未开浏览器，靠 typecheck/build/vitest） |
| `go test -tags=unit ./internal/service -run TestGemini`（#5755 skipped-policy write / failover） | 通过 |
| `go vet -tags integration ./internal/service`（#5755） | 通过 |
| `go test -tags=unit ./internal/service -run TestAdminServiceBulkUpdateAccounts`（#5721） | 通过 |
| `go vet -tags integration ./internal/service`（#5721） | 通过 |
| 前端 `vue-tsc --noEmit` + `pnpm build` + `BulkEditAccountModal.spec.ts`（#5721） | 通过（30/30；批量弹窗未开浏览器，靠 typecheck/build/vitest） |
| `go test -tags=unit ./internal/handler/dto`（#5749 PublicSettings schema） | 通过 |
| `go vet -tags integration ./internal/handler`（#5749） | 通过 |
| 前端 `vue-tsc --noEmit` + `pnpm build` + i18n vitest（#5636/#5749：`accountStatusExpired` + `localesMessageCompile`） | 通过（8/8；代理列表 expired 文案未开浏览器，靠 locale 单测 + typecheck/build） |
| 前端 `opsErrorTimeParams.spec.ts` + `CreateAccountModal.grok.spec.ts` + `vue-tsc` + `pnpm build`（#2148/#5839） | 通过（7/7 相关；custom 时间弹窗未开浏览器，靠单测 + typecheck/build） |
| 前端 `UserEditModal.spec.ts` + `vue-tsc`（#5838） | 通过（3/3） |
| `go test -tags=unit` fingerprint seed / account_repo SQL helper（#5738 子集） | 通过 |
| `go vet -tags integration ./internal/service ./internal/repository`（#5738 子集） | 通过 |
| 全量 `go test ./...` | 未跑（既有时长问题） |

## 相关

- 工作流：[`../standards/workflow.md`](../standards/workflow.md)
- 上一轮：[`upstream-sync-v0177-codex-compact-group-usage.md`](upstream-sync-v0177-codex-compact-group-usage.md)
