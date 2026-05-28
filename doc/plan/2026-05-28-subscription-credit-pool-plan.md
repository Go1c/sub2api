# Subscription Credit Pool Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建用户级订阅额度池。用户购买订阅后，可在任意可用分组消费；扣费时订阅额度优先，订阅不足部分自动从充值余额扣除。支持总额度、日/周限额、有效期、并存订阅（旧订阅未用完前不允许买新订阅）、ledger 审计与上限通知。

**Tech Stack:** Go, Gin, Ent, PostgreSQL, Redis billing cache, Vue 3, Pinia, Vite, Vitest, Go unit/integration tests.

**前提：订阅功能尚未上线**，本方案不考虑线上兼容，直接采用最简最终形态。订阅最长有效期 30 天。

---

## 实施进度（截至 2026-05-28）

**分支：** `feat/subscription-credit-pool`
**PR：** https://github.com/Go1c/sub2api/pull/44

| Task | 状态 | 备注 |
|------|------|------|
| Task 1 — DB migration + Ent schema | ✅ 完成 | commit `a21a610e` |
| Task 2 — 领域模型 + AllocateSubscriptionCredit 纯函数 | ✅ 完成 | commit `737940c3`，13 case 测试 PASS |
| Task 3 — Repository 查询 + ledger repo | ✅ 完成 | commit `ca6d4d84`，含 5 个 SubscriptionCreditExtension 方法 + 4 个 ledger repo 方法 + 集成测试 |
| Task 4 — 原子混合扣费（核心 SQL 改造） | ✅ 完成 | commit `625e372b`，事务内 `SELECT FOR UPDATE` + 混合拆分 + usage_log 拆分字段 |
| Task 5 — 鉴权 + 双资金源 | ✅ 完成 | 本次提交，用户级可消费订阅 + scope 校验 + 订阅/余额双资金源 fallback |
| Task 6 — 购买订阅履约 | 🟡 待实施 | 依赖：Task 3 ✅ |
| Task 7 — 过期销毁与审计 | 🟡 待实施 | 依赖：Task 3 ✅ + Task 8 ✅ |
| Task 8 — 通知 worker handler | ✅ 完成 | commit `ca6d4d84`，复用 `scheduler_outbox`，失败仅记日志不重试 |
| Task 9 — API DTO + 前端展示 | 🟡 待实施 | 依赖：Task 3-7 |
| Task 10 — 管理后台 + 浪费率统计页 | 🟡 待实施 | 依赖：Task 3 ✅，可并行启动 |

**已交付：**
- plan 文档 + window_reset 扩展 + schema + 纯函数 + repo&worker + Task 4 原子混合扣费
- 数据库 migration 141：`user_subscriptions` 改造 / `subscription_credit_ledger` 新表 / `payment_orders` 订阅快照字段 / `usage_logs` 拆分字段
- 部分唯一索引 `user_subscriptions_user_active_usable`：每用户最多 1 条可消费订阅
- service.SubscriptionCreditExtension + SubscriptionCreditLedgerRepository
- service.SubscriptionNotifyService + SubscriptionNotifyWorker（独立轮询，watermark in-memory）
- service.RenewalEligibility 类型 + 错误码
- 通知文案（中文站内信 + HTML 邮件，brand 渐变 `#4f8cff → #1a2f5a`）
- Task 4：扣费事务内锁定订阅、重置日/周窗口、计算订阅/余额拆分、写 consume / limit_reached / window_reset ledger、写 subscription notify outbox、回填 `usage_logs.subscription_cost_usd` / `balance_cost_usd`
- Task 5：鉴权改为始终查询用户级可消费订阅；新增 `SubscriptionCoversGroup`；订阅不可用时回落余额；记录 usage 时不再依赖分组 `subscription_type`

**已验证：**
- `go build ./...` / `go vet ./...` PASS
- `go test ./... -short` PASS（默认 tag）
- `go test -tags=unit ./... -short` PASS（修复了 7 处 stub 实现新接口）
- `go test -tags=integration` 编译 PASS（实际跑需 Docker）
- Task 4：`go test -tags=integration ./internal/repository -run TestUsageBilling -count=1` PASS
- Task 4：`go test -tags=unit ./internal/service -run 'TestBuildUsageBillingCommand|Test.*Subscription.*Billing' -count=1` PASS
- Task 4：`go test -tags=unit ./internal/service ./internal/repository ./internal/server/middleware -count=1` PASS
- Task 4：`go test ./internal/service ./internal/repository ./internal/server/middleware -count=1` PASS
- Task 5：`go test -tags=unit ./internal/server/middleware -run 'TestAPIKeyAuth|TestApiKeyAuthWithSubscriptionGoogle' -count=1` PASS
- Task 5：`go test -tags=unit ./internal/service -run 'TestBillingCacheService|TestCheckBillingEligibility|TestSubscriptionCoversGroup|Test.*RecordUsage.*Subscription' -count=1` PASS
- Task 5：`go test -tags=unit ./internal/service ./internal/server/middleware -count=1` PASS
- Task 5：`go test ./internal/service ./internal/server/middleware -count=1` PASS

**剩余 Task 6-7、9-10 可分两波启动：**
- 第二波（可并行）：Task 6（履约）/ Task 10 后端（浪费率聚合）
- 第三波：Task 7（过期任务）/ Task 9（前端）/ Task 10 前端

---

## 背景与现状

当前订阅实现是"分组订阅"：

- `user_subscriptions.group_id` 必填，并通过唯一索引限制同一用户同一分组只有一条订阅记录：`backend/ent/schema/user_subscription.go`。
- 订阅日限、周限、月限放在 `groups` 表：`backend/ent/schema/group.go`。
- 套餐 `subscription_plans.group_id` 绑定单个分组：`backend/ent/schema/subscription_plan.go`。
- 中间件只在 API Key 所属分组是 `subscription` 类型时读取该分组订阅：`backend/internal/server/middleware/api_key_auth.go`。
- 扣费层当前只能二选一：订阅扣 `SubscriptionCost`，余额扣 `BalanceCost`：`backend/internal/repository/usage_billing_repo.go`。

目标是把"订阅"变成一个用户级额度池：用户购买一个订阅后，可在任意可用分组消费；消费优先使用订阅额度，订阅额度不可用或不足时自动使用充值余额。

---

## 设计原则

1. **用户同时只有一条"可消费"订阅**：通过部分唯一索引在 DB 层强制保证。
2. **支持并存的"等过期"订阅**：旧订阅总额度耗尽（`exhausted_at IS NOT NULL`）但未到期，用户可购买新订阅；新旧订阅同时 `active`，但只有新订阅可消费。
3. **分组仍负责路由和价格**：API Key 仍绑定实际分组，分组倍率、用户专属倍率、账号倍率仍按现有逻辑计算本次费用。
4. **订阅额度优先，但不阻断余额消费**：订阅额度用完、过期、超出日/周限后，不直接拒绝请求；如果用户余额可用，则回落余额扣费。
5. **支持拆分扣费**：单次请求可同时扣订阅额度和充值余额，例如费用 `$3`，订阅可用 `$1`，余额扣 `$2`。
6. **额度和窗口限额使用套餐快照**：用户购买时把额度、有效期天数、日/周限和覆盖范围写入 `payment_orders` 和 `user_subscriptions`，管理员后续改套餐不影响已购订阅。
7. **所有额度变化必须有流水**：购买、消费、过期销毁、管理员调整都写 `subscription_credit_ledger`。
8. **只有"无可消费订阅"时才允许购买新订阅**：用户没有订阅 / 当前订阅总额度耗尽 / 当前订阅已过期 三种情况之一，方可下单。
9. **日/周限触顶不允许买新订阅**：日/周限是节流限制，会按窗口重置，不算"用完"。触顶时回落余额扣费，不释放下单门槛。
10. **订阅不支持退款**：第一版订阅订单一旦支付完成就不走退款流程；旧订阅过期时剩余额度直接销毁并记账。

---

## 数据模型

### 1. `user_subscriptions` 改为订阅额度池

```sql
-- 砍掉旧分组绑定（订阅功能未上线，可直接改）
ALTER TABLE user_subscriptions ALTER COLUMN group_id DROP NOT NULL;
DROP INDEX IF EXISTS user_subscriptions_user_group_unique_active;

-- credit pool 字段
ALTER TABLE user_subscriptions
    ADD COLUMN plan_id BIGINT REFERENCES subscription_plans(id) ON DELETE SET NULL,
    ADD COLUMN scope_type VARCHAR(32) NOT NULL DEFAULT 'all_available_groups',
    ADD COLUMN scope_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN quota_limit_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    ADD COLUMN quota_used_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    ADD COLUMN daily_limit_usd DECIMAL(20,10),
    ADD COLUMN weekly_limit_usd DECIMAL(20,10),
    ADD COLUMN exhausted_at TIMESTAMPTZ,
    ADD COLUMN expired_credit_logged_at TIMESTAMPTZ,
    ADD CONSTRAINT user_subscriptions_quota_limit_positive
        CHECK (quota_limit_usd > 0);

-- 删除原 monthly_* 字段（订阅最长 30 天，月限≡总额度，无需单独维护）
ALTER TABLE user_subscriptions
    DROP COLUMN IF EXISTS monthly_window_start,
    DROP COLUMN IF EXISTS monthly_usage_usd;

-- 部分唯一索引：每个用户最多 1 条"可消费"订阅
CREATE UNIQUE INDEX user_subscriptions_user_active_usable
    ON user_subscriptions(user_id)
    WHERE deleted_at IS NULL
      AND status = 'active'
      AND exhausted_at IS NULL;
```

**字段语义：**

- `status`：`active` / `expired` / `suspended`（**不**再有 `exhausted`，用 `exhausted_at` 时间戳替代）
- `exhausted_at`：总额度耗尽时刻；为 NULL 表示订阅当前仍可消费
- `quota_limit_usd`：套餐快照的总额度，必须 > 0（DB CHECK 约束）
- `quota_used_usd`：累计消费额度，由扣费 SQL 原子递增
- `daily_limit_usd` / `weekly_limit_usd`：套餐快照的窗口节流（NULL 表示该窗口无限制）
- `daily_window_start` / `weekly_window_start`：当前窗口起点，按 `date_trunc('day'/'week', $now)` 对齐
- `daily_usage_usd` / `weekly_usage_usd`：当前窗口累计用量
- `expired_credit_logged_at`：过期销毁 ledger 已写入的标记，避免重复写

