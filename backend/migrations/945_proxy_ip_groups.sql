-- Migration: 945_proxy_ip_groups
-- OpenAI OAuth IP 组：组表、组成员、账号外键。软删组；组成员随代理删除由仓储清理。

CREATE TABLE IF NOT EXISTS proxy_ip_groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    per_ip_concurrency INT NOT NULL DEFAULT 10
        CHECK (per_ip_concurrency >= 1 AND per_ip_concurrency <= 1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS proxy_ip_groups_name_alive
    ON proxy_ip_groups (name) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS proxy_ip_group_members (
    group_id BIGINT NOT NULL REFERENCES proxy_ip_groups(id),
    proxy_id BIGINT NOT NULL REFERENCES proxies(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (group_id, proxy_id)
);

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS proxy_ip_group_id BIGINT REFERENCES proxy_ip_groups(id);

CREATE INDEX IF NOT EXISTS accounts_proxy_ip_group_id_idx
    ON accounts (proxy_ip_group_id) WHERE deleted_at IS NULL AND proxy_ip_group_id IS NOT NULL;

COMMENT ON TABLE proxy_ip_groups IS 'OpenAI OAuth IP 组：统一每 IP 并发上限';
COMMENT ON COLUMN proxy_ip_groups.per_ip_concurrency IS '该组内每条代理对单个账号的并发上限';
COMMENT ON COLUMN accounts.proxy_ip_group_id IS 'OpenAI OAuth 账号选用的 IP 组；与 proxy_id 互斥';
