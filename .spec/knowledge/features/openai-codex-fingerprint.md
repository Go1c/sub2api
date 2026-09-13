---
name: openai-codex-fingerprint
description: 网关编译期 Codex CLI 规范身份与绑机指纹（0.154.0 Ubuntu 实测，无 trailer）；含 ChatGPT 上游 WS / 工具调用 / 侧信道抓包；改 UA / version / session 常量或对照 live 抓包时查
metadata:
  type: doc
  level: L2
  status: 已交付
---

# OpenAI Codex 编译期身份 / 绑机指纹

简介：网关出站 Codex 规范身份跟 **CLI**（无 trailer），绑机 installation / session 取自 staging 机 `~/.codex`。当前编译期值来自 **156.238.225.247 Ubuntu 22.04 x86_64 上官方 Codex CLI v0.154.0** 实测，不是 TUI/exec。

## 背景 / 目标

- 上游按 User-Agent、originator、version、installation / session 识别客户端。编译期常量必须跟真实 CLI 对齐，避免被当成非官方客户端。
- 规范身份是 **CLI / plugin / models / TUI HEAD 预检**，不是带 trailer 的 TUI 或 exec。
- 实测环境：Ubuntu 22.04 LTS，kernel 5.15.0-30-generic，x86_64，`codex --version` = `codex-cli 0.154.0`，`TERM=xterm-256color`。`os_info` 把 22.04 显示成 `22.4.0`。

## 设计

### 编译期规范身份

| 常量 | 文件 | 值 |
|------|------|----|
| `codexCLIVersion` | `backend/internal/service/openai_gateway_service.go` | `0.154.0` |
| `codexCLIUserAgent` | 同上 | `codex_cli_rs/0.154.0 (Ubuntu 22.4.0; x86_64) xterm-256color` |
| `boundCodexInstallationID` | `backend/internal/service/openai_codex_fingerprint.go` | `95f2ea03-3cf7-41d0-a661-81be6017db0c`（该机 `~/.codex/installation_id`） |
| `boundCodexSessionID` | 同上 | `01a09826-1bec-7002-afae-06afaf7ed030`（该机一条真实 user 会话，rollout 2026-09-13T00-23-49；每次 exec 会换会话，编译期不跟随） |

CLI UA 规则：

- OS 段是 `Ubuntu 22.4.0`（Codex `os_info` 对 22.04 的写法），不是 `22.04`。
- 终端段取 `TERM` 回落值 `xterm-256color`，不是设置页占位符 `WindowsTerminal`。
- CLI **没有** `(codex_cli_rs; 0.154.0)` trailer；trailer 只属于 TUI/exec。
- originator 是 `codex_cli_rs`。

该机各客户端对照（规范身份只用第一行）：

| 客户端 | User-Agent | originator |
|--------|------------|------------|
| CLI / plugin / models 预检 | `codex_cli_rs/0.154.0 (Ubuntu 22.4.0; x86_64) xterm-256color` | `codex_cli_rs` |
| TUI（带 trailer） | `codex-tui/0.154.0 (Ubuntu 22.4.0; x86_64) xterm-256color (codex-tui; 0.154.0)` | `codex-tui` |
| TUI 预检（无 trailer） | `codex-tui/0.154.0 (Ubuntu 22.4.0; x86_64) xterm-256color` | 有时仍发 `codex_cli_rs`（`GET /codex/models`、`/plugins/featured`） |
| exec（带 trailer） | `codex_exec/0.154.0 (Ubuntu 22.4.0; x86_64) xterm-256color (codex_exec; 0.154.0)` | `codex_exec` |
| exec（无 trailer） | `codex_exec/0.154.0 (Ubuntu 22.4.0; x86_64) xterm-256color` | `codex_exec` |
| MCP | `codex-mcp-client/0.154.0` | 父客户端 `codex_exec` |

### 实测目录（0.154.0 / 156）

抓包：本机环回假 provider（exec HTTP POST `/v1/responses`）+ 已登录 ChatGPT 经 mitmproxy HTTPS_PROXY 打 `chatgpt.com`（WS 默认）。**不要**把 `~/.codex/auth.json` 密钥写进仓库或日志。

`codex doctor --json`：`os version` = `22.4.0`，`os type` = `Ubuntu`，`TERM` = `xterm-256color`。

#### WS `GET /backend-api/codex/responses`（exec 实抓）