**订阅状态机：**

| 可消费状态 | 含义 |
|-----------|------|
| `status='active' AND exhausted_at IS NULL AND deleted_at IS NULL` | 当前扣费目标，每用户最多 1 条 |
| `status='active' AND exhausted_at IS NOT NULL` | 总额度耗尽，等待 `expires_at` 到达；用户已可购买新订阅 |
| `status='expired'` | 由后台过期任务推进 |
| `status='suspended'` | 管理员手动暂停 |

**scope_type 取值：**

- `all_available_groups`：覆盖用户当前可用的所有分组（默认）
- `selected_groups`：覆盖 `scope_config.group_ids`
- `platforms`：覆盖 `scope_config.platforms`，例如 `['anthropic', 'openai']`

### 2. `subscription_plans` 支持额度套餐

```sql
ALTER TABLE subscription_plans
    ADD COLUMN quota_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    ADD COLUMN daily_limit_usd DECIMAL(20,10),
    ADD COLUMN weekly_limit_usd DECIMAL(20,10),
    ADD COLUMN scope_type VARCHAR(32) NOT NULL DEFAULT 'all_available_groups',
    ADD COLUMN scope_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN validity_days INT NOT NULL DEFAULT 30,
    ADD CONSTRAINT subscription_plans_quota_positive
        CHECK (quota_usd > 0),
    ADD CONSTRAINT subscription_plans_validity_days_range
        CHECK (validity_days > 0 AND validity_days <= 30);

ALTER TABLE subscription_plans ALTER COLUMN group_id DROP NOT NULL;
```

`group_id` 改为 NULLABLE。额度套餐不绑定分组，由 `scope_type` / `scope_config` 描述覆盖范围。

### 3. 新增 `subscription_credit_ledger`

```sql
CREATE TABLE IF NOT EXISTS subscription_credit_ledger (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id BIGINT NOT NULL REFERENCES user_subscriptions(id) ON DELETE CASCADE,
    group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    api_key_id BIGINT REFERENCES api_keys(id) ON DELETE SET NULL,
    usage_log_id BIGINT REFERENCES usage_logs(id) ON DELETE SET NULL,
    order_id BIGINT REFERENCES payment_orders(id) ON DELETE SET NULL,
    type VARCHAR(32) NOT NULL,
    delta_usd DECIMAL(20,10) NOT NULL,
    balance_delta_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    remaining_after_usd DECIMAL(20,10) NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    event_key VARCHAR(128),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscription_credit_ledger_user_time
    ON subscription_credit_ledger(user_id, created_at DESC);

CREATE INDEX idx_subscription_credit_ledger_subscription_time
    ON subscription_credit_ledger(subscription_id, created_at DESC);

CREATE UNIQUE INDEX subscription_credit_ledger_event_key_unique
    ON subscription_credit_ledger(subscription_id, type, event_key)
    WHERE event_key IS NOT NULL AND event_key <> '';
```

`type` 固定值：

- `purchase`：购买订阅获得额度（`delta_usd > 0`）
- `consume`：请求消费订阅额度（`delta_usd < 0`，`balance_delta_usd < 0` 表示同次请求余额扣减部分）
- `limit_reached`：总额度、日限或周限到达上限（`delta_usd = 0`，幂等事件，靠 `event_key` 去重）
- `expire`：订阅到期销毁剩余额度（`delta_usd <= 0`，等于过期时刻的剩余）
- `window_reset`：日/周窗口重置时记录被重置窗口的浪费（`delta_usd = 0`，metadata 装 `window/limit_usd/used_before_reset_usd/wasted_usd/wasted_ratio`；用于定价分析）
- `admin_adjust`：管理员手动调整额度（符号视方向）

**`event_key` 格式（UTC RFC3339）：**

- 总额度耗尽：`total:{subscription_id}`
- 日限耗尽：`daily:{subscription_id}:{daily_window_start_rfc3339}`，例如 `daily:123:2026-05-28T00:00:00Z`
- 周限耗尽：`weekly:{subscription_id}:{weekly_window_start_rfc3339}`
- 过期销毁：`expire:{subscription_id}:{expires_at_rfc3339}`
- 窗口重置：`window_reset_{daily|weekly}:{subscription_id}:{old_window_start_rfc3339}`（归集到旧窗口起点）

**`delta_usd` 符号约定：**

- 增加额度（`purchase` / `admin_adjust` 正向）：正数
- 减少额度（`consume` / `expire` / `admin_adjust` 负向）：负数
- 幂等事件（`limit_reached`）：固定 0

### 4. `usage_logs` 支持混合扣费展示

```sql
ALTER TABLE usage_logs
    ADD COLUMN subscription_cost_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    ADD COLUMN balance_cost_usd DECIMAL(20,10) NOT NULL DEFAULT 0;
```

`billing_type` 扩展为：

- `0`：纯余额
- `1`：纯订阅
- `2`：订阅 + 余额混合

订阅功能未上线，新列不需要回填历史数据。

### 5. `payment_orders` 保存订阅快照

```sql
ALTER TABLE payment_orders
    ADD COLUMN subscription_quota_usd DECIMAL(20,10),
    ADD COLUMN subscription_daily_limit_usd DECIMAL(20,10),
    ADD COLUMN subscription_weekly_limit_usd DECIMAL(20,10),
    ADD COLUMN subscription_scope_type VARCHAR(32),
    ADD COLUMN subscription_scope_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN subscription_validity_days INT;
```

履约时**完全使用订单快照**生成订阅，不读 `subscription_plans` 当前值，避免管理员改套餐影响在途订单。

### 6. 通知队列复用 `scheduler_outbox`

不新建表，复用现有 `scheduler_outbox`（`migrations/036_scheduler_outbox.sql`）。新增 event_type 常量：

```go
const SchedulerOutboxEventSubscriptionNotify = "subscription_notify"
```

payload 结构：

```json
{
  "user_id": 123,
  "subscription_id": 456,
  "kind": "limit_reached_total"
}
```

`kind` 取值：`limit_reached_total` / `limit_reached_daily` / `limit_reached_weekly` / `expired`。`account_id` 和 `group_id` 留 NULL。

---

## 核心消费流程

1. API Key 鉴权后拿到用户、API Key、实际分组。
2. Service 层查"用户当前可消费订阅"（`status='active' AND exhausted_at IS NULL AND expires_at > NOW() AND deleted_at IS NULL`），结合 scope 校验当前分组是否被覆盖。
3. 资格预检（"双资金源"）：
   - 订阅覆盖且有可用额度 → 放行；
   - 订阅不覆盖或不可用，但余额满足现有余额门槛 → 放行；
   - 两者都不可用 → 返回结构化错误（见下文"错误返回"章节）。
4. 上游请求完成后计算 `ActualCost`。
5. 在 `UsageBillingRepository.Apply` 的同一事务内（详见下一章节"扣费 SQL"）：
   - `SELECT ... FOR UPDATE` 锁订阅行；
   - 用 `AllocateSubscriptionCredit` 纯函数算 `subscription_cost` / `balance_cost`；
   - UPDATE 订阅：写 `quota_used_usd` / 窗口用量 / 窗口重置 / `exhausted_at`，`RETURNING` 返回触顶跨越 bool；
   - UPDATE 用户余额（如有余额部分）；
   - INSERT `subscription_credit_ledger`（`consume` + 必要的 `limit_reached`）；
   - INSERT `scheduler_outbox`（`subscription_notify` 事件）；
   - INSERT `usage_logs` 拆分字段。
6. 事务提交后由 outbox worker 异步发送通知。

---

## 扣费 SQL 改造（核心）

替换 `backend/internal/repository/usage_billing_repo.go:148-174` 的 `incrementUsageBillingSubscription`。改造后分两步：

### 第 1 步：锁行 + 读状态

```sql
SELECT
  quota_limit_usd, quota_used_usd,
  daily_limit_usd, daily_usage_usd, daily_window_start,
  weekly_limit_usd, weekly_usage_usd, weekly_window_start,
  exhausted_at, expires_at, scope_type, scope_config
FROM user_subscriptions
WHERE id = $1 AND deleted_at IS NULL AND status = 'active'
FOR UPDATE
```

Go 侧基于读到的 window_start 判断"是否需要重置"（按 `date_trunc('day', $now)` 和 `date_trunc('week', $now)` 对齐 UTC 自然日/自然周）。然后调用 `AllocateSubscriptionCredit` 算 `subscription_cost` / `balance_cost`。

### 第 2 步：原子 UPDATE

```sql
UPDATE user_subscriptions SET
  quota_used_usd = quota_used_usd + $sub_cost,
  daily_usage_usd  = CASE WHEN $reset_daily  THEN $sub_cost ELSE daily_usage_usd  + $sub_cost END,
  weekly_usage_usd = CASE WHEN $reset_weekly THEN $sub_cost ELSE weekly_usage_usd + $sub_cost END,
  daily_window_start  = CASE WHEN $reset_daily  THEN $now_daily_start  ELSE daily_window_start  END,
  weekly_window_start = CASE WHEN $reset_weekly THEN $now_weekly_start ELSE weekly_window_start END,
  exhausted_at = CASE
    WHEN exhausted_at IS NULL
      AND quota_used_usd + $sub_cost >= quota_limit_usd
    THEN $now
    ELSE exhausted_at
  END,
  updated_at = $now
WHERE id = $sub_id AND deleted_at IS NULL
RETURNING
  quota_used_usd, quota_limit_usd,
  daily_usage_usd, daily_limit_usd,
  weekly_usage_usd, weekly_limit_usd,
  exhausted_at,
  -- 三个跨越 bool：本次扣费让某维度首次触顶
  (exhausted_at IS NOT NULL AND $pre_exhausted_at IS NULL) AS just_exhausted_total,
  (daily_limit_usd  IS NOT NULL AND daily_usage_usd  >= daily_limit_usd  AND $pre_daily_usage  < daily_limit_usd)  AS just_hit_daily,
  (weekly_limit_usd IS NOT NULL AND weekly_usage_usd >= weekly_limit_usd AND $pre_weekly_usage < weekly_limit_usd) AS just_hit_weekly
```

**关键设计点：**

