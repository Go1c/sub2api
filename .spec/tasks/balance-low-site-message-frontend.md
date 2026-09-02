---
status: completed
---

# 余额低告警站内信 — Frontend

## 做什么

在非管理员用户的个人设置「余额通知」卡片中，增加「余额低于阈值时发送站内信」勾选，并与后端字段同步。

## 设计依据

- `.spec/knowledge/features/balance-low-site-message-notify.md`
- 依赖后端字段：`balance_notify_site_message_enabled`

## 涉及范围

- `frontend/src/types/index.ts`（User）
- `frontend/src/api/user.ts`（updateProfile 参数）
- `frontend/src/components/user/profile/ProfileBalanceNotifyCard.vue`
- `frontend/src/views/user/ProfileView.vue`（非管理员 + 全局开关门控；传入 `siteMessagesEnabled`）
- `frontend/src/i18n/locales/{zh,en,zh-Hant}.ts`（及 zh 拆分若存在）
- 测试：`ProfileView.spec.ts`、卡片单测（若有）

## UI 定案

1. **谁看见整张余额通知卡**
   - `balance_low_notify_enabled`（public settings）为 true
   - 且当前用户 `role !== 'admin'`（非管理员）
2. **站内信勾选**
   - 仅当 `site_messages_enabled` 为 true 时显示
   - 文案建议：`余额低于阈值时，同时发送站内信`
   - 说明：需侧栏「站内信」入口开启；未读会红点
3. **与现有控件关系**
   - 总开关 `balance_notify_enabled` 关闭时，阈值 / 邮箱 / 站内信勾选均不可用或随现有 `v-if="notifyEnabled"` 折叠
   - 站内信勾选变更立即 `updateProfile`（与邮件 enable toggle 一致）
4. **不做**
   - 不新增独立「WebSocket」设置页
   - 不实现浏览器桌面通知 / WS 弹窗

## 验收标准

- [ ] 非管理员 + 全局余额通知开：可见余额通知卡
- [ ] 管理员：不显示该卡
- [ ] `site_messages_enabled` 开：可见站内信勾选；关：不可见
- [ ] 勾选变更调用 `updateProfile` 且带 `balance_notify_site_message_enabled`
- [ ] 用户自定义阈值保存行为与现有一致（回归）
- [ ] i18n 三语 key 齐全；相关单测通过
- [ ] `pnpm typecheck` 通过（改前端后）

## 依赖

- 后端任务 `balance-low-site-message-backend` 的 API 字段（可 mock 并行开发）
