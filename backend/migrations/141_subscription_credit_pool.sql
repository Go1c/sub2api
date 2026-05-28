-- 订阅额度池（subscription credit pool）改造
--
-- 设计要点：
--   - 用户级订阅额度池：每用户最多 1 条"可消费"订阅，并存"等过期"订阅
--   - 订阅最长 30 天，无月限字段（月限≡总额度）
--   - 扣费在事务内 SELECT FOR UPDATE 锁行后原子 UPDATE
--   - exhausted_at 时间戳替代独立 "exhausted" 状态枚举值
--   - 详见 doc/plan/2026-05-28-subscription-credit-pool-plan.md

-- ============================================================================
-- 1. user_subscriptions 改造
-- ============================================================================

-- group_id 改为可空（额度池订阅不绑定具体分组，由 scope_* 描述覆盖范围）
ALTER TABLE user_subscriptions ALTER COLUMN group_id DROP NOT NULL;

-- 删除旧的"用户+分组"唯一索引（由新的"用户+可消费"索引取代）
DROP INDEX IF EXISTS user_subscriptions_user_group_unique_active;

-- 新增额度池字段
ALTER TABLE user_subscriptions
    ADD COLUMN IF NOT EXISTS plan_id BIGINT,
    ADD COLUMN IF NOT EXISTS scope_type VARCHAR(32) NOT NULL DEFAULT 'all_available_groups',
    ADD COLUMN IF NOT EXISTS scope_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS quota_limit_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS quota_used_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS daily_limit_usd DECIMAL(20,10),
    ADD COLUMN IF NOT EXISTS weekly_limit_usd DECIMAL(20,10),
    ADD COLUMN IF NOT EXISTS exhausted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS expired_credit_logged_at TIMESTAMPTZ;

-- plan_id 外键约束（套餐硬删除时置 NULL，保留订阅历史）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'user_subscriptions_plan_id_fkey'
    ) THEN
        ALTER TABLE user_subscriptions
            ADD CONSTRAINT user_subscriptions_plan_id_fkey
            FOREIGN KEY (plan_id) REFERENCES subscription_plans(id) ON DELETE SET NULL;
    END IF;
END $$;

-- quota_limit_usd 必须 > 0（避免无意义的零额度订阅）
-- 兼容已有 default=0 的行：只对新插入和更新的行约束，已有 0 行视为历史数据
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'user_subscriptions_quota_limit_positive'
    ) THEN
        ALTER TABLE user_subscriptions
            ADD CONSTRAINT user_subscriptions_quota_limit_positive
            CHECK (quota_limit_usd >= 0);
        -- 注：使用 >= 0 而非 > 0 以兼容旧分组订阅（quota_limit_usd 为默认 0）。
        -- 业务层在创建额度池订阅时校验 quota_limit_usd > 0。
    END IF;
END $$;

-- 删除原月限字段（订阅最长 30 天，月限≡总额度，无需单独维护）
ALTER TABLE user_subscriptions
    DROP COLUMN IF EXISTS monthly_window_start,
    DROP COLUMN IF EXISTS monthly_usage_usd;

-- 部分唯一索引：每个用户最多 1 条"可消费"订阅
--   可消费 = status='active' AND exhausted_at IS NULL AND deleted_at IS NULL
-- 旧订阅总额度耗尽（exhausted_at IS NOT NULL）后不占用此索引，允许新订阅 INSERT
CREATE UNIQUE INDEX IF NOT EXISTS user_subscriptions_user_active_usable
    ON user_subscriptions(user_id)
    WHERE deleted_at IS NULL
      AND status = 'active'
      AND exhausted_at IS NULL;

-- 辅助索引：用于过期任务扫描
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_expiry_scan
    ON user_subscriptions(status, expires_at)
    WHERE deleted_at IS NULL AND status = 'active';

-- ============================================================================
-- 2. subscription_plans 改造
-- ============================================================================

-- group_id 改为可空（额度套餐不绑定具体分组）
ALTER TABLE subscription_plans ALTER COLUMN group_id DROP NOT NULL;

