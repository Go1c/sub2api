---
name: openai-first-token-streaming-20260706
description: OpenAI/Responses 流式首 token 口径修复记录——terminal/结构事件不能写入 first_token_ms，真实首 token 以第一个非空 delta 为准。
metadata:
  type: record
  date: 2026-07-06
  status: 归档
---

# OpenAI 流式首 Token 口径修复 — 2026-07-06

## 结论

GPT/OpenAI Responses 流式链路的 `first_token_ms` 必须在第一个真实输出增量到达时记录，也就是第一个带非空 `delta` 的事件。以下事件不得写入首 token：

- `response.completed` / `response.done` / `response.failed` / `response.incomplete` / `response.cancelled` / `response.canceled`
- `response.created` / `response.in_progress`
- `response.output_item.added` / `response.output_item.done` 等结构事件
- 空 `delta`

如果上游只返回 terminal/usage 而没有任何真实 delta，本次请求应保留 `first_token_ms = nil`，不能把总耗时误报成首 token。

## 触发场景

本地请求 dev 部署 A，A 路由到 publish 部署 B：`A -> A 路由账号 -> B`。B 自己记录正常，但 A 的下游用量日志出现 `first_token_ms == duration_ms`。根因在 A 的下游转发层把 terminal 事件当作首 token，导致没有可识别增量时在流结束处打点。

## 实现要求

- HTTP Responses 原生与 passthrough 流都只在第一个非空 `.delta` 上记录 `firstTokenMs`。
- 第一个真实 delta 到达时必须立即 flush，保证用户体感的第一个字与日志首 token 口径一致。
- terminal-only 流仍要透传最终事件与 usage，但不产生首 token。
- WS v2 relay 的 token/terminal 分类必须互斥，并且实际打点要求 payload 里有非空 `delta`。
