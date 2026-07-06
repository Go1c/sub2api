---
name: openai-http-stream-ttft-forwarding-20260706
description: GPT/OpenAI HTTP 流式转发首 token 修复记录——publish 转发层只在非空输出 delta 记 TTFT 并立即 flush,结构/终止事件不算。
metadata:
  type: record
  date: 2026-07-06
  status: 归档
---

# GPT/OpenAI HTTP 流式转发 TTFT 修复记录（2026-07-06）

## 背景

线上 A(dev) → B(publish) 级联时,GPT 5.5 的下游记录出现 `first_token_ms` 接近 `duration_ms`。问题不只在“自动透传”开关:普通 OpenAI Responses 转发、OpenAI passthrough、Chat Completions 兼容转换、`/v1/messages` 兼容转换都可能受影响。

本轮排查确认不能直接复制下一跳日志。A/B 每一跳都必须按自己收到并转发的 SSE 流重新测量 TTFT。publish 转发层如果把 `response.output_item.added`、`response.created`、`message_start` 或终止事件当成“输出开始”,就会污染本层 TTFT,并可能导致首个真实 delta 没有被立即 flush 给下一跳。

## 结论

HTTP 流式首 token 统一按“下游会收到的第一个非空输出 delta”记录:

- OpenAI Responses 计入:
  - `response.output_text.delta` 且 `delta != ""`
  - `response.reasoning_summary_text.delta` 且 `delta != ""`
  - `response.function_call_arguments.delta` 且 `delta != ""`
- Chat Completions 计入:
  - `delta.content`
  - `delta.reasoning_content`
  - `tool_calls[].function.arguments`
- Anthropic SSE 计入:
  - `content_block_delta` 的 `text_delta.text`
  - `thinking_delta.thinking`
  - `input_json_delta.partial_json`

以下事件不计 TTFT:

- `response.created` / `response.in_progress`
- `response.output_item.added` / `response.output_item.done`
- `response.completed` / `response.done` / `[DONE]`
- `message_start` / `content_block_start` / `message_delta` / `message_stop`
- usage-only / finish-only / role-only chunk

对于 publish 转发层,首个真实输出 delta 到达时必须立即 flush,不能因为之前结构事件已经写过内部状态而跳过首字 flush。

## 验证

- 新增回归测试覆盖:
  - OpenAI Responses 普通转发与 passthrough: `response.output_item.added` 先到,`response.output_text.delta` 延迟到,TTFT 必须等到 delta。
  - `/v1/messages` OpenAI 兼容转换: `response.created` / `output_item.added` 不算首 token。
  - Anthropic → Responses / Chat 转换: `message_start` / `content_block_start` 不算首 token。
  - Anthropic 原生透传与 API Key passthrough: `message_start` 不算首 token。
- 本地验证:
  - `go test -tags=unit ./internal/service`
  - `go test -tags=unit ./...`
  - `go test -tags=integration ./...`
  - `go vet -tags integration ./...`
  - `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0 run ./... --timeout=30m`
