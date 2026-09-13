# Codex 0.154.0 Linux 实测指纹报告

> 2026-09-13 在 Ubuntu 22.04 x86_64 上，用官方 `codex-cli 0.154.0` + 已登录 ChatGPT（Plus）经 mitmproxy 抓到的 **User-Agent / 头序 / 工具协议**。  
> 不含 access/refresh token、cookie、邮箱、`chatgpt-account-id`、`safety_identifier`。  
> 这是抓包目录，不是网关改动说明。

## 抓包环境

| 项 | 值 |
|---|---|
| 主机 | `C202609122282993`（`156.238.225.247`） |
| OS | Ubuntu 22.04 LTS jammy，kernel `5.15.0-30-generic`，`x86_64` |
| Codex `os_info` | `os type=Ubuntu`，**`os version=22.4.0`**（不是 `22.04`） |
| 二进制 | `/usr/local/bin/codex` → `codex-cli 0.154.0` |
| `TERM`（交互 SSH） | `xterm-256color` |
| locale | `C.UTF-8`；时区 `Etc/UTC` |
| 默认 shell（environment_context） | `bash` |
| `~/.codex/installation_id` | `95f2ea03-3cf7-41d0-a661-81be6017db0c` |
| `config.toml` | 不存在（全默认） |
| 鉴权 | ChatGPT 文件存储；无 API key |
| 抓包 | mitmproxy 11.0.2；`HTTPS_PROXY=http://127.0.0.1:<port>` + `SSL_CERT_FILE`（系统 CA + mitm CA） |
| 原始 jsonl | 机器上 `/tmp/codex-fp-0.154.0/`（不入库） |

`codex doctor --json` 补充：

- `respect system proxy` = disabled，但环境变量 `HTTPS_PROXY` 仍然生效。
- `network.websocket_reachability`：`wss://chatgpt.com/backend-api/…` 握手 **HTTP 101**。
- 功能开关（稳定 true，与本报告相关）：`enable_request_compression`、`remote_compaction_v2`、`image_generation`、`shell_tool`、`view_image`、`code_mode_host`。
- `responses_websockets` / `responses_websockets_v2` 在 features list 里是 **removed/false**，但 live 默认仍走 WebSocket。
- `search_tool` removed/false；`js_repl` removed/false。
- 本机没有 `/usr/local/bin/codex-code-mode-host`，exec 里 `exec` JS 沙箱会报 spawn 失败。

强制 HTTP：把 WS Upgrade 拦成 502 后，客户端日志为 `Falling back from WebSockets to HTTPS transport`，随后 `POST /backend-api/codex/responses`。

## User-Agent 矩阵

UA 格式：`<originator>/<version> (<os>; <arch>) <TERM> [optional trailer]`。

| 客户端 | User-Agent | originator | 何时出现 |
|---|---|---|---|
| CLI | `codex_cli_rs/0.154.0 (Ubuntu 22.4.0; x86_64) xterm-256color` | `codex_cli_rs` | `GET /ps/plugins/list`、`installed`；部分 TUI 预检把 originator 写成这个 |
| TUI 无 trailer | `codex-tui/0.154.0 (Ubuntu 22.4.0; x86_64) xterm-256color` | 有时仍是 `codex_cli_rs` | TUI 预检：`/codex/models`、`/plugins/featured` |
| TUI 带 trailer | `codex-tui/0.154.0 (Ubuntu 22.4.0; x86_64) xterm-256color (codex-tui; 0.154.0)` | `codex-tui` | TUI 启动后的 models / plugins |
| exec 无 trailer | `codex_exec/0.154.0 (Ubuntu 22.4.0; x86_64) xterm-256color` | `codex_exec` | exec 的部分 models / plugins |
| exec 带 trailer | `codex_exec/0.154.0 (Ubuntu 22.4.0; x86_64) xterm-256color (codex_exec; 0.154.0)` | `codex_exec` | WS/HTTP `/codex/responses`、analytics、wham/settings |
| MCP | `codex-mcp-client/0.154.0` | 父客户端 `codex_exec` | `POST /ps/mcp` |
| OTLP | `OTel-OTLP-Exporter-Rust/0.31.0` | （无 originator） | `POST ab.chatgpt.com/otlp/v1/metrics` |
| TUI + tmux | 同上，但 `<TERM>` 变成 **`screen`** | 同上 | tmux 默认 `TERM=screen`，UA 跟着变 |

规则：

- CLI **没有** `(codex_cli_rs; 0.154.0)` trailer。trailer 只属于 TUI / exec。
- OS 段必须是 `Ubuntu 22.4.0`，终端段等于当时的 `TERM`。
- 每次 `exec` 的 `session-id` / `thread-id` 都会换；`installation_id` 不变。

## 端点目录

