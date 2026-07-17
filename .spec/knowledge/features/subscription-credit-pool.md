---
name: subscription-credit-pool
description: 用户级订阅额度池——订阅额度优先消费、多订阅按时间顺序扣费、不足自动拆分扣余额、ledger 审计与上限通知 / 改订阅扣费或额度逻辑时查这篇
metadata:
  type: doc
  level: L2
  status: 已落地
---

# 用户级订阅额度池（Subscription Credit Pool）

简介：用户购买订阅后获得用户级额度池，可在任意可用分组消费；扣费时订阅额度优先，订阅不足部分自动从充值余额扣除。默认只允许用户保有一条可消费订阅；管理员开启多订阅购买后，用户可同时持有多条可消费订阅，扣费按订阅创建时间从早到晚消耗，遇到总额度不足或日/周限额触顶时继续尝试后一条订阅。支持总额度、日/周限额、有效期、ledger 审计与上限通知。

> 前提：订阅功能尚未上线，本方案不考虑线上兼容，直接采用最简最终形态。订阅最长有效期 30 天。技术栈：Go / Gin / Ent / PostgreSQL / Redis billing cache / Vue 3 / Pinia / Vite / Vitest。
> 实施状态：分支 `feat/subscription-credit-pool`，[PR #44](https://github.com/Go1c/sub2api/pull/44)。截至 2026-05-28 代码层 Task 1–10 全部完成并通过验证。2026-06-27 增补管理员多订阅购买开关与多订阅按时间顺序扣费能力。

## 背景 / 目标

当前订阅实现是「分组订阅」：

- `user_subscriptions.group_id` 必填，唯一索引限制同用户同分组只有一条订阅（`backend/ent/schema/user_subscription.go`）。
- 订阅日/周/月限放在 `groups` 表（`backend/ent/schema/group.go`）。
- 套餐 `subscription_plans.group_id` 绑定单个分组（`backend/ent/schema/subscription_plan.go`）。
- 中间件只在 API Key 所属分组是 `subscription` 类型时读取该分组订阅（`backend/internal/server/middleware/api_key_auth.go`）。
- 扣费层只能二选一：订阅扣 `SubscriptionCost` 或余额扣 `BalanceCost`（`backend/internal/repository/usage_billing_repo.go`）。

目标：把「订阅」变成用户级额度池——购买一个订阅后可在任意可用分组消费；消费优先用订阅额度，订阅不可用/不足时自动用充值余额。

## 设计

### 设计原则

1. **默认只允许一条「可消费」订阅**：未开启多订阅购买时，用户只有无可消费订阅、当前订阅总额度耗尽或过期时才能购买新订阅。
2. **管理员可开启多订阅购买**：系统设置 `subscription_multiple_purchases_enabled` 开启后，购买和履约不再拦截已有可消费订阅；多条订阅可同时 `active` 且可消费。
3. **分组仍负责路由和价格**：API Key 仍绑实际分组，分组/用户/账号倍率按现有逻辑计算本次费用。
4. **订阅额度优先，但不阻断余额消费**：订阅用完/过期/超日周限后不直接拒绝，余额可用则回落余额扣费。
5. **支持拆分扣费**：单次请求可同时扣订阅和余额（如费用 $3，订阅可用 $1，余额扣 $2）。
6. **额度和窗口限额用套餐快照**：购买时把额度、有效天数、日/周限、覆盖范围写入 `payment_orders` 和 `user_subscriptions`，管理员后续改套餐不影响已购订阅。
7. **所有额度变化必须有流水**：购买、消费、过期销毁、管理员调整都写 `subscription_credit_ledger`。
8. **多订阅按创建时间扣费**：鉴权阶段收集覆盖当前分组的可消费订阅，扣费事务内按 `created_at ASC, id ASC` 锁定并分配，先扣最早订阅。
9. **额度或窗口不足时向后递补**：最早订阅总额度不足、日限或周限剩余空间不足时，剩余费用继续尝试后一条订阅；所有订阅都不可用后才回落余额。
10. **未开启多订阅时，日/周限触顶不释放购买门槛**：日/周限是会按窗口重置的节流，不算「用完」；触顶时回落余额扣费，不释放下单门槛。
11. **订阅不支持退款**：第一版订单支付完成不走退款；旧订阅过期时剩余额度直接销毁并记账。

### 数据模型（实现面）

**1. `user_subscriptions` 改为额度池**：`group_id` 改 NULLABLE，删旧分组唯一索引；新增 `plan_id` / `scope_type` / `scope_config` / `quota_limit_usd`(>0 CHECK) / `quota_used_usd` / `daily_limit_usd` / `weekly_limit_usd` / `exhausted_at` / `expired_credit_logged_at`；删 `monthly_window_start` / `monthly_usage_usd`。最初用部分唯一索引保证每用户最多 1 条可消费订阅：

```sql
CREATE UNIQUE INDEX user_subscriptions_user_active_usable
    ON user_subscriptions(user_id)
    WHERE deleted_at IS NULL AND status = 'active' AND exhausted_at IS NULL;
```

2026-06-27 后，migration `155_allow_multiple_active_credit_subscriptions.sql` 删除 `user_subscriptions_user_active_usable`，是否允许用户购买多条可消费订阅改由管理员设置 `subscription_multiple_purchases_enabled` 控制。

状态机（`status` 取 `active`/`expired`/`suspended`，不再有 `exhausted`，由 `exhausted_at` 时间戳表达）：

| 可消费状态 | 含义 |
|-----------|------|
| `active` AND `exhausted_at IS NULL` AND `deleted_at IS NULL` | 可消费订阅；开启多订阅后同一用户可有多条 |
| `active` AND `exhausted_at IS NOT NULL` | 总额度耗尽等过期；用户已可买新订阅 |
| `expired` | 由后台过期任务推进 |
| `suspended` | 管理员手动暂停 |

`scope_type`：`all_available_groups`（默认）/ `selected_groups`（`scope_config.group_ids`）/ `platforms`（`scope_config.platforms` 如 `['anthropic','openai']`）。

**2. `subscription_plans`**：新增 `quota_usd`(>0) / `daily_limit_usd` / `weekly_limit_usd` / `scope_type` / `scope_config` / `validity_days`（CHECK 1–30）；`group_id` 改 NULLABLE。

**3. 新增 `subscription_credit_ledger`**：所有额度变化的流水表。`type` 固定值：

- `purchase`：购买获得额度（`delta>0`）
- `consume`：请求消费（`delta<0`，`balance_delta<0` 为同次余额扣减部分）
- `limit_reached`：总额度/日限/周限触顶（`delta=0`，幂等，靠 `event_key` 去重）
- `expire`：到期销毁剩余额度（`delta<=0`）
- `window_reset`：日/周窗口重置时记录被重置窗口的浪费（`delta=0`，metadata 装 `window/limit_usd/used_before_reset_usd/wasted_usd/wasted_ratio`，用于定价分析，不发通知）
- `admin_adjust`：管理员调整（符号视方向）

`event_key` 格式（UTC RFC3339）：总额度 `total:{sub_id}`；日限 `daily:{sub_id}:{daily_window_start}`；周限 `weekly:{sub_id}:{weekly_window_start}`；过期 `expire:{sub_id}:{expires_at}`；窗口重置 `window_reset_{daily|weekly}:{sub_id}:{old_window_start}`。唯一索引 `subscription_credit_ledger_event_key_unique(subscription_id,type,event_key)` 保证同一窗口只写一次。

**4. `usage_logs`**：新增 `subscription_cost_usd` / `balance_cost_usd`。`billing_type` 扩展：`0`=纯余额，`1`=纯订阅，`2`=订阅+余额混合。

**5. `payment_orders`**：新增 `subscription_quota_usd` / `subscription_daily_limit_usd` / `subscription_weekly_limit_usd` / `subscription_scope_type` / `subscription_scope_config` / `subscription_validity_days`。履约时**完全用订单快照**生成订阅，不读 `subscription_plans` 当前值。

**6. 通知队列复用 `scheduler_outbox`**：新增 event_type 常量 `subscription_notify`，payload `{user_id, subscription_id, kind}`，`kind` 取 `limit_reached_total`/`limit_reached_daily`/`limit_reached_weekly`/`expired`。

**7. 系统设置 `subscription_multiple_purchases_enabled`**：管理员设置页「用户订阅入口」区域暴露开关；后端通过 `SettingService.GetSubscriptionMultiplePurchasesEnabled()` 读取。默认 `false`，保持单订阅购买门槛；开启后允许购买和履约创建多条有效订阅。

### 核心消费流程

1. API Key 鉴权拿到用户、API Key、实际分组。
2. Service 查「用户当前可消费订阅」列表（`active AND exhausted_at IS NULL AND expires_at > NOW() AND deleted_at IS NULL`），按 scope 校验当前分组是否被覆盖，并按 `created_at ASC, id ASC` 排序。
3. 资格预检（双资金源）：任一覆盖订阅仍有可用额度/窗口 → 放行；订阅不可用但余额满足门槛 → 放行；两者都不可用 → 结构化错误。
4. 上游请求完成后算 `ActualCost`。
5. 在 `UsageBillingRepository.Apply` 同一事务内：`SELECT ... FOR UPDATE` 锁候选订阅行并按创建时间排序 → 对每条订阅用 `AllocateSubscriptionCredit` 分配可扣部分 → UPDATE 对应订阅（`quota_used_usd`/窗口用量/窗口重置/`exhausted_at`，`RETURNING` 触顶跨越 bool）→ 所有订阅后仍有剩余则 UPDATE 用户余额 → INSERT ledger（每条订阅的 `consume` + 必要 `limit_reached`）→ INSERT outbox（`subscription_notify`）→ INSERT `usage_logs` 拆分字段。
6. 事务提交后由 outbox worker 异步发通知。

### 扣费 SQL 改造（核心，实现面）

替换 `backend/internal/repository/usage_billing_repo.go` 的 `incrementUsageBillingSubscription`，单订阅与多订阅共用两步式扣费：

- **第 1 步**：`SELECT ... FOR UPDATE` 锁行读状态（quota/daily/weekly/exhausted/expires/scope）。多订阅路径锁定候选 ID 集并按 `created_at ASC, id ASC` 返回。Go 侧按配置窗口判断是否需要窗口重置，再调 `AllocateSubscriptionCredit` 算本订阅可扣金额和剩余金额。
- **第 2 步**：原子 UPDATE，`RETURNING` 三个跨越 bool（`just_exhausted_total` / `just_hit_daily` / `just_hit_weekly`），同一 UPDATE 内完成 `quota_used`/窗口用量/窗口重置/`exhausted_at` 写入。

关键设计点：行锁（混合扣费必须先读后算，无锁会两请求都读到同一余额各扣）；多订阅路径由 `UsageBillingCommand.SubscriptionIDs` 携带候选订阅 ID，幂等指纹包含归一化后的订阅 ID 列表；窗口重置收敛到扣费事务（不再用异步 `DoWindowMaintenance`）；触顶判定由 SQL `RETURNING` pre/post 跨越 bool 单源决定；ledger + outbox 事务内同步 INSERT（commit 失败一起回滚，杜绝「扣费成功但 ledger/通知丢失」）；时间源 `$now` 由 Go 端 `time.Now().UTC()` 传入。

跨越 bool 后续：Go 拿到 bool 后在同一事务内写 `limit_reached` ledger（带 `event_key`）+ enqueue outbox；ledger 唯一索引保证幂等（插不进去则整个分支无副作用）。窗口重置时在 UPDATE 之前先写 `window_reset` ledger，归集到旧窗口起点（用 `CreateLimitReachedEvent` 的 `ON CONFLICT DO NOTHING` 保证幂等）。

### 预扣费与额度不足策略

第一版不做「请求前扣款」（token 请求多在上游返回后才知准确用量，硬扣预估值带来退款/并发回滚/流式中断）。采用「预检 + 事后原子结算」：

1. 请求前只做资格预检，不实际扣订阅额度。
2. 订阅额度 < 预估费用时用双资金源判断：`subscription_available >= estimated` 放行；不足但余额过门槛 → 放行，结算时拆分；不足且余额不可用 → 固定价格请求拒绝，token 请求保持现有宽松策略。
3. 请求后在 DB 事务内按最新余额重新分配。
4. 余额不足时第一版不改后付费语义（差额按现有余额路径，可能扣成负余额；后续可加 `strict_balance_after_billing_enabled` 开关）。
5. 不为 token 请求做强预留（不按 `max_tokens` 预留，不建长事务锁）。

### 购买新订阅规则

未开启 `subscription_multiple_purchases_enabled` 时，允许下单条件（任一）：无 active 订阅 / 当前 active 订阅 `exhausted_at IS NOT NULL` / 当前 active 订阅 `expires_at <= NOW()`。不满足直接拒绝，返回 `SUBSCRIPTION_RENEWAL_NOT_ALLOWED`（带 `subscription_id` / `quota_remaining_usd` / `expires_at`）。日/周限触顶不释放下单门槛。

开启 `subscription_multiple_purchases_enabled` 时，支付下单和履约二次校验都跳过「已有可消费订阅」拦截，直接用订单快照 INSERT 新订阅并写 `purchase` ledger。`expires_at` 从**支付完成时刻**起算。DB 写入失败则回滚，订单回 `paid_unfulfilled` 等重试。

余额支付订阅：第一版不做 `/credit/convert`；走现有支付流程（订单 `payment_method='balance'`，后端扣 `users.balance`，履约同纯 INSERT）。

### 上限通知与错误返回

通知仅两处触发：请求后结算发现某维度首次触顶（SQL `RETURNING` 跨越 bool，事务内写 ledger + outbox）；过期任务发现订阅过期销毁剩余额度。预检阶段不写 ledger。幂等靠 `event_key` 唯一索引。worker 拉 `scheduler_outbox` 的 `subscription_notify` 事件发站内信 + 邮件，**失败仅记日志不重试**（通知非关键路径）。通知正文带「重新订阅」链接（来自设置项 `subscription_credit_pool_repurchase_url`，空则回退前端订阅页）；即使余额仍可用也发通知。

API 错误（订阅不可用且余额不可用）返回结构化 `error.code` + `error.details`（`reason` / `subscription_id` / `renewal_allowed` / `repurchase_url` / `recharge_url` / `reset_at` / `expires_at`）。错误码：`SUBSCRIPTION_CREDIT_EXHAUSTED` / `SUBSCRIPTION_DAILY_LIMIT_REACHED` / `SUBSCRIPTION_WEEKLY_LIMIT_REACHED` / `SUBSCRIPTION_EXPIRED` / `SUBSCRIPTION_RENEWAL_NOT_ALLOWED`。网关鉴权层把订阅不可用（包括日/周窗口限额触顶）统一视为无有效订阅，余额也不可用时返回 HTTP 403 / `SUBSCRIPTION_INVALID`；HTTP 429 保留给真正的速率限制语义。OpenAI/Anthropic 兼容接口尽量保持各自错误格式同时带 `code` 和 `details`。

### 管理员重置周限

2026-07-10 起，管理员订阅列表的 active 订阅提供独立「重置周限」操作。确认后复用 `POST /admin/subscriptions/:id/reset-quota`，请求体固定为 `{ daily: false, weekly: true, monthly: false }`，只清零当前每周用量窗口。该操作不修改总额度 `quota_limit_usd`、累计已用额度 `quota_used_usd`、每日用量、订阅状态或到期时间；成功后管理端重新加载订阅列表。原有日/周/月全部重置操作继续保留，两个动作通过独立确认文案区分。

**管理员路径不读写** `weekly_limit_user_reset_at`，不会消费用户自助重置机会。

### 用户自助重置周限

2026-07-17 起，用户订阅页（可用卡）提供「重置周限」：每个**订阅记录生命周期**（`starts_at`~`expires_at` 所在行）仅一次，不是自然月。

| 项 | 规则 |
| --- | --- |
| API | `POST /api/v1/subscriptions/:id/reset-weekly-limit`（用户 JWT） |
| 字段 | `user_subscriptions.weekly_limit_user_reset_at`（NULL=未用；非 NULL=本行已用过） |
| 响应 | `weekly_limit_reset_remaining`：有周限且 reset_at 为空 → `1`，否则 `0` |
| 效果 | 仅 `weekly_usage_usd=0` + `weekly_window_start=startOfDay(now)` + 写 `weekly_limit_user_reset_at`；**不改** `quota_used_usd` / `quota_limit_usd` / 日限 / status / expires_at |
| 权限 | 仅本人；订阅须 `IsUsable()`；须有 `weekly_limit_usd`；本周期未重置过（CAS：`weekly_limit_user_reset_at IS NULL`） |
| 错误码 | 非本人/不存在 → `SUBSCRIPTION_NOT_FOUND`；不可用 → `SUBSCRIPTION_NOT_USABLE`；无周限 → `SUBSCRIPTION_NO_WEEKLY_LIMIT`；已用过/并发 → `SUBSCRIPTION_WEEKLY_LIMIT_RESET_EXHAUSTED` |
| 前端 | `SubscriptionsView` 可用卡：红「重置周限」与绿「续费」并排；无周限隐藏；remaining=0 禁用；确认弹窗说明每周期一次 + remaining |
| 续费新行 | 新订阅行默认 `weekly_limit_user_reset_at=NULL`，重新获得 1 次机会 |

实现锚点：`SubscriptionService.UserResetWeeklyLimit`、`userSubscriptionRepository.UserResetWeeklyLimit`（CAS）、migration `187_user_subscription_weekly_limit_user_reset_at.sql`、用户路由 `routes/user.go`、前端 `api/subscriptions.ts` + `views/user/SubscriptionsView.vue`。

### 关键边界行为

| 场景 | 行为 |
| --- | --- |
| 订阅足够且未超限 | 全额扣订阅，`billing_type=1` |
| 订阅只够部分 | 订阅扣可用部分，余额扣剩余，`billing_type=2` |
| 开启多订阅且最早订阅只够部分 | 先扣最早订阅可用部分，剩余继续扣下一条订阅；所有订阅不足才扣余额 |
| 开启多订阅且最早订阅日/周限满 | 跳过该订阅的可扣空间，继续尝试下一条订阅 |
| 订阅 < 预估费用 | 不直接失败，预检用双资金源判断 |
| 订阅为 0 / 无可消费订阅 | 全额扣余额，`billing_type=0` |
| 日限满但总额度仍有 | 全额扣余额，发日限通知，**不**允许买新订阅 |
| 周限满但仍有效 | 全额扣余额，发周限通知，**不**允许买新订阅 |
| 管理员重置 active 订阅周限 | 仅清零当前每周用量窗口；总额度、累计已用额度、日用量、状态和到期时间不变；**不**消费用户自助机会 |
| 用户自助重置周限 | 每条订阅记录仅一次；仅清零周窗用量并写 `weekly_limit_user_reset_at`；总额度/累计已用不变 |
| 总额度耗尽（写 exhausted_at） | 全额扣余额，发 total 通知，**允许**买新订阅 |
| 订阅过期 | 不用订阅额度，余额可用则扣；过期任务写 `expire` ledger |
| 余额不足且订阅不可用 | 请求前拒绝 |
| 请求费用为 0 | 不扣订阅/余额，正常写使用日志 |
| 当前订阅未用完未过期时买新订阅 | 未开启多订阅时拒绝，返回 `SUBSCRIPTION_RENEWAL_NOT_ALLOWED`；开启后允许购买 |
| 旧订阅耗尽后买新订阅 | 旧订阅保持 active 等过期；新订阅可消费；开启多订阅后新旧可同时参与按时间顺序扣费 |

## 已决策

- **默认单条可消费订阅，管理员可开启多订阅**——默认保持原购买门槛；开启后允许多条可消费订阅并按 `created_at ASC, id ASC` 扣费。
- **多订阅先用订阅、后用余额**——先遍历所有覆盖当前分组的可消费订阅，最早订阅不足或触顶时继续后续订阅，最后才扣余额。
- **预检 + 事后原子结算，不做请求前硬扣**——token 用量请求后才准确，硬扣会引入退款/回滚/流式中断。
- **订阅优先、不足回落余额、支持单请求拆分扣费**——不阻断用户消费。
- **额度/窗口/有效期用订单快照履约**——管理员改套餐不影响在途订单与已购订阅。
- **触顶判定单源（SQL RETURNING 跨越 bool）**——纯分配函数不返回 Exhausted，避免双源不一致。
- **窗口重置收敛进扣费事务**——废弃异步 `DoWindowMaintenance`，所有窗口操作原子化。
- **ledger + outbox 事务内同步写**——杜绝扣费成功但流水/通知丢失。
- **通知幂等靠 event_key 唯一索引，失败仅记日志不重试**——通知非关键路径，最多发一次。
- **订阅不退款**——过期剩余额度销毁并记账。
- **expires_at 从支付完成时刻起算**——支付失败重试不白送等待时间。

## 待解决 / 实现

实施按 Task 1–10 推进，截至 2026-05-28 代码层全部完成（见各 commit）。2026-06-27 追加多订阅购买与按时间扣费能力。

- **Task 1** — DB migration `141_subscription_credit_pool.sql` + Ent schema（`user_subscription` / `subscription_plan` / `payment_order` / `usage_log` 改造 + 新建 `subscription_credit_ledger`，`go generate ./ent`）。commit `a21a610e`。
- **Task 2** — 领域模型（`subscription_credit.go` 常量/类型）+ `AllocateSubscriptionCredit` 纯函数（`subscription_credit_allocation.go`，仅算 cost 拆分，13 case 测试）。commit `737940c3`。
- **Task 3** — Repository 查询（`GetUsableCreditSubscription` / `HasUsableCreditSubscription` / `GetRenewalEligibility` / `MarkExpiredCreditLogged` / `LockUserForSubscriptionWrite`）+ ledger repo（`Create` / `CreateLimitReachedEvent` ON CONFLICT DO NOTHING / `ListByUserID` / `ListBySubscriptionID`）。commit `ca6d4d84`。
- **Task 4** — 原子混合扣费：事务内 `SELECT FOR UPDATE` + 两步 UPDATE + usage_log 拆分字段，删 `DoWindowMaintenance`。commit `625e372b`。
- **Task 5** — 鉴权 + 双资金源：始终查用户级可消费订阅，新增 `SubscriptionCoversGroup`，订阅不可用回落余额。
- **Task 6** — 购买订阅履约：套餐额度快照 + 续费拦截（`GetRenewalEligibility`）+ 额度池 INSERT + `purchase` ledger。
- **Task 7** — 过期销毁与审计：`ExpireCreditSubscriptions` 事务化过期 + `expire` ledger + `expired` 通知 outbox + `expired_credit_logged_at`。
- **Task 8** — 通知 worker handler：复用 `scheduler_outbox`，失败仅记日志不重试。commit `ca6d4d84`。
- **Task 9** — API DTO + 前端展示：DTO 暴露 `group_id=null`/`plan_id`/`plan_name`/`plan_product_name`/`quota_*`/`daily|weekly_*`/`scope_*`/`is_usable`/`exhausted_at`；用户订阅页/支付页/套餐卡改额度池展示，多订阅展示优先使用套餐名而不是空 `group_id` 的通用兜底名。
- **Task 10** — 管理后台 + 浪费率统计：admin 列表（状态/计划/邮箱/创建时间筛选）/ ledger 查看 / PATCH（调额度/限额/状态/过期时间，自动写 `admin_adjust` ledger）+ `/admin/subscriptions/waste-stats`(`/by-plan`/`/time-series`) 聚合与前端页面。
- **Task 11** — 多订阅购买与按时间扣费：管理员设置 `subscription_multiple_purchases_enabled`；migration `155_allow_multiple_active_credit_subscriptions.sql` 删除单可消费订阅唯一索引；鉴权收集候选订阅 ID；扣费事务按创建时间顺序跨订阅分配，额度或日/周限不足时递补到后一条订阅。

第一版仍不做：`/credit/convert` 余额转额度；按模型族复杂覆盖；复杂退款拆账自动化；通知重试；月限/月度窗口（订阅最长 30 天，月限≡总额度）；灰度/shadow/allowlist。

## 相关

- [[payment]]、[[subscription-pricing]]、[[subscription-admin]]、[[recharge-invoice-balance-gate]]、[[site-messages]]
- 来源：本文由 2026-05-28 订阅额度池实施计划迁移而来；落地见 [PR #44](https://github.com/Go1c/sub2api/pull/44)
- 核心代码：
  - DB/Ent：`backend/migrations/141_subscription_credit_pool.sql`、`backend/migrations/155_allow_multiple_active_credit_subscriptions.sql`、`backend/migrations/187_user_subscription_weekly_limit_user_reset_at.sql`、`backend/ent/schema/{user_subscription,subscription_plan,payment_order,usage_log,subscription_credit_ledger}.go`
  - Service：`backend/internal/service/{subscription_credit,subscription_credit_allocation,subscription_credit_purchase,usage_billing,billing_cache_service,payment_fulfillment,subscription_expiry_service,subscription_notify_worker,subscription_service}.go`（含 `UserResetWeeklyLimit` / `AdminResetQuota`）
  - Repository：`backend/internal/repository/{usage_billing_repo,user_subscription_repo,subscription_credit_ledger_repo,billing_cache}.go`
  - Middleware/Handler：`backend/internal/server/middleware/api_key_auth.go`、`backend/internal/handler/{subscription_handler,admin/subscription_handler,admin/payment_handler,payment_handler}.go`、`backend/internal/handler/dto/types.go`、`backend/internal/server/routes/user.go`
  - 前端：`frontend/src/{types,api,stores}/...`、`frontend/src/views/user/{SubscriptionsView,PaymentView}.vue`、`frontend/src/components/payment/SubscriptionPlanCard.vue`、`frontend/src/views/admin/{SubscriptionsView.vue,orders/PlanEditDialog.vue}`
