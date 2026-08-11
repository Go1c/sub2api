---
status: completed
title: 用户 Access Token 管理 API（创建/列表/撤销）
---

# 用户 Access Token 管理 API（创建/列表/撤销）

## 做什么

实现登录用户（JWT 会话）管理自己 Access Token 的 service + HTTP API。创建时返回**一次**明文；列表不回明文。

## 涉及范围

- `backend/internal/service/`（新 service 或挂在 user/auth service）
- `backend/internal/repository/`
- `backend/internal/handler/`（user 侧 handler）
- `backend/internal/server/routes/user.go`（挂到已认证 `/user` 组）
- 单元测试

## API 约定（路径可微调，语义固定）

在 **JWT 认证** 的用户路由下：

1. `POST /api/v1/user/access-tokens`
   - body: `{ "name": string, "expires_in_days"?: number }`
   - `expires_in_days` 缺省 = **7**；合法范围 **1–30**（含）
   - 201/200 + data 含完整 `token`（仅此一次）及 id/name/prefix/expires_at/created_at
2. `GET /api/v1/user/access-tokens`
   - 仅返回当前用户的 token 元数据（无完整 secret）
3. `DELETE /api/v1/user/access-tokens/:id`
   - 只能撤销自己的；他人 id → 404（推荐）或 403（一致即可）
   - 撤销后 `revoked_at` 有值，不可再用于鉴权

## 安全与业务规则

- 生成 token：高熵随机 + 可识别前缀（如 `uat_`）便于日志脱敏与识别
- 存储 `token_hash`（如 SHA-256 / 专用 hash；与现有 secret 处理风格对齐）
- 只能 list/revoke **自己的** token
- 创建接口受现有 audit middleware 覆盖即可；必要时写 audit 事件
- 可选：单用户活跃 token 数量上限（若加，文档写明；未要求可不加）

## 验收标准

- [x] 默认 `expires_in_days` 未传时为 7 天
- [x] `expires_in_days=0/31/负数` 返回 400
- [x] 创建响应含完整 token；列表响应不含完整 token
- [x] 用户 A 不能 list/revoke 用户 B 的 token
- [x] revoke 后同一 token 在鉴权层应失败（可与下一卡联调，本卡至少持久化为 revoked）
- [x] 相关 unit tests 覆盖创建校验 / 所有权 / 列表脱敏

## 依赖

- `user-access-token-schema`
