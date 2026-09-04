-- usage_logs.correlation_id stores downstream X-Sub2-Request-ID for reseller reconciliation.
-- Nullable, no default, no index: metadata-only ADD COLUMN; join happens on the caller side.
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS correlation_id VARCHAR(64) NULL;
