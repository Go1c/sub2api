ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS allowed_models JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN api_keys.allowed_models IS 'Models this API key is allowed to request; empty means unrestricted';
