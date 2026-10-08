-- 充值满赠：订单快照赠送额（履约幂等锚定 -B 兑换码，流水 type=recharge_bonus）
ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS bonus_amount DECIMAL(20,2) NOT NULL DEFAULT 0;
