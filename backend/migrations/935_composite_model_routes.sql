-- Composite 分组模型路由表（origin 172 建表 + 227 CN CHECK 折进 935）。
--
-- 背景：fork 没有 composite_model_routes。origin 172 只允许 5 个目标平台，
-- origin 227 再把 kimi/zhipu/deepseek 写进 CHECK。本 fork 最高迁移 933（渠道监控配额），
-- 直接以 935 一条落地，CHECK 含 anthropic/openai/gemini/antigravity/grok/
-- kimi/zhipu/deepseek。不要再用 origin 编号 172/227。
CREATE TABLE IF NOT EXISTS composite_model_routes (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    public_model VARCHAR(200) NOT NULL,
    match_type VARCHAR(20) NOT NULL DEFAULT 'exact',
    target_platform VARCHAR(50) NOT NULL,
    upstream_model VARCHAR(200) NOT NULL DEFAULT '',
    endpoint VARCHAR(50) NOT NULL DEFAULT 'any',
    priority INTEGER NOT NULL DEFAULT 100,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT composite_model_routes_match_type_check CHECK (match_type IN ('exact', 'prefix')),
    CONSTRAINT composite_model_routes_endpoint_check CHECK (endpoint IN ('any', 'messages', 'count_tokens', 'responses', 'chat_completions', 'embeddings', 'images', 'gemini')),
    CONSTRAINT composite_model_routes_target_platform_check CHECK (target_platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'grok',
                               'kimi', 'zhipu', 'deepseek'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_composite_model_routes_unique_active
    ON composite_model_routes (group_id, endpoint, match_type, public_model)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_composite_model_routes_group_enabled
    ON composite_model_routes (group_id, enabled)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_composite_model_routes_group_priority
    ON composite_model_routes (group_id, priority, id)
    WHERE deleted_at IS NULL;
