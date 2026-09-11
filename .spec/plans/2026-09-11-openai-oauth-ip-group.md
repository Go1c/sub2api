---
status: pending
---

# OpenAI OAuth IP 组 Implementation Plan

> **For agentic workers:** 先读 `.spec/knowledge/features/openai-oauth-ip-group.md` 与本文件。开工提示词在 `2026-09-11-openai-oauth-ip-group-AGENT-PROMPT.md`。TDD：先红后绿。

**Goal:** OpenAI OAuth 账号可分配一个 IP 组：同一套 Codex 指纹，只换出口；对话钉 IP；每账号每 IP 并发；默认（不选组）与现在线上字节级一致。

**Architecture:** 先选账号，再解析出口。未选组走现有 `account.ProxyID`。选组后用 Redis 把 `sessionHash → proxy_id` 钉在该账号上，并加 `(account_id, proxy_id)` Redis 槽。指纹路径零改动。

**Tech Stack:** Go (Gin / Ent / PostgreSQL / Redis)、Vue 3、Vitest、`go test -tags=unit`、`go vet -tags integration`

## Global Constraints

- 禁止整包 merge / cherry-pick https://github.com/Wei-Shaw/sub2api/pull/6650。
- 禁止改 Codex 指纹常量与 `enforceCodexIdentityHeaders` 行为。见 `.spec/knowledge/features/openai-codex-fingerprint.md`。
- 禁止直推 `main` / `publish`。业务 PR `--base dev`。
- 禁止官方 `234`–`240` 迁移文件名；本 fork 用 `945_…` 起。
- 禁止把 IP 组做成出站模板 / OS profile / 每槽一套 installation。
- 不改 `docker-compose.yml`。不做无关重构。
- 前端：`cd frontend && pnpm typecheck && pnpm build`；UI 用 curl 或浏览器走过。
- 后端：覆盖本卡的 `go test -tags=unit`；改 interface 补 stub；`go vet -tags integration ./...`。
- 完成后用 `spec-steward` 把 `openai-oauth-ip-group.md` 的 `status` 改为已交付，并核 `knowledge/README.md`。

## 推荐 schema

```sql
-- backend/migrations/945_proxy_ip_groups.sql

CREATE TABLE IF NOT EXISTS proxy_ip_groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    per_ip_concurrency INT NOT NULL DEFAULT 10
        CHECK (per_ip_concurrency >= 1 AND per_ip_concurrency <= 1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS proxy_ip_groups_name_alive
    ON proxy_ip_groups (name) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS proxy_ip_group_members (
    group_id BIGINT NOT NULL REFERENCES proxy_ip_groups(id),
    proxy_id BIGINT NOT NULL REFERENCES proxies(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (group_id, proxy_id)
);

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS proxy_ip_group_id BIGINT REFERENCES proxy_ip_groups(id);

CREATE INDEX IF NOT EXISTS accounts_proxy_ip_group_id_idx
    ON accounts (proxy_ip_group_id) WHERE deleted_at IS NULL AND proxy_ip_group_id IS NOT NULL;
```

保存 OpenAI OAuth 且 `proxy_ip_group_id != NULL` 时：**清空 `proxy_id`**。非 OpenAI OAuth 拒绝写入 `proxy_ip_group_id`。

## 对话绑定（Redis）

键建议：`openai:ip_group_bind:{accountID}:{sessionHash}` → `proxyID`  
TTL：与现有 `openaiStickySessionTTL` / `cfg.Gateway.OpenAIWS.StickySessionTTLSeconds` 对齐。

选 IP 伪代码：

```
if account 不是 OpenAI OAuth 或 group_id 为空:
    return 现有 account.Proxy
bound = redis.get(bindKey)
if bound 仍在组内且代理活着:
    试占 (account, bound) 槽（满了就排队/失败，不换 IP）
    return bound
candidates = 组内 active + 未过期
if bound 已死:
    从 candidates 另挑一条有空位的，set bind，占槽
    return new
# 新对话
按稳定顺序（例如 proxy_id 升序）找第一条该账号下未打满的 candidate
set bind，占槽
若都满：与现在账号满员相同处理，不换号语义之外的东西
```

「活着」= `status=active` 且 `expires_at` 为空或未过期且仍是组成员。  
单次 quality/latency probe 失败 **不是** 死。

## 并发

- 现有 `AcquireAccountSlot(accountID, account.Concurrency)` 必须仍先拿到。
- 新增 `AcquireAccountProxySlot(accountID, proxyID, group.PerIPConcurrency)`，Lua/ZSET 照抄账号槽，键分开，例如 `conc:acct-proxy:{accountID}:{proxyID}`。
- 两把锁都要在请求结束时释放（含 WS 生命周期）。
- 计数维度是账号 × IP，不是全站每 IP。

## 必须碰到的出站点（OpenAI OAuth 才解析组）

现有几乎都是：

```
if account.ProxyID != nil && account.Proxy != nil { proxyURL = account.Proxy.URL() }
```

抽一个 helper，例如 `resolveOpenAIAccountProxy(ctx, account, sessionHash) (proxy *Proxy, release func(), err)`，至少接到：

