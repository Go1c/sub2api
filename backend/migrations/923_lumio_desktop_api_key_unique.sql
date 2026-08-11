-- Migration: 923_lumio_desktop_api_key_unique
-- Keep one active account-level key under the reserved desktop name without
-- deleting or rotating any historical credentials.

LOCK TABLE api_keys IN SHARE ROW EXCLUSIVE MODE;

WITH ranked AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY user_id
            ORDER BY created_at, id
        ) AS rn
    FROM api_keys
    WHERE deleted_at IS NULL
      AND name = 'Lumio Codex Desktop'
)
UPDATE api_keys AS k
SET name = 'Lumio Codex Desktop (legacy ' || k.id || ')',
    updated_at = NOW()
FROM ranked
WHERE k.id = ranked.id
  AND ranked.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_lumio_desktop_active_unique
    ON api_keys (user_id, name)
    WHERE deleted_at IS NULL
      AND name = 'Lumio Codex Desktop';
