# Agent 开工提示词 — OpenAI OAuth IP 组（v1）

把下面「提示词正文」整段贴进新对话。仓库里已有同内容的设计与计划，先读文件再写代码。

---

## 提示词正文（复制从下一行到文末）

你是本仓库（Go1c/sub2api，品牌 LumioAPI）的实现 Agent。只做一件事：落地 **OpenAI OAuth IP 组 v1**。不要发挥，不要整包同步上游，不要合社区 PR。

### 0. 动手前必读（按顺序）

1. `.spec/AGENTS.md`
2. `.spec/knowledge/README.md`
3. `.spec/rules/system.md`
4. `.spec/skills/before-you-code/SKILL.md`（按它加载上下文）
5. `.spec/skills/test-driven-development/SKILL.md`
6. `.spec/knowledge/features/openai-oauth-ip-group.md`（产品决策，已锁定）
7. `.spec/plans/2026-09-11-openai-oauth-ip-group.md`（schema / 任务拆分 / 文件清单）
8. `.spec/knowledge/features/openai-codex-fingerprint.md`（**禁止改这里列出的常量与头策略**）
9. `.spec/knowledge/features/openai-capacity-shed-retry.md`（429/overloaded 不换 IP）
10. `.spec/knowledge/standards/testing.md` 与 `workflow.md`

读完再改文件。先写失败测试，再写最小实现。

### 1. 背景（为什么做、为什么不能合 #6650）

我们要让 **一个 OpenAI OAuth 号** 能走 **多条已有代理**（多出口 IP）。每条 IP 对这个号有统一并发上限（例如 10）。该 IP 对这个号满了，只有 **新对话** 换组里下一条。指纹（UA / version / installation / session）必须和现在线上完全一样。

社区 PR https://github.com/Wei-Shaw/sub2api/pull/6650（fork `gebdalaoli-arch/sub2api222`，分支 `codex/codex-device-slot-concurrency-v2`）做的是另一件事：出站模板、设备槽、按 OS/客户端换身份、重建号事务、CRS2 改动。官方 `main` **没有合并**它，`mergeable_state: dirty`，+42017 行。本 fork 还把官方 2xx 迁移 remap 成 `9xx`（例如官方 `225` → `930`）。#6650 却占用 `234`–`240`，而官方 `main` 的 234–238 已经是白名单 / MiniMax / OpenCode 等完全无关的文件。

**禁止** `git merge` / 整包 cherry-pick #6650 到 `dev`。只允许用浏览器或 GitHub 只读查看它的调度/租约思路。

本仓库发布纪律：`publish ⊆ dev`；日常 PR 只打 `dev`；**不要**碰 `publish` / `main`。合进 `dev` 不等于上线，但下一次 `dev → publish` 会带上迁移，所以默认路径（不选 IP 组）必须与现在行为一致。

### 2. 已锁定的产品决策（不要再讨论、不要改）

| # | 决策 |
|---|---|
| 1 | 默认 = 现在这套：账号一个 `proxy_id`，一号一出口。 |
| 2 | OpenAI OAuth 配 IP **二选一**：单独一条 IP，或一个 **IP 组**。选组 = 打开多 IP。不要再加第三个「启用」开关。 |
| 3 | 指纹不变，只换 IP。禁止改 `codexCLIVersion` / `codexCLIUserAgent` / `boundCodexInstallationID` / `boundCodexSessionID` / `enforceCodexIdentityHeaders` 的语义。 |
| 4 | 同一条对话钉死第一次选中的 IP。新对话才换。 |
| 5 | 「一条对话」= 现有 sticky：`session_id` / `conversation_id` / `prompt_cache_key` / 同一条 WS 连接。见 `GenerateSessionHash`（`openai_gateway_scheduling.go`）。 |
| 6 | 并发模型 A：账号 `concurrency` 仍是总闸；组上再设统一的每 IP 上限。两把锁都过才能发。 |
| 7 | 每 IP 上限是组级统一数字，不是每条代理单独设。 |
| 8 | 计数维度是 **账号 × IP**。号 A、号 B 共用一条法国 IP 时各算各的 10。 |
| 9 | 配置在代理管理里建 IP 组，账号只选组。中文叫「IP 组」，不要叫「分组」。 |
| 10 | v1 **只对 OpenAI OAuth**。Claude / Gemini / Grok / OpenAI API Key 不能选组；传入组 ID 必须 4xx。 |
| 11 | 绑定 IP **死亡** 时，进行中的对话才允许换组内另一条并改绑。 |
| 12 | 「死亡」= 代理 `inactive` / 非 active / `expires_at` 已过 / 已从该组删除。 |
| 13 | **不是**死亡：该 IP 对这个号槽满、账号总闸满、上游 429、overloaded、单次 latency/quality probe 失败。这些不换 IP。 |
| 14 | 选组后清空并忽略 `proxy_id`。一个号不能同时靠单 IP 和组出站。 |
| 15 | 现有 `proxy_fallback_origin_id`（过期才改写账号 proxy_id）不要复用。 |
| 16 | v1 不做 BulkEdit 选组、不做 CRS 选组。Settings 里 web search 的 ProxySelector 不要出现 IP 组。 |
| 17 | Spark shadow 代理跟父号：父号改组要传播 `proxy_ip_group_id`（对照 `propagateProxyToShadows`）。 |

