# 订阅管理 Issue 47 设计

## 目标

实现 GitHub issue #47，尽量减少对现有代码结构的影响：

- 管理员撤销的订阅不计入订阅浪费统计。
- 订阅日限额和周限额按管理员配置的 UTC 偏移和刷新小时重置。
- 订阅套餐管理放到“订阅管理”下，不再放在“订单管理”下。

## 当前状态

订阅浪费统计读取 `subscription_credit_ledger`，并关联 `user_subscriptions`。当前公共查询条件只按 ledger 时间、套餐、用户过滤，没有过滤订阅状态。所以订阅被管理员改成 `revoked` 后，它过去的 ledger 仍会进入汇总、按套餐统计和时间序列统计。

订阅扣费现在在 `backend/internal/repository/usage_billing_repo.go` 里用固定 UTC 窗口计算限额。日限额从 `00:00 UTC` 开始，周限额从每周一 `00:00 UTC` 开始。目前 `SettingService`、管理员设置 DTO 和前端设置表单都没有订阅限额刷新时间配置。

订阅套餐页面实际使用 `frontend/src/views/admin/orders/AdminPaymentPlansView.vue`，但路由和侧边栏入口是 `/admin/orders/plans`，属于订单管理。订阅管理现在已有用户订阅和浪费统计入口。

## 产品规则

- 状态为 `revoked` 的订阅从所有浪费统计视图中排除。
- 其他非活跃状态，例如 `expired`，仍保留在浪费统计里。以后如果产品另有要求，再单独调整。
- 订阅限额刷新配置只包含两个管理员字段：
  - UTC 偏移，按“分钟”存储。
  - 刷新小时，取值 `0` 到 `23`。
- 默认值保持现有行为：`UTC+00:00`，刷新小时 `0`。
- 日限额每天在“配置的 UTC 偏移 + 配置的小时”刷新。
- 周限额每周一在“配置的 UTC 偏移 + 配置的小时”刷新。
- 管理员保存设置后，新的扣费请求立即使用新配置。已有用量不会在保存时主动清零；下一次订阅扣费发现旧窗口早于当前窗口时，再重置窗口用量。
- 旧订阅套餐 URL 跳转到新 URL，避免收藏链接失效。

## 后端设计

### 浪费统计

在现有订阅状态常量旁新增：

```go
SubscriptionStatusRevoked = "revoked"
```

然后只在 `backend/internal/repository/subscription_waste_stats_repo.go` 的 `wasteStatsWhere()` 里增加一次排除条件。汇总、按套餐统计和时间序列统计都会调用这个 helper，所以一处修改即可保证所有浪费统计一致。

SQL 条件使用 `service.SubscriptionStatusRevoked`，避免散落字符串字面量。该条件和现有公共过滤条件放在一起，后续新增浪费统计查询也能自动继承。

### 设置模型

在 `backend/internal/service/domain_constants.go` 中新增两个系统设置 key：

- `subscription_quota_reset_utc_offset_minutes`
- `subscription_quota_reset_hour`

在管理员设置模型和 DTO 中新增两个字段：

- `SubscriptionQuotaResetUTCOffsetMinutes int`
- `SubscriptionQuotaResetHour int`

设置服务按以下默认值解析缺失或无效值：

- UTC 偏移：`0`
- 刷新小时：`0`

保存设置时做校验：

- UTC 偏移范围为 `-720` 到 `840` 分钟。
- UTC 偏移必须是 15 分钟的倍数。
- 刷新小时范围为 `0` 到 `23`。

这个范围覆盖真实 UTC 偏移，从 `UTC-12:00` 到 `UTC+14:00`，也支持半小时和 15 分钟时区。

这两个字段只通过管理员设置 API 暴露，不放到公开设置里。

### 扣费窗口计算

在 service 层新增一个小配置结构：

```go
type SubscriptionQuotaResetConfig struct {
	UTCOffsetMinutes int
	ResetHour        int
}
```

新增一个规范化 helper，统一处理默认值和边界。Gateway 构建 `UsageBillingCommand` 时从 `SettingService` 读取当前配置；如果 `SettingService` 不可用，则使用默认配置。