默认推理走 WebSocket，不是 HTTP POST。握手头序：

`Host, Connection, Upgrade, Sec-WebSocket-Version, Sec-WebSocket-Key, chatgpt-account-id, authorization, user-agent, originator, openai-beta, version, x-codex-beta-features, x-client-request-id, session-id, thread-id, x-codex-window-id, x-codex-turn-metadata, x-codex-routing-hint, sec-websocket-extensions`

- `openai-beta: responses_websockets=2026-02-06`
- `version: 0.154.0`
- `x-codex-beta-features: remote_compaction_v2`
- `x-codex-routing-hint: model=gpt-6-astra`（未见 `;tier=priority`）
- `session-id` = `thread-id` = `x-client-request-id`；`x-codex-window-id` = `"<session>:0"`
- **无** 独立 `x-codex-installation-id` HTTP 头
- `sec-websocket-extensions: permessage-deflate; client_max_window_bits`
- 预热握手 `x-codex-turn-metadata.request_kind=prewarm`，`turn_id=""`；真实 turn 才填 `turn_id` / `root_turn_id` / `turn_started_at_unix_ms`

#### `response.create` 帧（工具调用）

顶层键序（预热）：`type, model, input, tool_choice, parallel_tool_calls, reasoning, store, stream, include, prompt_cache_key, text, generate, client_metadata`

真实 turn 去掉 `generate`，在 `model` 后插入 `previous_response_id`。

- `model=gpt-6-astra`，`store=false`，`stream=true`，`tool_choice=auto`，`parallel_tool_calls=false`
- `reasoning={effort:low, context:all_turns}`，`text.verbosity=low`，`include=["reasoning.encrypted_content"]`
- `prompt_cache_key` = thread/session id
- 预热 `input` 含 `additional_tools` + `message`；真实 user turn 含 `<environment_context>`（字段：`timezone` / `shell` / `cwd` / `current_date` / `workspace_roots` / `filesystem` 等）；工具回传帧 `input=[{type:custom_tool_call_output, call_id:...}]`
- 配额走 WS 帧 `codex.rate_limits`（`plan_type, rate_limits, credits, promo`），不是每次都打 `GET /wham/usage`
- 服务端工具流：`response.custom_tool_call_input.delta` / `.done`，不是旧的 `function_call`
- `additional_tools` 展开（本机 exec）：`functions` 命名空间；custom `exec`；`wait` / `request_user_input` / `request_user_input_async`；`clock.sleep`；`collaboration` 下 `followup_task` / `interrupt_agent` / `list_agents` / `send_message` / `spawn_agent` / `wait_agent`
- `client_metadata` 含 `x-codex-installation-id`、`x-codex-window-id`、`x-codex-turn-metadata`、`ws_request_header_x_openai_internal_codex_responses_lite`、`x-codex-ws-stream-request-start-ms`；真实 turn 另有 `root_turn_id`

#### 侧信道 / 插件 / MCP / models

| 端点 | UA | originator | version | 其它 |
|------|----|------------|---------|------|
| `GET /wham/settings/user` | exec 带 trailer | **不发** | 无 | `cache-control: no-cache, no-store`；头序 `user-agent, authorization, chatgpt-account-id, cache-control, accept` |
| `GET /codex/models?client_version=0.154.0` | exec 带 trailer，或 TUI 无 trailer | exec 或 `codex_cli_rs` | `0.154.0` | 头序 `version, authorization, chatgpt-account-id, accept, originator, user-agent`；**无** OpenAI-Beta |
| `GET /ps/plugins/list` `installed` `suggested/codex` | CLI 无 trailer / exec / TUI | 配套 originator | 无 | `oai-product-sku: codex`（featured 没有 sku） |
| `GET /plugins/featured?platform=codex` | exec 或 TUI 无 trailer | 配套或 `codex_cli_rs` | 无 | 无 sku |
| `POST /ps/mcp` | `codex-mcp-client/0.154.0` | `codex_exec` | 无 | `x-openai-product-sku: codex`；后续带 `mcp-protocol-version`；方法 `initialize` / `notifications/initialized` / `tools/list` |
| `POST /codex/analytics-events/events` | exec 带 trailer | `codex_exec` | 无 | 头序 `authorization, chatgpt-account-id, content-type, accept, originator, user-agent` |

#### ChatGPT HTTP POST `/backend-api/codex/responses`（拦 WS→502 后回落）

