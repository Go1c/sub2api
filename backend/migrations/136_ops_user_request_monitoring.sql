-- Targeted user request monitoring: admin-created future request body captures.

CREATE TABLE IF NOT EXISTS ops_user_request_monitors (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_email VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'stopped')),
    duration_seconds INT NOT NULL,
    max_captures_per_minute INT NOT NULL,
    sample_rate_percent INT NOT NULL,
    retention_days INT NOT NULL DEFAULT 7,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at TIMESTAMPTZ NOT NULL,
    stopped_at TIMESTAMPTZ NULL,
    last_capture_at TIMESTAMPTZ NULL,
    capture_count BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_ops_user_request_monitors_active
    ON ops_user_request_monitors (user_id, status, starts_at, ends_at);

CREATE INDEX IF NOT EXISTS idx_ops_user_request_monitors_created_at
    ON ops_user_request_monitors (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_user_request_monitors_ends_at
    ON ops_user_request_monitors (ends_at);

CREATE TABLE IF NOT EXISTS ops_user_request_captures (
    id BIGSERIAL PRIMARY KEY,
    monitor_id BIGINT NOT NULL REFERENCES ops_user_request_monitors(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id BIGINT NULL REFERENCES api_keys(id) ON DELETE SET NULL,
    account_id BIGINT NULL REFERENCES accounts(id) ON DELETE SET NULL,
    group_id BIGINT NULL REFERENCES groups(id) ON DELETE SET NULL,
    request_id VARCHAR(64) NULL,
    model VARCHAR(100) NULL,
    inbound_endpoint VARCHAR(256) NULL,
    method VARCHAR(16) NULL,
    content_type VARCHAR(128) NULL,
    body TEXT NOT NULL,
    body_bytes INT NOT NULL,
    body_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    sample_rate_percent INT NOT NULL,
    capture_minute TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ops_user_request_captures_monitor_created
    ON ops_user_request_captures (monitor_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_user_request_captures_user_created
    ON ops_user_request_captures (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_user_request_captures_expires_at
    ON ops_user_request_captures (expires_at);

CREATE INDEX IF NOT EXISTS idx_ops_user_request_captures_request_id
    ON ops_user_request_captures (request_id);
