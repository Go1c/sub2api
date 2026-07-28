---
name: balance-low-site-message-notify
description: 非管理员用户可在个人设置自定义余额低于阈值时的站内信报警（扩展既有邮件余额告警）
metadata:
  type: doc
  level: L2
  status: 设计中
---

# 余额低于阈值 · 站内信通知

简介：在既有「余额低于阈值邮件告警」之上，为**非管理员用户**增加可自定义的**站内信**报警通道。用户在个人设置里勾选、设定阈值；扣费导致余额**首次跌破**阈值时，写入一条未读站内信。

## 背景 / 目标

### 现状（已落地，勿重复造轮子）

| 能力 | 位置 | 说明 |
|------|------|------|
| 全局开关 / 默认阈值 | 管理端设置 `balance_low_notify_*` | 关闭时所有用户余额告警不发 |
| 用户开关 / 自定义阈值 / 附加邮箱 | 用户页 `/profile` → `ProfileBalanceNotifyCard` | 字段：`balance_notify_enabled`、`balance_notify_threshold`、`balance_notify_extra_emails` |
| 触发时机 | `BalanceNotifyService.CheckBalanceAfterDeduction` | 仅在 `oldBalance >= thr && newBalance < thr` 时触发（首次穿越） |
| 通知通道 | **仅邮件** | `dispatchBalanceLowEmail` → `EmailService` |

### 需求（本功能）

- **入口**：用户页面（个人设置），**非管理员**可见/可配（管理员走管理端既有能力，不强制复用此卡）。
- **可自定义**：勾选启用；自定义「余额低于多少」阈值（沿用现有阈值语义：空/0 = 系统默认）。
- **通知方式**：收到**站内信**（侧栏未读红点 + 站内信收件箱）。
- **非目标（本版不做）**：
  - 真正的浏览器 WebSocket 实时弹窗推送（用户口语里的「WebSocket 设置」按「网页端通知设置」理解；通道是站内信，不是新 WS 协议）。
  - Telegram / 短信 / 第三方 webhook。
  - 余额持续低于阈值的重复轰炸（保持「首次穿越才通知」）。
  - 管理员账号专用配置页。

## 设计

### 产品语义

1. 管理端全局 `balance_low_notify_enabled` 仍为总闸；关闭则邮件与站内信均不发。
2. 用户级 `balance_notify_enabled` 仍为用户总开关。
3. 新增用户级通道开关：`balance_notify_site_message_enabled`（bool，默认 `false`，需用户显式勾选，避免静默刷站内信）。
4. 既有邮件行为不变：开启余额通知且邮箱通道可用时仍发邮件；站内信为**可选附加通道**，可与邮件并存。
5. 站内信功能总开关 `site_messages_enabled`：
   - 关闭时：前端隐藏「站内信」勾选；后端即使用户已勾选也**跳过**写站内信（不报错、不影响邮件）。
   - 开启时：用户勾选后穿越阈值即写站内信。

### 数据模型

在 `users` 表增加：

| 列 | 类型 | 默认 | 说明 |
|----|------|------|------|
| `balance_notify_site_message_enabled` | `boolean NOT NULL` | `false` | 用户是否用站内信收余额低告警 |

- Migration：新序号 SQL（与现有 migrations 衔接，不改历史 migration）。
- Ent `User` schema + domain `User` + DTO（profile 读写）同步。
- `PUT /api/v1/user`（或现有 update profile）接受可选字段 `balance_notify_site_message_enabled`。
- 管理员改用户资料若不需要此字段可忽略；用户自助更新必须支持。

### 后端

扩展 `BalanceNotifyService`（优先最小改动，不新建平行服务）：

1. 注入站内信写入能力：仿 `subscriptionNotifyMessengerAdapter`，直接 `SiteMessageRepository.Create`，`SenderID = RecipientID = user.ID`，**绕过** `SiteMessageService.Send` 的管理员/日限（系统通知语义，与订阅通知 / 抽奖一致）。
2. `CheckBalanceAfterDeduction` 在判定穿越后：
   - 保持 `dispatchBalanceLowEmail`；
   - 若 `user.BalanceNotifySiteMessageEnabled && site_messages_enabled`，再 `dispatchBalanceLowSiteMessage`。
3. 文案（中英可先中文主体 + 简短英文，或与邮件标题对齐）：
   - subject 例：`余额不足提醒 / Low balance alert`
   - content 含：当前余额、阈值、建议充值（可附 `balance_low_notify_recharge_url` 纯文本）。
4. 失败：站内信写入失败只打日志，不回滚扣费、不影响邮件。
5. 单元测试：
   - 勾选站内信 + 穿越 → Create 被调用一次；
   - 未勾选 / 全局关 / 站内信功能关 / 未穿越 → 不调用；
   - 与邮件通道互不阻塞。

Wire：在 `ProvideBalanceNotifyService` / `wire` 中接入 `SiteMessageRepository`（或窄接口 `BalanceLowSiteMessenger`）。

### 前端

- 组件：`ProfileBalanceNotifyCard.vue`（已有邮件设置）。
- 展示条件（`ProfileView.vue`）：
  - 全局 `balance_low_notify_enabled`；
  - 当前用户 **非 admin**（`role !== 'admin'`；与「用户页面非管理员」一致）。管理员若打开 `/profile` 可不显示此卡，或只读提示「请用管理端邮件通知配置」——**默认：管理员不显示该卡**，避免与管理设置混淆。
- 新增勾选：`使用站内信通知`（仅当 `site_messages_enabled` 时显示）。
- 保存：与现有 toggle 一样 `userAPI.updateProfile({ balance_notify_site_message_enabled })`。
- 类型：`User` 增加字段；i18n `zh` / `en` / `zh-Hant`。
- 测试：Profile 卡在非管理员 + 全局开时渲染；勾选变更会调 API；admin 不渲染。

### API 契约

- Profile 响应新增：`balance_notify_site_message_enabled: boolean`
- Update body 可选同字段
- 更新 `api_contract` golden（若仓库对该 DTO 有契约测试）

## 已决策

| 决策 | 理由 |
|------|------|
| 通道 = 站内信，不是新 WebSocket | 用户明确「收到站内信通知」；站内信已有未读红点；WS 实时弹窗成本高且非本版目标 |
| 复用首次穿越逻辑 | 与邮件一致，防刷 |
| 用户显式勾选站内信，默认关 | 避免历史用户突然收到系统自发站内信 |
| 系统通知直写 repo，不走用户发信限额 | 与 `subscription_notify_worker` 一致 |
| 管理员不展示用户侧此卡 | 需求限定非管理员；管理端已有全局阈值/邮件 |

## 待解决

- 站内信标题/正文最终文案是否要做 i18n 按用户 locale 生成（后端目前邮件多为中英混排固定模板）——**默认固定中英混排模板**，与现有余额邮件一致。
- 是否需要「仅站内信、不发邮件」细拆邮箱通道开关——**本版不拆**；邮件仍由 `balance_notify_enabled` + 收件人列表控制，站内信为独立勾选。

## 相关

- [[site-messages]]
- 代码：`backend/internal/service/balance_notify_service.go`、`subscription_notify_worker.go`（站内信适配器参考）、`frontend/src/components/user/profile/ProfileBalanceNotifyCard.vue`、`frontend/src/views/user/ProfileView.vue`
- 任务卡：`.spec/tasks/balance-low-site-message-backend.md`、`.spec/tasks/balance-low-site-message-frontend.md`
