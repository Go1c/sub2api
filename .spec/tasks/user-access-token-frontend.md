---
status: completed
title: 个人资料页 Access Token 管理 UI
---

# 个人资料页 Access Token 管理 UI

## 做什么

在用户个人资料（`ProfileView`）提供创建 / 展示一次 / 列表 / 撤销长效 Token 的 UI，并接后端 API。

## 涉及范围

- `frontend/src/views/user/ProfileView.vue`
- `frontend/src/components/user/profile/`（新卡片组件，风格对齐 Webhook / TOTP 卡）
- `frontend/src/api/user.ts` 或新 `accessTokens.ts`
- i18n 文案（中英若项目有双语文案）
- 必要的前端单测 / 组件测

## UI 要求

1. 卡片标题清晰（如「API 访问令牌 / Access Tokens」）
2. 说明：用于程序化管理**自己的** API Key；默认 7 天，最长 30 天；请妥善保管
3. 创建表单：
   - 名称（必填）
   - 有效天数（默认 7，步进 1，范围 1–30）
4. 创建成功后：
   - **醒目展示完整 token 一次**
   - 一键复制
   - 明确提示「离开后无法再查看完整 token」
5. 列表：名称、前缀、过期时间、创建时间、状态（有效/已撤销/已过期）、撤销按钮
6. 撤销需二次确认

## 非目标

- 不在 Keys 页重复做完整管理（以 Profile 为主即可）
- 不做复杂权限编辑器（范围固定为密钥管理）

## 验收标准

- [x] 登录用户在 Profile 可创建 token（默认 7，最长 30）
- [x] 创建后可复制完整 token；刷新后列表不再显示完整 token
- [x] 可撤销；撤销后列表状态更新
- [x] 非法天数有前端校验 + 后端错误展示
- [x] `pnpm typecheck` 通过；涉及 UI 时按项目要求做最小验证

## 依赖

- `user-access-token-service-api`（后端可用）
- 建议 `user-access-token-auth-scope` 完成后便于端到端自测
