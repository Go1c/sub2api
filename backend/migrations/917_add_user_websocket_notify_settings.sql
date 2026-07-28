-- User WebSocket notify preferences (independent from email balance notify)
ALTER TABLE users ADD COLUMN IF NOT EXISTS websocket_notify_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS websocket_balance_alert_enabled BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS websocket_balance_alert_threshold DECIMAL(20,8) DEFAULT NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS websocket_site_message_notify_enabled BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS websocket_announcement_notify_enabled BOOLEAN NOT NULL DEFAULT true;

COMMENT ON COLUMN users.websocket_notify_enabled IS 'User master switch for WebSocket realtime notifications; default off.';
COMMENT ON COLUMN users.websocket_balance_alert_enabled IS 'When WebSocket is enabled, push balance-below-threshold alerts; default on.';
COMMENT ON COLUMN users.websocket_balance_alert_threshold IS 'Independent WebSocket balance alert threshold (USD); NULL means use $10 default.';
COMMENT ON COLUMN users.websocket_site_message_notify_enabled IS 'When WebSocket is enabled, push on new site-message inbox items; default on.';
COMMENT ON COLUMN users.websocket_announcement_notify_enabled IS 'When WebSocket is enabled, push on new announcements; default on.';

-- System default for email balance-low threshold: new installs / missing key use 10.
-- Only fill when the setting row is absent; do not overwrite admin-configured values.
INSERT INTO settings (key, value, updated_at)
VALUES ('balance_low_notify_threshold', '10', NOW())
ON CONFLICT (key) DO NOTHING;
