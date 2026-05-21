-- Add backend-backed lottery campaigns, prize codes, and draw records.

CREATE TABLE IF NOT EXISTS lottery_campaigns (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL CHECK (LENGTH(BTRIM(name)) > 0),
    subtitle VARCHAR(240) NOT NULL CHECK (LENGTH(BTRIM(subtitle)) > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    prize_count INTEGER NOT NULL CHECK (prize_count > 0),
    max_participants INTEGER NOT NULL CHECK (max_participants > 0),
    joined_count INTEGER NOT NULL DEFAULT 0 CHECK (joined_count >= 0),
    winner_count INTEGER NOT NULL DEFAULT 0 CHECK (winner_count >= 0),
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ NULL,
    CHECK (max_participants >= prize_count),
    CHECK (joined_count <= max_participants),
    CHECK (winner_count <= prize_count)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_lottery_campaigns_one_active
    ON lottery_campaigns (status)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_lottery_campaigns_status
    ON lottery_campaigns (status);

CREATE INDEX IF NOT EXISTS idx_lottery_campaigns_created_at
    ON lottery_campaigns (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_lottery_campaigns_created_by
    ON lottery_campaigns (created_by);

CREATE TABLE IF NOT EXISTS lottery_codes (
    id BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT NOT NULL REFERENCES lottery_campaigns(id) ON DELETE CASCADE,
    code VARCHAR(128) NOT NULL CHECK (LENGTH(BTRIM(code)) > 0),
    assigned_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    assigned_draw_id BIGINT NULL,
    assigned_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_lottery_codes_campaign_code
    ON lottery_codes (campaign_id, code);

CREATE INDEX IF NOT EXISTS idx_lottery_codes_campaign_assigned
    ON lottery_codes (campaign_id, assigned_at, id);

CREATE INDEX IF NOT EXISTS idx_lottery_codes_assigned_user
    ON lottery_codes (assigned_user_id);

CREATE INDEX IF NOT EXISTS idx_lottery_codes_assigned_draw
    ON lottery_codes (assigned_draw_id);

CREATE TABLE IF NOT EXISTS lottery_draws (
    id BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT NOT NULL REFERENCES lottery_campaigns(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    won BOOLEAN NOT NULL DEFAULT FALSE,
    lottery_code_id BIGINT NULL REFERENCES lottery_codes(id) ON DELETE SET NULL,
    site_message_id BIGINT NULL REFERENCES site_messages(id) ON DELETE SET NULL,
    result_label VARCHAR(80) NOT NULL CHECK (LENGTH(BTRIM(result_label)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_lottery_draws_campaign_user
    ON lottery_draws (campaign_id, user_id);

CREATE INDEX IF NOT EXISTS idx_lottery_draws_campaign_created
    ON lottery_draws (campaign_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_lottery_draws_user_created
    ON lottery_draws (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_lottery_draws_code
    ON lottery_draws (lottery_code_id);

CREATE INDEX IF NOT EXISTS idx_lottery_draws_site_message
    ON lottery_draws (site_message_id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_lottery_codes_assigned_draw'
    ) THEN
        ALTER TABLE lottery_codes
            ADD CONSTRAINT fk_lottery_codes_assigned_draw
            FOREIGN KEY (assigned_draw_id)
            REFERENCES lottery_draws(id)
            ON DELETE SET NULL;
    END IF;
END $$;
