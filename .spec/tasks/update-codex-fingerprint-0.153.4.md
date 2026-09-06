---
status: completed
---

# Update Codex compiled-in fingerprint to live 0.153.4

## Task

把网关编译期 Codex 规范身份 / 绑机指纹从本机 0.146.0 抓包更新为本机 **OpenAI Codex CLI v0.153.4** 实测值。规范身份跟 **CLI**（无 trailer），不是 TUI/exec。

本机实测（iTerm.app 3.6.8，Mac OS 26.5.2，arm64，`codex --version` = `codex-cli 0.153.4`）：

| 客户端 | User-Agent | originator |
|--------|------------|------------|
| CLI / plugin / models / TUI HEAD 预检 | `codex_cli_rs/0.153.4 (Mac OS 26.5.2; arm64) iTerm.app/3.6.8` | `codex_cli_rs` |
| TUI | `codex-tui/0.153.4 (Mac OS 26.5.2; arm64) iTerm.app/3.6.8 (codex-tui; 0.153.4)` | `codex-tui` |
| exec（多数带 trailer） | `codex_exec/0.153.4 (Mac OS 26.5.2; arm64) iTerm.app/3.6.8 (codex_exec; 0.153.4)` | `codex_exec` |
| MCP HTTP | `codex-mcp-client/0.153.4` | 父客户端 `codex_exec` / `codex-tui` |

终端段是 `TERM_PROGRAM`/`TERM_PROGRAM_VERSION` = `iTerm.app/3.6.8`，**不是** 二进制终端表显示名 `iTerm2`。OS 段仍是 `"Mac OS"`（不是 `"Mac OS X"`）。CLI **没有** `(codex_cli_rs; 0.153.4)` trailer；TUI/exec 有。

绑机 ID：

- `boundCodexInstallationID` 保持 `12b0b072-d79b-45f9-98af-8fafbe3ef9f5`（与 `~/.codex/installation_id` 一致，已正确）
- `boundCodexSessionID` 从过期的 `01a005e9-75ac-7140-808a-7a91e2a43b33` 改为本机最近一条 **user** 会话（TUI seq 67）：`01a0755d-dbf8-7fb1-8295-17dbb9e447f2`

## Must change

1. `backend/internal/service/openai_gateway_service.go`
   - `codexCLIVersion = "0.153.4"`
   - `codexCLIUserAgent = "codex_cli_rs/0.153.4 (Mac OS 26.5.2; arm64) iTerm.app/3.6.8"`（无 trailer）
   - 重写上方注释：规范身份是 CLI；trailer 只属于 TUI/exec；终端取 `iTerm.app/3.6.8` 不是表名 `iTerm2`；来源是本机 0.153.4 实测。
2. `backend/internal/service/openai_codex_fingerprint.go`
   - `boundCodexSessionID = "01a0755d-dbf8-7fb1-8295-17dbb9e447f2"`
   - 注释写明来源（本机 0.153.4 TUI user 会话）。
3. `backend/internal/service/openai_codex_identity.go`
   - `codexClientVersionPattern` 注释示例从 0.146.0 改为 0.153.4（正则本身不用改）。
4. 测试：强化 `backend/internal/service/openai_codex_version_consistency_test.go`（TDD：先红后绿）
   - `codexCLIVersion == "0.153.4"`
   - `codexCLIUserAgent` 精确等于上面 CLI 串
   - UA **不以** `(codex_cli_rs; 0.153.4)` 结尾（防止再把 trailer 加回 CLI）
   - UA 含 `iTerm.app/3.6.8`，不含裸 `iTerm2` 作为终端段
   - 既有 `Contains(codex_cli_rs/+version)` / Canonical 同源断言保留
5. 既有测试凡引用 `codexCLIUserAgent` / `codexCLIVersion` 常量的会自动跟上，**不要** 改历史夹具 UA（`0.144.1` / `0.98.0` / `0.140.2` / Claude Code 的 `iTerm2.app` 等）。
6. 用 `spec-steward` 新建 `.spec/knowledge/features/openai-codex-fingerprint.md`，并在 `.spec/knowledge/README.md` 加一行。文档记录 0.153.4 实测目录（见下方「实测目录」），以及明确 **未改** 的行为（见 Out of scope）。

## Out of scope（本卡禁止改）

这些是协议/路径行为，不是编译期身份常量；实测有差异但改动面大、缺 HTTP POST `/codex/responses` 抓包，本卡只写入知识文档「已知差异」，**不要改代码**：

- 不要删 HTTP 路径的 `OpenAI-Beta: responses=experimental`（live HTTP models/search/plugins/quota **不带** 该头；WS 已用 `responses_websockets=2026-02-06`，网关 WS 已正确。HTTP POST `/codex/responses` 在 0.153.4 默认走 WS，本次未抓到）。
- 不要去掉 `applyCodexFingerprintHeaders` 的独立 `x-codex-installation-id` HTTP 头（live WS handshake **不发** 该头，installation 只在 turn-metadata + client_metadata）。
- 不要改 `openai_quota_service.go` 的 `openaiQuotaCodexOriginator = "Codex Desktop"`（live TUI `/wham/usage` 带 TUI UA、**无 originator**；配额探针是网关自己的通道，不是 CLI 规范身份）。
- 不要扩 `openaiAllowedHeaders` / 加 `x-codex-routing-hint` / 改 compact HTTP。
- 不要发明 TLS 指纹。不要把 `~/.codex/auth.json` 密钥写进仓库或日志。
- 不要改前端。不要顺手重构。不要 commit / 开 PR。

## 实测目录（写入知识文档，勿把密钥写进去）

抓包根：`/tmp/codex-fp-0.153.4/mitm_captures/`（本机 Codex 0.153.4，mitmproxy 10.4.2）。

