-- 渠道监控 provider 枚举增加 grok。
-- Grok/xAI 使用 OpenAI 兼容的 /v1/chat/completions 探测路径。
--
-- 两张表都有 provider CHECK 约束：
--   channel_monitors
--   channel_monitor_request_templates
-- DROP IF EXISTS + 新约束（旧值超集）保证可重入；存量行瞬时校验通过。

ALTER TABLE channel_monitors
    DROP CONSTRAINT IF EXISTS channel_monitors_provider_check;

ALTER TABLE channel_monitors
    ADD CONSTRAINT channel_monitors_provider_check
    CHECK (provider IN ('openai', 'anthropic', 'gemini', 'grok'));

ALTER TABLE channel_monitor_request_templates
    DROP CONSTRAINT IF EXISTS channel_monitor_request_templates_provider_check;

ALTER TABLE channel_monitor_request_templates
    ADD CONSTRAINT channel_monitor_request_templates_provider_check
    CHECK (provider IN ('openai', 'anthropic', 'gemini', 'grok'));
