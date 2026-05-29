CREATE INDEX IF NOT EXISTS idx_scheduler_outbox_subscription_notify_id
    ON scheduler_outbox (id)
    WHERE event_type = 'subscription_notify';
