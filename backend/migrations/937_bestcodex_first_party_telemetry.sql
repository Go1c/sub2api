-- Migration: 937_bestcodex_first_party_telemetry
-- First-party BestCodex ingest events and account first/last touch attribution.
-- Idempotent. No DROP. Does not touch users.signup_source.

CREATE TABLE IF NOT EXISTS telemetry_events (
    id                     BIGSERIAL    PRIMARY KEY,
    event                  VARCHAR(64)  NOT NULL,
    occurred_at            TIMESTAMPTZ  NOT NULL,
    ingested_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    client_source          VARCHAR(64)  NOT NULL DEFAULT 'unknown',
    route                  VARCHAR(128) NOT NULL DEFAULT '',
    auth_method            VARCHAR(16)  NOT NULL DEFAULT '',
    platform               VARCHAR(32)  NOT NULL DEFAULT '',
    destination            VARCHAR(16)  NOT NULL DEFAULT '',
    error_code             VARCHAR(64)  NOT NULL DEFAULT '',
    attribution_id         VARCHAR(40)  NOT NULL DEFAULT '',
    first_touch_source     VARCHAR(32)  NOT NULL DEFAULT '',
    first_touch_medium     VARCHAR(32)  NOT NULL DEFAULT '',
    first_touch_campaign   VARCHAR(32)  NOT NULL DEFAULT '',
    last_touch_source      VARCHAR(32)  NOT NULL DEFAULT '',
    last_touch_medium      VARCHAR(32)  NOT NULL DEFAULT '',
    last_touch_campaign    VARCHAR(32)  NOT NULL DEFAULT '',
    user_id                BIGINT       REFERENCES users(id) ON DELETE SET NULL,
    dedup_key              VARCHAR(128) NOT NULL DEFAULT '',
    ingest_source          VARCHAR(16)  NOT NULL DEFAULT 'client'
);

CREATE INDEX IF NOT EXISTS telemetry_events_occurred_at_event_idx
    ON telemetry_events (occurred_at, event);
CREATE INDEX IF NOT EXISTS telemetry_events_first_touch_campaign_occurred_at_idx
    ON telemetry_events (first_touch_campaign, occurred_at);
CREATE INDEX IF NOT EXISTS telemetry_events_client_source_occurred_at_idx
    ON telemetry_events (client_source, occurred_at);
CREATE INDEX IF NOT EXISTS telemetry_events_attribution_id_event_occurred_at_idx
    ON telemetry_events (attribution_id, event, occurred_at);
CREATE UNIQUE INDEX IF NOT EXISTS telemetry_events_dedup_key_uidx
    ON telemetry_events (dedup_key)
    WHERE dedup_key <> '';
CREATE INDEX IF NOT EXISTS telemetry_events_event_user_id_occurred_at_idx
    ON telemetry_events (event, user_id, occurred_at)
    WHERE user_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS user_first_party_attribution (
    user_id                BIGINT       PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    first_touch_source     VARCHAR(32)  NOT NULL DEFAULT '',
    first_touch_medium     VARCHAR(32)  NOT NULL DEFAULT '',
    first_touch_campaign   VARCHAR(32)  NOT NULL DEFAULT '',
    first_attribution_id   VARCHAR(40)  NOT NULL DEFAULT '',
    last_touch_source      VARCHAR(32)  NOT NULL DEFAULT '',
    last_touch_medium      VARCHAR(32)  NOT NULL DEFAULT '',
    last_touch_campaign    VARCHAR(32)  NOT NULL DEFAULT '',
    last_attribution_id    VARCHAR(40)  NOT NULL DEFAULT '',
    first_touch_at         TIMESTAMPTZ,
    last_touch_at          TIMESTAMPTZ
);

COMMENT ON TABLE telemetry_events IS 'BestCodex first-party telemetry ingest. Client events are observations; register/login authority is this table.';
COMMENT ON TABLE user_first_party_attribution IS 'Per-account first-touch write-once and last-touch update attribution for BestCodex.';
