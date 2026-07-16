-- 182_ops_error_log_api_key_prefix.sql
-- Fork remapping of upstream 147_ops_error_log_api_key_prefix.sql
-- (fork 147 unused / number space occupied by other fork migrations).

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

ALTER TABLE ops_error_logs
    ADD COLUMN IF NOT EXISTS api_key_prefix VARCHAR(32);
