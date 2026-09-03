---
name: openai-capacity-shed-retry
description: OpenAI/Codex 容量降载（overloaded）的网关重试：同账号只静默再打 1 次再换号，避免连打把 RPM 打光，也避免立刻回客户端导致重连
metadata:
  type: doc
  level: L2
  status: 已交付
---

# OpenAI 容量降载重试

简介：上游返回 `server_is_overloaded` / `slow_down` / “servers are currently overloaded” 时，网关在当前请求内静默恢复，不把降载码原样丢给 Codex。

## 背景 / 目标

- 默认池模式同账号重试 3 次（加首次共 4 次）会在降载时把 ChatGPT/Codex 短时限流打满，后续变成 `Rate limit exceeded`。
- 若网关完全不重试、立刻回客户端：HTTP 会走 Codex 内置重试，WS 会 `TryAgainLater` 重连，用户有感。
- 目标：偶发降载对用户无感；每个账号最多额外打 1 次。

## 设计

- **实现面**：`applyOpenAICapacityShedLimitedRetry` 把降载标成请求级瞬时故障（不冷却账号）。`SameAccountRetryMax = 1`，间隔沿用 handler 默认 500ms；同账号用尽后仍可换号（同一 HTTP/WS 请求未结束）。
- **交互面**：尚未写出语义输出时，客户端一直等首包。整池耗尽才回 `503` + `code=server_error`（Codex 对 `server_is_overloaded` 判致命，必须改写）。WS 耗尽才关连接。
- **设计面**：降载通常是上游容量问题，换号救不了成片过载，但能挡住单号毛刺且不把错误提前抛给客户端。

## 已决策

- 同账号只多打 1 次，不再用默认 3 次。连打 4 次会烧 RPM。
- 允许换号，不在第一次失败就 `NextAccountStop`。否则 WS 会立刻重连，用户有感。
- 账号不因降载被标过载/摘号。这是请求级压力，不是号坏了。
- 耗尽后回 `server_error`，保留 overloaded 原文。

## 待解决

- 若日志里大量「第一次降载、隔几秒再打就成功」，再考虑把静默间隔从 500ms 拉长。

## 相关

- `backend/internal/service/openai_gateway_upstream_errors.go`（`applyOpenAICapacityShedLimitedRetry`）
- `backend/internal/service/openai_gateway_passthrough.go`（流内降载码改写为 `server_error`）
- `backend/internal/handler/openai_gateway_handler.go`（耗尽回包 / WS 关闭）
