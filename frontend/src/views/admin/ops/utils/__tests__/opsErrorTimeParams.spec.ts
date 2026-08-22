import { describe, expect, it } from 'vitest'
import { buildOpsErrorTimeParams } from '../opsFormatters'

describe('buildOpsErrorTimeParams', () => {
  it('passes preset ranges through as time_range', () => {
    expect(buildOpsErrorTimeParams('1h')).toEqual({ time_range: '1h' })
    expect(buildOpsErrorTimeParams('24h')).toEqual({ time_range: '24h' })
  })

  it('uses start_time/end_time for a complete custom window', () => {
    expect(
      buildOpsErrorTimeParams('custom', '2026-08-21T00:00:00.000Z', '2026-08-21T01:00:00.000Z')
    ).toEqual({
      start_time: '2026-08-21T00:00:00.000Z',
      end_time: '2026-08-21T01:00:00.000Z'
    })
  })

  it('falls back to 1h when custom bounds are missing', () => {
    expect(buildOpsErrorTimeParams('custom')).toEqual({ time_range: '1h' })
    expect(buildOpsErrorTimeParams('custom', '2026-08-21T00:00:00.000Z', null)).toEqual({
      time_range: '1h'
    })
  })
})