Host 默认 `chatgpt.com`。下列头序去掉了 cookie / authorization 的**值**，只保留名字。

### 1. WebSocket `GET /backend-api/codex/responses`

默认推理路径。本号抓到的握手全是 **exec UA + originator=`codex_exec`**。非交互 TUI 只打了 models/plugins，**没有**完成一次 TUI `/codex/responses` 回合。

状态：`101 Switching Protocols`。

头序：

```
Host, Connection, Upgrade, Sec-WebSocket-Version, Sec-WebSocket-Key,
chatgpt-account-id, authorization, user-agent, originator, openai-beta,
version, x-codex-beta-features, x-client-request-id, session-id, thread-id,
x-codex-window-id, x-codex-turn-metadata, x-codex-routing-hint,
sec-websocket-extensions
```

| 头 | 值 |
|---|---|
| `openai-beta` | `responses_websockets=2026-02-06` |
| `version` | `0.154.0` |
| `x-codex-beta-features` | `remote_compaction_v2` |
| `x-codex-routing-hint` | `model=gpt-6-astra`（未见 `;tier=priority`） |
| `sec-websocket-extensions` | `permessage-deflate; client_max_window_bits` |
| `x-codex-installation-id` | **不发**（installation 只在 turn-metadata / client_metadata） |

ID 关系：`session-id` = `thread-id` = `x-client-request-id`；`x-codex-window-id` = `"<session>:0"`。

`x-codex-turn-metadata`（JSON 字符串）字段包括：

`installation_id, session_id, thread_id, agent_name, turn_id, window_id, window_number, context_window_id, request_kind, root_turn_id, thread_source, sandbox, sandbox_mode, auto_review_enabled, node_repl_auto_review_required, node_repl_disabled, turn_started_at_unix_ms`

- 预热：`request_kind=prewarm`，`turn_id=""`。
- 真回合：`request_kind=turn`，填 `turn_id` / `root_turn_id` / `turn_started_at_unix_ms`。
- 本机 sandbox：`sandbox=seccomp`，`sandbox_mode=read-only`。
- `agent_name` 实测为工作目录名（`/root`）。

客户端只发 `response.create`。服务端帧类型：

| 方向 | type |
|---|---|
| 服务端 | `codex.rate_limits`，`codex.response.metadata`，`response.created`，`response.in_progress`，`responsesapi.websocket_timing`，`response.completed`，`response.output_item.added` / `.done`，`response.custom_tool_call_input.delta` / `.done`，`response.content_part.added` / `.done`，`response.output_text.delta` / `.done`，偶发 `error` |
| 客户端 | `response.create`；工具结果作为下一帧 `input=[{type:custom_tool_call_output,...}]` |

`codex.rate_limits` 顶层键：`type, plan_type, rate_limits, code_review_rate_limits, additional_rate_limits, credits, promo`。Plus 路径的配额走这条 WS 帧，**不是**每次都打 `GET /wham/usage`。

WS `response.create` 顶层键序（预热）：

`type, model, input, tool_choice, parallel_tool_calls, reasoning, store, stream, include, prompt_cache_key, text, generate, client_metadata`

真回合去掉 `generate`，在 `model` 后插入 `previous_response_id`。

公共字段：`model=gpt-6-astra`，`store=false`，`stream=true`，`tool_choice=auto`，`parallel_tool_calls=false`，`reasoning={effort:low, context:all_turns}`，`text.verbosity=low`，`include=["reasoning.encrypted_content"]`，`prompt_cache_key` = thread/session id。

### 2. HTTP POST `/backend-api/codex/responses`（WS 失败后回落）

本号在拦 WS→502 后抓到。请求体 **zstd**（magic `28b52ffd`，frame 不带 content size，标准流式帧）。解码后约 74KB JSON。`Accept: text/event-stream`，响应是 SSE。

**无** `openai-beta`。**有** `x-openai-internal-codex-responses-lite: true`。仍无独立 `x-codex-installation-id`。

首次 POST 头序：

```
version, x-codex-beta-features, x-codex-window-id, x-codex-turn-metadata,
x-openai-internal-codex-responses-lite, x-codex-routing-hint,
x-client-request-id, session-id, thread-id, accept, content-encoding,
content-type, authorization, chatgpt-account-id, originator, user-agent,
cookie, content-length
```

后续带工具/多消息的 POST 在 `x-codex-beta-features` 后插入 **`x-codex-turn-state`**（服务端下发的不透明 blob，Fernet 形 `gAAAAA…`，不要当固定指纹回放）。

| 头 | 值 |
|---|---|
| `version` | `0.154.0` |
| `x-codex-beta-features` | `remote_compaction_v2` |
| `x-codex-routing-hint` | `model=gpt-6-astra` |
| `x-openai-internal-codex-responses-lite` | `true` |
| `accept` | `text/event-stream` |
| `content-encoding` | `zstd` |
| `content-type` | `application/json` |
| UA / originator | exec 带 trailer / `codex_exec` |

