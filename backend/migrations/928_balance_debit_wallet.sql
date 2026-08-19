-- Migration: 928_balance_debit_wallet
-- External balance consumers, immutable successful debit ledger, and durable
-- cache invalidation. No plaintext client secret or idempotency key is stored.

CREATE TABLE IF NOT EXISTS balance_debit_clients (
    id               BIGSERIAL PRIMARY KEY,
    client_id        UUID         NOT NULL,
    name             VARCHAR(100) NOT NULL,
    secret_hash      CHAR(64)     NOT NULL,
    secret_prefix    VARCHAR(32)  NOT NULL,
    allowed_purposes JSONB        NOT NULL,
    status           VARCHAR(16)  NOT NULL DEFAULT 'active',
    last_used_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT balance_debit_clients_status_check
        CHECK (status IN ('active', 'inactive')),
    CONSTRAINT balance_debit_clients_allowed_purposes_check
        CHECK (jsonb_typeof(allowed_purposes) = 'array' AND jsonb_array_length(allowed_purposes) > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS balance_debit_clients_client_id_key
    ON balance_debit_clients (client_id);
CREATE UNIQUE INDEX IF NOT EXISTS balance_debit_clients_secret_hash_key
    ON balance_debit_clients (secret_hash);
CREATE INDEX IF NOT EXISTS balance_debit_clients_status_idx
    ON balance_debit_clients (status);

CREATE TABLE IF NOT EXISTS balance_debit_transactions (
    id                       BIGSERIAL PRIMARY KEY,
    txn_id                   UUID         NOT NULL,
    user_id                  BIGINT       NOT NULL,
    balance_client_id        BIGINT       NOT NULL REFERENCES balance_debit_clients(id) ON DELETE RESTRICT,
    idempotency_key_hash     CHAR(64)     NOT NULL,
    request_fingerprint      CHAR(64)     NOT NULL,
    amount                   NUMERIC(20,2) NOT NULL,
    currency                 VARCHAR(3)   NOT NULL,
    purpose                  VARCHAR(64)  NOT NULL,
    ref                      VARCHAR(128) NOT NULL,
    balance_before           NUMERIC(20,8) NOT NULL,
    balance_after            NUMERIC(20,8) NOT NULL,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT balance_debit_transactions_amount_check CHECK (amount > 0),
    CONSTRAINT balance_debit_transactions_currency_check CHECK (currency = 'CNY'),
    CONSTRAINT balance_debit_transactions_balance_check
        CHECK (balance_before >= amount AND balance_after = balance_before - amount),
    UNIQUE (balance_client_id, user_id, idempotency_key_hash)
);

CREATE UNIQUE INDEX IF NOT EXISTS balance_debit_transactions_txn_id_key
    ON balance_debit_transactions (txn_id);
CREATE INDEX IF NOT EXISTS balance_debit_transactions_user_created_id_idx
    ON balance_debit_transactions (user_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS balance_debit_transactions_user_ref_idx
    ON balance_debit_transactions (user_id, ref);
CREATE INDEX IF NOT EXISTS balance_debit_transactions_client_created_idx
    ON balance_debit_transactions (balance_client_id, created_at DESC);

-- One pending row per user coalesces repeated balance changes. Workers claim
-- with FOR UPDATE SKIP LOCKED and retry failures with exponential backoff.
CREATE TABLE IF NOT EXISTS balance_cache_invalidation_outbox (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       NOT NULL,
    attempts        INTEGER      NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    claimed_at      TIMESTAMPTZ,
    claim_token     VARCHAR(36),
    last_error      TEXT         NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS balance_cache_invalidation_outbox_due_idx
    ON balance_cache_invalidation_outbox (next_attempt_at, id);

COMMENT ON TABLE balance_debit_clients IS
    'External server-side consumers allowed to debit user wallet balances; secret hash only.';
COMMENT ON TABLE balance_debit_transactions IS
    'Immutable successful wallet debits. user_id is retained as financial audit data without a cascading users FK.';
COMMENT ON TABLE balance_cache_invalidation_outbox IS
    'Durable coalesced invalidation of billing balance and API-key auth caches.';
