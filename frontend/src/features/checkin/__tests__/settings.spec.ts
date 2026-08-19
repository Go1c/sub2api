import { describe, expect, it } from 'vitest'
import { hasDailyCapRisk, maximumSingleReward } from '../settings'
describe('check-in settings calculations', () => {
  it('adds the largest milestone bonus to maximum reward', () => expect(maximumSingleReward({ max_reward: '0.5000', milestones: [{ day: 7, bonus: '1.2500' }, { day: 30, bonus: '2.2500' }] })).toBe('2.7500'))
  it('warns only for a positive insufficient cap', () => {
    const settings = { max_reward: '0.5000', milestones: [{ day: 7, bonus: '1.0000' }] }
    expect(hasDailyCapRisk({ ...settings, daily_cap: '1.4999' })).toBe(true); expect(hasDailyCapRisk({ ...settings, daily_cap: '1.5000' })).toBe(false); expect(hasDailyCapRisk({ ...settings, daily_cap: '0' })).toBe(false)
  })
})