| 点 | 选择 | 理由 |
|----|------|------|
| 行锁 | `SELECT FOR UPDATE` 事务内持有 | 混合扣费必须先读后算，无锁会两请求都读到 $1 余额各扣 $1 |
| 窗口重置 | 事务内同一 UPDATE 完成 | 不再使用现有 `DoWindowMaintenance` 异步路径；所有窗口操作收敛到扣费事务 |
| 触顶判定 | SQL `RETURNING` 返回 pre/post 跨越 bool | 学现有 `incrementUsageBillingAccountQuota` 的 `crossedTotal/crossedDaily/crossedWeekly` 模式 |
| ledger + outbox 写入 | 事务内同步 INSERT | commit 失败一起回滚；不会出现"扣费成功但 ledger / 通知丢失" |
| 时间源 | `$now` 由 Go 端传入（`time.Now().UTC()`） | 单次扣费所有"现在"是同一个值，方便测试和审计 |
| 窗口对齐 | `date_trunc('day' / 'week', $now)` UTC | 与现有 `incrementUsageBillingAccountQuota` 一致，前端展示"今日已用/本周已用"易实现 |

### 跨越 bool 后续处理

Go 拿到 `just_exhausted_total` / `just_hit_daily` / `just_hit_weekly` 后，在同一事务内：

```go
if just_exhausted_total {
    insertLedger(tx, type="limit_reached", event_key="total:{sub_id}", delta=0)
    enqueueSchedulerOutbox(tx, "subscription_notify", payload{user_id, sub_id, kind:"limit_reached_total"})
}
if just_hit_daily {
    eventKey := fmt.Sprintf("daily:%d:%s", subID, dailyWindowStart.UTC().Format(time.RFC3339))
    insertLedger(tx, type="limit_reached", event_key=eventKey, delta=0)
    enqueueSchedulerOutbox(tx, "subscription_notify", payload{user_id, sub_id, kind:"limit_reached_daily"})
}
// weekly 同理
```

ledger `event_key` 唯一索引保证幂等：如果 ledger 这次插不进去（已存在），整个 if 分支无副作用。

### 窗口重置时写浪费记录

当 `resetDaily=true` 或 `resetWeekly=true` 时，**在同一事务内、UPDATE 之前**先写一条 `window_reset` ledger，归集到旧窗口起点（语义"那一周/那一天的浪费"）：

```go
if resetDaily && state.DailyLimitUSD != nil && state.DailyWindowStart != nil {
    oldUsed := state.DailyUsageUSD
    limit := *state.DailyLimitUSD
    wasted := limit - oldUsed
    if wasted < 0 { wasted = 0 }
    eventKey := fmt.Sprintf("window_reset_daily:%d:%s", subID, state.DailyWindowStart.UTC().Format(time.RFC3339))
    ledgerRepo.CreateLimitReachedEvent(ctx, tx, &SubscriptionCreditLedgerEntry{
        UserID: cmd.UserID, SubscriptionID: subID,
        Type: SubscriptionCreditLedgerWindowReset,
        DeltaUSD: 0, RemainingAfterUSD: state.QuotaLimitUSD - state.QuotaUsedUSD,
        EventKey: &eventKey,
        Reason: "daily window reset",
        Metadata: map[string]any{
            "window":                "daily",
            "limit_usd":             limit,
            "used_before_reset_usd": oldUsed,
            "wasted_usd":            wasted,
            "wasted_ratio":          wasted / limit,
            "old_window_start":      state.DailyWindowStart.UTC().Format(time.RFC3339),
        },
    })
}
// weekly 同理
```

`CreateLimitReachedEvent`（沿用 `ON CONFLICT DO NOTHING` 机制）保证幂等：并发请求同时触发重置时只会留一条 ledger。`window_reset` 不发通知，仅供后台统计。

---

## 预扣费与额度不足策略

第一版不做"请求前扣款"。token 请求大多在上游返回后才知准确用量，请求前硬扣预估值会带来退款、并发回滚、流式中断等复杂问题。

采用"预检 + 事后原子结算"：

1. **请求前只做资格预检，不实际扣订阅额度**。
2. **订阅额度小于预估费用时，预检使用双资金源判断**：
   - `subscription_available >= estimated_cost` → 放行；
   - `subscription_available < estimated_cost` 且用户余额通过现有余额门槛 → 放行，结算时拆分；
   - `subscription_available < estimated_cost` 且余额也不可用 → 固定价格请求直接拒绝，token 请求保持现有宽松策略。
3. **请求后在数据库事务内按最新余额重新分配**（详见上一章节"扣费 SQL"）。
4. **余额不足时第一版不改变线上后付费语义**：订阅不足的差额按现有余额扣费路径处理，可能扣成负余额。后续可加 `strict_balance_after_billing_enabled` 开关收紧。
5. **不为 token 请求做强预留**：不按 `max_tokens` 预留最大费用，不为流式请求建立长事务锁。

### 典型例子

| 场景 | 请求前判断 | 请求后结算 |
| --- | --- | --- |
| 预估 `$3`，订阅可用 `$10` | 放行 | 实际 `$2.8`，全额扣订阅 |
| 预估 `$3`，订阅可用 `$1`，余额可用 | 放行 | 实际 `$2.8`，订阅扣 `$1`，余额扣 `$1.8` |
| 预估 `$3`，订阅可用 `$1`，余额不可用，固定价格请求 | 拒绝 | 不请求上游 |
| token 请求无法准确预估，订阅有少量余额，钱包余额为正 | 放行 | 实际费用按事务内最新额度拆分 |
| 并发导致结算时订阅从 `$1` 变为 `$0` | 请求已放行 | 本次全额扣余额 |

---

## 购买新订阅规则

### 允许购买的条件

满足任一条件即可下单：

- 用户没有 active 订阅；
- 当前 active 订阅 `exhausted_at IS NOT NULL`（总额度已耗尽，等待过期）；
- 当前 active 订阅 `expires_at <= NOW()`（已过期，待后台任务推进为 `expired`）。

不满足时直接拒绝订单：

```json
{
  "code": "SUBSCRIPTION_RENEWAL_NOT_ALLOWED",
  "message": "当前订阅仍可使用，只有额度用完或过期后才可以重新订阅",
  "data": {
    "subscription_id": 123,
    "quota_remaining_usd": 38.5,
    "expires_at": "2026-06-28T00:00:00Z"
  }
}
```

**日/周限触顶不释放下单门槛**：日/周窗口会重置，订阅本身仍可消费，不允许提前买新订阅。

### 履约：纯 INSERT，不动旧订阅

```go
func FulfillSubscriptionOrder(ctx, order) error {
    return tx.RunInTx(func(tx) {
        // 1. 锁用户行，防并发
        lockUser(tx, order.UserID)

        // 2. 二次校验"无可消费订阅"（理论必然满足，做防御）
        if hasUsableSubscription(tx, order.UserID) {
            return ErrAlreadyHasUsableSubscription  // 触发 ops 告警 + 人工对账
        }

        // 3. INSERT 新订阅，使用订单快照
        sub := insertNewSubscription(tx, NewSubscriptionInput{
            UserID:        order.UserID,
            PlanID:        order.PlanID,
            ScopeType:     order.SubscriptionScopeType,
            ScopeConfig:   order.SubscriptionScopeConfig,
            QuotaLimitUSD: order.SubscriptionQuotaUSD,
            DailyLimit:    order.SubscriptionDailyLimitUSD,
            WeeklyLimit:   order.SubscriptionWeeklyLimitUSD,
            StartsAt:      now,
            ExpiresAt:     now.AddDate(0, 0, order.SubscriptionValidityDays),
            Status:        "active",
            // exhausted_at = NULL, quota_used_usd = 0, window fields NULL
        })

        // 4. 写 purchase ledger
        insertLedger(tx, "purchase", +order.SubscriptionQuotaUSD, subscription_id=sub.ID, order_id=order.ID)
    })
}
```

**关键设计点：**

| 点 | 选择 | 理由 |
|----|------|------|
| 不动旧订阅 | 旧订阅已 `exhausted_at IS NOT NULL` 或 `expired`，自然不占唯一索引 | 唯一索引 `user_subscriptions_user_active_usable` 兜底防双订阅 |
| 用户行加锁 | `SELECT FOR UPDATE users WHERE id=?` | 防 webhook 重试 + 用户手动重试并发进入履约 |
| 二次校验 | 必做防御 | 下单到支付窗口理论不该有可消费订阅复活，但加防御成本极低 |
| `expires_at` 起算 | **支付完成时刻**（履约时 `now`） | 用户支付失败重试不会"白送"等待时间 |
| 套餐快照来源 | 完全使用 `payment_orders.subscription_*` | 管理员改套餐不影响在途订单 |
| `validity_days` 快照 | 必须快照到 `payment_orders.subscription_validity_days` | 否则套餐改 days 后履约时长被改变 |

### 履约失败处理

- `ErrAlreadyHasUsableSubscription`：理论不该发生。一旦发生，订单状态置 `fulfillment_blocked`，触发 ops 告警 + 人工对账。不自动退款。
- DB 写入失败：事务回滚，订单状态回到 `paid_unfulfilled`，由 webhook 重试或人工介入。

### 余额支付订阅

第一版**不**做"余额转换订阅额度"接口（`/credit/convert`）。如果运营需要"用余额买订阅"，走现有支付流程：
1. 订阅订单选 `payment_method='balance'`；
2. 后端扣 `users.balance`，履约走上面的纯 INSERT ��程；
3. 与外部支付完全同构。

---

## 上限通知与错误返回

### 通知触发时机

仅两处：

1. **请求后结算发现本次消费让某维度首次触顶**（由扣费 SQL `RETURNING` 跨越 bool 触发，事务内同步写 ledger + outbox）。
2. **过期任务发现订阅已过期并销毁剩余额度**（事务内写 `expire` ledger + outbox）。

**不**在请求前预检阶段写 ledger。预检只判资格不写库。

### 幂等保证

每条 `limit_reached` / `expire` ledger 都有 `event_key`，唯一索引 `subscription_credit_ledger_event_key_unique` 保证同一窗口只写一次。outbox 入队失败不影响扣费事务，但 ledger 写入失败会回滚整个事务。

### 通知发送

后台 worker 拉取 `scheduler_outbox` 中 `event_type='subscription_notify'` 的事件：

```go
case "subscription_notify":
    payload := parsePayload(event.Payload)
    siteMessageService.Send(payload.UserID, title, body)
    emailService.Send(payload.UserID, subject, body)
    // 失败仅记日志，不重试
```

**通知失败仅记日志不重试。** 理由：通知不是关键路径；用户后续请求会再次命中"订阅不可用"，前端/API 错误码已告知用户。

