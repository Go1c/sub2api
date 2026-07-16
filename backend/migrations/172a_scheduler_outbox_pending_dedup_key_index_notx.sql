-- 172a_scheduler_outbox_pending_dedup_key_index_notx.sql
-- Fork remapping of upstream 153_scheduler_outbox_pending_dedup_key_index_notx.sql
-- (fork 153 is add_api_key_fallback_key_id). CONCURRENTLY requires *_notx.sql.

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_scheduler_outbox_pending_dedup_key
    ON scheduler_outbox (dedup_key)
    WHERE dedup_key IS NOT NULL;
