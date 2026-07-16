-- 177_add_ops_system_logs_api_key_id.sql
-- Fork remapping of upstream 154_add_ops_system_logs_api_key_id.sql
-- (fork 154 is add_group_exhausted_fallback / previously also collided with spark-shadow).
-- Required by ops system log sink flush (api_key_id column).

ALTER TABLE ops_system_logs
  ADD COLUMN IF NOT EXISTS api_key_id BIGINT;
