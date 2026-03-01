import { NextResponse } from 'next/server';
import { getEnv } from '@/lib/config';
import { queryMethodLimits } from '@/lib/order/limits';

/**
 * GET /api/limits
 * 返回各支付渠道今日限额使用情况，公开接口（无需鉴权）。
 *
 * Response:
 * {
 *   methods: {
 *     alipay: { dailyLimit: 10000, used: 3500, remaining: 6500, available: true },
 *     wxpay:  { dailyLimit: 10000, used: 10000, remaining: 0,    available: false },
 *     stripe: { dailyLimit: 0,     used: 500,  remaining: null,  available: true }
 *   },
 *   resetAt: "2026-03-02T00:00:00.000Z"  // UTC 次日零点（限额重置时间）
 * }
 */
export async function GET() {
  const env = getEnv();
  const types = env.ENABLED_PAYMENT_TYPES;

  const todayStart = new Date();
  todayStart.setUTCHours(0, 0, 0, 0);
  const resetAt = new Date(todayStart);
  resetAt.setUTCDate(resetAt.getUTCDate() + 1);

  const methods = await queryMethodLimits(types);

  return NextResponse.json({ methods, resetAt });
}
