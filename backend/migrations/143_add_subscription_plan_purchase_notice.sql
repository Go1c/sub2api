ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS purchase_notice text NOT NULL DEFAULT '';
