-- 170a_account_spark_shadow_indexes_notx.sql
-- Fork remapping of upstream 154a_account_spark_shadow_indexes_notx.sql.
-- CONCURRENTLY indexes must run outside a transaction (*_notx.sql).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_accounts_parent_account_id
    ON accounts (parent_account_id) WHERE parent_account_id IS NOT NULL;

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS uq_accounts_spark_shadow_per_parent
    ON accounts (parent_account_id)
    WHERE parent_account_id IS NOT NULL AND quota_dimension = 'spark' AND deleted_at IS NULL;