### 通知文案

- 站内消息：`SiteMessageService`，标题包含"订阅额度已用完"或"订阅已到达日/周限"或"订阅已过期"。
- 邮件：`EmailService`，收件人使用用户主邮箱和已验证的额外通知邮箱。
- 通知正文带"重新订阅"链接；链接来自设置项 `subscription_credit_pool_repurchase_url`，为空时回退到前端订阅页。
- 即使余额仍可用，仍发通知，提醒用户可重新订阅。

### API 错误返回

当订阅不可用且余额也不可用时返回结构化错误：

```json
{
  "error": {
    "code": "SUBSCRIPTION_WEEKLY_LIMIT_REACHED",
    "message": "订阅已到达周限，且账户余额不足，请重新订阅或充值后继续使用",
    "details": {
      "reason": "weekly_limit_reached",
      "subscription_id": 123,
      "renewal_allowed": false,
      "repurchase_url": "https://example.com/subscriptions",
      "recharge_url": "https://example.com/payment",
      "reset_at": "2026-06-01T00:00:00Z",
      "expires_at": "2026-06-28T00:00:00Z"
    }
  }
}
```

错误码：

- `SUBSCRIPTION_CREDIT_EXHAUSTED`：总额度耗尽
- `SUBSCRIPTION_DAILY_LIMIT_REACHED`：日限耗尽
- `SUBSCRIPTION_WEEKLY_LIMIT_REACHED`：周限耗尽
- `SUBSCRIPTION_EXPIRED`：订阅过期
- `SUBSCRIPTION_RENEWAL_NOT_ALLOWED`：当前订阅仍可使用，不能提前购买新订阅

OpenAI/Anthropic 兼容接口应尽量保持各自错误格式，同时带上 `code` 和 `details`。

---

## 关键边界行为

| 场景 | 行为 |
| --- | --- |
| 订阅额度足够且未超限 | 全额扣订阅，`billing_type=1` |
| 订阅额度只够部分费用 | 订阅扣可用部分，余额扣剩余，`billing_type=2` |
| 订阅额度小于请求前预估费用 | 不直接失败；预检用双资金源判断 |
| 订阅额度为 0 / 无可消费订阅 | 全额扣余额，`billing_type=0` |
| 订阅日限满但总额度仍有 | 全额扣余额，发送日限通知，**不**允许买新订阅 |
| 订阅周限满但仍在有效期 | 全额扣余额，发送周限通知，**不**允许买新订阅 |
| 订阅总额度耗尽（exhausted_at 写入） | 全额扣余额，发送 total 通知，**允许**买新订阅 |
| 订阅过期 | 不使用订阅额度，余额可用则扣余额；过期任务写 `expire` ledger |
| 余额不足且订阅也不可用 | 请求前拒绝 |
| 请求费用为 0 | 不扣订阅、不扣余额，正常写使用日志 |
| 订阅到期仍有剩余 | 过期任务写 `expire` 流水，`expired_credit_logged_at` 标记 |
| 当前订阅未用完且未过期时购买新订阅 | 拒绝订单，返回 `SUBSCRIPTION_RENEWAL_NOT_ALLOWED` |
| 旧订阅总额度耗尽后买新订阅 | 旧订阅保持 active 等过期；新订阅 INSERT 后成为唯一可消费 |

---

## 文件改造地图

### 数据库与 Ent

- Create: `backend/migrations/141_subscription_credit_pool.sql`
- Modify: `backend/ent/schema/user_subscription.go`
- Modify: `backend/ent/schema/subscription_plan.go`
- Modify: `backend/ent/schema/payment_order.go`
- Modify: `backend/ent/schema/usage_log.go`
- Create: `backend/ent/schema/subscription_credit_ledger.go`
- Modify: Ent generated 文件需要 `go generate ./ent` 重新生成（命令见 `ent/generate.go:6`）

### 服务层

- Modify: `backend/internal/service/user_subscription.go`
- Modify: `backend/internal/service/user_subscription_port.go`
- Modify: `backend/internal/service/subscription_service.go`（删除 `DoWindowMaintenance` 异步路径）
- Create: `backend/internal/service/subscription_credit.go`（常量 + 类型）
- Create: `backend/internal/service/subscription_credit_allocation.go`（纯函数）
- Create: `backend/internal/service/subscription_credit_purchase.go`（履约逻辑）
- Modify: `backend/internal/service/usage_billing.go`
- Modify: `backend/internal/service/gateway_service.go`
- Modify: `backend/internal/service/openai_gateway_service.go`
- Modify: `backend/internal/service/billing_cache_service.go`
- Modify: `backend/internal/service/payment_config_plans.go`
- Modify: `backend/internal/service/payment_order.go`
- Modify: `backend/internal/service/payment_fulfillment.go`
- Modify: `backend/internal/service/subscription_expiry_service.go`
- Create: `backend/internal/service/subscription_notify_worker.go`（处理 `subscription_notify` outbox 事件的 handler）

### Repository

- Modify: `backend/internal/repository/user_subscription_repo.go`
- Modify: `backend/internal/repository/usage_billing_repo.go`（核心改造）
- Create: `backend/internal/repository/subscription_credit_ledger_repo.go`
- Modify: `backend/internal/repository/billing_cache.go`

### Middleware 与 Handler

- Modify: `backend/internal/server/middleware/api_key_auth.go`
- Modify: `backend/internal/server/middleware/api_key_auth_google.go`
- Modify: `backend/internal/handler/subscription_handler.go`
- Modify: `backend/internal/handler/admin/subscription_handler.go`
- Modify: `backend/internal/handler/dto/types.go`
- Modify: `backend/internal/handler/admin/payment_handler.go`
- Modify: `backend/internal/handler/payment_handler.go`

### 前端

- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/types/payment.ts`
- Modify: `frontend/src/api/subscriptions.ts`
- Modify: `frontend/src/api/payment.ts`
- Modify: `frontend/src/stores/subscriptions.ts`
- Modify: `frontend/src/views/user/SubscriptionsView.vue`
- Modify: `frontend/src/views/user/PaymentView.vue`
- Modify: `frontend/src/components/payment/SubscriptionPlanCard.vue`
- Modify: `frontend/src/views/admin/orders/PlanEditDialog.vue`
- Modify: `frontend/src/views/admin/SubscriptionsView.vue`

---

## 实施任务

### Task 1: 数据库迁移与 Ent schema ✅ 已完成（commit a21a610e）

**Files:**
- Create: `backend/migrations/141_subscription_credit_pool.sql`
- Modify: `backend/ent/schema/user_subscription.go`
- Modify: `backend/ent/schema/subscription_plan.go`
- Modify: `backend/ent/schema/payment_order.go`
- Modify: `backend/ent/schema/usage_log.go`
- Create: `backend/ent/schema/subscription_credit_ledger.go`

- [ ] **Step 1: 写迁移文件**

Create `backend/migrations/141_subscription_credit_pool.sql`，按"数据模型"章节的 SQL 顺序：

1. `user_subscriptions` 改造（DROP NOT NULL、DROP 旧索引、ADD 新字段、DROP 月限字段、CREATE 部分唯一索引、CHECK 约束）；
2. `subscription_plans` 改造（ADD 字段、ALTER group_id NULLABLE、CHECK 约束）；
3. CREATE `subscription_credit_ledger` 表 + 索引；
4. `usage_logs` 新增 `subscription_cost_usd` / `balance_cost_usd`；
5. `payment_orders` 新增订阅快照字段。

订阅功能未上线，不需要 backfill 历史数据。

- [ ] **Step 2: 更新 Ent schema**

Update `UserSubscription`：把 `group_id` 改为 Optional + Nillable；新增 `plan_id`, `scope_type`, `scope_config`, `quota_limit_usd`, `quota_used_usd`, `daily_limit_usd`, `weekly_limit_usd`, `exhausted_at`, `expired_credit_logged_at`；删除 `monthly_window_start`, `monthly_usage_usd`。状态值仍为 `active` / `expired` / `suspended`，"耗尽"由 `exhausted_at` 时间戳表达。

Update `SubscriptionPlan`：新增 `quota_usd`, `daily_limit_usd`, `weekly_limit_usd`, `scope_type`, `scope_config`, `validity_days`；`group_id` 改 Optional + Nillable。

Update `PaymentOrder`：新增 `subscription_quota_usd`, `subscription_daily_limit_usd`, `subscription_weekly_limit_usd`, `subscription_scope_type`, `subscription_scope_config`, `subscription_validity_days`。

Update `UsageLog`：新增 `subscription_cost_usd`, `balance_cost_usd`。

Create `SubscriptionCreditLedger` schema，字段对照 SQL。

- [ ] **Step 3: 生成 Ent 代码**

```bash
cd backend
go generate ./ent
```

完整命令在 `ent/generate.go:6`：`go run -mod=mod entgo.io/ent/cmd/ent generate --feature sql/upsert,intercept,sql/execquery,sql/lock --idtype int64 ./schema`

Expected: 生成的 Ent 文件包含 `subscriptioncreditledger` 包和新字段 setter/getter；`usersubscription` 包的 group_id 变 Optional。

- [ ] **Step 4: 运行 schema 测试**

```bash
cd backend
go test ./internal/repository -run 'TestMigrations|TestUserSubscriptionSchema' -count=1
```

Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add backend/migrations/141_subscription_credit_pool.sql backend/ent
git commit -m "feat: add subscription credit pool schema"
```

### Task 2: 订阅领域模型与纯分配逻辑 ✅ 已完成（commit 737940c3）

**Files:**
- Modify: `backend/internal/service/user_subscription.go`
- Modify: `backend/internal/service/user_subscription_port.go`
- Create: `backend/internal/service/subscription_credit.go`
- Create: `backend/internal/service/subscription_credit_allocation.go`
- Test: `backend/internal/service/subscription_credit_allocation_test.go`

- [ ] **Step 1: 增加服务层字段**

Extend `service.UserSubscription`：新增 `PlanID`, `ScopeType`, `ScopeConfig`, `QuotaLimitUSD`, `QuotaUsedUSD`, `DailyLimitUSD`, `WeeklyLimitUSD`, `ExhaustedAt`, `ExpiredCreditLoggedAt`。`GroupID` 改为 `*int64`。删除月限相关字段。

- [ ] **Step 2: 定义核心类型**

Create `backend/internal/service/subscription_credit.go`：