- `openai_gateway_forward.go`
- `openai_gateway_passthrough.go`
- `openai_ws_forwarder_ingress.go` / `openai_ws_forwarder_v2.go` / `openai_ws_http_bridge.go`
- `openai_ws_v2_passthrough_adapter.go`
- `openai_codex_models_service.go`
- `openai_quota_service.go`（配额探针：无会话则组内挑一条活的，不必钉对话）
- `openai_oauth_service.go`（授权/换票：组内挑一条活的）
- `openai_alpha_search.go` / `openai_agent_identity.go` 等仍用账号代理的 OpenAI 路径

非 OpenAI 平台 **不要** 走 helper 的组分支。

`httpUpstream.Do(..., account.ID, account.Concurrency)` 的连接池按账号计即可，不必为每个 IP 新建池。

## 管理 API（建议）

挂在现有 `/api/v1/admin/proxies` 旁，例如：

- `GET/POST /api/v1/admin/proxy-ip-groups`
- `GET/PUT/DELETE /api/v1/admin/proxy-ip-groups/:id`
- `PUT /api/v1/admin/proxy-ip-groups/:id/members` body `{ proxy_ids: number[] }`

账号创建/更新：`CreateAccountInput` / `UpdateAccountInput` 增加 `ProxyIPGroupID *int64`。`0` 或显式空 = 清组。校验：仅 `platform=openai && type=oauth`。

## 前端

- `frontend/src/views/admin/ProxiesView.vue`：增加「IP 组」管理（建组、改名、每 IP 上限、勾选已有代理）。
- `CreateAccountModal.vue` / `EditAccountModal.vue`：仅当 OpenAI OAuth 时，配 IP 区二选一（单独 IP / IP 组）。其它平台仍只用 `ProxySelector`。
- `frontend/src/api/admin/` 补 API；`frontend/src/types` 补类型。
- i18n：`frontend/src/i18n/locales/{zh,en}/admin/` 的 proxies + accounts。文案用「IP 组」/ “IP group”。
- v1 可不做 BulkEdit / CRS 选组（保持单 IP）。若父号改组，shadow 跟后端传播即可。
- Settings 里 web search 的 `ProxySelector` 不要出现 IP 组。

## 参考 #6650（只读，禁止合入）

https://github.com/Wei-Shaw/sub2api/pull/6650  
fork `gebdalaoli-arch/sub2api222` 分支 `codex/codex-device-slot-concurrency-v2`

**可以看：**

- `backend/internal/service/codex_slot_scheduling.go` — 满了换槽、已绑定不换
- `concurrency_service.go` 里按槽占 Redis 租约
- 对话绑定「首写生效」

**不要带：**

- `codex_identity_*` / OS profile / template / client version
- 对 `openai_codex_fingerprint.go` / `openai_codex_identity.go` 的改写
- `AccountProvisioningSpec` / 重建号事务
- `234`–`240` 迁移、`account_codex_device_*` 表
- CRS2 `crs_sync_service.go` / `SyncFromCrsModal.vue` 改动
- 前端 `codex-identity/` 整目录

## Task 1: IP 组 schema + admin API

**Files:**

- Create: `backend/migrations/945_proxy_ip_groups.sql` + 对应 migration test
- Create: `backend/ent/schema/proxy_ip_group.go`（及 member；或 member 只走 SQL 仓储）
- Modify: `backend/ent/schema/account.go` 增加 `proxy_ip_group_id`
- Generate: `backend/ent/`
- Modify: repository / admin handler / routes / dto / wire

验收：

- [ ] 能 CRUD 组、改成员、改 `per_ip_concurrency`
- [ ] 删组是软删；有账号仍引用时拒绝删除或要求先解绑（选一种并写测试）
- [ ] 删代理时同步去掉成员行（FK 或仓储清理，写测试）
- [ ] `go test -tags=unit` 覆盖 repo/handler

## Task 2: 账号二选一

**Files:** `admin_account.go`、`CreateAccountInput` / `UpdateAccountInput`、account mapper、Create/EditAccountModal、i18n、account API types

验收：

- [ ] OpenAI OAuth 可选组；选组后 `proxy_id` 为空
- [ ] 非 OpenAI OAuth 传 `proxy_ip_group_id` → 4xx
- [ ] 不选组的创建/编辑与现在响应字段兼容
- [ ] 父号改组则 shadow 跟着变（对照 `propagateProxyToShadows`）
- [ ] 前端 typecheck

## Task 3: 出站选 IP + 钉对话 + 双槽

**Files:** 新建 `openai_ip_group_proxy.go`（名字可调）+ 单测；改各 OpenAI `proxyURL` 点；`concurrency_cache.go` / `concurrency_service.go`

验收：

- [ ] 无组：`proxyURL` 与现在相同（单测锁死）
- [ ] 有组：同 `sessionHash` 两次解析得到同一 `proxy_id`
- [ ] 新 `sessionHash` 在首选 IP 已满时可落到组内另一条
- [ ] 账号总闸满时不再因组而超发
- [ ] 号 A、号 B 共用同一 proxy 时各有各的 10
- [ ] 出站头仍是 0.153.4 绑机栈（跑现有 `openai_codex_version_consistency_test.go`）

## Task 4: 出口死亡才换绑

验收：

- [ ] 绑定代理被设 inactive / 过期 / 移出组后，同 session 改绑到组内另一活 IP
- [ ] 绑定代理仍活但槽满 / 模拟 429 / overloaded：**不**改绑
- [ ] 单次 probe 失败不改绑

## 建议顺序

1 → 2 → 3 → 4。Task 3 是唯一热路径；无组时必须与现在一致。
