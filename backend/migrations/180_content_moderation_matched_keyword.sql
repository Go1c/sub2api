-- 180_content_moderation_matched_keyword.sql
-- Fork remapping of upstream 156_content_moderation_matched_keyword.sql
-- (fork 156 is site_message_compensation_batches).

ALTER TABLE content_moderation_logs ADD COLUMN IF NOT EXISTS matched_keyword VARCHAR(255) NOT NULL DEFAULT '';
