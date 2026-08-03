-- 171a_channel_monitor_jitter.sql
-- Fork remapping of upstream 151_channel_monitor_jitter.sql
-- (fork 151 is subscription_backfill_exhausted_at).
--
-- ± [0, jitter_seconds] uniform offset on interval_seconds; 0 = fixed interval.

ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS jitter_seconds INTEGER NOT NULL DEFAULT 0;
