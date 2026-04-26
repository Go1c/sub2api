-- Affiliate signup bonus controls, daily/global guardrails, and audit logs.

INSERT INTO settings (key, value)
VALUES
    ('affiliate_signup_bonus_enabled', 'false'),
    ('affiliate_signup_bonus_amount', '0.00'),
    ('affiliate_signup_bonus_total_cap', '0.00'),
    ('affiliate_signup_bonus_daily_cap', '0.00'),
    ('balance_usage_gate_enabled', 'false'),
    ('balance_usage_gate_min_balance', '0.00'),
    ('balance_usage_gate_min_recharge', '0.00')
ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS affiliate_invite_logs (
    id BIGSERIAL PRIMARY KEY,
    inviter_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    invitee_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    affiliate_code VARCHAR(64) NOT NULL DEFAULT '',
    success BOOLEAN NOT NULL DEFAULT false,
    failure_reason VARCHAR(128) NOT NULL DEFAULT '',
    bonus_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    fingerprint_hash VARCHAR(128) NOT NULL DEFAULT '',
    ip_address VARCHAR(64) NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_affiliate_invite_logs_inviter_id ON affiliate_invite_logs(inviter_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_affiliate_invite_logs_invitee_id ON affiliate_invite_logs(invitee_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_affiliate_invite_logs_fingerprint_hash ON affiliate_invite_logs(fingerprint_hash)
    WHERE fingerprint_hash <> '';
CREATE INDEX IF NOT EXISTS idx_affiliate_invite_logs_created_at ON affiliate_invite_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_signup_user ON user_affiliate_ledger(user_id, created_at DESC)
    WHERE action = 'signup_bonus';
CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_signup_created_at ON user_affiliate_ledger(created_at DESC)
    WHERE action = 'signup_bonus';

COMMENT ON TABLE affiliate_invite_logs IS '邀请注册绑定与注册送余额审计日志';
COMMENT ON COLUMN affiliate_invite_logs.failure_reason IS '失败或跳过注册送余额原因，例如 invalid_code|fingerprint_reused|daily_total_cap_reached';
COMMENT ON COLUMN user_affiliate_ledger.action IS 'accrue|transfer|signup_bonus';
