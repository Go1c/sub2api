---
status: completed
---

# Fix Codex Models for CPA Upstreams

## Task

修复 OpenAI API Key 账号配置自定义 `base_url`（CPA/CLIProxyAPI）时，Codex 模型清单错误读取 OAuth token 并直连 ChatGPT 的问题；保留 OAuth 账号现有直连行为。

## Scope

- `backend/internal/service/openai_codex_models_service.go`
- `backend/internal/service/openai_codex_models_service_test.go`
- 必要时最小修改 `backend/internal/handler/openai_codex_models_handler.go` 及对应测试，以实现能力账号选择/一次 failover。

## Acceptance

- API Key + 自定义 `base_url` 使用 `Authorization: Bearer <api_key>` 请求规范化后的上游 `/models?client_version=...`，不要求 `access_token`，不访问 ChatGPT。
- OAuth + `access_token` 继续请求 ChatGPT Codex models endpoint，并保留现有 Codex headers。
- 缺少可用 OAuth token 或 API Key/custom base URL 时返回准确的 `TOKEN_MISSING` 或 `UPSTREAM_NOT_CONFIGURED` 错误。
- API Key 上游的响应 body、ETag 与 304 行为正确透传。
- 优先复用现有 base URL 校验/URL 拼接/HTTP client 约定；不泄露凭证，不做无关重构。
- TDD 记录红→绿；相关 unit tests、`gofmt`、`git diff --check` 通过，并尽可能运行仓库要求的 integration-tag 校验。

## Upstream Review

- `upstream/main` 仅新增 `resolveCredentialAccount` 与 ChatGPT account headers，仍强制 OAuth token 并直连 ChatGPT；截至 2026-07-12 未发现覆盖 CPA Codex models 的已合并或未合并 PR。

## Knowledge

纯 bugfix，沿用现有 OpenAI-compatible `base_url` 转发模式；若实现未引入新设计模式，可不新增 knowledge 文档。

## Results

- TDD 红灯：`go test -json -tags=unit ./internal/service -run 'TestFetchCodexModelsManifestAPIKeyCustomBaseURL' -count=1` 稳定复现 API Key + CPA 账号错误返回 `OPENAI_CODEX_MODELS_TOKEN_MISSING`。
- API Key + 显式自定义 `base_url` 现经 `validateUpstreamBaseURL` 校验，并用 `buildOpenAIEndpointURL(..., "/v1/models")` 拼接 CPA models 路径；使用 Bearer API key，透传 `client_version`、`If-None-Match`、ETag、body 与 304。
- OAuth + access token 保持 ChatGPT `/backend-api/codex/models`、Codex headers 与 account header 行为。
- API Key 无自定义 `base_url`、OAuth/API Key 缺凭证返回精确的 `UPSTREAM_NOT_CONFIGURED` / `TOKEN_MISSING` reason 与 message。
- Handler 对上述两类账号配置错误最多排除首个账号并重选一次；网络或上游响应错误不盲目 failover。
- 通过：
  - `go test -tags=unit ./...`
  - `go test -tags=integration ./...`
  - `go test -tags=unit ./internal/service`
  - `go test -tags=unit ./internal/handler`
  - `go test -tags=unit ./internal/server/routes`
  - `go test -tags=integration ./internal/service ./internal/handler ./internal/server/routes`
  - `go vet -tags integration ./...`
  - `gofmt`（改动 Go 文件）
  - `git diff --check`
- 未运行：`golangci-lint run ./internal/service ./internal/handler ./internal/server/routes`，当前环境未安装 `golangci-lint`（`command not found`）。
