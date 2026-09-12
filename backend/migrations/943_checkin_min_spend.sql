-- Migration: 943_checkin_min_spend
-- Historical billed-spend gate for daily check-in. 0 disables the gate.

ALTER TABLE daily_checkin_settings
    ADD COLUMN IF NOT EXISTS min_spend NUMERIC(20,8) NOT NULL DEFAULT 0;

ALTER TABLE daily_checkin_settings
    DROP CONSTRAINT IF EXISTS daily_checkin_settings_min_spend_nonnegative;

ALTER TABLE daily_checkin_settings
    ADD CONSTRAINT daily_checkin_settings_min_spend_nonnegative CHECK (min_spend >= 0);

COMMENT ON COLUMN daily_checkin_settings.min_spend IS
    'Historical billed spend (usage_logs.actual_cost) must be strictly greater than this amount to check in. 0 disables the gate.';
