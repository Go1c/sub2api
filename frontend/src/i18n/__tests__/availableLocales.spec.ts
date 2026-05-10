import { describe, expect, it } from 'vitest'

import {
  normalizeAvailableLocaleCodes,
  pickLocaleForAvailability,
} from '../index'

describe('frontend locale availability', () => {
  it('normalizes configured locale aliases and removes duplicates', () => {
    expect(normalizeAvailableLocaleCodes(['zh-TW', 'zh-CN', 'en', 'zh-Hant'])).toEqual([
      'zh-Hant',
      'zh',
      'en',
    ])
  })

  it('falls back to the first enabled locale when current locale is disabled', () => {
    expect(pickLocaleForAvailability('zh', ['zh-Hant'])).toBe('zh-Hant')
  })

  it('keeps the current locale when it remains enabled', () => {
    expect(pickLocaleForAvailability('zh-Hant', ['en', 'zh-Hant'])).toBe('zh-Hant')
  })
})
