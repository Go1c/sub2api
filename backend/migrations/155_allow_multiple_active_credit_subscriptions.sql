-- Allow multiple active usable credit-pool subscriptions.
--
-- The single-subscription restriction is now controlled by the
-- subscription_multiple_purchases_enabled setting in the purchase flow. When
-- disabled, fulfillment still locks the users row and checks for an existing
-- usable subscription before inserting.
DROP INDEX IF EXISTS user_subscriptions_user_active_usable;
