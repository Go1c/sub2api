---
status: completed
---

# OpenAI Model Test Codex Headers

## Task

让普通 OpenAI 模型连通性测试为 API Key 与 OAuth 请求统一携带 Codex Responses 请求头，并统一 Codex 探针版本常量。

## Acceptance

- API Key 模型测试请求携带 `Accept: text/event-stream`、`OpenAI-Beta: responses=experimental`、`Originator: codex_cli_rs`、`Version: 0.144.0`、`User-Agent: codex_cli_rs/0.144.0`。
- OAuth 普通模型测试沿用相同 Codex 请求头。
- 账户配置的非空自定义 `user_agent` 优先于默认 Codex User-Agent。
- Codex CLI 全局版本和用量探针版本统一为 `0.144.0`，不增加动态 metadata/session headers。

## Verification

- 先运行新增 API Key header 测试，确认修复前失败。
- 运行最小 targeted unit tests。
- 运行 `gofmt` 与 `git diff --check`。

## Knowledge

纯 bugfix，沿用现有 Codex Responses 请求头约定，不更新 knowledge。

## Verification Results

- 红：新增 API Key header 测试修复前失败，`Accept` 实际为空。
- 绿：`go test -tags=unit ./internal/service -run '^TestAccountTestService_OpenAIAPIKeyModelTestUsesCodexHeaders$' -count=1` 通过。
- 相关 Go 文件已运行 `gofmt`，`git diff --check` 通过。
