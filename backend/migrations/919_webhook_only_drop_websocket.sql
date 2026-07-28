-- Webhook-only user notify: drop browser WebSocket prefs; expand webhook with site-message/announcement.
-- Keeps webhook_balance_* from 918; adds sub-toggles previously on websocket.

ALTER TABLE users ADD COLUMN IF NOT EXISTS webhook_site_message_notify_enabled BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS webhook_announcement_notify_enabled BOOLEAN NOT NULL DEFAULT true;

COMMENT ON COLUMN users.webhook_site_message_notify_enabled IS 'When webhook is enabled, POST on new site-message inbox items; default on.';
COMMENT ON COLUMN users.webhook_announcement_notify_enabled IS 'When webhook is enabled, POST on new announcements; default on.';

-- Drop user WebSocket preference columns (browser realtime path removed).
ALTER TABLE users DROP COLUMN IF EXISTS websocket_notify_enabled;
ALTER TABLE users DROP COLUMN IF EXISTS websocket_balance_alert_enabled;
ALTER TABLE users DROP COLUMN IF EXISTS websocket_balance_alert_threshold;
ALTER TABLE users DROP COLUMN IF EXISTS websocket_site_message_notify_enabled;
ALTER TABLE users DROP COLUMN IF EXISTS websocket_announcement_notify_enabled;
