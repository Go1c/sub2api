-- 179_group_web_search_price_per_call.sql
-- Fork remapping of upstream 174_group_web_search_price_per_call.sql
-- (fork 174 is usage_log long-context billing / api_key ip index).
-- Codex alpha/search per-call override; NULL = built-in default.

ALTER TABLE groups ADD COLUMN IF NOT EXISTS web_search_price_per_call DECIMAL(20,8);
