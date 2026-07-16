-- 178_add_group_peak_rate_multiplier.sql
-- Fork remapping of upstream 158_add_group_peak_rate_multiplier.sql
-- (fork 158 unused; number collision with other upstream 158).
-- Required by scheduler snapshot rebuild (groups.peak_rate_* via ent).

ALTER TABLE groups ADD COLUMN IF NOT EXISTS peak_rate_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS peak_start VARCHAR(5) NOT NULL DEFAULT '';
ALTER TABLE groups ADD COLUMN IF NOT EXISTS peak_end VARCHAR(5) NOT NULL DEFAULT '';
ALTER TABLE groups ADD COLUMN IF NOT EXISTS peak_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;
