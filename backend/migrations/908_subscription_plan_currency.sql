-- Display-only ISO 4217 currency label for subscription plan prices.
-- Empty string means "no label"; does not affect billing or settlement currency.
ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS currency VARCHAR(3) NOT NULL DEFAULT '';
