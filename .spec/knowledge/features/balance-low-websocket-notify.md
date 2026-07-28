---
status: in_progress
title: 个人资料 WebSocket 设置 — 实现中
---

# 个人资料 · WebSocket 设置

## 已确认口径

- 入口：个人资料，卡片名 **WebSocket 设置**
- 总开关默认 **关**
- 余额阈值与邮件 **独立**
- 连接 **全站**
- 站内信：**全部**收件箱新消息
- 公告：独立子开关
- 与邮件余额提醒解耦；邮件系统默认阈值 $10

## 已实现（本分支）

### Backend

- migration `917_add_user_websocket_notify_settings.sql`
- Ent user fields + domain/DTO/profile update
- `UserWebsocketHub` + `UserWebsocketNotifyService`
- `GET /api/v1/user/ws/notifications` + `POST /api/v1/user/websocket-notify/test`
- JWT middleware accepts WS subprotocol token
- Hooks: balance deduct / site message create / announcement create|update active
- unit tests: hub + threshold + test gates
- wire_gen wiring via setters

### Frontend

- types + `updateProfile` fields + `sendWebsocketNotifyTest`
- `ProfileWebsocketNotifyCard` on `/profile` (non-admin)
- `useUserWebsocketNotify` full-site connection from `App.vue`
- i18n zh / en / zh-Hant
- admin settings form default threshold 10
- `pnpm typecheck` green

## 文档

`.spec/knowledge/features/balance-low-websocket-notify.md`
