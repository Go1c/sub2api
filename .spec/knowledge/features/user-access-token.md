---
name: user-access-token
description: 用户长效 opaque Access Token——个人资料创建/撤销、仅密钥管理 API、与 sk- 网关密钥分离
metadata:
  type: doc
  level: L2
  status: 已实现
---

# 用户长效 Access Token

用户可在**个人资料**创建可撤销的长效 **opaque** Token，供外部程序通过管理面 API 管理**自己的** API Key（创建/更新/删除密钥、读 available groups / rates）。明文仅创建时返回一次；与网关推理用 `sk-` API Key 完全分离。

## 背景 / 目标

- 用户需要用脚本 / CI 管理自己的密钥，而不把浏览器 JWT 会话交给程序。
- 范围严格限制在密钥管理，禁止 profile / 支付 / 用量 / admin 等。

## 产品决策

| 项 | 决策 |
|----|------|
| 形态 | opaque（`uat_` + 高熵 hex），非 JWT-only |
| 默认有效期 | 7 天 |
| 最长 | 30 天（`expires_in_days` 1–30） |
| 存储 | 仅 SHA-256 hex（`token_hash`）+ 展示前缀 `token_prefix`；无明文列 |
| 撤销 | `revoked_at` 非空即失效 |
| 分组 | 只能选 available groups；不能自建 group |

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

**失效：** 过期 → 401 `TOKEN_EXPIRED`；撤销 → 401 `TOKEN_REVOKED`；无效 → 401 `INVALID_TOKEN`；用户 inactive → 401 `USER_INACTIVE`。

可选 best-effort 更新 `last_used_at`（异步，不阻塞请求）。

## 本人隔离

`/keys` 等 handler 继续用 `AuthSubject.UserID`；access token 只能操作创建者本人资源。用户 A 的 token 不能读写用户 B 的 key（与现网一致 404/403）。

## 与 sk- API Key 的区别

| | User Access Token (`uat_`) | Gateway API Key (`sk-`) |
|--|---------------------------|-------------------------|
| 用途 | 管理面：管理自己的 keys | 网关推理 / 上游模型调用 |
| 鉴权入口 | JWT 中间件旁路（用户路由） | API Key 网关中间件 |
| 存储表 | `user_access_tokens`（hash） | `api_keys`（key 明文或既有策略） |

## 实现面（关键路径）

- Schema / migration：`backend/ent/schema/user_access_token.go`，`backend/migrations/922_add_user_access_tokens.sql`
- Service：`backend/internal/service/user_access_token_service.go`
- Repo：`backend/internal/repository/user_access_token_repo.go`
- Handler / routes：`backend/internal/handler/user_access_token_handler.go`，`backend/internal/server/routes/user.go`
- Auth：`backend/internal/server/middleware/jwt_auth.go`（`ContextKeyAuthMethod`）
- 前端：`frontend/src/api/accessTokens.ts`，`frontend/src/components/user/profile/ProfileAccessTokensCard.vue`，挂在 `ProfileView.vue`

## 已决策

- 不 soft-delete 表行；撤销用 `revoked_at`。
- 不把 access token 当作网关密钥；不开放自建 Group / Admin。
- 改密 / TokenVersion 不自动吊销 access token（独立 `revoked_at` 生命周期）；需要时可后续对齐。

## 待解决

- 单用户活跃 token 数量上限（未要求，未实现）。
- 创建 rate limit 专门配额（依赖现有 audit / 通用限流即可）。

## 相关

- 任务总览：`.spec/tasks/user-access-token-overview.md`
- 网关 API Key 模型限制：[`api-key-model-restriction.md`](./api-key-model-restriction.md)
