-- users.frozen_balance: hold amount reserved for async billing (e.g. batch image).
--
-- History:
-- - Upstream / origin/dev already has 160_add_user_frozen_balance.sql with the same column.
-- - Production publish lineage (through 919_webhook_only_drop_websocket.sql) shipped Ent
--   field frozen_balance without this column in any embedded migration, causing
--   pq: column users.frozen_balance does not exist on login / balance paths.
-- - 920 is an idempotent safety net for environments that applied the 9xx notify
--   migrations but never applied 160 (and for the manual prod hotfix to become a
--   recorded schema_migrations row with a stable checksum).
--
-- Safe to re-run: ADD COLUMN IF NOT EXISTS. No index required (lookups are by user id).

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS frozen_balance DECIMAL(20,8) NOT NULL DEFAULT 0;

COMMENT ON COLUMN users.frozen_balance IS
    'Amount of balance currently held/frozen (e.g. batch-image holds); available balance is balance, not balance-frozen_balance unless app logic subtracts.';
