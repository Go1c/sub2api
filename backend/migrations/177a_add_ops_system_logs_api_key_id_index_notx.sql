-- 177a_add_ops_system_logs_api_key_id_index_notx.sql
-- Fork remapping of upstream 155_add_ops_system_logs_api_key_id_index_notx.sql
-- (fork 155 is allow_multiple_active_credit_subscriptions).
-- CONCURRENTLY requires *_notx.sql.

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ops_system_logs_api_key_id_created_at
  ON ops_system_logs (api_key_id, created_at DESC);