```go
package service

import "time"

const (
    SubscriptionScopeAllAvailableGroups = "all_available_groups"
    SubscriptionScopeSelectedGroups     = "selected_groups"
    SubscriptionScopePlatforms          = "platforms"

    SubscriptionCreditLedgerPurchase     = "purchase"
    SubscriptionCreditLedgerConsume      = "consume"
    SubscriptionCreditLedgerLimitReached = "limit_reached"
    SubscriptionCreditLedgerExpire       = "expire"
    SubscriptionCreditLedgerWindowReset  = "window_reset"
    SubscriptionCreditLedgerAdminAdjust  = "admin_adjust"

    // LimitReached 事件类型（用于 event_key 前缀和 outbox payload kind）
    SubscriptionLimitReachedTotal  = "total"
    SubscriptionLimitReachedDaily  = "daily"
    SubscriptionLimitReachedWeekly = "weekly"

    SchedulerOutboxEventSubscriptionNotify = "subscription_notify"
)

type SubscriptionScopeConfig struct {
    GroupIDs  []int64  `json:"group_ids,omitempty"`
    Platforms []string `json:"platforms,omitempty"`
}

type SubscriptionCreditLedgerEntry struct {
    ID                int64
    UserID            int64
    SubscriptionID    int64
    GroupID           *int64
    APIKeyID          *int64
    UsageLogID        *int64
    OrderID           *int64
    Type              string
    DeltaUSD          float64
    BalanceDeltaUSD   float64
    RemainingAfterUSD float64
    Reason            string
    EventKey          *string
    Metadata          map[string]any
    CreatedAt         time.Time
}
```

- [ ] **Step 3: 实现纯分配函数**

Create `backend/internal/service/subscription_credit_allocation.go`：

```go
package service

import "math"

type SubscriptionCreditWindowState struct {
    LimitUSD *float64
    UsedUSD  float64
}

type SubscriptionCreditAllocationInput struct {
    ActualCostUSD float64
    QuotaLimitUSD float64
    QuotaUsedUSD  float64
    Daily         SubscriptionCreditWindowState
    Weekly        SubscriptionCreditWindowState
}

// 仅做 cost 拆分计算。触顶判定由 SQL RETURNING 跨越 bool 单源决定，
// 本函数不返回 Exhausted 字段，避免双源不一致。
type SubscriptionCreditAllocation struct {
    SubscriptionCostUSD float64
    BalanceCostUSD      float64
}

func AllocateSubscriptionCredit(in SubscriptionCreditAllocationInput) SubscriptionCreditAllocation {
    cost := math.Max(in.ActualCostUSD, 0)
    available := math.Max(in.QuotaLimitUSD-in.QuotaUsedUSD, 0)
    available = minByWindow(available, in.Daily)
    available = minByWindow(available, in.Weekly)
    subCost := math.Min(cost, available)
    return SubscriptionCreditAllocation{
        SubscriptionCostUSD: subCost,
        BalanceCostUSD:      math.Max(cost-subCost, 0),
    }
}

func minByWindow(current float64, w SubscriptionCreditWindowState) float64 {
    if w.LimitUSD == nil || *w.LimitUSD <= 0 {
        return current
    }
    remaining := math.Max(*w.LimitUSD-w.UsedUSD, 0)
    if remaining < current {
        return remaining
    }
    return current
}
```

- [ ] **Step 4: 写分配测试**

Create `backend/internal/service/subscription_credit_allocation_test.go`，覆盖：

- 全额扣订阅（available 充足）
- 混合扣费（available < cost）
- 总额度耗尽（available=0 → subCost=0，全扣余额）
- 日限收紧 available（daily.LimitUSD - daily.UsedUSD < quota_remaining）
- 周限收紧 available
- 零费用请求（cost=0 → 全 0）
- quota_used > quota_limit 容错（available 取 max(0, ...)）
- LimitUSD = nil 视为无限制

- [ ] **Step 5: Run tests**

```bash
cd backend
go test ./internal/service -run TestAllocateSubscriptionCredit -count=1
```

Expected: PASS。

- [ ] **Step 6: Commit**

```bash
git add backend/internal/service/subscription_credit*.go backend/internal/service/user_subscription*.go
git commit -m "feat: model subscription credit allocation"
```

### Task 3: Repository 查询与流水写入 ✅ 已完成（commit ca6d4d84）

**Files:**
- Modify: `backend/internal/repository/user_subscription_repo.go`
- Create: `backend/internal/repository/subscription_credit_ledger_repo.go`
- Test: `backend/internal/repository/user_subscription_repo_integration_test.go`
- Test: `backend/internal/repository/subscription_credit_ledger_repo_test.go`

- [ ] **Step 1: 更新现有 Repository**

Update `userSubscriptionEntityToService` 和 `Create`/`Update` 方法以匹配新 schema（`group_id` 可空、新增字段）。

- [ ] **Step 2: 新增 repo 方法**

Extend `UserSubscriptionRepository`：

```go
GetUsableCreditSubscription(ctx context.Context, userID int64) (*UserSubscription, error)
HasUsableCreditSubscription(ctx context.Context, userID int64) (bool, error)
GetRenewalEligibility(ctx context.Context, userID int64) (RenewalEligibility, error)
MarkExpiredCreditLogged(ctx context.Context, id int64, loggedAt time.Time) error
LockUserForSubscriptionWrite(ctx context.Context, tx *sql.Tx, userID int64) error
```

`GetUsableCreditSubscription` 查询条件：

```go
usersubscription.UserIDEQ(userID),
usersubscription.StatusEQ(service.SubscriptionStatusActive),
usersubscription.ExhaustedAtIsNil(),
usersubscription.ExpiresAtGT(time.Now()),
usersubscription.DeletedAtIsNil(),
```

`GetRenewalEligibility` 返回 enum 而非错误码字符串：

```go
type RenewalReason string
const (
    RenewalReasonNoSubscription RenewalReason = "no_subscription"
    RenewalReasonExhausted      RenewalReason = "exhausted"
    RenewalReasonExpired        RenewalReason = "expired"
    RenewalReasonNotExhausted   RenewalReason = "not_exhausted"  // 不允许买新订阅
)

type RenewalEligibility struct {
    Allowed      bool
    Reason       RenewalReason
    Subscription *UserSubscription // 当前 active 订阅，nil 表示没有
}
```

错误码 `SUBSCRIPTION_RENEWAL_NOT_ALLOWED` 由 handler 层基于 `Reason` 映射。

- [ ] **Step 3: 新增 ledger repo**

Create `SubscriptionCreditLedgerRepository`：

```go
Create(ctx context.Context, exec sqlExecutor, entry *service.SubscriptionCreditLedgerEntry) error
CreateLimitReachedEvent(ctx context.Context, exec sqlExecutor, entry *service.SubscriptionCreditLedgerEntry) (created bool, err error)
ListByUserID(ctx context.Context, userID int64, params pagination.PaginationParams) ([]service.SubscriptionCreditLedgerEntry, *pagination.PaginationResult, error)
ListBySubscriptionID(ctx context.Context, subscriptionID int64, params pagination.PaginationParams) ([]service.SubscriptionCreditLedgerEntry, *pagination.PaginationResult, error)
```

`CreateLimitReachedEvent` 使用 `ON CONFLICT DO NOTHING`，返回 `created` bool 表示是否首次写入。所有 Create 方法接受 `sqlExecutor`，支持在已有事务内调用。

- [ ] **Step 4: Run repository tests**

```bash
cd backend
go test ./internal/repository -run 'TestUserSubscriptionRepository|TestSubscriptionCreditLedger' -count=1
```

Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add backend/internal/repository backend/internal/service/user_subscription_port.go
git commit -m "feat: add subscription credit repositories"
```

### Task 4: 原子混合扣费 ✅ 已完成（本次提交）

**Files:**
- Modify: `backend/internal/service/usage_billing.go`
- Modify: `backend/internal/repository/usage_billing_repo.go`
- Modify: `backend/internal/service/gateway_service.go`
- Modify: `backend/internal/service/openai_gateway_service.go`
- Modify: `backend/internal/service/subscription_service.go`（删除 `DoWindowMaintenance`）
- Test: `backend/internal/repository/usage_billing_repo_integration_test.go`
- Test: `backend/internal/service/gateway_service_subscription_billing_test.go`

- [x] **Step 1: 复用现有 `UsageBillingCommand`**

不新增 `PreferredSubscriptionID` 和 `AllowMixedBilling`。继续使用现有 `SubscriptionID *int64` 字段：

- `SubscriptionID != nil` → 走订阅扣费 + 可能的余额拆分；
- `SubscriptionID == nil` → 跳过订阅环节，纯扣余额。

由 service 层在调用前查好"用户当前可消费订阅"赋值给 `SubscriptionID`。`BalanceCost` 和 `SubscriptionCost` 字段保留作为"预计算"输入；当 `SubscriptionID != nil` 时，repository 层会在事务内重新计算并覆盖。

- [x] **Step 2: 改造 `applyUsageBillingEffects` 订阅扣费分支**

替换原有 `incrementUsageBillingSubscription` 调用为新的两步流程：

```go
if cmd.SubscriptionID != nil {
    // 第 1 步：SELECT FOR UPDATE 锁行 + 读状态
    state, err := lockAndReadSubscription(ctx, tx, *cmd.SubscriptionID)
    if err != nil { return err }

    // 计算窗口重置（按 date_trunc UTC 自然日/周）
    resetDaily  := needResetDaily(state.DailyWindowStart, now)
    resetWeekly := needResetWeekly(state.WeeklyWindowStart, now)

    // 调用纯函数算拆分
    actualCost := cmd.SubscriptionCost + cmd.BalanceCost
    alloc := AllocateSubscriptionCredit(SubscriptionCreditAllocationInput{
        ActualCostUSD: actualCost,
        QuotaLimitUSD: state.QuotaLimitUSD,
        QuotaUsedUSD:  state.QuotaUsedUSD,
        Daily: SubscriptionCreditWindowState{
            LimitUSD: state.DailyLimitUSD,
            UsedUSD:  effectiveDailyUsed(state, resetDaily),
        },
        Weekly: SubscriptionCreditWindowState{
            LimitUSD: state.WeeklyLimitUSD,
            UsedUSD:  effectiveWeeklyUsed(state, resetWeekly),
        },
    })

    // 第 2 步：原子 UPDATE，RETURNING 三个跨越 bool
    post, err := updateSubscriptionUsage(ctx, tx, updateInput{
        SubID: *cmd.SubscriptionID, SubCost: alloc.SubscriptionCostUSD,
        ResetDaily: resetDaily, ResetWeekly: resetWeekly,
        Now: now,
        PreExhaustedAt: state.ExhaustedAt,
        PreDailyUsage: effectiveDailyUsed(state, resetDaily),
        PreWeeklyUsage: effectiveWeeklyUsed(state, resetWeekly),
    })
    if err != nil { return err }

    // 触顶事件：事务内同步写 ledger + outbox
    if post.JustExhaustedTotal {
        eventKey := fmt.Sprintf("total:%d", *cmd.SubscriptionID)
        created, err := ledgerRepo.CreateLimitReachedEvent(ctx, tx, &SubscriptionCreditLedgerEntry{
            UserID: cmd.UserID, SubscriptionID: *cmd.SubscriptionID,
            Type: SubscriptionCreditLedgerLimitReached,
            DeltaUSD: 0, RemainingAfterUSD: 0,
            EventKey: &eventKey,
        })
        if err != nil { return err }
        if created {
            enqueueSchedulerOutbox(ctx, tx, SchedulerOutboxEventSubscriptionNotify, nil, nil, map[string]any{
                "user_id": cmd.UserID, "subscription_id": *cmd.SubscriptionID,
                "kind": "limit_reached_" + SubscriptionLimitReachedTotal,
            })
        }
    }
    if post.JustHitDaily { /* 同理，event_key = daily:{id}:{daily_window_start_rfc3339} */ }
    if post.JustHitWeekly { /* 同理 */ }

    // 写 consume ledger
    insertConsumeLedger(ctx, tx, alloc, post)

    // 把订阅拆分结果回写到 cmd，供后续 balance 扣减使用
    cmd.SubscriptionCost = alloc.SubscriptionCostUSD
    cmd.BalanceCost = alloc.BalanceCostUSD
}

