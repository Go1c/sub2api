-- 151_subscription_backfill_exhausted_at.sql
--
-- 存量补盖：修复「订阅总额度已用满却没盖 exhausted_at，导致无法续购」的精度边界 bug。
--
-- 根因：扣费 UPDATE 把带浮点误差的 quota_used + subCost 存进 DECIMAL(20,10) 时四舍五入
-- 凑成 quota_limit（remaining=0），但同句未舍入的跨越判定看到的是「差一丝没到 limit」，
-- 于是 exhausted_at 该盖未盖；下游续购门槛只认 exhausted_at IS NOT NULL，把用户永久拦住。
--
-- 本 migration 给所有「已在 1e-6 容差内用满总额度、却仍未盖 exhausted_at」的可消费订阅
-- 补盖 exhausted_at。容差 1e-6 与扣费/续购路径的 SubscriptionQuotaExhaustionEpsilonUSD 一致。
--
-- 纯数据修复：只 UPDATE 本表，不写 scheduler_outbox，不发任何通知。
-- 幂等：exhausted_at IS NULL 前提保证已盖过的行不会被重复处理。
UPDATE user_subscriptions
SET exhausted_at = COALESCE(updated_at, NOW())
WHERE deleted_at IS NULL
    AND status = 'active'
    AND exhausted_at IS NULL
    AND quota_limit_usd > 0
    AND quota_used_usd >= quota_limit_usd - 1e-6;
