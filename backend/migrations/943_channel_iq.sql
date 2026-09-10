-- Migration: 943_channel_iq
-- 渠道智商检测：分组纳入列表、每账号最近一次 SVG 成图结果、定时自动检测配置

CREATE TABLE IF NOT EXISTS channel_iq_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    group_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    auto_enabled BOOLEAN NOT NULL DEFAULT false,
    interval_seconds INTEGER NOT NULL DEFAULT 1800,
    prompt TEXT NOT NULL DEFAULT 'Generate an SVG of a pelican riding a bicycle',
    model VARCHAR(200) NOT NULL DEFAULT 'gpt-6-astra',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO channel_iq_settings (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS channel_iq_results (
    account_id BIGINT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'idle',
    test_count INTEGER NOT NULL DEFAULT 0,
    model VARCHAR(200) NOT NULL DEFAULT '',
    reasoning_effort VARCHAR(32) NOT NULL DEFAULT 'low',
    duration_ms BIGINT NOT NULL DEFAULT 0,
    total_tokens BIGINT NOT NULL DEFAULT 0,
    svg TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    last_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE channel_iq_settings IS '渠道智商检测全局配置（单行）';
COMMENT ON TABLE channel_iq_results IS '渠道智商检测每账号最近一次结果';