### 3. 现在代码长什么样（改之前先建立这张地图）

**代理是平铺列表，没有组。**

- Schema：`backend/ent/schema/proxy.go`（`proxies`：name/protocol/host/port/…/`fallback_mode`/`backup_proxy_id`）
- 账号一个出口：`backend/ent/schema/account.go` 的 `proxy_id` + `proxy_fallback_origin_id`
- 领域对象：`backend/internal/service/proxy.go`、`account.go`（`ProxyID *int64`）
- Admin：`backend/internal/handler/admin/proxy_handler.go`
- 路由：`backend/internal/server/routes/admin.go` 里 `/api/v1/admin/proxies`
- 前端列表：`frontend/src/views/admin/ProxiesView.vue`
- 账号选代理：`frontend/src/components/common/ProxySelector.vue`（创建/编辑账号、Settings web search 都在用）
- 创建账号仍是「先 `accountRepo.Create` 再绑分组」，见 `admin_account.go` `CreateAccount`。**不要**改成 #6650 的 `AccountProvisioningSpec`。
- `CreateAccountInput` / `UpdateAccountInput` 在 `backend/internal/service/admin_service.go`，目前只有 `ProxyID`。

**出站现在这样取代理（多处重复）：**

```
proxyURL := ""
if account.ProxyID != nil && account.Proxy != nil {
    proxyURL = account.Proxy.URL()
}
```

至少出现在：

- `openai_gateway_forward.go`（主 HTTP）
- `openai_gateway_passthrough.go`
- `openai_ws_forwarder_ingress.go` / `openai_ws_forwarder_v2.go` / `openai_ws_http_bridge.go`
- `openai_ws_v2_passthrough_adapter.go`
- `openai_codex_models_service.go`
- `openai_quota_service.go`
- `openai_oauth_service.go`
- `openai_alpha_search.go`、`openai_agent_identity.go` 以及其它仍读 `account.Proxy` 的 OpenAI 路径

抽 **一个** helper 接到这些 OpenAI 路径。非 OpenAI 平台不要走组逻辑。

**并发现在只有账号级 Redis 槽：**

- `ConcurrencyService.AcquireAccountSlot`
- `repository/concurrency_cache.go` `AcquireAccountSlot`（Lua + ZSET）
- 照这个模式加 `(accountID, proxyID)` 槽，不要抄 #6650 整段 concurrency 补丁。

**会话粘性已经存在（账号级，不是 IP 级）：**

- `GenerateSessionHash` / `BindStickySession`：`openai_gateway_scheduling.go`
- 调度：`openai_account_scheduler.go`
- IP 组绑定是 **同一 sessionHash 上再钉 proxy_id**，不要重写选号器。

**指纹（禁止改文件语义，测试必须保持绿）：**

- `openai_gateway_service.go`：`codexCLIVersion=0.153.4`，`codexCLIUserAgent=codex_cli_rs/0.153.4 (Mac OS 26.5.2; arm64) iTerm.app/3.6.8`
- `openai_codex_fingerprint.go`：`boundCodexInstallationID`、`boundCodexSessionID`
- `openai_codex_identity.go`、`openai_codex_version_consistency_test.go`

**迁移号：**

- 官方风格 2xx 在本 fork 只同步到大约 `187`，更新的官方迁移已 remap 到 `9xx`
- 当前最高 fork 迁移：`backend/migrations/944_channel_iq_excluded.sql`
- 本功能用 **`945_proxy_ip_groups.sql`**（若 945 已被别人占用，用下一个空的 9xx，不要用 234–240）