// 后续按现有逻辑扣 balance
if cmd.BalanceCost > 0 {
    newBalance, err := deductUsageBillingBalance(ctx, tx, cmd.UserID, cmd.BalanceCost)
    ...
}
```

删除现有 `incrementUsageBillingSubscription` 实现，按"扣费 SQL 改造"章节的两步流程重写。

- [x] **Step 3: 设置 usage log 拆分字段**

`usage_logs.subscription_cost_usd` 和 `balance_cost_usd` 在写入 usage_log 之前根据 `cmd.SubscriptionCost` / `cmd.BalanceCost` 设置。

`billing_type` 计算：
- `SubscriptionCost > 0 && BalanceCost == 0` → 1
- `SubscriptionCost == 0 && BalanceCost > 0` → 0
- `SubscriptionCost > 0 && BalanceCost > 0` → 2
- 两者都为 0（免费请求）→ 按原逻辑分配

- [x] **Step 4: 删除 `DoWindowMaintenance` 异步路径**

删除 `backend/internal/service/subscription_service.go:848` 附近的 `DoWindowMaintenance` 函数及其所有调用点（middleware 中）。窗口重置完全收敛到扣费事务内 UPDATE 语句。

- [x] **Step 5: 缓存终结**

`finalizePostUsageBilling` 更新两类缓存：

```go
if result.SubscriptionCost > 0 { QueueUpdateSubscriptionCreditUsage(...) }
if result.BalanceCost > 0 { QueueDeductBalance(...) }
```

`UsageBillingApplyResult` 新增字段：

```go
SubscriptionCost  float64
BalanceCost       float64
LimitReachedKinds []string  // 本次扣费触发的 limit kinds，供 monitoring
```

- [x] **Step 6: Run billing tests**

```bash
cd backend
go test ./internal/repository -run TestUsageBilling -count=1
go test ./internal/service -run 'TestBuildUsageBillingCommand|Test.*Subscription.*Billing' -count=1
```

Expected: PASS。集成测试覆盖：

- 全额扣订阅；
- 混合扣费；
- 触顶（total / daily / weekly）跨越 bool 准确；
- 触顶事件 ledger + outbox 同事务写入；
- 并发两请求结算同一订阅（FOR UPDATE 序列化正确）；
- 窗口重置（跨日 / 跨周）。

- [x] **Step 7: Commit**

```bash
git add backend/internal/service/usage_billing.go backend/internal/repository/usage_billing_repo.go backend/internal/service/gateway_service.go backend/internal/service/openai_gateway_service.go backend/internal/service/subscription_service.go
git commit -m "feat: support mixed subscription and balance billing"
```

### Task 5: 鉴权与资格检查改造 ✅ 已完成（本次提交）

**Files:**
- Modify: `backend/internal/server/middleware/api_key_auth.go`
- Modify: `backend/internal/server/middleware/api_key_auth_google.go`
- Modify: `backend/internal/service/billing_cache_service.go`
- Modify: `backend/internal/repository/billing_cache.go`
- Test: `backend/internal/server/middleware/api_key_auth_test.go`
- Test: `backend/internal/server/middleware/api_key_auth_google_test.go`
- Test: `backend/internal/service/billing_cache_service_test.go`

- [x] **Step 1: 中间件查可消费订阅**

Middleware 不再依赖"API Key 所属分组的 subscription_type"判断是否查订阅。统一调用：

```go
sub, err := subscriptionService.GetUsableCreditSubscription(ctx, userID)
```

`GetUsableCreditSubscription` 内部应用 scope 校验（见 Step 3），返回的订阅必须覆盖当前请求分组才算可用。

- [x] **Step 2: 资格检查改"双资金源"**

`CheckBillingEligibility` 允许请求当且仅当任一条件满足：

- 可消费订阅覆盖当前分组且窗口内仍有可用额度；
- 钱包余额通过现有余额门槛。

如果订阅存在但不可用（耗尽 / 过期 / 日周限触顶 / scope 不覆盖），并且余额可用 → 放行扣余额。

仅当两者都不可用时，按当前订阅不可用的具体原因返回对应错误码（见"API 错误返回"章节）。

- [x] **Step 3: Scope 校验**

新增 `SubscriptionCoversGroup(sub, group, user) bool`：

- `all_available_groups`：使用现有用户/分组可见性规则；
- `selected_groups`：分组 ID 必须在 `scope_config.group_ids`；
- `platforms`：分组 platform 必须在 `scope_config.platforms`。

未在 scope 中的分组视为"订阅不可用于此请求"，回落余额。

- [x] **Step 4: 不再依赖分组 subscription_type**

删除 middleware 中 `apiKey.Group.IsSubscriptionType()` 分支判断。所有用户请求统一查 `GetUsableCreditSubscription`，由 service 层和 scope 决定是否扣订阅。

- [x] **Step 5: Run middleware tests**

```bash
cd backend
go test ./internal/server/middleware -run 'TestAPIKeyAuth|TestApiKeyAuthWithSubscriptionGoogle' -count=1
go test ./internal/service -run 'TestBillingCacheService|TestCheckBillingEligibility' -count=1
```

Expected: PASS。

- [x] **Step 6: Commit**

```bash
git add backend/internal/server/middleware backend/internal/service/billing_cache_service.go backend/internal/repository/billing_cache.go
git commit -m "feat: prefer subscription credit during auth"
```

### Task 6: 购买订阅履约

**Files:**
- Modify: `backend/internal/service/payment_config_plans.go`
- Modify: `backend/internal/service/payment_order.go`
- Modify: `backend/internal/service/payment_fulfillment.go`
- Create: `backend/internal/service/subscription_credit_purchase.go`
- Modify: `backend/internal/handler/payment_handler.go`
- Modify: `backend/internal/handler/subscription_handler.go`
- Test: `backend/internal/service/payment_fulfillment_test.go`
- Test: `backend/internal/service/subscription_credit_purchase_test.go`

- [ ] **Step 1: 套餐 DTO 返回额度字段**

Expose `quota_usd`, `daily_limit_usd`, `weekly_limit_usd`, `scope_type`, `scope_config`, `validity_days` 到套餐 DTO 和 checkout API。

- [ ] **Step 2: 下单时校验 + 写订单快照**

`CreateSubscriptionOrder` 流程：

1. 调用 `GetRenewalEligibility(userID)`：
   - `Allowed=false` → 返回 `SUBSCRIPTION_RENEWAL_NOT_ALLOWED`，附 `subscription_id` / `quota_remaining_usd` / `expires_at`。
2. 把套餐当前值快照写入 `payment_orders.subscription_*`：`quota_usd` / `daily_limit_usd` / `weekly_limit_usd` / `scope_type` / `scope_config` / `validity_days`。

- [ ] **Step 3: 履约：纯 INSERT**

Create `backend/internal/service/subscription_credit_purchase.go`：

```go
func (s *SubscriptionCreditPurchaseService) FulfillOrder(ctx context.Context, order *PaymentOrder) error {
    return s.tx.RunInTx(ctx, func(tx *sql.Tx) error {
        // 1. 锁用户行
        if err := s.repo.LockUserForSubscriptionWrite(ctx, tx, order.UserID); err != nil {
            return err
        }

        // 2. 二次校验：理论上一定满足，做防御
        hasUsable, err := s.repo.HasUsableCreditSubscription(ctx, order.UserID)
        if err != nil { return err }
        if hasUsable {
            return ErrAlreadyHasUsableSubscription
        }

        // 3. INSERT 新订阅（订单快照）
        now := time.Now().UTC()
        sub, err := s.repo.InsertSubscription(ctx, tx, &UserSubscription{
            UserID:        order.UserID,
            PlanID:        &order.PlanID,
            ScopeType:     order.SubscriptionScopeType,
            ScopeConfig:   order.SubscriptionScopeConfig,
            QuotaLimitUSD: order.SubscriptionQuotaUSD,
            DailyLimitUSD: order.SubscriptionDailyLimitUSD,
            WeeklyLimitUSD: order.SubscriptionWeeklyLimitUSD,
            StartsAt:      now,
            ExpiresAt:     now.AddDate(0, 0, order.SubscriptionValidityDays),
            Status:        SubscriptionStatusActive,
            // exhausted_at: NULL, quota_used_usd: 0, windows: NULL
        })
        if err != nil { return err }

        // 4. 写 purchase ledger
        return s.ledgerRepo.Create(ctx, tx, &SubscriptionCreditLedgerEntry{
            UserID:            order.UserID,
            SubscriptionID:    sub.ID,
            OrderID:           &order.ID,
            Type:              SubscriptionCreditLedgerPurchase,
            DeltaUSD:          order.SubscriptionQuotaUSD,
            RemainingAfterUSD: order.SubscriptionQuotaUSD,
            Reason:            fmt.Sprintf("payment order %d", order.ID),
        })
    })
}
```

`payment_fulfillment.go` 检测到订单是订阅类型时调用此服务。失败处理：
- `ErrAlreadyHasUsableSubscription` → 订单状态置 `fulfillment_blocked`，触发 ops 告警；
- 其他 DB 错误 → 事务回滚，订单回到 `paid_unfulfilled`，由 webhook 重试。

- [ ] **Step 4: 不实现余额转换接口**

第一版不实现 `/credit/convert` 接口。如果运营要"用余额买订阅"，走现有支付流程的 `payment_method='balance'` 分支即可，履约逻辑与外部支付完全同构。

- [ ] **Step 5: Run tests**

```bash
cd backend
go test ./internal/service -run 'TestSubscriptionCreditPurchase|TestPayment.*Subscription' -count=1
go test ./internal/handler -run TestSubscription -count=1
```

Expected: PASS。测试覆盖：

- 无可消费订阅时下单成功；
- 已有可消费订阅时下单返回 `SUBSCRIPTION_RENEWAL_NOT_ALLOWED`；
- 旧订阅 `exhausted_at IS NOT NULL` 时允许下单且履约成功（并存）；
- 旧订阅 `expires_at <= NOW()` 时允许下单且履约成功；
- 履约二次校验阻塞异常路径；
- `expires_at = now + validity_days`，与下单时刻无关。

- [ ] **Step 6: Commit**

```bash
git add backend/internal/service/payment_* backend/internal/service/subscription_credit_purchase.go backend/internal/handler/payment_handler.go backend/internal/handler/subscription_handler.go
git commit -m "feat: fulfill subscription credit purchases"
```

### Task 7: 过期销毁与审计

**Files:**
- Modify: `backend/internal/service/subscription_expiry_service.go`
- Modify: `backend/internal/repository/user_subscription_repo.go`
- Test: `backend/internal/service/subscription_credit_expiry_test.go`

- [ ] **Step 1: 扩展过期任务**

过期任务发现 `expires_at <= NOW() AND status='active'` 的订阅时：

1. 事务内 SELECT FOR UPDATE 订阅行；
2. 计算 `remaining := math.Max(QuotaLimitUSD - QuotaUsedUSD, 0)`；
3. UPDATE `status='expired'`；
4. 如果 `remaining > 0` 且 `expired_credit_logged_at IS NULL`：
   - INSERT `expire` ledger，`delta_usd = -remaining`, `remaining_after_usd = 0`, `event_key = expire:{id}:{expires_at_rfc3339}`；
   - INSERT `scheduler_outbox`，event_type `subscription_notify`，payload kind `expired`；
   - UPDATE `expired_credit_logged_at = NOW()`。

`event_key` 唯一索引保证过期事件幂等（多个 worker 同时运行也安全）。

- [ ] **Step 2: Run expiry tests**

```bash
cd backend
go test ./internal/service -run TestSubscriptionCreditExpiry -count=1
```

Expected: PASS。

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/subscription_expiry_service.go backend/internal/repository/user_subscription_repo.go
git commit -m "feat: log expired subscription credit"
```

