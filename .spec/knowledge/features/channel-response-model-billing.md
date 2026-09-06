---
name: channel-response-model-billing
description: 渠道计费基准「按上游响应模型计费」：只降不升、仅确定性识别、媒体请求不采纳
metadata:
  type: doc
  level: L2
  status: 已实现
---

# 按上游响应模型计费

渠道可以把 `billing_model_source` 设为 `response_model`，按上游成功响应里自报的模型名计价，而不是按渠道映射名或发往上游的最终模型名。这是从 origin `main` 选择性移植的能力，不是整包 merge。

## 背景 / 目标

- origin（Wei-Shaw/sub2api）在 2026-08 增加了观察器 + `billing_model_source=response_model`：上游在 SSE/JSON 里声明实际服务了哪个模型时，渠道可以选择按该名字查价。
- fork 此前只有三项：`channel_mapped`（默认）、`requested`、`upstream`（「以最终模型计费」= 账号映射后**发出去**的名字，不是响应 JSON 里的名字）。
- 目的：在可信上游会运行时降级（例如 Opus → Sonnet）时，允许按更便宜的响应模型收费；绝不让上游自报一个更贵的名字抬高用户费用。

## 设计

- **交互面**：渠道表单计费基准增加第四项「按上游响应模型计费」。
- **观察**：每次转发在 gin context 上挂 `upstreamResponseModelObserver`，在把响应 `model` 改写成客户端可见名**之前**读取 Anthropic / OpenAI / Gemini 声明。一次请求内声明冲突则打 conflict，计费回落基线。
- **限制预检**：调度时还看不到响应，`response_model` 的 Restrict Models 预检仍用渠道映射名（与 `channel_mapped` 相同）。
- **计费准入**（全部满足才采纳，否则静默回落开启本模式前的基线）：
  1. 渠道显式选择 `response_model`；
  2. 无 in-stream 冲突；
  3. 不是图片 / 视频 / 网页搜索这类按次按量请求；
  4. 响应模型能被价格表**确定性识别**（精确键 / 已知别名 / 剥日期后缀；**禁止** `GetModelPricing` 的系列子串兜底，避免伪造 `haiku` 名落到系列价）；
  5. 重算成本不得更贵（epsilon 吸收浮点末位），不得把本应计费的请求归零，不得从渠道定价切到全局价格表。
- **落盘**：`usage_logs.upstream_response_model`、`upstream_model_mismatch`。请求/发送模型字段不因计费切换改写。
- **不移植**：origin 后续「fast/priority 按上游响应档位只降不升」未纳入本次。

## 已决策

- 从 origin 选择性移植观察器 + 计费门 + 渠道第四项，不整包 merge `main`。
- 限制预检在响应到来前使用渠道映射名。
- 确定性识别必须走 `HasIdentifiedTokenPricing` / `GetIdentifiedModelPricing`，不用家族兜底。

## 待解决

- 无。

## 相关

- origin PRs：`#5396` 观察器、`#5439` `billing_model_source=response_model`、`#5742` Grok `-build` 审计别名。
- 代码：`backend/internal/service/upstream_response_model.go`、`response_model_billing.go`、`gateway_usage_billing.go`、`openai_gateway_usage.go`。
