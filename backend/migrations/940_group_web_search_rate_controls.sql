-- Add independent group rate controls for Codex/Grok web search per-call billing.

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS web_search_rate_independent BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS web_search_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;

COMMENT ON COLUMN groups.web_search_rate_independent IS '网页搜索是否使用独立倍率；false 表示共享分组有效倍率';
COMMENT ON COLUMN groups.web_search_rate_multiplier IS '网页搜索独立倍率，仅 web_search_rate_independent=true 时生效';
