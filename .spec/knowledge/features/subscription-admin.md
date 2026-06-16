---
name: subscription-admin
description: issue #47 订阅管理三项修复——撤销不计浪费、限额刷新时间可配、套餐入口迁移
metadata:
  type: doc
  level: L2
  status: 设计中
---

# 订阅管理（Issue #47）
简介：实现 GitHub issue #47，尽量减少对现有代码结构的影响：管理员撤销的订阅不计入浪费统计、订阅日 / 周限额刷新时间可由管理员按 UTC 偏移和刷新小时配置、订阅套餐入口从「订单管理」移到「订阅管理」。

## 背景 / 目标
三个要求：

- 管理员撤销的订阅不计入订阅浪费统计。
- 订阅日限额和周限额按管理员配置的 UTC 偏移和刷新小时重置。
- 订阅套餐管理放到「订阅管理」下，不再放在「订单管理」下。

**当前状态**：
- 订阅浪费统计读 `subscription_credit_ledger` 并关联 `user_subscriptions`，公共查询条件只按 ledger 时间、套餐、用户过滤，没过滤订阅状态。所以订阅被改成 `revoked` 后，其过去的 ledger 仍进入汇总、按套餐统计和时间序列统计。
- 订阅扣费在 `backend/internal/repository/usage_billing_repo.go` 用固定 UTC 窗口算限额：日限额从 `00:00 UTC` 开始，周限额从每周一 `00:00 UTC` 开始。`SettingService`、管理员设置 DTO 和前端设置表单都没有刷新时间配置。
- 订阅套餐页实际用 `frontend/src/views/admin/orders/AdminPaymentPlansView.vue`，但路由和侧边栏入口是 `/admin/orders/plans`，挂在订单管理下。订阅管理现已有用户订阅和浪费统计入口。

技术栈：Go 1.26、Gin、Ent/raw SQL、Vitest/Vue 3、Vue Router、pnpm、PowerShell。

整体架构：后端把订阅刷新时间作为系统设置写入 `UsageBillingCommand`，repository 只根据 command 计算窗口；浪费统计在公共 SQL where helper 中排除 `revoked`；前端只扩展已有设置页、路由和侧边栏，不重命名支付套餐 API。

## 设计

### 产品规则
- 状态 `revoked` 的订阅从所有浪费统计视图排除。
- 其他非活跃状态（如 `expired`）仍保留在浪费统计里，未来另有要求再单独调整。
- 订阅限额刷新配置只含两个管理员字段：UTC 偏移（按分钟存储）、刷新小时（`0`–`23`）。
- 默认值保持现有行为：`UTC+00:00`，刷新小时 `0`。
- 日限额每天在「配置 UTC 偏移 + 配置小时」刷新；周限额每周一在同一时刻刷新。
- 管理员保存后，新扣费请求立即用新配置；已有用量不在保存时主动清零，下一次订阅扣费发现旧窗口早于当前窗口时再重置窗口用量。
- 旧订阅套餐 URL 跳转到新 URL，避免收藏链接失效。

### 后端设计

**浪费统计**：在现有订阅状态常量旁新增 `SubscriptionStatusRevoked = "revoked"`，然后只在 `subscription_waste_stats_repo.go` 的 `wasteStatsWhere()` 加一次排除条件。汇总、按套餐统计和时间序列统计都调这个 helper，一处修改即可一致。SQL 条件用 `service.SubscriptionStatusRevoked` 避免散落字面量：

```go
args := []any{query.StartTime, query.EndTime, service.SubscriptionStatusRevoked}
parts := []string{"l.created_at >= $1", "l.created_at < $2", "us.status <> $3"}
```

**设置模型**：在 `domain_constants.go` 新增两个系统设置 key：`subscription_quota_reset_utc_offset_minutes`、`subscription_quota_reset_hour`。在管理员设置模型和 DTO 新增 `SubscriptionQuotaResetUTCOffsetMinutes int`、`SubscriptionQuotaResetHour int`。缺失或无效值默认解析为偏移 `0`、刷新小时 `0`。保存校验：UTC 偏移范围 `-720`–`840` 分钟、必须是 15 分钟倍数、刷新小时 `0`–`23`（覆盖 UTC-12:00 到 UTC+14:00，支持半小时和 15 分钟时区）。两个字段只通过管理员设置 API 暴露，不放公开设置。校验错误 reason：`INVALID_SUBSCRIPTION_QUOTA_RESET_UTC_OFFSET`、`INVALID_SUBSCRIPTION_QUOTA_RESET_HOUR`。

**扣费窗口计算**：service 层新增配置结构和规范化 helper：

```go
type SubscriptionQuotaResetConfig struct {
	UTCOffsetMinutes int
	ResetHour        int
}
```

Gateway 构建 `UsageBillingCommand` 时从 `SettingService` 读当前配置（不可用则用默认），把规范化后的配置放进 command。Repository 不再直接调 `startOfUTCDay(now)` / `startOfUTCWeek(now)`，改为按配置计算当前日窗口和周窗口开始时间（周窗口仍以周一为第一天），helper 最终返回 UTC 时间用于存储。默认配置下返回值与当前 UTC helper 完全一致。`subscriptionQuotaWindowStarts(now, cfg)` 先归一化配置，按偏移把 UTC 时间转本地、取当天刷新小时作为日窗口起点（若本地时间早于该点则回退一天），再回退到本周一构造周窗口，最后各自减去偏移转回 UTC。

