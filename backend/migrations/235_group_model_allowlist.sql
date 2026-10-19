-- 235: 新增 groups.model_allowlist，并从 models_list_config 拷贝已有配置。
--
-- 不重命名、不删除 models_list_config。新程序的 ent 仍同时查询这两列，
-- 线上旧镜像也仍查询旧列。改名会在新进程听端口之前就让密钥和用量列表 500。
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_attribute
        WHERE attrelid = 'groups'::regclass
          AND attname = 'model_allowlist'
          AND NOT attisdropped
    ) THEN
        ALTER TABLE groups
            ADD COLUMN model_allowlist JSONB NOT NULL DEFAULT '{}'::jsonb;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_attribute
        WHERE attrelid = 'groups'::regclass
          AND attname = 'models_list_config'
          AND NOT attisdropped
    ) THEN
        UPDATE groups
           SET model_allowlist = models_list_config
         WHERE COALESCE(model_allowlist, '{}'::jsonb) = '{}'::jsonb
           AND COALESCE(models_list_config, '{}'::jsonb) <> '{}'::jsonb;
    END IF;
END
$$;

COMMENT ON COLUMN groups.model_allowlist IS
    'Group model allowlist: constrains both model listing responses and request admission';