HTTP JSON **没有** WS 那种 `type: response.create` 包装，顶层直接是 Responses 体：

`model, input, tool_choice, parallel_tool_calls, reasoning, store, stream, include, prompt_cache_key, text, client_metadata`

`client_metadata` 键：`thread_id, x-codex-turn-metadata, root_turn_id, session_id, x-codex-installation-id, x-codex-window-id, turn_id`。

HTTP 200 响应头（配额面，WS 成功时不一定出现这组 HTTP 头）：

| 响应头 | 含义（本号 Plus 样本） |
|---|---|
| `x-codex-plan-type` | `plus` |
| `x-codex-active-limit` | `premium` |
| `x-codex-primary-window-minutes` | `300`（5h） |
| `x-codex-secondary-window-minutes` | `10080`（7d） |
| `x-codex-primary-used-percent` / `x-codex-secondary-used-percent` | 有数值 |
| `x-codex-primary-reset-after-seconds` / `x-codex-secondary-reset-after-seconds` | 有数值 |
| `x-codex-primary-reset-at` / `x-codex-secondary-reset-at` | unix 秒 |
| `x-codex-credits-has-credits` / `x-codex-credits-balance` / `x-codex-credits-unlimited` | 有 |
| `x-codex-safety-buffering-enabled` | `true` |
| `x-codex-safety-buffering-faster-model` | `gpt-5.6-luna` |
| `x-codex-turn-state` | 不透明；下次 POST 带回 |
| `x-models-etag` | 有 |

SSE 第一帧：`event: response.created`，`data` 为 `response.created` JSON。`prompt_cache_retention=24h`，`service_tier=auto`，`reasoning.mode=standard`。SSE `data` 里还有 `safety_identifier`（用户级标识，**不要仿、不要入库**）。

### 3. `GET /backend-api/codex/models?client_version=0.154.0`

头序：`version, authorization, chatgpt-account-id, accept, originator, user-agent`  
`version=0.154.0`；**无** OpenAI-Beta；`Accept: */*`。  
UA 随调用方：exec / TUI（有无 trailer）都见过；originator 可以是 `codex_exec` / `codex-tui` / `codex_cli_rs`。

### 4. `GET /backend-api/wham/settings/user`

仅见 exec 带 trailer。**不发** originator / version。  
头序：`user-agent, authorization, chatgpt-account-id, cache-control, accept`  
`cache-control: no-cache, no-store`。  
**未出现** `GET /wham/usage`、`/wham/rate-limit-reset-credits`。

### 5. 插件

| 路径 | sku | Accept |
|---|---|---|
| `GET /backend-api/ps/plugins/list` | `oai-product-sku: codex` | `*/*` |
| `GET /backend-api/ps/plugins/installed` | `oai-product-sku: codex` | `*/*` |
| `GET /backend-api/ps/plugins/suggested/codex` | `oai-product-sku: codex` | `*/*` |
| `GET /backend-api/plugins/featured?platform=codex` | **无 sku** | `*/*` |

头序（list/installed/suggested）：`originator, user-agent, authorization, chatgpt-account-id, oai-product-sku, accept`  
featured：`originator, user-agent, authorization, chatgpt-account-id, accept`  
TUI 启动会连打多页 `plugins/list`。

### 6. `POST /backend-api/ps/mcp`

UA：`codex-mcp-client/0.154.0`；originator：`codex_exec`。  
`x-openai-product-sku: codex`。  
`Accept: text/event-stream, application/json`。  
后续请求带 `mcp-protocol-version`。  
方法：`initialize` → `notifications/initialized`（常见 **204**）→ `tools/list`。  
本号还见过 **451**（部分 MCP 握手被拒），与 200/204 并存。

头序样本：

```
user-agent, originator, x-openai-product-sku, authorization, chatgpt-account-id, accept, content-type
```

带协议版本时在 originator 后插入 `mcp-protocol-version`。

### 7. 其它侧信道

| 端点 | 说明 |
|---|---|
| `POST /backend-api/codex/analytics-events/events` | exec 带 trailer；头序 `authorization, chatgpt-account-id, content-type, accept, originator, user-agent` |
| `POST ab.chatgpt.com/otlp/v1/metrics` | UA `OTel-OTLP-Exporter-Rust/0.31.0`；另有 `statsig-api-key`（值不记录）；HTTP 202 |
| `GET raw.githubusercontent.com/openai/codex/main/announcement_tip.toml` | TUI/exec 启动拉公告 |
| `GET api.github.com/repos/openai/codex/releases/latest` | TUI 更新检查；doctor 缓存 latest=`0.154.0` |

