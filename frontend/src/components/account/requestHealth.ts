export const REQUEST_HEALTH_WINDOW_OPTIONS = [8, 12, 16, 20] as const
export const DEFAULT_REQUEST_HEALTH_WINDOW = 12
export const REQUEST_HEALTH_WINDOW_KEY = 'account-request-health-window'

export type RequestHealthSlot = 'ok' | 'fail' | 'empty'
export type HealthBadge = 'rate_limited' | 'overloaded' | 'cooling' | 'executing' | 'idle'
export type AccountProxyMode = 'single' | 'ip_group'

export interface RequestFailError {
  statusCode: number
  message: string
  occurredAt: string
  model?: string
  endpoint?: string
}

export interface RequestOutcome {
  slot: 'ok' | 'fail'
  error?: RequestFailError
}

export type WindowSlot = RequestOutcome | { slot: 'empty' }

export interface RequestHealthSnapshot {
  outcomes: RequestOutcome[]
  current: number
  max: number
  cooldownUntil?: number
  rateLimited?: boolean
  overloaded?: boolean
}

export interface RequestHealthOutcomeDTO {
  slot: 'ok' | 'fail'
  status_code?: number
  message?: string
  occurred_at?: string
  model?: string
  endpoint?: string
}

export interface RequestHealthLineDTO {
  proxy_id?: number
  ip: string
  outcomes: RequestHealthOutcomeDTO[]
  current: number
  max: number
  cooldown_until?: string | null
  rate_limited?: boolean
  overloaded?: boolean
}

export interface AccountRequestHealthDTO {
  account_id: number
  mode: AccountProxyMode
  ip_group_name?: string
  lines: RequestHealthLineDTO[]
}

export interface AccountHealthRow {
  accountId: number
  mode: AccountProxyMode
  ipGroupName?: string
  lines: Array<{
    ip: string
    health: RequestHealthSnapshot
  }>
}

export function clampRequestHealthWindow(value: number): number {
  return (REQUEST_HEALTH_WINDOW_OPTIONS as readonly number[]).includes(value)
    ? value
    : DEFAULT_REQUEST_HEALTH_WINDOW
}

export function sliceWindow(outcomes: RequestOutcome[], windowSize: number): WindowSlot[] {
  const size = clampRequestHealthWindow(windowSize)
  const recent = outcomes.slice(-size)
  if (recent.length >= size) return recent
  return [
    ...recent,
    ...Array.from({ length: size - recent.length }, () => ({ slot: 'empty' as const }))
  ]
}

export function formatCountdown(until: number, now: number): string {
  const remain = Math.max(0, Math.ceil((until - now) / 1000))
  const minutes = Math.floor(remain / 60)
  const seconds = remain % 60
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}

export function resolveBadge(health: RequestHealthSnapshot, now: number): HealthBadge {
  if (health.cooldownUntil && health.cooldownUntil > now) return 'cooling'
  if (health.overloaded) return 'overloaded'
  if (health.rateLimited) return 'rate_limited'
  if (health.current > 0) return 'executing'
  return 'idle'
}

export function toRequestOutcomes(outcomes: RequestHealthOutcomeDTO[] | undefined): RequestOutcome[] {
  return (outcomes ?? []).map((item) => {
    if (item.slot === 'fail') {
      return {
        slot: 'fail' as const,
        error: {
          statusCode: item.status_code ?? 0,
          message: item.message ?? '',
          occurredAt: item.occurred_at ?? '',
          model: item.model,
          endpoint: item.endpoint
        }
      }
    }
    return { slot: 'ok' as const }
  })
}

export function toHealthSnapshot(line: RequestHealthLineDTO): RequestHealthSnapshot {
  const cooldown = line.cooldown_until ? Date.parse(line.cooldown_until) : NaN
  return {
    outcomes: toRequestOutcomes(line.outcomes),
    current: line.current ?? 0,
    max: line.max ?? 1,
    cooldownUntil: Number.isFinite(cooldown) ? cooldown : undefined,
    rateLimited: line.rate_limited,
    overloaded: line.overloaded
  }
}

export function toAccountHealthRow(item: AccountRequestHealthDTO): AccountHealthRow {
  return {
    accountId: item.account_id,
    mode: item.mode === 'ip_group' ? 'ip_group' : 'single',
    ipGroupName: item.ip_group_name,
    lines: (item.lines ?? []).map((line) => ({
      ip: line.ip || '—',
      health: toHealthSnapshot(line)
    }))
  }
}

export function emptyAccountHealth(accountId: number): AccountHealthRow {
  return {
    accountId,
    mode: 'single',
    lines: [{ ip: '—', health: { outcomes: [], current: 0, max: 1 } }]
  }
}
