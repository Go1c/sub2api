-- 186_account_autopause_expiry_index_notx.sql
-- Fork remapping of upstream 151_account_autopause_expiry_index_notx.sql
-- (fork 151 is subscription_backfill_exhausted_at).

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_accounts_autopause_expiry_due
    ON accounts (expires_at)
    WHERE deleted_at IS NULL
      AND schedulable = TRUE
      AND auto_pause_on_expired = TRUE
      AND expires_at IS NOT NULL;
