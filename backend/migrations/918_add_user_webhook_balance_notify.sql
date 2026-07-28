-- User external robot/webhook balance notify (WeCom primary; independent from email & browser WebSocket)
ALTER TABLE users ADD COLUMN IF NOT EXISTS webhook_balance_notify_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS webhook_balance_notify_url TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS webhook_balance_notify_threshold DECIMAL(20,8) DEFAULT NULL;

COMMENT ON COLUMN users.webhook_balance_notify_enabled IS 'Enable external webhook/robot balance-low alerts; default off.';
COMMENT ON COLUMN users.webhook_balance_notify_url IS 'Webhook URL (WeCom robot: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...).';
COMMENT ON COLUMN users.webhook_balance_notify_threshold IS 'Independent threshold for webhook balance alerts; NULL uses $10 default.';
