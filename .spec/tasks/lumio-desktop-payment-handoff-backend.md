---
status: completed
title: Lumio Desktop 一次性支付交接后端
---

# Lumio Desktop 一次性支付交接后端

## 做什么

增加 JWT 鉴权的交接签发端点和公开单次消费端点，用 Redis 哈希键保证 60 秒有效、跨实例单次消费，并通过 HttpOnly Cookie 建立当前账号的网站会话。

## 涉及范围

- `backend/internal/service/desktop_payment_handoff.go`
- `backend/internal/repository/desktop_payment_handoff_store.go`
- `backend/internal/handler/desktop_payment_handoff_handler.go`
- `backend/internal/server/routes/desktop.go`
- `backend/internal/server/middleware/jwt_auth.go`
- `backend/internal/handler/auth_handler.go`
- Wire 生成源与生成物
- 对应 unit / Redis / route / middleware tests

## 接口

- Produces: `POST /api/v1/desktop/payment-handoff`
- Produces: `GET /api/v1/desktop/payment-handoff/consume?token=...`
- Produces: HttpOnly `lumio_web_session` JWT Cookie
- Consumes: `SettingService.GetLumioDesktopConfig`
- Consumes: existing JWT validation and token-version checks

## 验收标准

- [ ] 签发端点只接受 JWT 用户并返回 60 秒同源相对 handoff URL
- [ ] Redis 只存 SHA-256 哈希键和 user ID，不存原始 token/JWT/API Key
- [ ] 消费是跨实例原子单次操作，过期/重复/伪造统一返回 410
- [ ] 消费后 Cookie 为 HttpOnly、SameSite=Lax、host-only、TLS 下 Secure
- [ ] 消费只登录 token 绑定账号，禁用账号与关闭支付均 fail-close
- [ ] 重定向只来自服务端安全 payment_url，恶意 redirect 参数无效
- [ ] JWT 中间件无 Authorization 时可使用会话 Cookie，显式 Header 仍优先
- [ ] Logout 清除会话 Cookie
- [ ] 后端相关测试通过

## 依赖

依赖已完成的 `lumio-desktop-config`；前端卡消费 Cookie 会话。