**未出现：** `POST /backend-api/codex/responses/compact`、`POST /codex/alpha/search`、`GET /wham/usage`。

## 工具协议（additional_tools）

`input[0]` 是 `type=additional_tools`（`role=developer`），不是顶层 `tools` 数组。  
扁平 walk 看到的顶层：

```
namespace functions
  custom exec          ← JS/V8 isolate（code-mode），不是直接 shell
  function wait
  function request_user_input
  function request_user_input_async
namespace clock
  function sleep
namespace collaboration
  function followup_task, interrupt_agent, list_agents,
           send_message, spawn_agent, wait_agent
```

`exec` 描述里挂的嵌套工具（`tools.<name>(...)`），**不会**出现在上面那层 flatten 里：

`apply_patch`, `create_goal`, `exec_command`, `get_goal`, `list_mcp_resources`, `list_mcp_resource_templates`, `read_mcp_resource`, `request_plugin_install`, `update_goal`, `view_image`, `write_stdin`, `clock__curr_time`, `image_gen__imagegen`, `web__run`

所以：

- `image_generation` 功能开着，声明名是 **`image_gen__imagegen`**，走 `exec` 命名空间，不是独立 `additional_tools` 项。
- 网页搜索是 **`web__run`**，同样挂在 `exec` 下；独立 `search_tool` 已 removed。本号 prompt「不要搜」时模型回 `NO_SEARCH`，不会打 `/codex/alpha/search`。
- 没有独立 `python` 工具（`js_repl` 已 removed）。`image_gen` 文案里提到 python，只是禁止用它来改图。

服务端工具流是 **`response.custom_tool_call_input.delta/done`**，不是旧 `function_call`。

## `<environment_context>`

真 user turn 的 message 里带 XML 块。本号解码出的标签（含工具描述误匹配的词，以小写实用字段为准）：

实用字段：`timezone`, `shell`, `cwd`, `current_date`, `workspace_roots`, `filesystem`, `collaboration_mode`, `multi_agent_mode`, `multi_agent_role`, `recommended_plugins`, `skills_instructions`

本号值：`timezone=Etc/UTC`，`shell=bash`，`cwd` 为启动目录 `/root`。

## TLS / JA3

`/tmp/codex-fp-0.154.0/tls.pcap`（LINUX_SLL2）里至少三条 ClientHello，**不能**钉成「Codex rustls 的唯一 JA3」：

| JA3 md5 | ClientHello 指纹（version,ciphers,extensions,groups,points） |
|---|---|
| `0b85eb0d4981e69064e40753e4f0ac5f` | 长套件（更像 OpenSSL / 其它进程） |
| `ee99c8d932f8ed475172937406b91869` | 短套件 + rustls 风格 |
| `6473a3c7256e371759eca1c599f54621` | 短套件，扩展顺序不同 |

网关不要发明或写死 JA3。

## 本号仍未打到的面

| 面 | 原因 |
|---|---|
| TUI 的 `GET/WS /codex/responses` | TUI 在 pty/expect/tmux 里只完成了 plugins/models 预检，没有真正提交一轮推理。协议以 exec WS/HTTP 为准 |
| `GET /wham/usage` | Plus 配额走 WS `codex.rate_limits` + HTTP 回落时的 `x-codex-*` 响应头 |
| `/responses/compact` | 短对话未触发；`remote_compaction_v2` 只作为请求头出现 |
| `POST /codex/alpha/search` | `search_tool` 已移除；搜索在 `exec`→`web__run` |
| 独立 `python` 工具 | 未出现 |
| HTTP POST 带 TUI UA | 回落实验用的是 `codex exec` |

## 和网关对照（只记录差异，不改代码）

仓库编译期规范身份是 **CLI、无 trailer**：

`codex_cli_rs/0.154.0 (Ubuntu 22.4.0; x86_64) xterm-256color`

绑机 `installation_id` 与上表一致。`session-id` 编译期冻的是该机一条历史会话，**不会**跟着每次 exec 变。

live 还和网关不一致、且这次抓包确认了的：

1. ChatGPT `/codex/responses` 默认 **WS**；HTTP 只在 WS 失败后出现，且 body **zstd**、`openai-beta` 换成（HTTP 上干脆不发）WS 的 `responses_websockets=2026-02-06`。
2. live **不发** HTTP 头 `x-codex-installation-id`；installation 在 metadata。
3. live WS/HTTP 都发 `x-codex-routing-hint: model=gpt-6-astra`。
4. HTTP 回落发 `x-openai-internal-codex-responses-lite: true`。
5. 配额：WS 用 `codex.rate_limits`；HTTP 回落用 `x-codex-plan-type` 等响应头。未见 `originator: Codex Desktop`。