**CLI UA（规范，无 trailer）**

`codex_cli_rs/0.153.4 (Mac OS 26.5.2; arm64) iTerm.app/3.6.8`

出现：GET `/backend-api/codex/models?client_version=0.153.4`（seq 44/46）、HEAD `/backend-api/codex/responses`（seq 40，头序 `originator, user-agent, accept`）、TUI 预检 WS seq 41（短握手，无 x-codex-* 块）、plugins installed/suggested/featured。

**WS turn** `GET wss://chatgpt.com/backend-api/codex/responses` HTTP/1.1 Upgrade

完整握手头序（exec seq 7 / TUI seq 67/82）：

`Host, Connection, Upgrade, Sec-WebSocket-Version, Sec-WebSocket-Key, chatgpt-account-id, authorization, user-agent, originator, openai-beta, version, x-codex-beta-features, x-client-request-id, session-id, thread-id, x-codex-window-id, x-codex-turn-metadata, x-codex-routing-hint, sec-websocket-extensions`

- `openai-beta: responses_websockets=2026-02-06`
- `x-codex-beta-features: remote_compaction_v2`
- `x-codex-routing-hint: model=gpt-6-astra;tier=priority`（TUI seq 82 第二次预热为 `model=gpt-5.6-luna;tier=priority`）
- **无** 独立 `x-codex-installation-id` 头
- TUI seq 41 CLI-UA 预检握手只到 `version` 再 `sec-websocket-extensions`

`response.create` 顶层：`type, model, tool_choice=auto, parallel_tool_calls=false, reasoning={effort:medium,context:all_turns}, store=false, stream=true, include=["reasoning.encrypted_content"], service_tier=priority, prompt_cache_key=<thread>, text={verbosity:low}`；预热帧另有 `generate=false`；真实 turn 省略 `generate`，后续工具轮次带 `previous_response_id`。

`client_metadata` 预热：`x-codex-turn-metadata, ws_request_header_x_openai_internal_codex_responses_lite=true, session_id, x-codex-ws-stream-request-start-ms, turn_id(空), thread_id, x-codex-installation-id, x-codex-window-id`。真实 turn 增加 `root_turn_id`，填 `turn_id`。

**Search** `POST /backend-api/codex/alpha/search`（exec，带 trailer UA）

头序：`version, x-codex-turn-metadata, authorization, chatgpt-account-id, content-type, accept, originator, user-agent, cookie, content-length`

无 OpenAI-Beta、无 session-id 头。Body：`{id, model, input[user message], commands.search_query[{q}], commands.response_length, settings.allowed_callers=["direct"], settings.external_web_access=false, max_output_tokens}`。

**Models** `GET /backend-api/codex/models?client_version=0.153.4`

头序：`version, authorization, chatgpt-account-id, accept, originator, user-agent`。

**Quota** TUI `GET /wham/usage` 与 `/wham/rate-limit-reset-credits`：UA=TUI，**无 originator**，头序 `user-agent, authorization, chatgpt-account-id, accept`。`/wham/settings/user` 另有 `cache-control: no-cache, no-store`。

**Plugins** CLI UA 无 trailer + `oai-product-sku: codex`；TUI 带 trailer。

**MCP** `POST chatgpt.com/backend-api/ps/mcp` UA=`codex-mcp-client/0.153.4`，`x-openai-product-sku: codex`，initialize `clientInfo.name=codex-mcp-client` version `0.153.4`；后续 `mcp-protocol-version: 2025-06-18`。`POST developers.openai.com/mcp?source=codex` 无 originator。

**Analytics** `codex_rs_version=0.153.4`, `runtime_os=macos`, `runtime_os_version=26.5.2`, `runtime_arch=aarch64`。

未抓到：`auth.openai.com/oauth/token`（access JWT 仍有效）、HTTP POST `/codex/responses`、`/responses/compact`、独立 `x-codex-installation-id` HTTP 头。二进制有但本机 turn 未见：`x-codex-turn-state`, `x-codex-parent-thread-id`, `x-codex-protocol-version`, `x-codex-host-device-kind`, `x-codex-server-id`, `x-codex-name`, `x-codex-image-turn-id`, `conversation_id`。

## Acceptance

- [x] `codexCLIVersion` / `codexCLIUserAgent` 等于上表 CLI 实测串（无 trailer，终端 `iTerm.app/3.6.8`）。
- [x] `boundCodexSessionID` 为 `01a0755d-dbf8-7fb1-8295-17dbb9e447f2`；installation 不变。
- [x] 版本一致性测试锁死 0.153.4 + 无 CLI trailer + `iTerm.app/3.6.8`。
- [x] `cd backend && go test -tags=unit ./internal/service ./internal/pkg/openai` 通过（至少覆盖 identity / fingerprint / version-consistency / 引用这些常量的测试）。
- [x] `gofmt` 已跑；`git diff --check` 通过。
- [x] 知识文档已建并挂到 `knowledge/README.md`。
- [x] Out of scope 列表里的代码一行未改。

## Skills

动手前跑 `before-you-code`。测试走 `test-driven-development`（先红：新断言在改常量前失败；再改常量转绿）。沉淀走 `spec-steward`。

## Verification Results

- 红：新断言在改常量前失败，`expected "0.153.4"` vs `actual "0.146.0"`。
- 绿：
  - `go test -tags=unit ./internal/service -run 'TestCodexVersionConstants_Consistency' -count=1` → ok
  - `go test -tags=unit ./internal/service ./internal/pkg/openai -count=1` → ok（service ~151s，pkg/openai ~1.6s）
  - `gofmt -l` 四个改动 Go 文件为空；`git diff --check` 通过
- 收口审查（lumio:reviewer）：通过，无 P0/P1，未越界。
