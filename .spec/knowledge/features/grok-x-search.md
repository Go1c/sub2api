---
name: grok-x-search
description: Grok x_search 三条路径（独立 /v1/x_search、Chat tools、Responses tools）的现网契约、抽取、计费与管理端按次价
metadata:
  type: doc
  level: L2
  status: 已实现
---

# Grok x_search

Grok 文本搜索走 xAI Responses 的 `tools: [{type:"x_search"}]`。独立搜索、Chat Completions 嵌入、Responses 嵌入三条路径都要落到这条上游契约，不能整包合 `main` 的独立搜索实现。

## 背景 / 目标

- 现网探测：`POST /v1/responses` 带 `x_search` 可用；`include: ["x_search_call.action.sources"]` 会被 xAI 以 HTTP 400 拒绝；独立 `POST /v1/x_search` 因此 502；Chat Completions `tools: [{type:"x_search"}]` 因桥拒绝非 function 工具而走 raw Chat，上游 422。
- `origin/main` 的独立搜索仍带被拒的 `include`，Chat 桥也只接受 `type: function`。fork 已有独立 handler + `DoGrokNativeResponsesJSON`，只补契约缺口。

## 设计

- **独立搜索**：`POST /v1/x_search` 与 `POST /x_search`。`buildGrokXSearchResponsesBody` 不发 `include`。`DoGrokNativeResponsesJSON` 与 `patchGrokResponsesBody` 再剥一次 `x_search_call.action.sources`，其它 include（如 `reasoning.encrypted_content`）保留。
- **抽取**：优先用消息里的结构化 `{"results":[...]}`，URL 必须落在 native `web_search_call` / `x_search_call` / `custom_tool_call` 的 `action.sources` 或 `url_citation` 上。现网常见形态是 `custom_tool_call`（`x_keyword_search` / `x_user_search`）+ 消息 `url_citation`。没有 native 源列表时，直接采用结构化 JSON 结果。
- **Chat 桥**：`grokChatFunctionDeclarationsBridgeable` 接受 `type: x_search` 以及 handles / 日期 / 图视频字段；对象 `tool_choice: {"type":"x_search"}` 可桥。无 tools 时 `tool_choice` 为 x_search 则回退 `x_search_tool_choice_without_tools`。转换层 `ChatCompletionsToResponses` 已保留 `x_search`。
- **计费**：独立搜索成功后按 `grok-x-search`、`WebSearchCalls=1` 走 `CalculateWebSearchCost`。默认官方 $0.01/次；分组可覆盖 `web_search_price_per_call`（nil 用默认，0 免费）。倍率走 `web_search_rate_independent` / `web_search_rate_multiplier`：独立开启用搜索独立倍率，否则用不含高峰的分组有效倍率（用户专属 > 分组 `rate_multiplier` > 系统默认）。嵌入 Responses/Chat 的 `x_search` 仍按 Grok 文本 token 计。
- **管理端**：分组创建/编辑在 `platform === 'openai' || platform === 'grok'` 时显示按次搜索价 + 独立倍率开关。

## 已决策

- 不整包 merge `main` 的 `openai_x_search.go`；fork 独立 handler 不复用 `forwardGrokResponses`（会把上游 Responses 直接写给客户端）。
- 独立搜索默认模型继续走运行时 `ResolveDefaultTextModel`，计费模型固定 `grok-x-search`。
- 嵌入搜索不改成按次，避免把一次对话里的文本 token 和搜索次费叠两次口径。
- 搜索倍率独立于 token `rate_multiplier`，对齐 image/video：独立开启时不乘分组倍率和高峰因子。

## 待解决

- 无。

## 相关

- 同步记录：[`../records/upstream-sync-grok-v0176-jwt-xsearch-pricing.md`](../records/upstream-sync-grok-v0176-jwt-xsearch-pricing.md)
- `backend/internal/handler/openai_x_search.go`
- `backend/internal/service/openai_gateway_grok_native.go`
- `backend/internal/service/openai_gateway_grok_chat_bridge.go`
- `backend/ent/schema/group.go`
- `backend/migrations/940_group_web_search_rate_controls.sql`
- `backend/internal/service/image_billing_multiplier.go`
- `frontend/src/views/admin/groupsWebSearchPricing.ts`
