-- Migration: 944_channel_iq_excluded
-- 智商检测：从列表移除账号（不改分组），并删除该账号已存 SVG

ALTER TABLE channel_iq_settings
    ADD COLUMN IF NOT EXISTS excluded_account_ids JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN channel_iq_settings.excluded_account_ids IS '不监测的账号 ID 列表（仍留在分组里）';
