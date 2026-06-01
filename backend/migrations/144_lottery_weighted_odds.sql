-- Add configurable lottery odds weighting.

ALTER TABLE lottery_campaigns
    ADD COLUMN IF NOT EXISTS early_boost_participant_percent INTEGER NOT NULL DEFAULT 25;

ALTER TABLE lottery_campaigns
    ADD COLUMN IF NOT EXISTS recharge_boost_cap_percent INTEGER NOT NULL DEFAULT 0;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_lottery_campaigns_early_boost_percent'
    ) THEN
        ALTER TABLE lottery_campaigns
            ADD CONSTRAINT chk_lottery_campaigns_early_boost_percent
            CHECK (early_boost_participant_percent >= 0 AND early_boost_participant_percent <= 100);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_lottery_campaigns_recharge_boost_cap'
    ) THEN
        ALTER TABLE lottery_campaigns
            ADD CONSTRAINT chk_lottery_campaigns_recharge_boost_cap
            CHECK (recharge_boost_cap_percent >= 0 AND recharge_boost_cap_percent <= 50);
    END IF;
END $$;
