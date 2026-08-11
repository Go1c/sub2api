---
status: completed
title: 用户长效 Access Token（总览）
---

# 用户长效 Access Token（总览）

## 目标

用户可在**个人资料**中创建**可撤销的长效 opaque Token**，用于外部程序通过 API 管理**自己的**密钥：

- 创建 / 更新 / 删除 API Key
- 查看自己的 API Key
- 选择管理员开放的可用分组（`/groups/available`）
- 设置 key 的额度 / 限速等（沿用现有 `/keys` 能力）

## 已确认产品决策

| 项 | 决策 |
|----|------|
| Token 形态 | 可撤销 **opaque token** + 列表管理（非 JWT-only） |
| 默认有效期 | **7 天** |
| 最长有效期 | **30 天** |
| 授权范围 | **仅密钥管理**：`/api/v1/keys*`、`/api/v1/groups/available`（及同组 rates 如需要） |
| 分组语义 | **只能选择**管理员开放的 available groups；**不能**自建平台分组 |
| 数据隔离 | Token 只能操作**创建者本人**的资源；访问他人资源必须 404/403 |
| 明文 | 创建时**只返回一次**完整 token；之后列表仅展示前缀 / 元数据 |

## 非目标

- 不开放用户自建 Group / Channel / Admin 能力
- 不让此 Token 访问密码、支付、订阅、站内信、TOTP、管理员接口
- 不改现有 sk- API Key 网关鉴权（调用上游模型的 `sk-...` 机制保持不变）
- 不把此 Token 当作网关推理密钥使用

## 建议实现要点（供 coder 参考，不锁死方案）

1. **存储**：新表如 `user_access_tokens`：`id, user_id, name, token_hash, token_prefix, expires_at, last_used_at, revoked_at, created_at`；只存 hash，明文仅创建响应返回。
2. **鉴权**：JWT 中间件旁路或统一 Auth 入口支持 `Bearer <opaque>`：校验 hash → 加载 user → 写入与 JWT 相同的 `AuthSubject`；并标记 auth method = `user_access_token`。
3. **范围守卫**：对 `user_access_token` 认证的请求，仅放行白名单路径；其余 403。
4. **管理 API**（需登录 JWT，放在 profile 流程）：
   - `GET /api/v1/user/access-tokens`
   - `POST /api/v1/user/access-tokens` `{ name, expires_in_days }`（默认 7，范围 1–30）
   - `DELETE /api/v1/user/access-tokens/:id`（撤销）
5. **前端**：`ProfileView` 增加管理卡片（创建、复制一次、列表、撤销、到期展示）。
6. **安全**：token 熵足够（≥ 256 bit）；rate limit 创建；审计可选；改密是否吊销 access token 可与 `TokenVersion` 策略对齐或独立 `revoked`。

## 任务拆分与顺序

1. `user-access-token-schema` — 表 / Ent / migration
2. `user-access-token-service-api` — service + handler + 管理路由
3. `user-access-token-auth-scope` — opaque 鉴权 + 路径白名单 + 本人隔离测试
4. `user-access-token-frontend` — 个人资料 UI + API 客户端
5. `user-access-token-knowledge` — 知识沉淀 + 索引

依赖：1 → 2 → 3 → 4；5 可在 3/4 完成后。

## 整体验收

- [x] 用户在个人资料可创建 token（默认 7 天，最长 30 天）
- [x] 用该 token 可对本账号执行 keys CRUD + 读 available groups + 设额度
- [x] 用该 token 不能访问他人资源、不能访问非白名单接口
- [x] 撤销 / 过期后立即失效
- [x] 明文只在创建时出现一次
- [x] 相关测试与本地验证通过
