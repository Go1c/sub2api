-- Add fallback_key_id: when this key's group has no usable upstream account,
-- requests/billing fall back to the referenced key.
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS fallback_key_id BIGINT;
CREATE INDEX IF NOT EXISTS idx_api_keys_fallback_key_id ON api_keys (fallback_key_id);
