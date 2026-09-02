---
status: completed
---

# 任务：国模第二刀（网关转发）

依赖第一刀常量 / Account.GetAPIProtocol / IsCNProvider。把 origin 的 CN 网关转发迁到 fork。

## 做什么

从 origin/main HEAD 移植并适配：

- `openai_gateway_messages_anthropic_native.go`（含 #5919 reasoning_effort）
- `openai_gateway_chat_completions_anthropic_native.go`
- `openai_gateway_responses_anthropic_native.go`
- `openai_gateway_anthropic_native_pump.go`（origin HEAD 后续抽出的 native 流泵，901a 初版没有）
- 现有 `openai_gateway_{messages,chat_completions,forward,request_body,count_tokens,scheduling,passthrough}.go` 的 CN 钩子
- origin 后续 #5730 五项 CN 分组/计费/断开/count_tokens 修复（网关部分）
- origin #5842 adaptive 协议的网关侧（account_test_service 可放到第四刀）

不要整文件覆盖 fork 的 gateway（fork 已有 sticky/WS/Grok/Luna 适配）。

## 验收标准

- [ ] CN anthropic 协议账号走 native `/v1/messages` 直通
- [ ] chat_completions / responses / adaptive 按 origin 语义分流
- [ ] `go test -tags=unit ./internal/service ./internal/handler` 相关测试绿
- [ ] `go vet -tags integration` 改动包
- [ ] 不碰 docker-compose、支付、publish

## 依赖

v0179-cn-1-constants-schema-probe