完整目录见 [`docs/codex-linux-fingerprint-0.154.0.md`](../../../docs/codex-linux-fingerprint-0.154.0.md)。要点：

- 请求体 **zstd**；`Accept: text/event-stream`；响应 SSE `event: response.created`。
- **无** `openai-beta`；**有** `x-openai-internal-codex-responses-lite: true`、`x-codex-routing-hint: model=gpt-6-astra`。
- 头序：`version, x-codex-beta-features, x-codex-window-id, x-codex-turn-metadata, x-openai-internal-codex-responses-lite, x-codex-routing-hint, x-client-request-id, session-id, thread-id, accept, content-encoding, content-type, authorization, chatgpt-account-id, originator, user-agent, cookie, content-length`；后续 POST 在 beta-features 后插入 `x-codex-turn-state`。
- JSON 无 `type: response.create` 包装；顶层 `model, input, …, client_metadata`。
- HTTP 200 带 `x-codex-plan-type` / `x-codex-active-limit` / 主副窗口分钟数 / used-percent / reset / credits / `x-codex-safety-buffering-*`。
- `image_gen__imagegen` / `web__run` 挂在 `additional_tools` → `functions.exec` 的 JS 工具表里，不在顶层 flatten。

#### exec HTTP POST `/v1/responses`（本机环回，非 ChatGPT）

`x-codex-beta-features: remote_compaction_v2`，window / turn-metadata / session-id / thread-id 同上。**无** 独立 `x-codex-installation-id` HTTP 头。

#### 未抓到

- TUI 自己的 `/codex/responses` WS（pty 里 TUI 只完成 plugins/models 预检）
- `/responses/compact`
- `GET /wham/usage`（Plus 走 WS `codex.rate_limits` 或 HTTP 回落的 `x-codex-*` 响应头）
- `POST /codex/alpha/search`（独立 search_tool 已 removed）
- 独立 `python` 工具

pcap 里混有多条 TLS ClientHello（rustls 短套件两条 + 一条 OpenSSL 长套件），**未**把单一 JA3 钉进网关。

## 已决策

- 规范身份锁定 CLI 0.154.0 实测串（无 trailer）；Canonical UA / version 与这两个常量同源。
- installation / session 绑该 staging 机 `~/.codex`，不再用旧 Mac 0.153.4 抓包。
- 版本一致性测试锁死 `0.154.0`、精确 CLI UA、不以 `(codex_cli_rs; 0.154.0)` 结尾、含 `xterm-256color`、不含 `WindowsTerminal`。历史夹具 UA（`0.144.1` / `0.98.0` / `0.140.2` / Claude Code 的 `iTerm2.app`）不改。

## 已知差异（未改代码）

这些是协议/路径行为，不是编译期身份常量：

- HTTP 路径仍发 `OpenAI-Beta: responses=experimental`。live HTTP models/search/plugins/quota **不带** 该头；WS 已用 `responses_websockets=2026-02-06`。
- `applyCodexFingerprintHeaders` 仍发独立 `x-codex-installation-id` HTTP 头。live WS / HTTP **都不发** 该头，installation 只在 turn-metadata + client_metadata。
- `openai_quota_service.go` 的 `openaiQuotaCodexOriginator = "Codex Desktop"` 未改。
- 未扩 `openaiAllowedHeaders`、未加 `x-codex-routing-hint`（live WS 有 `model=gpt-6-astra`）、未改 compact HTTP。
- 未发明 TLS 指纹。未改前端。

## 待解决

- TUI `/codex/responses` WS 与 `/responses/compact` 仍无完成回合的抓包。
- live WS/HTTP **都无** 独立 `x-codex-installation-id` 头；是否从网关 HTTP 指纹头里拿掉需单独任务卡。

## 相关

- `backend/internal/service/openai_gateway_service.go`（`codexCLIVersion` / `codexCLIUserAgent`）
- `backend/internal/service/openai_codex_fingerprint.go`（绑机 ID、`applyCodexFingerprintHeaders`）
- `backend/internal/service/openai_codex_identity.go`（Canonical 身份、version 校验）
- `backend/internal/service/openai_codex_version_consistency_test.go`
- 配额探针 originator：`backend/internal/service/openai_quota_service.go`
- Linux 实测全文（UA / WS / HTTP zstd / 工具表）：`docs/codex-linux-fingerprint-0.154.0.md`
