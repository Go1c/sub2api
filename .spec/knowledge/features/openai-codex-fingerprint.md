---
name: openai-codex-fingerprint
description: 网关编译期 Codex CLI 规范身份与绑机指纹（0.153.4 实测）；改 UA / version / session 常量或对照 live 抓包时查
metadata:
  type: doc
  level: L2
  status: 已交付
---

# OpenAI Codex 编译期身份 / 绑机指纹

简介：网关出站 Codex 规范身份跟 **CLI**（无 trailer），绑机 installation / session 取自本机 `~/.codex`。当前编译期值来自本机 **OpenAI Codex CLI v0.153.4** 实测，不是 TUI/exec。

## 背景 / 目标

- 上游按 User-Agent、originator、version、installation / session 识别客户端。编译期常量必须跟真实 CLI 抓包对齐，避免被当成非官方客户端。
- 规范身份是 **CLI / plugin / models / TUI HEAD 预检**，不是带 trailer 的 TUI 或 exec。
- 本机实测环境：iTerm.app 3.6.8，Mac OS 26.5.2，arm64，`codex --version` = `codex-cli 0.153.4`。抓包根：`/tmp/codex-fp-0.153.4/mitm_captures/`（mitmproxy 10.4.2）。

## 设计

### 编译期规范身份

| 常量 | 文件 | 值 |
|------|------|----|
| `codexCLIVersion` | `backend/internal/service/openai_gateway_service.go` | `0.153.4` |
| `codexCLIUserAgent` | 同上 | `codex_cli_rs/0.153.4 (Mac OS 26.5.2; arm64) iTerm.app/3.6.8` |
| `boundCodexInstallationID` | `backend/internal/service/openai_codex_fingerprint.go` | `12b0b072-d79b-45f9-98af-8fafbe3ef9f5`（`~/.codex/installation_id`） |
| `boundCodexSessionID` | 同上 | `01a0755d-dbf8-7fb1-8295-17dbb9e447f2`（本机 0.153.4 TUI user 会话 seq 67） |

CLI UA 规则：

- OS 段是 `"Mac OS"`，不是 `"Mac OS X"`。
- 终端段取 `TERM_PROGRAM`/`TERM_PROGRAM_VERSION` = `iTerm.app/3.6.8`，**不是** 二进制终端表显示名 `iTerm2`。
- CLI **没有** `(codex_cli_rs; 0.153.4)` trailer；trailer 只属于 TUI/exec。
- originator 是 `codex_cli_rs`。

本机各客户端对照（规范身份只用第一行）：

| 客户端 | User-Agent | originator |
|--------|------------|------------|
| CLI / plugin / models / TUI HEAD 预检 | `codex_cli_rs/0.153.4 (Mac OS 26.5.2; arm64) iTerm.app/3.6.8` | `codex_cli_rs` |
| TUI | `codex-tui/0.153.4 (Mac OS 26.5.2; arm64) iTerm.app/3.6.8 (codex-tui; 0.153.4)` | `codex-tui` |
| exec（多数带 trailer） | `codex_exec/0.153.4 (Mac OS 26.5.2; arm64) iTerm.app/3.6.8 (codex_exec; 0.153.4)` | `codex_exec` |
| MCP HTTP | `codex-mcp-client/0.153.4` | 父客户端 `codex_exec` / `codex-tui` |

### 实测目录（0.153.4）

**CLI UA（规范，无 trailer）** 出现于：GET `/backend-api/codex/models?client_version=0.153.4`（seq 44/46）、HEAD `/backend-api/codex/responses`（seq 40，头序 `originator, user-agent, accept`）、TUI 预检 WS seq 41（短握手，无 x-codex-* 块）、plugins installed/suggested/featured。

**WS turn** `GET wss://chatgpt.com/backend-api/codex/responses` HTTP/1.1 Upgrade。完整握手头序（exec seq 7 / TUI seq 67/82）：

`Host, Connection, Upgrade, Sec-WebSocket-Version, Sec-WebSocket-Key, chatgpt-account-id, authorization, user-agent, originator, openai-beta, version, x-codex-beta-features, x-client-request-id, session-id, thread-id, x-codex-window-id, x-codex-turn-metadata, x-codex-routing-hint, sec-websocket-extensions`

- `openai-beta: responses_websockets=2026-02-06`
- `x-codex-beta-features: remote_compaction_v2`
- `x-codex-routing-hint: model=gpt-6-astra;tier=priority`（TUI seq 82 第二次预热为 `model=gpt-5.6-luna;tier=priority`）
- **无** 独立 `x-codex-installation-id` 头；installation 只在 turn-metadata + `client_metadata`
- TUI seq 41 CLI-UA 预检握手只到 `version` 再 `sec-websocket-extensions`

`response.create` 顶层：`type, model, tool_choice=auto, parallel_tool_calls=false, reasoning={effort:medium,context:all_turns}, store=false, stream=true, include=["reasoning.encrypted_content"], service_tier=priority, prompt_cache_key=<thread>, text={verbosity:low}`；预热帧另有 `generate=false`；真实 turn 省略 `generate`，后续工具轮次带 `previous_response_id`。

