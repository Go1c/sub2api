---
name: site-messages
description: 站内信功能，类轻量邮件：用户收发读回复，红点未读提醒，管理员可全局开关并从用户表发信。
metadata:
  type: doc
  level: L2
  status: 已交付
---

# 站内信（Site Messages）
简介：在 Sub2API 内提供类邮件的站内信功能——用户可收件、发件、阅读、回复；管理员可全局开关，并从用户管理表向选定用户发信。带未读红点、每日发送上限、30 天默认保留期。

## 背景 / 目标
让用户在站内进行轻量邮件式沟通；管理员可全局启停并从用户管理表向某用户发信。

需求要点：
- 功能开启时侧栏出现"站内信"入口；当前用户有未读时显示红点。
- 用户可查看收件箱/发件箱，打开消息看内容与收发方元数据并回复。
- 用户通过输入完整邮箱或数字用户 ID 给另一用户发信；普通用户不能模糊搜索或枚举用户。
- 每收件人有读/未读状态，打开收到的消息即标记已读。
- 管理员可在系统设置启停；关闭时前端隐藏入口，后端所有站内信 API 返回功能未开启错误。
- 管理员从用户表"更多"菜单发信，对话框含标题与内容，收件人固定为选中用户。
- 消息默认保留 30 天；过期消息不被列表/详情 API 返回，可由仓储清理方法删除。
- 非管理员默认每日最多发 10 条，该上限可在管理设置配置；管理员不受每日上限限制。

非目标：实时推送/websocket；附件、富 HTML、草稿、标签、文件夹、批量选择；多收件人；普通用户模糊搜索；完整的管理员消息管理收件箱。

## 设计

### 设置（DB-backed）
- `site_messages_enabled`：布尔，默认 `false`，暴露到 public settings + SSR 注入（便于侧栏/路由守卫在异步设置请求完成前隐藏 UI），注册为 opt-in feature flag。
- `site_messages_daily_send_limit`：整数，默认 `10`，仅管理/系统设置。
- `site_messages_retention_days`：整数，默认 `30`，仅管理/系统设置。

### 数据模型
Ent schema 对应表 `site_messages`（迁移 `backend/migrations/137_add_site_messages.sql`）：
- `id` 主键、`sender_id` 必填、`recipient_id` 必填、`parent_id` 可选（回复）
- `subject` string max 200 必填、`content` text 必填
- `read_at` nullable timestamptz（`null` 表示该收件人未读）
- `created_at` 不可变、`updated_at`
- 外键：`sender`/`recipient` 来自 `User`，`parent`/`replies` 自引用边
- 索引：`(recipient_id, created_at)` 收件箱；`(sender_id, created_at)` 发件箱；`(recipient_id, read_at)` 未读计数；`(parent_id, created_at)` 回复链；`(created_at)` 清理
- 保留期基于 `created_at`，超出窗口的消息从正常读取中排除并由清理删除。

### 后端架构
`SiteMessageService` 职责：解析收件人、强制功能开关、强制普通用户每日发送上限、创建消息/回复、分页列收件箱/发件箱、仅当当前用户是发件人或收件人时加载详情、标记收到的消息已读、统计未读、清理过期消息。所有设置读取走现有 setting service；功能未开启用稳定域错误 `SITE_MESSAGES_DISABLED`。

仓储接口：`Create`、`GetVisibleByID(messageID, userID, retentionCutoff)`、`ListInbox`、`ListSent`、`ListThread`、`MarkRead`、`CountUnread`、`CountSentSince(userID, since)`、`DeleteOlderThan(cutoff)`。

域错误：
```go
ErrSiteMessageNotFound          = infraerrors.NotFound("SITE_MESSAGE_NOT_FOUND", ...)
ErrSiteMessagesDisabled         = infraerrors.Forbidden("SITE_MESSAGES_DISABLED", ...)
ErrSiteMessageRecipientNotFound = infraerrors.NotFound("SITE_MESSAGE_RECIPIENT_NOT_FOUND", ...)
ErrSiteMessageDailyLimitExceeded = infraerrors.Forbidden("SITE_MESSAGE_DAILY_LIMIT_EXCEEDED", ...)
```

### API 设计
用户 API：
- `GET /api/v1/site-messages/inbox?page=&page_size=`
- `GET /api/v1/site-messages/sent?page=&page_size=`
- `GET /api/v1/site-messages/unread-count`
- `GET /api/v1/site-messages/:id`
- `POST /api/v1/site-messages`
- `POST /api/v1/site-messages/:id/reply`
- `POST /api/v1/site-messages/:id/read`
- `GET /api/v1/site-messages/recipient/resolve?query=`

普通用户收件人解析仅接受：精确数字用户 ID；精确邮箱（规范化后大小写不敏感）。

管理员 API：
- `POST /api/v1/admin/site-messages/users/:id`（用户表发信用此接口，行已固定收件人）
- `GET /api/v1/admin/site-messages/recipients?q=`（支持未来管理员撰写流程，模糊匹配邮箱/用户名）
- `POST /api/v1/admin/site-messages/compensation-batches` 支持 `send_email` 发送邮箱副本；全员模式可带 `inactive_days`，筛选最近 N 天没有使用记录的 active 用户。

