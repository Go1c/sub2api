---
status: completed
---

# T2 — native remote compaction v2

## 做什么

把上游 `#5641` 的 native compaction v2 线型带进 `dev`：流式 `/responses` + `compaction_trigger` 不再提升到 legacy `/responses/compact`；探测与出站协商头对齐真实 Codex。

## 涉及范围

- `backend/internal/service/openai_compact_body_signal.go`
- `backend/internal/service/openai_compaction_context.go`（新增）
- `backend/internal/service/openai_compact_probe.go`
- `backend/internal/service/account_test_service.go`
- `backend/internal/handler/openai_gateway_handler.go`
- `backend/internal/handler/openai_gateway_compact_body_signal_test.go`
- `backend/internal/service/openai_gateway_scheduling.go`
- `backend/internal/service/openai_ws_forwarder_payload.go`

## 验收标准

- [x] 裸 `/responses` + `stream:true` + `compaction_trigger` 保持原路径，不追加 `/compact`。
- [x] 非流式 / 无 trigger 的旧 body-signal 仍可提升到 legacy `/responses/compact`。
- [x] compact 探测打流式 `/responses`，2xx 且无 compaction 输出项记为不支持。
- [x] 出站在原生 v2 或 OAuth 未声明时补注 `x-codex-beta-features: remote_compaction_v2`。
- [x] native v2 不走 `compact_model_mapping`；渠道限制用转发后的模型。
- [x] compact body-signal / probe 相关测试通过。

## 依赖

T1（共享 beta-feature 与指纹暂存辅助函数）。
