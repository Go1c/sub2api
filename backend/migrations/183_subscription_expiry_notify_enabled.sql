-- 183_subscription_expiry_notify_enabled.sql
-- Fork remapping of upstream 141_subscription_expiry_notify_enabled.sql
-- (fork 141 is subscription_credit_pool).

INSERT INTO settings (key, value, updated_at)
VALUES ('subscription_expiry_notify_enabled', 'true', NOW())
ON CONFLICT (key) DO NOTHING;
