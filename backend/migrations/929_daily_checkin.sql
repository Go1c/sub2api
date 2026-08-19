-- Migration: 929_daily_checkin
-- Independent daily check-in settings, audit records, and serialized daily budget.

CREATE TABLE IF NOT EXISTS daily_checkin_settings (
    id          SMALLINT      PRIMARY KEY DEFAULT 1,
    enabled     BOOLEAN       NOT NULL DEFAULT FALSE,
    min_reward  NUMERIC(20,8) NOT NULL DEFAULT 0.1000,
    max_reward  NUMERIC(20,8) NOT NULL DEFAULT 0.5000,
    timezone    VARCHAR(64)   NOT NULL DEFAULT 'Asia/Shanghai',
    daily_cap   NUMERIC(20,8) NOT NULL DEFAULT 0,
    milestones  JSONB         NOT NULL DEFAULT '[]'::jsonb,
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT daily_checkin_settings_singleton CHECK (id = 1),
    CONSTRAINT daily_checkin_settings_rewards_nonnegative CHECK (min_reward >= 0 AND max_reward >= 0 AND daily_cap >= 0),
    CONSTRAINT daily_checkin_settings_reward_range CHECK (min_reward <= max_reward),
    CONSTRAINT daily_checkin_settings_milestones_array CHECK (jsonb_typeof(milestones) = 'array')
);

INSERT INTO daily_checkin_settings (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS daily_checkin_records (
    id                BIGSERIAL     PRIMARY KEY,
    user_id           BIGINT        REFERENCES users(id) ON DELETE SET NULL,
    user_email        VARCHAR(255)  NOT NULL,
    username          VARCHAR(255)  NOT NULL DEFAULT '',
    business_date     DATE          NOT NULL,
    checked_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    timezone          VARCHAR(64)   NOT NULL,
    streak_days       INTEGER       NOT NULL,
    cycle_day         INTEGER       NOT NULL,
    milestone_day     INTEGER,
    base_reward       NUMERIC(20,8) NOT NULL,
    milestone_bonus   NUMERIC(20,8) NOT NULL,
    actual_reward     NUMERIC(20,8) NOT NULL,
    status            VARCHAR(32)   NOT NULL,
    balance_after     NUMERIC(20,8) NOT NULL,
    client_ip         VARCHAR(64)   NOT NULL DEFAULT '',
    user_agent        TEXT          NOT NULL DEFAULT '',
    CONSTRAINT daily_checkin_records_streak_positive CHECK (streak_days > 0 AND cycle_day > 0),
    CONSTRAINT daily_checkin_records_rewards_nonnegative CHECK (base_reward >= 0 AND milestone_bonus >= 0 AND actual_reward >= 0),
    CONSTRAINT daily_checkin_records_status_check CHECK (status IN ('awarded', 'budget_exhausted')),
    UNIQUE (user_id, business_date)
);

CREATE INDEX IF NOT EXISTS daily_checkin_records_user_date_idx ON daily_checkin_records (user_id, business_date DESC);
CREATE INDEX IF NOT EXISTS daily_checkin_records_business_date_idx ON daily_checkin_records (business_date DESC);
CREATE INDEX IF NOT EXISTS daily_checkin_records_checked_at_idx ON daily_checkin_records (checked_at DESC);
CREATE INDEX IF NOT EXISTS daily_checkin_records_status_idx ON daily_checkin_records (status);

CREATE TABLE IF NOT EXISTS daily_checkin_daily_counters (
    business_date DATE          PRIMARY KEY,
    awarded_total NUMERIC(20,8) NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT daily_checkin_daily_counters_nonnegative CHECK (awarded_total >= 0)
);

COMMENT ON TABLE daily_checkin_records IS 'Independent immutable audit source for daily check-in rewards.';
COMMENT ON TABLE daily_checkin_daily_counters IS 'Rows locked inside check-in transactions to serialize global daily budget allocation.';