### Task 8: 通知 worker ✅ 已完成（commit ca6d4d84）

**Files:**
- Create: `backend/internal/service/subscription_notify_worker.go`
- Modify: 现有 scheduler outbox worker 注册新 handler
- Test: `backend/internal/service/subscription_notify_worker_test.go`

- [ ] **Step 1: 实现 handler**

```go
func handleSubscriptionNotify(ctx context.Context, event SchedulerOutboxEvent) error {
    var payload struct {
        UserID         int64  `json:"user_id"`
        SubscriptionID int64  `json:"subscription_id"`
        Kind           string `json:"kind"`
    }
    if err := json.Unmarshal(event.Payload, &payload); err != nil {
        logger.Errorf("subscription_notify: invalid payload: %v", err)
        return nil  // 不重试，记日志
    }

    title, body := buildNotificationContent(payload.Kind, payload.SubscriptionID)

    if err := siteMessageService.Send(ctx, payload.UserID, title, body); err != nil {
        logger.Errorf("subscription_notify: site message failed: user=%d sub=%d kind=%s err=%v",
            payload.UserID, payload.SubscriptionID, payload.Kind, err)
    }
    if err := emailService.Send(ctx, payload.UserID, title, body); err != nil {
        logger.Errorf("subscription_notify: email failed: user=%d sub=%d kind=%s err=%v",
            payload.UserID, payload.SubscriptionID, payload.Kind, err)
    }
    return nil  // 任一渠道失败都记日志、不重试
}
```

注册到现有 scheduler outbox worker 的事件分发表中。

- [ ] **Step 2: 配置项**

仅一个设置：

```json
{
  "subscription_credit_pool_repurchase_url": ""
}
```

为空时通知正文使用前端订阅页 URL。

- [ ] **Step 3: Run tests**

```bash
cd backend
go test ./internal/service -run TestSubscriptionNotifyWorker -count=1
```

Expected: PASS。测试覆盖：

- payload 解析正确；
- 站内消息和邮件都调用；
- 任一渠道失败不影响另一渠道；
- 失败仅记日志返回 nil。

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/subscription_notify_worker.go
git commit -m "feat: subscription credit notification worker"
```

### Task 9: API DTO 与前端展示

**Files:**
- Modify: `backend/internal/handler/dto/types.go`
- Modify: `backend/internal/handler/subscription_handler.go`
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/types/payment.ts`
- Modify: `frontend/src/api/subscriptions.ts`
- Modify: `frontend/src/api/payment.ts`
- Modify: `frontend/src/stores/subscriptions.ts`
- Modify: `frontend/src/views/user/SubscriptionsView.vue`
- Modify: `frontend/src/views/user/PaymentView.vue`
- Modify: `frontend/src/components/payment/SubscriptionPlanCard.vue`

- [ ] **Step 1: 订阅 DTO 增加额度字段**

```json
{
  "id": 123,
  "status": "active",
  "is_usable": true,
  "exhausted_at": null,
  "starts_at": "2026-05-28T10:00:00Z",
  "expires_at": "2026-06-27T10:00:00Z",
  "quota_limit_usd": 100,
  "quota_used_usd": 25,
  "quota_remaining_usd": 75,
  "daily_limit_usd": 10,
  "daily_usage_usd": 3.2,
  "daily_reset_at": "2026-05-29T00:00:00Z",
  "weekly_limit_usd": 50,
  "weekly_usage_usd": 12.5,
  "weekly_reset_at": "2026-06-01T00:00:00Z",
  "scope_type": "all_available_groups",
  "scope_config": {}
}
```

`is_usable = (exhausted_at IS NULL AND expires_at > NOW())`，前端展示折叠用。

- [ ] **Step 2: 订阅页 UI**

`SubscriptionsView.vue` 展示规则：

- **当前可消费订阅**（1 张大卡片）：
  - 总额度进度条 + 剩余金额；
  - 日/周额度进度条（无配置时不显示）；
  - 到期时间；
  - 覆盖范围 summary；
  - ledger 入口或内联最近 ledger 列表。
- **未到期但已耗尽的订阅**（折叠区域）：
  - 标题："已耗尽订阅（等待过期）"；
  - 灰显，显示"额度已用完，X 月 Y 日过期"。
- **已过期 / 暂停订阅**：不在订阅页展示，使用记录页可查 ledger。

- [ ] **Step 3: 套餐卡片**

`SubscriptionPlanCard.vue` 显示：

- `quota_usd` 作为主要额度；
- `daily_limit_usd` / `weekly_limit_usd` 作为节流提示；
- `validity_days` 作为有效期；
- 覆盖范围 badge（基于 `scope_type` / `scope_config`）。

- [ ] **Step 4: 错误码映射**

前端 API 错误处理需新增映射：

- `SUBSCRIPTION_CREDIT_EXHAUSTED` / `SUBSCRIPTION_EXPIRED` → "订阅已用完/过期，请重新订阅" + 跳转按钮；
- `SUBSCRIPTION_DAILY_LIMIT_REACHED` / `SUBSCRIPTION_WEEKLY_LIMIT_REACHED` → "订阅已达日/周限，请稍后再试或充值" + 充值按钮；
- `SUBSCRIPTION_RENEWAL_NOT_ALLOWED` → "当前订阅仍可使用，无法重复购买" + 详情。

- [ ] **Step 5: Run frontend checks**

```bash
cd frontend
pnpm typecheck
pnpm build
```

Expected: 两个命令都 PASS。

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handler/dto/types.go backend/internal/handler/subscription_handler.go frontend/src
git commit -m "feat: show subscription credit pool"
```

### Task 10: 管理后台

**Files:**
- Modify: `backend/internal/handler/admin/subscription_handler.go`
- Modify: `backend/internal/handler/admin/payment_handler.go`
- Create: `backend/internal/handler/admin/subscription_waste_stats_handler.go`
- Create: `backend/internal/service/subscription_waste_stats.go`
- Create: `backend/internal/repository/subscription_waste_stats_repo.go`
- Modify: `frontend/src/views/admin/orders/PlanEditDialog.vue`
- Modify: `frontend/src/views/admin/SubscriptionsView.vue`
- Create: `frontend/src/views/admin/SubscriptionWasteStatsView.vue`
- Modify: `frontend/src/router/index.ts`（新增菜单项）

- [ ] **Step 1: 套餐编辑器**

`PlanEditDialog.vue` 新增字段：

- `quota_usd`（必填，> 0）
- `daily_limit_usd`（可空）
- `weekly_limit_usd`（可空）
- `scope_type` 下拉（`all_available_groups` / `selected_groups` / `platforms`）
- `scope_config`（根据 scope_type 动态显示分组多选 或 平台多选）
- `validity_days`（默认 30，范围 1-30）

- [ ] **Step 2: 订阅管理页**

`SubscriptionsView.vue`（admin 版）展示所有用户订阅：

- 列：用户邮箱 / 状态（active+可消费 / active+已耗尽 / expired / suspended）/ `exhausted_at` / `expires_at` / `quota_limit_usd` / `quota_used_usd` / 剩余 / 当前日用量 / 当前周用量 / 最近 30 天浪费金额
- 筛选：状态 / 套餐 / 邮箱关键字 / 创建时间范围
- 操作：调整额度（`admin_adjust` ledger）/ 暂停 / 恢复（修改 status）/ 查看 ledger 详情

- [ ] **Step 3: 浪费率统计页（新增独立菜单项）**

**菜单：** 订阅管理 → 浪费率统计

**后端 service：**

`subscription_waste_stats.go` 提供聚合查询，输入参数：

```go
type WasteStatsQuery struct {
    StartTime time.Time
    EndTime   time.Time
    PlanID    *int64    // 可选：按套餐过滤
    UserID    *int64    // 可选：按用户过滤
    Window    string    // "daily" / "weekly" / "total" / "all"
}

