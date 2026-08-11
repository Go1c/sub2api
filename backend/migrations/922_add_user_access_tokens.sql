-- Migration: 922_add_user_access_tokens
-- User long-lived opaque access tokens for programmatic key management.
-- Plaintext is returned only once on create; DB stores SHA-256 hash + display prefix.

CREATE TABLE IF NOT EXISTS user_access_tokens (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         VARCHAR(100) NOT NULL,
    token_hash   VARCHAR(64)  NOT NULL,
    token_prefix VARCHAR(32)  NOT NULL,
    expires_at   TIMESTAMPTZ  NOT NULL,
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS user_access_tokens_token_hash_key
    ON user_access_tokens (token_hash);

CREATE INDEX IF NOT EXISTS idx_user_access_tokens_user_id
    ON user_access_tokens (user_id);

CREATE INDEX IF NOT EXISTS idx_user_access_tokens_user_created
    ON user_access_tokens (user_id, created_at DESC);

COMMENT ON TABLE user_access_tokens IS
    'User opaque access tokens (uat_*) for keys/groups management APIs; hash only, never plaintext.';
COMMENT ON COLUMN user_access_tokens.token_hash IS
    'SHA-256 hex of full opaque token; unique O(1) auth lookup.';
COMMENT ON COLUMN user_access_tokens.token_prefix IS
    'Display prefix for list UI; not secret.';
COMMENT ON COLUMN user_access_tokens.revoked_at IS
    'When set, token is revoked and must not authenticate.';
