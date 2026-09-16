import { describe, expect, it } from 'vitest'
import {
  clampRequestHealthWindow,
  emptyAccountHealth,
  formatCountdown,
  resolveBadge,
  sliceWindow,
  toAccountHealthRow,
  type RequestOutcome
} from '../requestHealth'

describe('requestHealth helpers', () => {
  it('clamps window to allowed sizes', () => {
    expect(clampRequestHealthWindow(12)).toBe(12)
    expect(clampRequestHealthWindow(9)).toBe(12)
    expect(clampRequestHealthWindow(20)).toBe(20)
  })

  it('pads recent outcomes to the right with empty slots', () => {
    const outcomes: RequestOutcome[] = [{ slot: 'ok' }, { slot: 'fail', error: { statusCode: 500, message: 'boom', occurredAt: '2026-01-01T00:00:00Z' } }]
    const slots = sliceWindow(outcomes, 8)
    expect(slots).toHaveLength(8)
    expect(slots[0]).toEqual({ slot: 'ok' })
    expect(slots[1]?.slot).toBe('fail')
    expect(slots[7]?.slot).toBe('empty')
  })

  it('prefers cooling over executing', () => {
    expect(resolveBadge({
      outcomes: [],
      current: 2,
      max: 4,
      cooldownUntil: Date.now() + 5000
    }, Date.now())).toBe('cooling')
  })

  it('formats countdown as mm:ss', () => {
    expect(formatCountdown(Date.now() + 90_000, Date.now())).toBe('01:30')
  })

  it('maps dto rows and empty fallback', () => {
    const row = toAccountHealthRow({
      account_id: 3,
      mode: 'ip_group',
      ip_group_name: 'france',
      lines: [{
        ip: '1.2.*.*',
        outcomes: [{ slot: 'fail', status_code: 429, message: 'limited', occurred_at: '2026-01-01T00:00:00Z' }],
        current: 0,
        max: 10,
        rate_limited: true
      }]
    })
    expect(row.mode).toBe('ip_group')
    expect(row.lines[0]?.health.rateLimited).toBe(true)
    expect(row.lines[0]?.health.outcomes[0]?.error?.statusCode).toBe(429)
    expect(emptyAccountHealth(9).lines).toHaveLength(1)
  })
})
