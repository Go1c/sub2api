-- Keep timeouts distinct from wrong answers and other test errors.
ALTER TABLE channel_monitor_histories
    DROP CONSTRAINT IF EXISTS channel_monitor_histories_iq_status_check;
ALTER TABLE channel_monitor_histories
    ADD CONSTRAINT channel_monitor_histories_iq_status_check
    CHECK (iq_status IN ('', 'iq_ok', 'iq_down', 'test_timeout', 'test_error', 'monitor_network'));

COMMENT ON COLUMN channel_monitor_histories.iq_status IS
    '智商检测：iq_ok / iq_down / test_timeout / test_error / monitor_network；非 iq 检测为空串';
