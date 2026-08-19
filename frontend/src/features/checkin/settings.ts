import type { CheckinMilestone } from './types'

interface RewardSettings { max_reward: string; milestones: CheckinMilestone[] }
interface CapSettings extends RewardSettings { daily_cap: string }

function amountUnits(value: string): bigint {
  const match = value.trim().match(/^(\d+)(?:\.(\d{0,4}))?$/)
  if (!match) return 0n
  return BigInt(match[1]) * 10000n + BigInt((match[2] ?? '').padEnd(4, '0') || '0')
}

function unitsAmount(value: bigint): string {
  return `${value / 10000n}.${String(value % 10000n).padStart(4, '0')}`
}

export function maximumSingleReward(settings: RewardSettings): string {
  const maximumBonus = settings.milestones.reduce((maximum, item) => {
    const bonus = amountUnits(item.bonus)
    return bonus > maximum ? bonus : maximum
  }, 0n)
  return unitsAmount(amountUnits(settings.max_reward) + maximumBonus)
}

export function hasDailyCapRisk(settings: CapSettings): boolean {
  const cap = amountUnits(settings.daily_cap)
  return cap > 0n && cap < amountUnits(maximumSingleReward(settings))
}
