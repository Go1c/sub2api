-- Migration: 946_channel_monitor_iq
-- 渠道监控第四种检查方式「状态 + 智商」(check_mode=iq)：
--   一次糖果题请求同时记下服务器状态与智商四态。
--   检测间隔仍为 15–3600 秒（与探活相同，不另收紧）。

ALTER TABLE channel_monitors DROP CONSTRAINT IF EXISTS channel_monitors_check_mode_check;
ALTER TABLE channel_monitors
    ADD CONSTRAINT channel_monitors_check_mode_check
    CHECK (check_mode IN ('probe', 'quota', 'quota_probe', 'iq'));

ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS iq_question TEXT NOT NULL DEFAULT '';
ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS iq_answer VARCHAR(200) NOT NULL DEFAULT '';
ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS iq_fuzzy_match BOOLEAN NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN channel_monitors.check_mode IS
    'probe = LLM 探活（默认）；quota = 仅查关联账号用量；quota_probe = 探活 + 配额；iq = 状态 + 智商（糖果题一次请求填两条时间线）';
COMMENT ON COLUMN channel_monitors.iq_question IS
    '智商测试题；check_mode=iq 时空串按默认糖果题填充';
COMMENT ON COLUMN channel_monitors.iq_answer IS
    '智商测试标准答案；check_mode=iq 时空串按 21 填充';
COMMENT ON COLUMN channel_monitors.iq_fuzzy_match IS
    '智商判定是否模糊包含标准答案；默认 true';

ALTER TABLE channel_monitor_histories
    ADD COLUMN IF NOT EXISTS iq_status VARCHAR(32) NOT NULL DEFAULT '';

ALTER TABLE channel_monitor_histories DROP CONSTRAINT IF EXISTS channel_monitor_histories_iq_status_check;
ALTER TABLE channel_monitor_histories
    ADD CONSTRAINT channel_monitor_histories_iq_status_check
    CHECK (iq_status IN ('', 'iq_ok', 'iq_down', 'test_error', 'monitor_network'));

COMMENT ON COLUMN channel_monitor_histories.iq_status IS
    '智商四态：iq_ok / iq_down / test_error / monitor_network；非 iq 检测为空串';
