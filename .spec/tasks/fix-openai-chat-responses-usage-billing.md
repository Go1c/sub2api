---
status: completed
---

# Fix OpenAI Chat To Responses Usage Billing

## Task

修复 OpenAI API Key 账号把入站 `/v1/chat/completions` 转成上游 `/v1/responses` 时，流式终止事件 usage 兼容形状未被计费链路解析、图片输出未计数的问题。

## Acceptance

- Chat Completions 转 Responses 的流式与 buffered 路径复用直通 Responses 的 usage 提取口径。
- 支持 `response.usage`、顶层 `usage`、`data.usage`、`data.response.usage` 及 `prompt_tokens` / `completion_tokens` 别名。
- 支持 SSE `event:` 提供事件类型、JSON body 缺少 `type` 的终止事件。
- 流式 Chat Completions 转 Responses 对 `image_generation_call` 输出设置 `ImageCount`。
- `copyOpenAIUsageFromResponsesUsage` 保留 cache 语义并复制 image input/output tokens。
- 终止事件缺少 usage 时仍成功返回零 token，不推算 usage。

## Verification

- 先运行新增 focused tests，确认兼容包装 usage、image count 与 image token copy 在修复前失败。
- 运行相关 service unit tests、Go format 与 `git diff --check`。
- 按仓库政策运行后端 unit、integration、vet 与 lint；环境性失败需如实记录。

## Knowledge

纯 bugfix，复用既有 Responses usage 解析与图片计数约定，不更新 knowledge。

## Operations Handoff

运维侧对 CPA-pro-0.18（账号 113）的“强制 Chat Completions”仅用于止血；代码合入并验证 `/v1/chat/completions` 调度到账号 113 后 token 不再为 0，才可改回自动探测。

## Verification Results

- 红：`data.usage` / `data.response.usage`、buffered/messages 包装 usage、image token copy 与 `ImageCount` 用例均按预期失败为 0。
- 绿：`go test -tags=unit ./internal/pkg/apicompat ./internal/service` 通过。
- `go test -tags=integration ./...` 通过。
- `go vet -tags integration ./...` 通过。
- `golangci-lint v2.9.0 --new ./...` 通过（0 issues）。
- 全量 unit 被当前仓库 `internal/server/TestAPIContracts` 的 settings contract 基线差异阻断（期望空模型，实际 `gpt-5.4`）；隔离运行仍可复现。
- 全量 lint 被当前仓库既有测试文件的 SA5011 告警阻断；本次 diff 的 v2.9 `--new` 检查为 0 issues。
- `gofmt` 与 `git diff --check` 通过。
