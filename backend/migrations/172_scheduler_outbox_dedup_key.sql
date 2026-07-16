-- 172_scheduler_outbox_dedup_key.sql
-- Fork remapping of upstream 152_scheduler_outbox_dedup_key.sql
-- (fork 152 is add_account_error_histories). Required by scheduler outbox poll/insert.

ALTER TABLE scheduler_outbox
    ADD COLUMN IF NOT EXISTS dedup_key TEXT;
