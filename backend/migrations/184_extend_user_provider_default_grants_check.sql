-- 184_extend_user_provider_default_grants_check.sql
-- Fork remapping of upstream 140_extend_user_provider_default_grants_check.sql
-- (fork 140 is add_lottery). Align provider_type check with auth identity tables.

ALTER TABLE user_provider_default_grants
    DROP CONSTRAINT IF EXISTS user_provider_default_grants_provider_type_check;

ALTER TABLE user_provider_default_grants
    ADD CONSTRAINT user_provider_default_grants_provider_type_check
    CHECK (provider_type IN ('email', 'linuxdo', 'wechat', 'oidc', 'github', 'google', 'dingtalk'));
