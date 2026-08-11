---
name: lumio-desktop-client
description: Lumio Codex 桌面客户端的服务端引导契约：公开配置白名单、安全默认、缓存与全局开关；接入或调整桌面启动配置时查
metadata:
  type: doc
  level: L2
  status: 实施中
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

`registration` 同时受全局注册开关与 backend mode 压制；`payment_handoff` 同时受全局支付开关压制。尚未具备完整服务端支撑的 `payment_handoff` 与 `key_provisioning` 默认关闭，由对应能力交付后再显式开启。

## 缓存

响应带内容 SHA-256 ETag，并设置：

```text
Cache-Control: public, max-age=300, stale-if-error=86400
```

请求携带匹配的 `If-None-Match` 时返回 304。客户端仍需缓存最近一次成功配置并保留内置安全回退；HTTP 缓存不是离线状态的唯一来源。

## 实现位置

- Service/规范化：`backend/internal/service/lumio_desktop_config.go`
- 公开 handler：`backend/internal/handler/setting_handler.go`
- 公开路由：`backend/internal/server/routes/public.go`
- 管理员映射：`backend/internal/handler/admin/setting_handler.go`、`setting_handler_update.go`

## 已决策

- 不扩充浏览器用的 `/settings/public`，桌面使用独立窄契约，避免兼容耦合与未来字段误暴露。
- 相关兼容配置以单个 JSON 文档原子保存；全局业务开关仍是最终 kill switch。
- 对已持久化坏值 fail-safe 回退，对管理员新写入 fail-fast 拒绝。

## 待解决

- 账号级唯一 Desktop Key、一次性支付登录交接完成后，分别开启对应 feature flag。
