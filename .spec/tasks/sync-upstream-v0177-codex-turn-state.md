---
status: completed
---

# T1 — Codex turn-state 透传与跨账号守卫

## 做什么

把上游 `#5668` 的 `x-codex-turn-state` 回传、溯源与 failover 换号剥离带进 `dev`。指纹收敛只补暂存 / 透传 raw 改写，**默认模式仍是 `session`**。

## 涉及范围

- `backend/internal/service/openai_codex_turn_state.go`（新增）
- `backend/internal/service/openai_codex_turn_state_test.go`（新增）
- `backend/internal/service/openai_codex_fingerprint.go`
- `backend/internal/service/openai_gateway_service.go`
- `backend/internal/service/openai_gateway_forward.go`
- `backend/internal/service/openai_gateway_passthrough.go`
- `backend/internal/service/openai_gateway_response_handling.go`

## 验收标准

- [x] 上游响应里的 `x-codex-turn-state` 会写回下游；首输出守卫路径只在真正提交时记溯源。
- [x] 同一下游会话 failover 换号后，出站会剥离已知由其他账号铸造的回带值。
- [x] 上游本次响应没有该头时，会清掉 writer 上可能残留的上一 attempt 值。
- [x] `GetCodexFingerprintMode` 未设置 extra 键时仍返回 `session`，测试不断言 `off` 为默认。
- [x] `go test` 覆盖 turn-state 单测与相关 gateway 包。

## 依赖

无。
