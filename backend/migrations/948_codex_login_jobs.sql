-- Private login materials are deliberately outside accounts.credentials/extra exports.
CREATE TABLE IF NOT EXISTS codex_login_jobs (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    material_encrypted TEXT NOT NULL,
    options JSONB NOT NULL DEFAULT '{}'::jsonb,
    account_id BIGINT REFERENCES accounts(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    error_message VARCHAR(255) NOT NULL DEFAULT '',
    attempts INTEGER NOT NULL DEFAULT 0,
    run_after TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lease_token VARCHAR(64),
    lease_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS codex_login_jobs_account ON codex_login_jobs(account_id) WHERE account_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS codex_login_jobs_pending ON codex_login_jobs(status, run_after);
