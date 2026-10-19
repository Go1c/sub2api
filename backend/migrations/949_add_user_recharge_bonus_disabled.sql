-- users.recharge_bonus_disabled: admin can hide recharge-bonus tags and skip the gift.
-- Default FALSE keeps existing users eligible for the configured bonus.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS recharge_bonus_disabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN users.recharge_bonus_disabled IS
    'When true, the user can still recharge but does not see recharge-bonus tags and is not granted bonus balance.';
