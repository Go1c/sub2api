---
name: upstream-sync-grok-v0176-jwt-xsearch-pricing
description: Grok 主题选择性同步（JWT 档位、独立 /x_search、分组 model_pricing、SuperGrokPro、用量快照、4.5/4.6 官方价卡）——范围、fork 适配、排除项与验证。
metadata:
  type: record
  date: 2026-08-13
  status: 已实现，待 Review 合入 dev
---

# Grok 主题同步 — JWT / x_search / 分组定价 / SuperGrokPro（2026-08-13）

## 目标与策略

从 `origin/main` / upstream 按主题 port 剩余 Grok 能力到 fork `dev`，**以上游实现为主**，不做整包 merge。fork 已有的支付、订阅、OpenAI 网关和 grok-4.6 官方回退价卡保留。交付走 topic 分支 `sync/v0176-grok-jwt-xsearch-pricing` → `--base dev` 的 Review PR。

## 已纳入范围

- **JWT 订阅档位**（已单独提交 `d6cc299f3`）：从 Grok Build access token 解码 `tier`，刷新后覆盖失效订阅。
- **grok-4.6 目录与官方价卡**（已单独提交 `0f9883ffb`）：`/models` 接入 4.6 别名；独立回退价 `$2 / $0.50 cache read / $6`，≥200k 全项 2×；保留 `reasoning_effort`。
- **独立 `/v1/x_search` 与 `/x_search`**：不复用 `forwardGrokResponses`（该函数会写客户端）。新增 `DoGrokNativeResponsesJSON` 拉非流式 Responses JSON，handler 抽出 `x_search` / `web_search` sources 后返回统一搜索结果。独立搜索默认模型硬编码 `grok-4.5`，计费模型 `grok-x-search`。
- **Chat ↔ Responses `x_search` 工具桥**：声明与回传保留 `type: x_search`，以及 handles / 日期 / 图视频理解字段。
- **分组 `model_pricing` + `long_context_pricing_enabled`**：migration `923_group_model_pricing.sql`；解析链 **Group → Channel → LiteLLM → Fallback**。分组 token 价卡只覆盖首档/一口价，区间 `Intervals` 在 resolver 剥离（管理端 `hide-token-intervals`）。
- **鉴权缓存 v16**：快照写入分组 `ModelPricing` 与 `LongContextPricingEnabled`。
- **SuperGrokPro 消歧**：`NormalizeSubscriptionTier` 识别 `supergrok_pro`；`CanonicalGrokPlan` 在 JWT 已先落地后，对模糊付费档（`supergrok` / `supergrok_pro` / `paid` / `pro`）用 **新鲜的 grok-4.5 Responses 窗口**（8300 req / 53M tokens → Heavy，否则 SuperGrok）。月限额 $150/$1500 仍优先。
- **用量快照 extra key 保持 fork 名 `grok_usage_snapshot`**（不是 main 的 `grok_quota_snapshot`）。`stampGrokQuotaSnapshotForPlan` 用 `StripGrokProviderPrefix` 盖模型（fork 无 `ResolveGrokTextResponsesModelID`）。
- **网关盖章**：Responses / Chat raw / chat bridge / Messages / CC pipeline / media generate / WS HTTP bridge / native search 成功路径写入请求模型上下文后再 `updateGrokUsageFromResponse`。
- **计费**：grok-4.5 补官方 200k / 2× 长上下文阶梯；未知文本族（`grok-5`、带日期快照、`xai/` 前缀等）回退到 4.5 价卡，避免新模型零计费。cache read 维持 fork `$0.50`，不跟 main `$0.30`。
- **管理端**：账号列表优先读 `grok_usage_snapshot`（兼容旧 `grok_quota_snapshot`）；`buildGrokUsageRefreshKey` 驱动自动刷新；分组页可配 `model_pricing`。
- **路由配套**：geo-block / embed SPA bypass / inbound endpoint 归一化 / prompt-audit 覆盖均登记 `/x_search`。

同分支上另有两笔已存在的 cherry-pick，不是本主题新写、但已在 ahead of `origin/dev` 的历史上：

- `1cc458cd7` — 探测 incomplete/failed 不再误标「上游不支持 Responses」（#5371）
- `e627cd6d3` — 渠道定价冲突检测与 cache key 归一化对齐

## 明确排除（不要在 Review 里当遗漏）

- 整包 merge upstream / 直推 `main` / `publish`。
- Voice / audio 分组列、`MaxReasoningEffort`、利润控制、`VideoModelPrices`、`AllowLive`、`HasIdentifiedTokenPricing`。
- 全账号 model-quota-block / `engine_overloaded` / `persistGrokTransientModelCooldown`。
- main 独有的 `LongContextThresholdInclusive`。
- 独立 `/x_search` 复用 `forwardGrokResponses`（会把上游 Responses 直接写给客户端）。
- 覆盖 fork grok-4.6 官方回退价卡。

## fork 适配决策

| 点 | 决定 |
|---|---|
| 搜索默认模型 | 硬编码 `grok-4.5`，不跟账号映射绕一圈 |
| 快照 extra key | 继续 `grok_usage_snapshot` |
| 4.5 cache read | `$0.50`（官方/fork），不是 main `$0.30` |
| 4.5 长上下文 | 采用官方 200k / 2×（上游优先） |
| 模型 stamp | `StripGrokProviderPrefix`，不引入 main 的 Resolve helper |
| 长上下文账号开关 | `openAILongContextBillingGate` 对非 OpenAI 返回 nil，Grok 只看分组开关 |
| SuperGrokPro | JWT 数字档优先；模糊付费档才看 4.5 Responses 窗口 |

## 验证

| 门禁 | 结果 |
|---|---|
| `frontend` `pnpm typecheck` + `pnpm build` | 通过 |
| 前端单测：`accountUsageRefresh` + `AccountsView.sparkShadow` | 18/18 通过 |
| 后端相关包 unit（`xai` / `apicompat` / `handler` / `service` / `routes` / `web`） | 通过（`service` ~149s） |
| `go vet -tags integration`（改动包含 ent / repository / middleware） | 通过 |
| `golangci-lint` | 本机未安装，未跑 |
| 全量 `go test ./...` 并行 | 先前约 11m 被杀，归类为既有时长问题，不作为本主题回归 |
| 管理端 Groups 定价 UI | 无浏览器工具；靠 typecheck / 单测与代码审查 |

## 交付约束

- PR：`gh pr create --repo Go1c/sub2api --base dev`。
- 不 merge、不推 `main` / `publish`。
- 合入后再考虑 `release/dev-to-publish-*` promotion。

## 相关

- 前一轮 Grok 同步：[`upstream-sync-grok-20260711.md`](upstream-sync-grok-20260711.md)
- Grok API Key 透传：[`grok-apikey-upstream-20260711.md`](grok-apikey-upstream-20260711.md)
- 工作流：[`../standards/workflow.md`](../standards/workflow.md)
