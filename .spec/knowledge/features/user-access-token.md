---
name: user-access-token
description: 用户长效 opaque Access Token——个人资料创建/撤销、密钥管理 + 只读用量/余额/订阅；含活跃数量上限、用户侧 RPM、usage 查询护栏
metadata:
  type: doc
  level: L2
  status: 已实现
---

# 用户长效 Access Token

用户可在**个人资料**创建可撤销的长效 **opaque** Token，供外部程序通过管理面 API 管理**自己的** API Key（创建/更新/删除密钥、读 available groups / rates），并**只读**查询用量日志、钱包余额与订阅额度。明文仅创建时返回一次；与网关推理用 `sk-` API Key 完全分离。

## 背景 / 目标

- 用户需要用脚本 / CI 管理自己的密钥，而不把浏览器 JWT 会话交给程序。
- 脚本也需要查使用日志、钱包余额、订阅剩余额度做运维/告警。
- 禁止写资料、支付、自助重置周限、access-token 自管理、admin 等写操作 / 敏感面。

## 产品决策

| 项 | 决策 |
|----|------|
| 形态 | opaque（`uat_` + 高熵 hex），非 JWT-only |
| 默认有效期 | 7 天 |
| 最长 | 30 天（`expires_in_days` 1–30） |
| 存储 | 仅 SHA-256 hex（`token_hash`）+ 展示前缀 `token_prefix`；无明文列 |
| 撤销 | `revoked_at` 非空即失效 |
| 分组 | 只能选 available groups；不能自建 group |
| 扩展只读 | 用量 `/usage*`、余额 `GET /user/profile` + `GET /auth/me`、订阅 `GET /subscriptions*`（不含 reset） |
| 活跃数量上限 | 每用户最多 **10** 个未撤销且未过期 token（`USER_ACCESS_TOKEN_LIMIT`） |
| 校验缓存 | 进程内正向缓存约 **30s**；`last_used_at` 写节流 **≥60s** |

## 管理 API（仅 JWT 会话）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/user/access-tokens` | body `{ name, expires_in_days? }`；响应含一次性 `token` |
| `GET` | `/api/v1/user/access-tokens` | 列表元数据，无完整 secret |
| `DELETE` | `/api/v1/user/access-tokens/:id` | 撤销自己的；他人 id → 404 |

用 access token 调用上述路径 → **403**（路径白名单不含 access-tokens 管理，且 handler 二次拒绝）。

## 鉴权行为

1. `Authorization: Bearer <token>`
2. 若 token 以 `uat_` 开头 → opaque 路径：hash 查库 → 未撤销 / 未过期 → 用户 active → 写 `AuthSubject` + `auth_method=user_access_token`
3. 否则走既有 JWT `ValidateToken`
4. `auth_method=user_access_token` 时仅放行白名单路径，否则 403 `ACCESS_TOKEN_SCOPE_DENIED`

**允许路径：**

- `GET/POST /api/v1/keys`
- `GET/PUT/DELETE /api/v1/keys/:id`
- `GET /api/v1/groups/available`
- `GET /api/v1/groups/rates`
- `GET /api/v1/user/profile`（含 `balance` / `frozen_balance` 钱包余额字段）
- `GET /api/v1/auth/me`（前端主路径；同样含 `balance` / `frozen_balance`；与 profile **共用** 余额轮询 RPM）
- `GET /api/v1/usage`、`GET /api/v1/usage/:id`、`GET /api/v1/usage/stats`
- `GET /api/v1/usage/dashboard/stats|trend|models`
- `POST /api/v1/usage/dashboard/api-keys-usage`（批量查询，只读语义）
- `GET /api/v1/subscriptions`、`/active`、`/progress`、`/summary`
- `GET /api/v1/user/balance/transactions`、`/transactions/:txn_id`（本人成功扣款流水）

**明确不放行：** `PUT /user/profile`、access-tokens 管理、`POST /auth/revoke-all-sessions`、`POST /subscriptions/:id/reset-weekly-limit`、支付 / redeem / admin 等。

