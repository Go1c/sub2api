---
status: completed
---

# Access Token 鉴权接入与权限范围隔离

## 做什么

让 `Authorization: Bearer <user_access_token>` 能通过用户管理面鉴权，并**严格限制**可访问路径与资源所有权。

与总览同一功能：密钥管理 + **只读**用量日志 / 钱包余额 / 订阅额度。

## 涉及范围

- `backend/internal/server/middleware/jwt_auth.go` 或新建 `user_access_token_auth` 并在路由层组合
- `backend/internal/server/routes/user.go`（keys/groups/usage/profile/subscriptions 须接受该 token）
- `backend/internal/service/`（ValidateAccessToken）
- 鉴权 / 范围 / 隔离相关测试

## 行为要求

### 1. 鉴权成功时

- 解析 Bearer
- 若是 JWT → 现有逻辑
- 若是 user access token（前缀或 hash 查库）→ 校验：
  - 存在
  - 未撤销
  - 未过期
  - 用户 active
- 写入与现有一致的 `AuthSubject{UserID,...}`，使现有 handler 的 `subject.UserID` 继续生效

### 2. 范围白名单（user_access_token 专用）

**允许：**

- `GET/POST /api/v1/keys`
- `GET/PUT/DELETE /api/v1/keys/:id`
- `GET /api/v1/groups/available`
- `GET /api/v1/groups/rates`
- `GET /api/v1/user/profile`（钱包余额字段）
- `GET /api/v1/usage`、`GET /api/v1/usage/:id`、`GET /api/v1/usage/stats`
- `GET /api/v1/usage/dashboard/stats|trend|models`
- `POST /api/v1/usage/dashboard/api-keys-usage`（只读批量查询）
- `GET /api/v1/subscriptions`、`/active`、`/progress`、`/summary`

**拒绝**（示例，非穷尽）：

- `PUT /api/v1/user/profile`、password、webhook、totp
- `POST /api/v1/subscriptions/:id/reset-weekly-limit`
- payment、admin、redeem、access-tokens 管理 等
- 返回 **403**（推荐）并带明确 code/message

> 管理 Access Token 自身的 CRUD 必须用 **JWT 登录会话**，不得用 access token 自举创建更多 token。

### 3. 本人隔离

现有 keys / usage / subscriptions / profile API 已按 `subject.UserID` 校验所有权；本卡需测试确保：

- 用户 A 的 access token 创建的 key 的 `user_id=A`
- 用户 A 的 access token 不能读/改/删 用户 B 的 key（404 或 403，与现网一致）
- 不能通过伪造 group/user 参数操作他人资源

### 4. 失效

- 过期 → 401 TOKEN_EXPIRED 或等价
- 撤销 → 401 TOKEN_REVOKED / INVALID_TOKEN
- 用户 inactive → 401

## 验收标准

- [x] 有效 access token 可：list/create/update/delete 自己的 keys；读 available groups
- [x] 有效 access token 可只读：profile 余额、usage 日志/统计、subscriptions 列表/进度/summary
- [x] 有效 access token 访问 access-tokens 管理 / 改资料 / 重置周限 / admin 等 → 403
- [x] access token **不能** 调用 `POST /user/access-tokens` 创建新 token
- [x] 用户 A token 无法操作用户 B 的 key
- [x] 过期 / 撤销 token 无法鉴权
- [x] JWT 登录行为不受回归破坏（现有 jwt 测试仍过）
- [x] 相关 unit/integration 测试通过

## 依赖

- `user-access-token-schema`
- `user-access-token-service-api`
