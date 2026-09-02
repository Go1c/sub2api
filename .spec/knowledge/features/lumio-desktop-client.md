---
name: lumio-desktop-client
description: Lumio Codex 桌面客户端服务端契约：公开配置、账号级唯一 Key、一次性支付交接，以及 GET /v1/models 不查余额；接入桌面启动或充值流程时查
metadata:
  type: doc
  level: L2
  status: 已交付
---

# Lumio Codex 桌面客户端服务端契约

## 公开配置

`GET /api/v1/desktop/config` 无需 JWT，只返回固定白名单：

- `default_model`
- `payment_url`
- `min_client_version`
- `update_notice`
- `feature_flags.registration`
- `feature_flags.payment_handoff`
- `feature_flags.key_provisioning`

配置以 JSON 存在 `settings.lumio_desktop_config`，管理员现有 `GET/PUT /api/v1/admin/settings` 通过同名 typed 字段读取或整体替换。PUT 省略该字段时保留原值。

## 安全回退

- 默认模型优先继承 `ccswitch_default_model_openai`，再回退 `gpt-5.4`。
- `payment_url` 只能是同源绝对路径；外部 URL、协议相对 URL和反斜杠路径均回退 `/payment`。
- 最低版本必须是 semver；无效持久化值回退 `0.0.0`，无效管理员写入返回 400。
- 更新提示去首尾空白并最多保留 2000 个 Unicode 字符。
- 配置 JSON 破损时从内置默认恢复，不向公开响应透出原始设置或未知字段。

`registration` 同时受全局注册开关与 backend mode 压制；`payment_handoff` 默认开启，但同时受持久化桌面配置和全局支付开关压制，任一关闭都立即阻止签发与消费。账号级唯一 Key 约束已落地，因此 `key_provisioning` 默认开启。

## 账号级桌面 Key

客户端继续调用已鉴权的 `POST /api/v1/keys`，名称固定为 `Lumio Codex Desktop`。服务端把这个精确名称解释为账号级 get-or-create：已有未删除记录时原样返回；不存在时走普通创建流程；并发插入冲突时读取并返回数据库中的胜出记录。

PostgreSQL 部分唯一索引只约束 `deleted_at IS NULL` 且名称精确匹配的记录，因此不同用户可各有一条，软删除后可创建替代 Key，普通名称行为不变。迁移按 `created_at, id` 保留最早记录的名称，后续历史重复记录仅改为 `Lumio Codex Desktop (legacy <id>)`，不删除或轮换凭证。

## 模型目录不查余额

桌面 onboarding 第 3 步是 `GET /v1/models`（OpenAI 兼容目录，不是 Gemini）。目录是发现接口，不是计费消费：有效 Key 即可列出模型，零余额、无订阅也不得返回 `403 INSUFFICIENT_BALANCE`。真正发起 `messages` / `responses` / `chat` / 生图 / 视频时才拦余额。鉴权通过后仍会 `TouchLastUsed`。

实现：`isGatewayModelsListPath` 覆盖网关 GET 目录别名（`/v1/models`、`/models`、`/backend-api/codex/models` 及 Gemini/Antigravity 只读目录）；主中间件与 Gemini 中间件的 `skipBilling` 与 `/v1/usage` 同类。不送新用户余额，也不改 403 信封格式。

## 一次性支付交接

已登录桌面客户端用普通 JWT 调用 `POST /api/v1/desktop/payment-handoff`。成功响应只返回 60 秒有效的同源相对路径：

```text
/api/v1/desktop/payment-handoff/consume?token=dph_<opaque>
```

`dph_` 后是 32 个随机字节的 raw base64url。服务端只把原始 token 的 SHA-256 十六进制摘要作为 Redis key，值只含 `user_id`；不保存原始 token、JWT、refresh token 或 API Key。浏览器访问公开的 `GET /api/v1/desktop/payment-handoff/consume` 后以 Redis `GETDEL` 原子消费，同一 token 跨实例只有一个请求能成功，过期、伪造和重复消费统一返回 410。