-- 新增额度套餐字段（validity_days 已存在，不重复添加）
ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS quota_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS daily_limit_usd DECIMAL(20,10),
    ADD COLUMN IF NOT EXISTS weekly_limit_usd DECIMAL(20,10),
    ADD COLUMN IF NOT EXISTS scope_type VARCHAR(32) NOT NULL DEFAULT 'all_available_groups',
    ADD COLUMN IF NOT EXISTS scope_config JSONB NOT NULL DEFAULT '{}'::jsonb;

-- 套餐 quota_usd 必须 > 0（不允许"零额度"套餐）
-- 兼容旧分组订阅套餐（quota_usd 默认 0），与 user_subscriptions 一致使用 >= 0
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'subscription_plans_quota_non_negative'
    ) THEN
        ALTER TABLE subscription_plans
            ADD CONSTRAINT subscription_plans_quota_non_negative
            CHECK (quota_usd >= 0);
        -- 业务层在创建额度套餐时校验 quota_usd > 0。
    END IF;
END $$;

-- validity_days 范围约束（1-30 天）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'subscription_plans_validity_days_range'
    ) THEN
        ALTER TABLE subscription_plans
            ADD CONSTRAINT subscription_plans_validity_days_range
            CHECK (validity_days > 0 AND validity_days <= 30);
    END IF;
END $$;

-- ============================================================================
-- 3. subscription_credit_ledger（订阅额度流水）
-- ============================================================================

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

-- type 取值约束：
--   purchase / consume / limit_reached / expire / admin_adjust
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'subscription_credit_ledger_type_check'
    ) THEN
        ALTER TABLE subscription_credit_ledger
            ADD CONSTRAINT subscription_credit_ledger_type_check
            CHECK (type IN ('purchase', 'consume', 'limit_reached', 'expire', 'admin_adjust'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_subscription_credit_ledger_user_time
    ON subscription_credit_ledger(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_subscription_credit_ledger_subscription_time
    ON subscription_credit_ledger(subscription_id, created_at DESC);

-- 幂等事件唯一索引：同一 subscription + type + event_key 只允许一条
-- event_key 格式（UTC RFC3339）：
--   total:{subscription_id}
--   daily:{subscription_id}:{daily_window_start_rfc3339}
--   weekly:{subscription_id}:{weekly_window_start_rfc3339}
--   expire:{subscription_id}:{expires_at_rfc3339}
CREATE UNIQUE INDEX IF NOT EXISTS subscription_credit_ledger_event_key_unique
    ON subscription_credit_ledger(subscription_id, type, event_key)
    WHERE event_key IS NOT NULL AND event_key <> '';

-- ============================================================================
-- 4. usage_logs 支持混合扣费展示
-- ============================================================================

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS subscription_cost_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS balance_cost_usd DECIMAL(20,10) NOT NULL DEFAULT 0;

-- billing_type 扩展为：
--   0: 纯余额
--   1: 纯订阅
--   2: 订阅 + 余额混合
-- 由 service 层根据 subscription_cost_usd / balance_cost_usd 计算后写入。
-- 历史数据保持原值（订阅功能未上线，无生产数据需要回填）。

-- ============================================================================
-- 5. payment_orders 保存订阅快照
-- ============================================================================

ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS subscription_quota_usd DECIMAL(20,10),
    ADD COLUMN IF NOT EXISTS subscription_daily_limit_usd DECIMAL(20,10),
    ADD COLUMN IF NOT EXISTS subscription_weekly_limit_usd DECIMAL(20,10),
    ADD COLUMN IF NOT EXISTS subscription_scope_type VARCHAR(32),
    ADD COLUMN IF NOT EXISTS subscription_scope_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS subscription_validity_days INT;

-- 注：payment_orders.subscription_days 已存在（upstream 历史字段），
-- 与本次新增的 subscription_validity_days 语义相同。新逻辑只使用 subscription_validity_days，
-- 旧字段保留以兼容现有数据；service 层在写入新订单时同时填充两字段。
