-- Add per-user invoice application support.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS invoice_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS invoice_requests (
    id BIGSERIAL PRIMARY KEY,
    order_no VARCHAR(32) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    user_email VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL CHECK (LENGTH(BTRIM(title)) > 0),
    tax_number VARCHAR(100) NOT NULL CHECK (LENGTH(BTRIM(tax_number)) > 0),
    amount NUMERIC(20,2) NOT NULL CHECK (amount > 0),
    recipient_email VARCHAR(255) NOT NULL CHECK (LENGTH(BTRIM(recipient_email)) > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'processing',
    file_name VARCHAR(255) NOT NULL DEFAULT '',
    file_path TEXT NOT NULL DEFAULT '',
    file_size BIGINT NOT NULL DEFAULT 0,
    content_type VARCHAR(100) NOT NULL DEFAULT '',
    tax_rate NUMERIC(10,6) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(20,2) NOT NULL DEFAULT 0,
    failure_reason TEXT NOT NULL DEFAULT '',
    completed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT invoice_requests_status_check CHECK (status IN ('processing', 'completed', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_invoice_requests_user_created
    ON invoice_requests (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_invoice_requests_status_created
    ON invoice_requests (status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_invoice_requests_created
    ON invoice_requests (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_invoice_requests_user_status
    ON invoice_requests (user_id, status);
