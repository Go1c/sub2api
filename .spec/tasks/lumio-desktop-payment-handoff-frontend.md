---
status: completed
title: Lumio Desktop 支付 Cookie 会话前端恢复
---

# Lumio Desktop 支付 Cookie 会话前端恢复

## 做什么

让网站在无 localStorage token 时从 HttpOnly Cookie 恢复当前用户；桌面交接标记出现时先清除旧网站账号，避免进入错误用户的充值页，并让 `/payment` 复用现有支付页面。

## 涉及范围

- `frontend/src/api/auth.ts`
- `frontend/src/stores/auth.ts`
- `frontend/src/router/index.ts`
- 对应 Vitest 测试

## 接口

- Consumes: HttpOnly `lumio_web_session` Cookie
- Consumes: `GET /api/v1/auth/me`
- Consumes: `desktop_handoff=1` 非敏感查询标记
- Produces: `/payment` → existing `PaymentView.vue`

## 验收标准

- [ ] `/payment` 命中现有购买/充值页面
- [ ] 无本地 token 的受保护路由可通过 Cookie `/auth/me` 恢复用户
- [ ] `desktop_handoff=1` 会先清除旧本地账号，再恢复 Cookie 账号
- [ ] 恢复过程中 JavaScript 不读取或持久化 Cookie JWT
- [ ] 标记完成后从地址栏移除，其他 query/hash 保留
- [ ] Cookie 会话 logout 一定请求后端并清除前端状态
- [ ] 无效/过期 Cookie 回到原登录流程，不形成跳转循环
- [ ] `pnpm typecheck`、相关 Vitest 与 build 通过

## 依赖

依赖 `lumio-desktop-payment-handoff-backend`。