### 前端设计

**管理员设置**：在现有「用户订阅入口」设置区域下新增两个表单项——UTC 偏移选择器（如 `UTC+00:00`、`UTC+08:00` 等有效偏移，由常量数组生成）、刷新小时数字输入（`0`–`23`）。文案明确说明周限额固定在每周一所选小时刷新。前端通过 `frontend/src/api/admin/settings.ts` 发送这两个数字字段。

**订阅套餐导航**：在 `AppSidebar.vue` 把订阅套餐菜单从「订单管理」移到「订阅管理」。新增路由 `/admin/subscriptions/plans`，继续渲染 `AdminPaymentPlansView.vue`（仍用支付套餐 API，保留 `requiresPayment` guard）。旧路由 `/admin/orders/plans` 重定向到新路由。现有 `nav.paymentPlans` 文案中文已是「订阅套餐」，继续沿用。

### 数据流程
1. 管理员在设置页更新订阅限额刷新 UTC 偏移和刷新小时。
2. 管理员设置 handler 校验并经 `SettingService` 保存。
3. 网关请求触发订阅扣费时构建 `UsageBillingCommand`。
4. command 携带当前订阅刷新配置。
5. `UsageBillingRepository.Apply` 锁定订阅记录，计算当前日 / 周窗口开始时间，旧窗口早于当前窗口时重置用量。
6. 浪费 ledger 和消费 ledger 数据结构不变，只改窗口边界。

### 错误处理
- 管理员提交非法设置返回现有设置校验错误响应。
- 设置 key 缺失用默认值，不影响扣费。
- 网关扣费读设置临时失败用默认配置，并经现有网关日志记录。
- 浪费统计不会因排除 `revoked` 失败，撤销订阅记录只是不出现在结果中。

## 已决策
- 设计覆盖 issue #47 的三个要求，默认值保持现有扣费行为。
- 刷新时间配置只通过管理员设置 API 暴露。
- 扣费使用固定 UTC 偏移，不用时区名称或夏令时规则。
- 路由变更通过重定向保持兼容。
- 不在本次范围：按套餐 / 按用户配置刷新时间、配置周刷新星期（固定周一）、Cron 表达式或任意刷新分钟、改变 expired 订阅在浪费统计中的行为、重命名支付套餐 API 或组件。

## 实现
按 TDD 分六个任务：

1. **浪费统计排除 revoked**：改 `domain_constants.go`（加 `SubscriptionStatusRevoked`）、`subscription_waste_stats_repo.go`（`wasteStatsWhere()` 加 `us.status <> $3`）；测试 `TestWasteStatsWhere_ExcludesRevokedSubscriptions`。
2. **订阅刷新配置和窗口计算**：改 `usage_billing.go`（加配置结构、`SubscriptionQuotaResetMin/MaxOffsetMinutes` 常量、`NormalizeSubscriptionQuotaResetConfig`、`UsageBillingCommand` 字段）、`usage_billing_repo.go`（加 `subscriptionQuotaWindowStarts`，在 `applyUsageBillingSubscription()` 中调用）；测试默认匹配 UTC、UTC+08 日窗口、UTC+08 周一窗口。
3. **设置服务保存 / 读取配置**：改 `domain_constants.go`、`settings_view.go`、`setting_service.go`（key、字段、默认、`parseSettings`、`buildSystemSettingsUpdates` 校验与写入）；测试更新 / 拒绝非法 / parse 默认与取值。
4. **Gateway 和管理员设置 API 贯通**：实现 `GetSubscriptionQuotaResetConfig(ctx)`（内部 `GetMultiple`，失败或非法返回默认），`buildUsageBillingCommand()` 经 helper 从 setting service 取配置（必要时在 `billingDeps` 加 `settingService *SettingService` 由 Gateway/OpenAI Gateway 填充）；管理员 DTO/handler 映射 `subscription_quota_reset_utc_offset_minutes` / `subscription_quota_reset_hour`。
5. **前端设置表单、路由、侧边栏**：改 `api/admin/settings.ts`（`SystemSettings` 加两字段）、`SettingsView.vue`（默认值、payload、两个控件）、`router/index.ts`（新路由 + 旧路由重定向）、`AppSidebar.vue`（菜单移到订阅管理组）、i18n（zh/en/zh-Hant）。先用字段触发 `pnpm typecheck` 红灯再补 interface。
6. **全量验证**：`go test -tags unit ./internal/repository ./internal/service ./internal/handler/admin`、`pnpm typecheck`、`pnpm build`、检查 `git status --short` / `git diff --stat` 仅含 issue #47 文件。

## 相关
- 交叉链接：[[subscription-credit-pool]]、[[subscription-pricing]]
- 代码路径（后端）：`backend/internal/service/{domain_constants,settings_view,setting_service,usage_billing,gateway_service}.go`、`backend/internal/repository/{usage_billing_repo,subscription_waste_stats_repo}.go`、`backend/internal/handler/dto/settings.go`、`backend/internal/handler/admin/setting_handler.go`
- 代码路径（前端）：`frontend/src/api/admin/settings.ts`、`frontend/src/views/admin/SettingsView.vue`、`frontend/src/views/admin/orders/AdminPaymentPlansView.vue`、`frontend/src/router/index.ts`、`frontend/src/components/layout/AppSidebar.vue`、`frontend/src/i18n/locales/{zh,en,zh-Hant}.ts`
- 外链：GitHub issue #47
