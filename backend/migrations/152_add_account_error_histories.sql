-- Migration: 152_add_account_error_histories
-- 账号错误历史：记录每个账号来自不同来源（网关真实请求 / 账号测试 / 限流 / token 刷新 /
-- 调度 / 管理员手动置错）的错误，供管理后台「更多 > 错误历史」弹窗按需排查。
--
-- 设计要点：
--   - 这是 best-effort 监控数据：写入异步、非阻塞，丢几条可接受；不在热路径做同步写。
--   - 字段可空性按来源不同而不同：只有 gateway 源（真实请求）才齐全
--     （user_id / user_email / model / upstream_status_code），其余来源对应字段留 NULL。
--   - 防刷爆 DB 三件套（在 repository 层实现）：
--       1) 60s 窗口内同 (account_id, source, upstream_status_code, fingerprint) 折叠为一行，dup_count+1；
--       2) 每账号最小写入间隔节流；
--       3) 每账号裁剪保留最近 ~50 条。
--   - history 通过 ON DELETE CASCADE 随账号删除自动清理。
--   - (account_id, created_at DESC) 索引服务最近 N 条查询；
--     (account_id, source, upstream_status_code, fingerprint, created_at) 服务 60s 去重查找。
--
-- 注意：Ent schema 仅用于 ORM 读写，建表以本 SQL 为准；两边字段/类型必须严格一致。

CREATE TABLE IF NOT EXISTS account_error_histories (
    id                   BIGSERIAL PRIMARY KEY,
    account_id           BIGINT       NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    user_id              BIGINT,                              -- 真实请求才有
    user_email           VARCHAR(255),                        -- 邮箱快照，避免 join
    model                VARCHAR(200),                        -- 真实请求才有
    upstream_status_code INT,                                 -- 上游 HTTP 状态码，仅真实请求齐全
    source               VARCHAR(20)  NOT NULL,               -- gateway/test/ratelimit/refresh/schedule/admin
    message              TEXT         NOT NULL DEFAULT '',     -- 错误内容/上游返回
    fingerprint          VARCHAR(64)  NOT NULL DEFAULT '',     -- 去重指纹：message 归一后 hash
    dup_count            INT          NOT NULL DEFAULT 1,      -- 折叠计数
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),  -- 去重命中时刷新
    CONSTRAINT account_error_histories_source_check
        CHECK (source IN ('gateway', 'test', 'ratelimit', 'refresh', 'schedule', 'admin'))
);

CREATE INDEX IF NOT EXISTS idx_account_error_histories_account_created
    ON account_error_histories (account_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_account_error_histories_dedup
    ON account_error_histories (account_id, source, upstream_status_code, fingerprint, created_at);
