-- 154_add_group_exhausted_fallback.sql
-- 添加账号全部不可用时兜底分组配置

-- 添加 fallback_group_id_on_exhausted 字段：上游账号全部不可用时兜底使用的分组
ALTER TABLE groups
ADD COLUMN IF NOT EXISTS fallback_group_id_on_exhausted BIGINT REFERENCES groups(id) ON DELETE SET NULL;

-- 添加索引优化查询
CREATE INDEX IF NOT EXISTS idx_groups_fallback_group_id_on_exhausted
ON groups(fallback_group_id_on_exhausted) WHERE deleted_at IS NULL AND fallback_group_id_on_exhausted IS NOT NULL;

-- 添加字段注释
COMMENT ON COLUMN groups.fallback_group_id_on_exhausted IS '上游账号全部不可用时兜底使用的分组 ID';