### 前端设计
- 导航：`AppSidebar.vue` 加"站内信"项，`featureFlags.ts` 注册 `site_messages_enabled`，未读数 >0 时渲染红点；在 public settings 与认证就绪后拉未读数，读/发/回复后刷新。store（`stores/siteMessages.ts`）暴露 `unreadCount`、`hasUnread`、`refreshUnreadCount()`。
- 用户页 `SiteMessagesView.vue`：收件/发件标签页；收件行显示未读态/发件人/主题/预览/时间；发件行显示收件人等；详情含读态与回复；撰写表单校验收件人/主题/内容，收件人仅精确邮箱或 ID。
- 管理员用户管理：`UsersView.vue` 的"更多"菜单加"发送站内信"，弹 `UserSiteMessageModal.vue`，收件人固定，提交标题+内容到 `POST /admin/site-messages/users/:id`。
- 管理员站内信管理：`/admin/site-message-management` 仅管理员可见，侧栏放在"公告"后。页面提供"历史补偿"与"新增补偿"两个视图；发布版默认不预置任何补偿历史、收件人或补偿码数据，真实发送后由后端持久化补偿批次，刷新页面通过 `GET /api/v1/admin/site-messages/compensation-batches` 重新加载。新增补偿页支持指定邮箱 / 全员站内信、标题 / 内容、是否同时发送邮箱副本、是否补偿、补偿额度、补偿码粘贴与发送前提示。提交走 `POST /api/v1/admin/site-messages/compensation-batches`，后端按收件人循环创建真实站内信；指定邮箱按精确邮箱解析，全员模式分页拉取 active 用户逐个发送，若传 `inactive_days > 0` 则只拉取最近 N 天没有 `usage_logs` 记录的 active 用户（从未使用过服务的用户也包含）。该页不在站内信页面生成补偿码，只引用后台已有记录：`/admin/redeem` 生成的 `balance` 兑换码需一人一码并校验状态 `unused`、类型 `balance`、面值匹配；`/admin/promo-codes` 中的优惠码可单码复用，后端校验优惠码状态、过期、全局次数上限、本批次已分配次数、收件用户是否已使用过该优惠码，以及赠送余额是否匹配。单个收件人、补偿码或邮件入队失败不会阻塞整批，接口返回并持久化每条成功 / 失败结果供前端历史详情展示。
- 设置页：站内信卡片，含启停开关、每日发送上限（默认 10）、保留天数（默认 30）。
- 路由 `/site-messages` 带 `requiresSiteMessages: true`，关闭时重定向到 `/dashboard`。

### 错误处理
- 功能关闭：后端返回 typed 功能未开启错误；前端陈旧页面调用时显示"站内信功能未开启"。
- 收件人不存在：普通用户见"收件人不存在或不可用"，不暴露部分匹配数据。
- 自发自收允许（除非实现发现项目既有规则禁止），便于管理员测试和用户自留笔记。
- 主题空/超长、内容空返回校验错误；超每日上限返回含配置上限的 typed 错误。

## 已决策
- 功能默认关闭；现有部署收到新表与设置，但管理员启用前不出现用户侧导航。
- `site_messages_enabled` 进 public settings + SSR 注入；另外两个设置仅管理侧。
- 服务层独立于 Ent（用内存 stub 测试）。
- 保留期与发送上限按 `created_at` / 当日计数计算；过滤值保留 `affiliate_balance` 之类既有契约不破坏（指 settings 暴露与审计 diff 更新）。
- 管理员补偿站内信只引用后台已有补偿码，不临时伪造或绕过记录。余额兑换码来源为 `/admin/redeem` 生成的 `balance` 兑换码（一人一码）；优惠码来源为 `/admin/promo-codes`，可在用户侧 `/redeem` 页面兑换，并按优惠码自身使用次数上限与每用户最多一次控制。
- 管理员回归邮件筛选口径固定为 `usage_logs.created_at`，不是 `users.last_active_at`；即"使用服务"指 API 调用记录，登录过但最近没有 API 调用的用户仍会进入最近 N 天未使用筛选。
- `site_message_compensation_batches.amount` 是本批成功发出的补偿合计，不是每人单价。管理后台写入 `round(compensation_amount × success_count, 2)`（走现有 `moneyCents`）；无补偿或全部失败为 0。运维脚本按人不同面值时已经写入 `sum(value)`。历史页「累计补偿」对非 cancelled 批次直接加 `amount`，禁止再乘 `code_count`。「按此再发」在 `codeCount > 0` 时用 `amount / codeCount` 还原每人额度填进草稿。请求字段 `compensation_amount` 与站内信正文里的补偿额度仍是每人面值。

## 相关
- [[admin-settings-idempotency]]
- 迁移：`backend/migrations/137_add_site_messages.sql`
- 后端：`backend/ent/schema/site_message.go`、`backend/internal/domain/site_message.go`、`backend/internal/service/site_message_service.go`、`backend/internal/repository/site_message_repo.go`、`backend/internal/handler/site_message_handler.go`、`backend/internal/handler/admin/site_message_handler.go`、路由 `backend/internal/server/routes/user.go` + `admin.go`、设置 `backend/internal/service/setting_service.go` 等
- 前端：`frontend/src/api/siteMessages.ts`、`frontend/src/api/admin/siteMessages.ts`、`frontend/src/stores/siteMessages.ts`、`frontend/src/views/user/SiteMessagesView.vue`、`frontend/src/components/admin/user/UserSiteMessageModal.vue`、`frontend/src/views/admin/SiteMessageManagementView.vue`、`frontend/src/views/admin/SettingsView.vue`、`frontend/src/components/layout/AppSidebar.vue`
- 技术栈：Go + Gin + Ent + Wire、Vue 3 + Pinia + Vue Router + TS、Vitest、pnpm