`client_metadata` 预热：`x-codex-turn-metadata, ws_request_header_x_openai_internal_codex_responses_lite=true, session_id, x-codex-ws-stream-request-start-ms, turn_id(空), thread_id, x-codex-installation-id, x-codex-window-id`。真实 turn 增加 `root_turn_id`，填 `turn_id`。

**Search** `POST /backend-api/codex/alpha/search`（exec，带 trailer UA）：头序 `version, x-codex-turn-metadata, authorization, chatgpt-account-id, content-type, accept, originator, user-agent, cookie, content-length`。无 OpenAI-Beta、无 session-id 头。Body：`{id, model, input[user message], commands.search_query[{q}], commands.response_length, settings.allowed_callers=["direct"], settings.external_web_access=false, max_output_tokens}`。

**Models** `GET /backend-api/codex/models?client_version=0.153.4`：头序 `version, authorization, chatgpt-account-id, accept, originator, user-agent`。

**Quota** TUI `GET /wham/usage` 与 `/wham/rate-limit-reset-credits`：UA=TUI，**无 originator**，头序 `user-agent, authorization, chatgpt-account-id, accept`。`/wham/settings/user` 另有 `cache-control: no-cache, no-store`。

**Plugins** CLI UA 无 trailer + `oai-product-sku: codex`；TUI 带 trailer。

**MCP** `POST chatgpt.com/backend-api/ps/mcp` UA=`codex-mcp-client/0.153.4`，`x-openai-product-sku: codex`，initialize `clientInfo.name=codex-mcp-client` version `0.153.4`；后续 `mcp-protocol-version: 2025-06-18`。`POST developers.openai.com/mcp?source=codex` 无 originator。

**Analytics** `codex_rs_version=0.153.4`, `runtime_os=macos`, `runtime_os_version=26.5.2`, `runtime_arch=aarch64`。

未抓到：`auth.openai.com/oauth/token`（access JWT 仍有效）、HTTP POST `/codex/responses`、`/responses/compact`、独立 `x-codex-installation-id` HTTP 头。二进制有但本机 turn 未见：`x-codex-turn-state`, `x-codex-parent-thread-id`, `x-codex-protocol-version`, `x-codex-host-device-kind`, `x-codex-server-id`, `x-codex-name`, `x-codex-image-turn-id`, `conversation_id`。

**不要**把 `~/.codex/auth.json` 密钥写进仓库或日志。

## 已决策

- 规范身份锁定 CLI 0.153.4 实测串；Canonical UA / version 与这两个常量同源。
- installation 保持本机 `~/.codex/installation_id`；session 换成最近一条 TUI user 会话。
- 版本一致性测试锁死 `0.153.4`、精确 CLI UA、不以 `(codex_cli_rs; 0.153.4)` 结尾、含 `iTerm.app/3.6.8`、不含裸 `iTerm2` 终端段。历史夹具 UA（`0.144.1` / `0.98.0` / `0.140.2` / Claude Code 的 `iTerm2.app`）不改。

## 已知差异（未改代码）

这些是协议/路径行为，不是编译期身份常量；实测有差异但改动面大、缺 HTTP POST `/codex/responses` 抓包，**不要当回归去改**：

- HTTP 路径仍发 `OpenAI-Beta: responses=experimental`。live HTTP models/search/plugins/quota **不带** 该头；WS 已用 `responses_websockets=2026-02-06`，网关 WS 已正确。HTTP POST `/codex/responses` 在 0.153.4 默认走 WS，本次未抓到。
- `applyCodexFingerprintHeaders` 仍发独立 `x-codex-installation-id` HTTP 头。live WS handshake **不发** 该头，installation 只在 turn-metadata + client_metadata。
- `openai_quota_service.go` 的 `openaiQuotaCodexOriginator = "Codex Desktop"` 未改。live TUI `/wham/usage` 带 TUI UA、**无 originator**；配额探针是网关自己的通道，不是 CLI 规范身份。
- 未扩 `openaiAllowedHeaders`、未加 `x-codex-routing-hint`、未改 compact HTTP。
- 未发明 TLS 指纹。未改前端。

## 待解决

- HTTP POST `/codex/responses` 与 `/responses/compact` 尚无 0.153.4 抓包；有抓包后再评估 OpenAI-Beta 与 compact 头。
- live WS 无独立 `x-codex-installation-id` 头；是否从 HTTP 指纹头里拿掉需单独任务卡。

## 相关

- `backend/internal/service/openai_gateway_service.go`（`codexCLIVersion` / `codexCLIUserAgent`）
- `backend/internal/service/openai_codex_fingerprint.go`（绑机 ID、`applyCodexFingerprintHeaders`）
- `backend/internal/service/openai_codex_identity.go`（Canonical 身份、version 校验）
- `backend/internal/service/openai_codex_version_consistency_test.go`
- 配额探针 originator：`backend/internal/service/openai_quota_service.go`