把规范化后的配置放进 `UsageBillingCommand`。Repository 不再直接调用 `startOfUTCDay(now)` 和 `startOfUTCWeek(now)`，而是改为根据配置计算：

- 当前日限额窗口开始时间。
- 当前周限额窗口开始时间。周窗口仍以周一为第一天。

helper 最终返回 UTC 时间用于数据库存储。默认配置下，返回值与当前 UTC helper 完全一致。

## 前端设计

### 管理员设置

在现有“用户订阅入口”设置区域下面新增两个表单项：

- UTC 偏移选择器，例如 `UTC+00:00`、`UTC+08:00` 和其他有效偏移。
- 刷新小时数字输入，范围 `0` 到 `23`。

页面文案明确说明：周限额固定在每周一的所选小时刷新。前端通过 `frontend/src/api/admin/settings.ts` 发送这两个数字字段。

### 订阅套餐导航

在 `frontend/src/components/layout/AppSidebar.vue` 中，把订阅套餐菜单从“订单管理”移动到“订阅管理”。

新增路由：

- `/admin/subscriptions/plans`

这个路由继续渲染 `AdminPaymentPlansView.vue`。页面仍使用支付套餐 API，所以保留现有 `requiresPayment` guard。旧路由 `/admin/orders/plans` 改为重定向到 `/admin/subscriptions/plans`。

现有 `nav.paymentPlans` 文案在中文里已经是“订阅套餐”，可以继续使用。

## 数据流程

1. 管理员在设置页更新订阅限额刷新 UTC 偏移和刷新小时。
2. 管理员设置 handler 校验并通过 `SettingService` 保存这两个值。
3. 网关请求触发订阅扣费时，构建 `UsageBillingCommand`。
4. command 携带当前订阅刷新配置。
5. `UsageBillingRepository.Apply` 锁定订阅记录，计算当前日窗口和周窗口开始时间，并在旧窗口早于当前窗口时重置用量。
6. 浪费 ledger 和消费 ledger 的数据结构不变，只改变窗口边界。

## 错误处理

- 管理员提交非法设置时，返回现有的设置校验错误响应。
- 设置 key 缺失时使用默认值，不影响扣费。
- 网关扣费读取设置临时失败时使用默认配置，并通过现有网关日志记录失败。
- 浪费统计不会因为排除 `revoked` 状态而失败；撤销订阅的记录只是不出现在结果中。

## 测试

后端测试：

- Repository 级浪费统计测试证明 `revoked` 订阅被所有浪费统计 SQL 路径排除。
- 订阅窗口测试证明默认配置与当前 UTC 行为一致。
- 订阅窗口测试证明 `UTC+08:00` 且刷新小时为 `4` 时，在本地 `04:00` 前的日窗口从前一天 `20:00 UTC` 开始。
- 订阅窗口测试证明周窗口在配置偏移和配置小时下，从周一开始。
- 设置服务测试覆盖默认值、合法更新、非法 UTC 偏移和非法小时。
- 管理员设置 handler 测试证明新字段可以通过管理员 API 往返。

前端检查：

- 类型检查证明设置 DTO 和表单模型一致。
- 构建验证证明新路由和侧边栏移动能正常编译。
- 如果本地前端服务可用，用浏览器验证“用户订阅入口”下出现刷新时间设置，且订阅套餐入口出现在“订阅管理”下。

## 不在本次范围内

- 按套餐配置刷新时间。
- 按用户配置刷新时间。
- 配置周刷新星期。周窗口固定周一刷新。
- Cron 表达式或任意刷新分钟。
- 改变 expired 订阅在浪费统计中的行为。
- 重命名支付套餐 API 或组件。

## 自查

- 设计覆盖 issue #47 的三个要求。
- 默认值保持现有扣费行为。
- 刷新时间配置只通过管理员设置 API 暴露。
- 扣费使用固定 UTC 偏移，不使用时区名称或夏令时规则。
- 路由变更通过重定向保持兼容。
