-- 171_channel_monitor_openai_api_mode.sql
-- Fork remapping of upstream 138_channel_monitor_openai_api_mode.sql
-- (fork 138 is invoice_requests). Required by channel monitor service/ent.
--
-- OpenAI protocol mode:
--   chat_completions -> /v1/chat/completions + messages
--   responses        -> /v1/responses + instructions/input
-- Historical rows default to chat_completions to preserve existing behavior.

ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS api_mode VARCHAR(32) NOT NULL DEFAULT 'chat_completions';

ALTER TABLE channel_monitor_request_templates
    ADD COLUMN IF NOT EXISTS api_mode VARCHAR(32) NOT NULL DEFAULT 'chat_completions';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'channel_monitors_api_mode_check'
          AND table_name = 'channel_monitors'
    ) THEN
        ALTER TABLE channel_monitors
            ADD CONSTRAINT channel_monitors_api_mode_check
            CHECK (api_mode IN ('chat_completions', 'responses'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'channel_monitor_request_templates_api_mode_check'
          AND table_name = 'channel_monitor_request_templates'
    ) THEN
        ALTER TABLE channel_monitor_request_templates
            ADD CONSTRAINT channel_monitor_request_templates_api_mode_check
            CHECK (api_mode IN ('chat_completions', 'responses'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_channel_monitors_provider_api_mode
    ON channel_monitors (provider, api_mode);

CREATE INDEX IF NOT EXISTS idx_channel_monitor_templates_provider_api_mode
    ON channel_monitor_request_templates (provider, api_mode);