**失效：** 过期 → 401 `TOKEN_EXPIRED`；撤销 → 401 `TOKEN_REVOKED`；无效 → 401 `INVALID_TOKEN`；用户 inactive → 401 `USER_INACTIVE`。

可选 best-effort 更新 `last_used_at`（异步，不阻塞请求）。

## 本人隔离

`/keys`、`/usage`、`/subscriptions`、`/user/profile`、`/auth/me` 等 handler 继续用 `AuthSubject.UserID`；access token 只能操作创建者本人资源。用户 A 的 token 不能读写用户 B 的 key / 日志 / 余额（与现网一致 404/403）。

## 与 sk- API Key 的区别

| | User Access Token (`uat_`) | Gateway API Key (`sk-`) |
|--|---------------------------|-------------------------|
| 用途 | 管理面：keys + 只读用量/余额/订阅 | 网关推理 / 上游模型调用 |
| 鉴权入口 | JWT 中间件旁路（用户路由） | API Key 网关中间件 |
| 存储表 | `user_access_tokens`（hash） | `api_keys`（key 明文或既有策略） |

## 实现面（关键路径）

- Schema / migration：`backend/ent/schema/user_access_token.go`，`backend/migrations/922_add_user_access_tokens.sql`
- Service：`backend/internal/service/user_access_token_service.go`
- Repo：`backend/internal/repository/user_access_token_repo.go`
- Handler / routes：`backend/internal/handler/user_access_token_handler.go`，`backend/internal/server/routes/user.go`
- Auth 白名单：`backend/internal/server/middleware/jwt_auth.go`（`isUserAccessTokenAllowedPath`）
- 前端：`frontend/src/api/accessTokens.ts`，`frontend/src/components/user/profile/ProfileAccessTokensCard.vue`，挂在 `ProfileView.vue`
- 对外文档：`docs/user-access-token-api.md`

## 滥用护栏（与 JWT 会话共用用户侧配额）

按 `user_id`（缺省回退 IP）Redis 限流，Redis 故障 **fail-open**：

| 配额 key | RPM | 覆盖 |
|----------|-----|------|
| `user-usage-list` | 60 | `GET /usage`、`GET /usage/:id` |
| `user-usage-agg` | 20 | `stats` + dashboard 聚合 |
| `user-wallet-balance` | **30** | `GET /auth/me` + `GET /user/profile`（**共享同一桶**，防交替刷） |
| `user-keys` / `user-groups` / `user-subscriptions-read` | 120 | keys CRUD、groups、订阅只读 |
| `user-access-token-create` | 10 | `POST /user/access-tokens` |

用户 usage 查询额外硬限制（handler）：

- `page_size` 上限 **100**（非通用 1000）
- 自定义 `start_date`/`end_date` 跨度上限 **90 天**（list / stats / dashboard trend & models）

## 已决策

- 不 soft-delete 表行；撤销用 `revoked_at`。
- 不把 access token 当作网关密钥；不开放自建 Group / Admin。
- 改密 / TokenVersion 不自动吊销 access token（独立 `revoked_at` 生命周期）；需要时可后续对齐。
- 钱包余额：`GET /auth/me`（前端主路径）与 `GET /user/profile` 都可读；二者共用 `user-wallet-balance` **30 RPM/用户**，不单独开 balance 端点。
- 钱包流水：允许两个 `GET /user/balance/transactions*` 路径；`POST /user/balance/debit` 仍只接受 Header JWT。
- 订阅只读；周限重置仍需 JWT 会话。
- 活跃 token 上限 10；创建与只读扩展面带 RPM；usage 查询有 page/range 护栏。

## 待解决

- 若 profile 字段对脚本过宽，可后续拆 `GET /user/balance` 并收紧 profile 出白名单。
- 跨实例校验缓存（当前为进程内 30s）；若多副本对「撤销立刻失效」要求更严，可改 Redis 或缩短 TTL。

## 相关

- 任务总览：`.spec/tasks/user-access-token-overview.md`
- 网关 API Key 模型限制：[`api-key-model-restriction.md`](./api-key-model-restriction.md)
- 订阅额度池：[`subscription-credit-pool.md`](./subscription-credit-pool.md)
