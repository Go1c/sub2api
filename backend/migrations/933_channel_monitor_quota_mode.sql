-- Migration: 933_channel_monitor_quota_mode
-- fork remap of origin 226_channel_monitor_quota_mode.sql
-- 渠道监控配额模式（只保留 openai / anthropic / gemini / grok）：
--   1. 不扩 provider CHECK（fork 已是四家，见 migration 906）
--   2. check_mode：probe（默认，现状探活）/ quota（仅查关联账号用量，零 LLM 成本）
--      / quota_probe（探活 + 配额并存）
--   3. account_id 关联已有账号（配额模式的数据源，复用账号侧用量服务）；
--      账号删除时置空，监控保留并报「账号未关联」
--   4. channel_monitor_histories.quota 持久化归一化配额快照（JSONB）
--   5. 新增公开设置 channel_monitor_show_quota（默认关闭）：
--      控制用户端监控页是否展示配额/余额；管理端始终可见

ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS check_mode VARCHAR(32) NOT NULL DEFAULT 'probe';

ALTER TABLE channel_monitors
    ADD CONSTRAINT channel_monitors_check_mode_check
    CHECK (check_mode IN ('probe', 'quota', 'quota_probe'));

ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_channel_monitors_account_id ON channel_monitors(account_id);

COMMENT ON COLUMN channel_monitors.check_mode IS
    'probe = LLM 探活（默认）；quota = 仅查关联账号用量；quota_probe = 探活 + 配额';
COMMENT ON COLUMN channel_monitors.account_id IS
    '配额模式关联的账号 ID（数据源）；账号删除时置空';

ALTER TABLE channel_monitor_histories
    ADD COLUMN IF NOT EXISTS quota JSONB;

COMMENT ON COLUMN channel_monitor_histories.quota IS
    '配额模式监控的归一化配额快照（domain.MonitorQuotaSnapshot）；探活模式为 NULL';

-- 用户端是否展示配额/余额（默认关闭，fail-closed 解析：仅 "true" 视为开启）。
-- 管理端不受此开关影响。
INSERT INTO settings (key, value)
VALUES ('channel_monitor_show_quota', 'false')
ON CONFLICT (key) DO NOTHING;
