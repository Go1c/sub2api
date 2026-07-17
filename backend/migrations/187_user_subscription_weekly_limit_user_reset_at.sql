-- 187_user_subscription_weekly_limit_user_reset_at.sql
-- 用户手动重置周限：每订阅周期仅一次机会。
-- NULL = 本行生命周期内尚未手动重置；非 NULL = 已使用过（存执行时刻）。

ALTER TABLE user_subscriptions
    ADD COLUMN IF NOT EXISTS weekly_limit_user_reset_at TIMESTAMPTZ NULL;
