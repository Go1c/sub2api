-- Lottery campaign promo copy and public HTTPS image URL.

ALTER TABLE lottery_campaigns
    ADD COLUMN IF NOT EXISTS promo_text VARCHAR(240) NOT NULL DEFAULT '';

ALTER TABLE lottery_campaigns
    ADD COLUMN IF NOT EXISTS promo_image_url VARCHAR(2048) NOT NULL DEFAULT '';