消费时服务端重新检查账号状态、桌面功能开关和全局支付开关，重定向目标只取规范化后的服务端 `payment_url`，忽略调用者提供的 redirect。成功后设置 host-only `lumio_web_session`：`HttpOnly`、`SameSite=Lax`、`Path=/`，HTTPS 下带 `Secure`，再以 303 跳转到支付路径并附加非敏感的 `desktop_handoff=1` 标记。

网站把 `/payment` 映射到现有支付页。因为服务端配置可指向其他同源受保护支付页，路由在任一需要鉴权的目标路由识别交接标记：先清除旧 localStorage 账号，再用不经过 Axios 401 跳转拦截器的 `/auth/me` Cookie 探测恢复账号，成功或失败都移除标记并保留其他 query/hash。Cookie JWT 始终不可被 JavaScript 读取或写入 localStorage；Cookie-only 会话退出时仍调用后端清除 Cookie。普通受保护路由在没有本地 token 时也可从 Cookie 恢复，Cookie 无效则回到原登录流程。

Redis 或配置读取失败一律 fail-close，不签发无法保证单次消费的 token。消费先于账号与开关复查，因此账号被禁用或支付被关闭后，已消费 token 也不能重试。

## 缓存

响应带内容 SHA-256 ETag，并设置：

```text
Cache-Control: public, max-age=300, stale-if-error=86400
```

请求携带匹配的 `If-None-Match` 时返回 304。客户端仍需缓存最近一次成功配置并保留内置安全回退；HTTP 缓存不是离线状态的唯一来源。

## 实现位置

- Service/规范化：`backend/internal/service/lumio_desktop_config.go`
- Key get-or-create：`backend/internal/service/api_key_service.go`
- Key 查询与约束：`backend/internal/repository/api_key_repo.go`、`backend/ent/schema/api_key.go`
- 数据迁移：`backend/migrations/923_lumio_desktop_api_key_unique.sql`
- 支付交接 service/store：`backend/internal/service/desktop_payment_handoff.go`、`backend/internal/repository/desktop_payment_handoff_store.go`
- 支付交接 handler/route：`backend/internal/handler/desktop_payment_handoff_handler.go`、`backend/internal/server/routes/desktop.go`
- Cookie JWT 回退：`backend/internal/server/middleware/jwt_auth.go`
- 公开 handler：`backend/internal/handler/setting_handler.go`
- 公开路由：`backend/internal/server/routes/public.go`
- 管理员映射：`backend/internal/handler/admin/setting_handler.go`、`setting_handler_update.go`
- 网站 Cookie 恢复：`frontend/src/api/auth.ts`、`frontend/src/stores/auth.ts`、`frontend/src/router/index.ts`

## 已决策

- 不扩充浏览器用的 `/settings/public`，桌面使用独立窄契约，避免兼容耦合与未来字段误暴露。
- 相关兼容配置以单个 JSON 文档原子保存；全局业务开关仍是最终 kill switch。
- 对已持久化坏值 fail-safe 回退，对管理员新写入 fail-fast 拒绝。
- 桌面 Key 的并发串行化交给 PostgreSQL 部分唯一索引，应用层只负责查询与冲突后读取胜出记录。
- 支付交接 URL 只放短时 opaque token，Redis 只存哈希；不把 JWT、refresh token 或 API Key 放进 URL。
- 网站会话只复用现有 access JWT，不增加 refresh Cookie、通用服务端 session、OAuth 或设备表。

## 验证边界

带 `integration` build tag 的测试和编译校验始终执行；需要容器的数据库/Redis harness 在 Docker 不可用时按仓库现有逻辑跳过，因此这种环境下不能声称完成了外部容器验证。Redis 单次消费与 TTL 另有 miniredis 覆盖。

## 剩余客户端范围

- Lumio Codex 原生客户端仍需接入签发端点、用系统浏览器打开返回路径，并完成操作系统凭据存储、账户 UI 和离线启动范围；服务端与网站交接契约已就绪。