### 4. 要做成什么样（实现契约）

按 `.spec/plans/2026-09-11-openai-oauth-ip-group.md` 的 schema、Redis 键、选 IP 伪代码、四个 Task 做。摘要：

**库**

- `proxy_ip_groups(id, name, per_ip_concurrency, timestamps, deleted_at)`
- `proxy_ip_group_members(group_id, proxy_id)`
- `accounts.proxy_ip_group_id` 可空 FK
- 组名单存活唯一；`per_ip_concurrency` 1–1000，默认 10

**运行**

```
未选组或非 OpenAI OAuth → 原样 account.Proxy
已选组 + 已有绑定且代理活着 → 用绑定 IP（槽满不换）
已选组 + 绑定已死 → 组内另挑有空位的活 IP，改绑
已选组 + 新对话 → 按 proxy_id 升序找该账号未打满的活 IP，写入绑定
```

配额探针 / OAuth 换票：没有 sessionHash 时组内挑一条活代理即可，不必钉对话。

`httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)` 连接池继续按账号，不必每 IP 一个 pool。

**API**

- `/api/v1/admin/proxy-ip-groups` CRUD + 改成员
- 账号 create/update 增加 `proxy_ip_group_id`；`0`/null 清组
- 仅 `platform=openai && type=oauth` 可写

**前端**

- 代理页管理 IP 组
- 仅 OpenAI OAuth 的创建/编辑账号：配 IP 二选一
- i18n 中英：「IP 组」/ “IP group”

### 5. 怎么参考 #6650（允许看、禁止合）

用 GitHub 只读打开 PR 文件，**不要**把那 224 个文件引进工作树。

可以看：

- `backend/internal/service/codex_slot_scheduling.go` — 已绑定不换、满了新对话换
- 对话绑定首写生效
- 每槽 Redis 租约的形状（再自己用更小的 API 实现）

绝对不要带：

- `codex_identity_*`、OS profile、template、client version、frontend `codex-identity/`
- 对 fingerprint / identity 文件的修改
- `AccountProvisioningSpec`
- `234`–`240` 迁移和 `account_codex_device_*`
- `crs_sync_service.go`、`SyncFromCrsModal.vue`

### 6. 明确不要做

- 不改指纹、不改容量降载语义（overloaded 仍同号再试，不换 IP）
- 不整包 merge upstream / #6650
- 不直推 `main`/`publish`，不开 `--base publish` 的业务 PR
- 不重构 CreateAccount 事务模型
- 不做每代理单独上限、不做账号多选多个 IP 组
- 不把 IP 组开给全平台
- 不把单次探测失败当成出口死亡
- 不在知识文档里复制一份第二套规则；实现后只更新 `openai-oauth-ip-group.md` 的 status 与锚点路径

### 7. Git / 验证 / 收口

- 从最新 `origin/dev` 开分支（Cloud Agent 用 `cursor/<descriptive-name>-6c93` 规则）。
- 提交信息：`feat(proxy): OpenAI OAuth IP groups for multi-egress`（或按任务拆 commit）。
- PR `--base dev`，标题说明这是 IP 组而不是 #6650。
- 本地：
  - `cd backend && go test -tags=unit ./internal/service ./internal/repository ./internal/handler/admin ./migrations`
  - `go vet -tags integration ./...`
  - `cd frontend && pnpm typecheck && pnpm build`
  - 用现有 `openai_codex_version_consistency_test.go` 证明指纹未动
- 无组回归：加单测锁死「OpenAI OAuth 无 `proxy_ip_group_id` 时 proxyURL == account.Proxy.URL()」
- 有 UI：代理页建组、账号选组这条路径用 curl 或浏览器走一遍
- 用 `spec-steward` 把 `.spec/knowledge/features/openai-oauth-ip-group.md` 标成已交付，补实现锚点，核 `knowledge/README.md`

### 8. 建议执行顺序（一次做完也可以，但测试按这个红绿）

1. 迁移 + Ent + 组 CRUD 测试与 API  
2. 账号字段 + 校验 + OpenAI OAuth 表单二选一  
3. helper + Redis 绑定 + 双槽 + 接到出站点  
4. 死亡换绑测试  

现在开始实现。先开分支，先写 Task 1 的失败测试。
