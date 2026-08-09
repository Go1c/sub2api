-- users.subscription_purchase_disabled: admin can ban a user from purchasing subscriptions.
-- Default FALSE keeps existing users allowed to purchase.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS subscription_purchase_disabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN users.subscription_purchase_disabled IS
    'When true, the user cannot create subscription purchase orders (admin-controlled ban).';