type WasteStatsResult struct {
    // 总览
    TotalSubscriptionsPurchased int64   // 期间内购买订阅数
    TotalQuotaPurchasedUSD      float64 // 总额度（purchase ledger 累计）
    TotalQuotaConsumedUSD       float64 // 实际消费（consume ledger 累计）
    TotalQuotaWastedUSD         float64 // 总额度浪费（expire ledger 累计）

    // 窗口浪费（来自 window_reset ledger）
    DailyResetCount        int64
    DailyAverageWasteRatio float64  // 平均浪费率（0-1）
    DailyTotalWastedUSD    float64
    WeeklyResetCount       int64
    WeeklyAverageWasteRatio float64
    WeeklyTotalWastedUSD   float64

    // 维度聚合（按套餐）
    ByPlan []PlanWasteBucket  // 每个套餐的浪费率排行

    // 时间序列（按周 bucket）
    TimeSeries []WasteTimeBucket
}

type PlanWasteBucket struct {
    PlanID                  int64
    PlanName                string
    PurchaseCount           int64
    AverageDailyWasteRatio  float64
    AverageWeeklyWasteRatio float64
    TotalQuotaWastedRatio   float64  // SUM(expire.wasted_usd) / SUM(purchase.delta_usd)
}

type WasteTimeBucket struct {
    BucketStart           time.Time
    DailyAverageWasteRatio  float64
    WeeklyAverageWasteRatio float64
    TotalWastedUSD        float64
}
```

**SQL 实现（subscription_waste_stats_repo.go）：**

主查询从 `subscription_credit_ledger` 聚合，**按 created_at 落在 `[StartTime, EndTime]` 区间内的 `window_reset` / `expire` / `purchase` / `consume` 事件**统计。

关键 SQL：

```sql
-- 日浪费率：window='daily' 的 window_reset 事件
WITH daily_resets AS (
  SELECT
    subscription_id,
    (metadata->>'wasted_usd')::DECIMAL AS wasted_usd,
    (metadata->>'wasted_ratio')::DECIMAL AS wasted_ratio
  FROM subscription_credit_ledger
  WHERE type = 'window_reset'
    AND metadata->>'window' = 'daily'
    AND created_at >= $1 AND created_at < $2
)
SELECT
  COUNT(*) AS reset_count,
  AVG(wasted_ratio) AS avg_waste_ratio,
  SUM(wasted_usd) AS total_wasted_usd
FROM daily_resets;
```

**按套餐聚合：**

```sql
SELECT
  us.plan_id,
  sp.name,
  COUNT(DISTINCT us.id) AS purchase_count,
  -- 套餐下所有订阅的窗口浪费聚合
  AVG(CASE WHEN scl.metadata->>'window'='daily'  THEN (scl.metadata->>'wasted_ratio')::DECIMAL END) AS avg_daily_waste_ratio,
  AVG(CASE WHEN scl.metadata->>'window'='weekly' THEN (scl.metadata->>'wasted_ratio')::DECIMAL END) AS avg_weekly_waste_ratio,
  SUM(CASE WHEN scl.type='expire' THEN -scl.delta_usd END) /
    NULLIF(SUM(CASE WHEN scl.type='purchase' THEN scl.delta_usd END), 0) AS total_quota_wasted_ratio
FROM user_subscriptions us
LEFT JOIN subscription_plans sp ON sp.id = us.plan_id
LEFT JOIN subscription_credit_ledger scl ON scl.subscription_id = us.id
  AND scl.created_at >= $1 AND scl.created_at < $2
WHERE us.plan_id IS NOT NULL
  AND us.created_at >= $1
GROUP BY us.plan_id, sp.name
ORDER BY total_quota_wasted_ratio DESC NULLS LAST;
```

**前端 SubscriptionWasteStatsView.vue 展示：**

1. **筛选条**：时间范围（最近 7/30/90 天 / 自定义）/ 套餐下拉 / 用户搜索
2. **总览卡片**（4 张）：
   - 总购买额度 / 总消费 / 总浪费金额 / 整体浪费率
3. **窗口浪费率卡片**（2 张）：
   - 日限平均浪费率 + reset 次数 + 浪费金额
   - 周限平均浪费率 + reset 次数 + 浪费金额
4. **套餐浪费排行表**：按 `total_quota_wasted_ratio` 降序排列，列：套餐名 / 购买数 / 日浪费率 / 周浪费率 / 总浪费率
5. **时间趋势图**（ECharts 折线）：横轴时间 bucket，纵轴浪费率，三条线（日/周/总）

- [ ] **Step 4: API 端点**

```
GET /admin/subscriptions
  ?status=&plan_id=&email=&page=&page_size=
GET /admin/subscriptions/:id/ledger
  ?type=&page=&page_size=
PATCH /admin/subscriptions/:id
  body: {quota_limit_usd?, daily_limit_usd?, weekly_limit_usd?, expires_at?, status?, reason}
GET /admin/subscriptions/waste-stats
  ?start=&end=&plan_id=&user_id=&window=
GET /admin/subscriptions/waste-stats/by-plan
  ?start=&end=
GET /admin/subscriptions/waste-stats/time-series
  ?start=&end=&bucket=week
```

PATCH 调整额度时自动写 `admin_adjust` ledger（`delta_usd` = new_limit - old_limit）。

- [ ] **Step 5: Run tests**

```bash
cd backend
go test ./internal/service -run 'TestSubscriptionWasteStats' -count=1
go test ./internal/handler/admin -run 'TestAdminSubscription|TestAdminPayment.*Plan|TestAdminWasteStats' -count=1
cd ../frontend
pnpm typecheck
pnpm build
```

Expected: PASS。后端测试覆盖：

- 浪费率计算正确（mock ledger 数据后查聚合）；
- 时间范围过滤；
- 按套餐分组聚合；
- 空数据返回 NaN 处理（avg_ratio 用 NULLIF 防除零）。

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handler/admin backend/internal/service/subscription_waste_stats* backend/internal/repository/subscription_waste_stats_repo.go frontend/src/views/admin frontend/src/router
git commit -m "feat: admin subscription credit management with waste stats"
```

---

## 测试矩阵

### 后端单元测试

```bash
cd backend
go test ./internal/service -run 'TestAllocateSubscriptionCredit|Test.*Subscription.*Credit|TestBuildUsageBillingCommand|TestSubscriptionNotifyWorker' -count=1
```

Expected: PASS。

### 后端 repository / migration 测试

```bash
cd backend
go test ./internal/repository -run 'TestUsageBilling|TestUserSubscription|TestSubscriptionCreditLedger|TestMigrations' -count=1
```

Expected: PASS。

### 中间件测试

```bash
cd backend
go test ./internal/server/middleware -run 'TestAPIKeyAuth|TestApiKeyAuthWithSubscriptionGoogle' -count=1
```

Expected: PASS。

### 全量后端测试

```bash
cd backend
go test ./...
```

Expected: PASS。

### 前端测试

```bash
cd frontend
pnpm typecheck
pnpm build
pnpm test -- --run
```

Expected: PASS。如果仓库未配置 `pnpm test`，至少运行 payment/subscription 相关 Vitest spec。

---

## 验收标准

- 用户购买一次订阅后，订阅页只看到 1 张"可消费订阅"卡片。
- 用户在任意可用分组（scope 覆盖范围内）请求时，优先扣订阅额度。
- 订阅额度不足时，同一次请求自动拆分扣订阅和余额，`usage_logs.billing_type=2`。
- 订阅日 / 周限达到后，不拒绝余额可用的用户请求；触顶时发送 limit_reached 通知（站内 + 邮件）；不允许提前购买新订阅。
- 订阅总额度耗尽后，`exhausted_at` 立即写入；发送 total 通知；用户可购买新订阅。
- 旧订阅总额度耗尽后买新订阅 → 旧订阅保持 active 等过期，新订阅成为唯一可消费；DB 唯一索引 `user_subscriptions_user_active_usable` 保证不会出现两条可消费订阅。
- 订阅过期后，剩余额度不可再消费；过期任务写 `expire` ledger 并发送 expired 通知。
- 使用记录能区分纯订阅（`billing_type=1`）、纯余额（`billing_type=0`）、混合扣费（`billing_type=2`）。
- 管理后台能配置额度型套餐（`quota_usd` / `daily_limit_usd` / `weekly_limit_usd` / `scope_type` / `validity_days`）并查看用户订阅额度状态。
- API 错误返回包含 `code` / `details` 结构化字段，前端能映射成对应文案 + 跳转按钮。
- 触顶或过期通知**最多发一次**（靠 `subscription_credit_ledger_event_key_unique` 索引保证）。

---

## 第一版建议范围

第一版只做：

- 一个用户级订阅额度池（DB 唯一索引强制）；
- 套餐购买增加订阅额度和有效期（最长 30 天）；
- 消费优先订阅，额度不足拆分扣余额；
- 订阅额度日 / 周限（无月限，因订阅最长 30 天，月限≡总额度）；
- 订阅到期销毁剩余额度并写流水；
- 用户前端展示总额度和窗口额度，已耗尽订阅折叠展示；
- 触顶 / 过期通知走 `scheduler_outbox`，失败仅记日志；
- 管理后台配置套餐和查看订阅状态。

第一版不做：

- 多个订阅叠加优先级（用户始终只有 1 条可消费订阅）；
- 余额转换订阅额度接口（`/credit/convert`，统一用"余额支付订阅套餐"代替）；
- 按模型族做复杂覆盖范围；
- 复杂退款拆账自动化；
- 通知重试机制（失败仅记日志）；
- 月限 / 月度窗口（订阅最长 30 天，无意义）；
- 灰度 / shadow mode / allowlist（未上线，无需灰度）。

这些能力可以在第一版稳定后基于 ledger 和 scope 扩展。
