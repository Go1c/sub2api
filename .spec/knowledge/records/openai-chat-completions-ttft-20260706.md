---
name: openai-chat-completions-ttft-20260706
description: GPT Chat Completions 流式首 token 统计修复记录——role / finish / usage 不算 TTFT,非空输出 delta 才算。
metadata:
  type: record
  date: 2026-07-06
  status: 归档
---

# GPT Chat Completions TTFT 修复记录（2026-07-06）

## 背景

线上 dev → publish 级联时,GPT 分组下游使用记录出现 `first_token_ms` 接近或等于 `duration_ms`。前一版修复误改了更宽的 Responses / WSv2 路径,发布后导致 publish 上游也出现体验回归,已先通过 PR #116 回滚。

## 结论

本次窄修复只覆盖 GPT Chat Completions 两条实际链路:

- raw `/v1/chat/completions` 透传路径。
- Responses → Chat Completions 兼容转换路径。

TTFT 只在下游实际会收到非空输出增量时记录:

- 计入: `delta.content`、`delta.reasoning_content`、`tool_calls[].function.arguments`。
- 不计入: `delta.role`、空 delta、finish-only chunk、usage-only chunk、`response.created` / `response.in_progress` 等非输出事件。

这样记录值和流式实际首字体验保持一致:role / terminal / usage 可以照常透传,但不能污染首 token 指标。

## 验证

- 先写失败测试复现:
  - role-only chunk 提前到达,content delta 延迟到达时,TTFT 必须等到 content delta。
  - finish-only + usage 流不得写入 `first_token_ms`。
  - `response.created` 转出的 role chunk 不算 TTFT,首个 `response.output_text.delta` 才算。
- 本地验证:
  - `go test -tags=unit ./internal/service -run 'TestForwardAsRawChatCompletions_FirstTokenWaitsForOutputDelta|TestForwardAsRawChatCompletions_FinishOnlyDoesNotSetFirstToken|TestForwardAsChatCompletions_FirstTokenWaitsForOutputDelta|TestOpenAIChatStreamPayloadHasOutputDelta|TestIsOpenAIChatUsageOnlyStreamChunk'`
  - `go test -tags=unit ./internal/service`
