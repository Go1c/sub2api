import { prisma } from '@/lib/db';
import { getEnv } from '@/lib/config';

/**
 * 获取指定支付渠道的每日全平台限额（0 = 不限制）。
 * 优先读 config（Zod 验证），兜底读 process.env，适配未来动态注册的新渠道。
 */
export function getMethodDailyLimit(paymentType: string): number {
  const env = getEnv();
  const key = `MAX_DAILY_AMOUNT_${paymentType.toUpperCase()}` as keyof typeof env;
  const val = env[key];
  if (typeof val === 'number') return val;

  // 兜底：支持动态渠道（未在 schema 中声明的 MAX_DAILY_AMOUNT_* 变量）
  const raw = process.env[`MAX_DAILY_AMOUNT_${paymentType.toUpperCase()}`];
  if (raw !== undefined) {
    const num = Number(raw);
    return Number.isFinite(num) && num >= 0 ? num : 0;
  }
  return 0; // 默认不限制
}

export interface MethodLimitStatus {
  /** 每日限额，0 = 不限 */
  dailyLimit: number;
  /** 今日已使用金额 */
  used: number;
  /** 剩余额度，null = 不限 */
  remaining: number | null;
  /** 是否还可使用（false = 今日额度已满） */
  available: boolean;
}

/**
 * 批量查询多个支付渠道的今日使用情况。
 * 一次 DB groupBy 完成，调用方按需传入渠道列表。
 */
export async function queryMethodLimits(
  paymentTypes: string[],
): Promise<Record<string, MethodLimitStatus>> {
  const todayStart = new Date();
  todayStart.setUTCHours(0, 0, 0, 0);

  const usageRows = await prisma.order.groupBy({
    by: ['paymentType'],
    where: {
      paymentType: { in: paymentTypes },
      status: { in: ['PAID', 'RECHARGING', 'COMPLETED'] },
      paidAt: { gte: todayStart },
    },
    _sum: { amount: true },
  });

  const usageMap = Object.fromEntries(
    usageRows.map((r) => [r.paymentType, Number(r._sum.amount ?? 0)]),
  );

  const result: Record<string, MethodLimitStatus> = {};
  for (const type of paymentTypes) {
    const dailyLimit = getMethodDailyLimit(type);
    const used = usageMap[type] ?? 0;
    const remaining = dailyLimit > 0 ? Math.max(0, dailyLimit - used) : null;
    result[type] = {
      dailyLimit,
      used,
      remaining,
      available: dailyLimit === 0 || used < dailyLimit,
    };
  }
  return result;
}
