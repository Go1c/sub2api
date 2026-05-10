-- Add configurable site messages with inbox, sent mail, replies, read state, and retention cleanup support.

CREATE TABLE IF NOT EXISTS site_messages (
    id BIGSERIAL PRIMARY KEY,
    sender_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    recipient_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    parent_id BIGINT NULL REFERENCES site_messages(id) ON DELETE SET NULL,
    subject VARCHAR(200) NOT NULL CHECK (LENGTH(BTRIM(subject)) > 0),
    content TEXT NOT NULL CHECK (LENGTH(BTRIM(content)) > 0),
    read_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_site_messages_recipient_created
    ON site_messages (recipient_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_site_messages_sender_created
    ON site_messages (sender_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_site_messages_recipient_read_created
    ON site_messages (recipient_id, read_at, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_site_messages_parent_created
    ON site_messages (parent_id, created_at ASC, id ASC);

CREATE INDEX IF NOT EXISTS idx_site_messages_created_at
    ON site_messages (created_at);
