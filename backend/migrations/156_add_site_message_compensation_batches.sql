CREATE TABLE IF NOT EXISTS site_message_compensation_batches (
    id BIGSERIAL PRIMARY KEY,
    batch_id VARCHAR(64) NOT NULL,
    subject VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    mode VARCHAR(20) NOT NULL,
    audience TEXT NOT NULL,
    recipient_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    code_count INTEGER NOT NULL DEFAULT 0,
    operator_email VARCHAR(255) NOT NULL DEFAULT '',
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    codes JSONB NOT NULL DEFAULT '[]'::jsonb,
    results JSONB NOT NULL DEFAULT '[]'::jsonb,
    message_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_site_message_compensation_batches_sent_at
    ON site_message_compensation_batches(sent_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_site_message_compensation_batches_batch_id
    ON site_message_compensation_batches(batch_id);
