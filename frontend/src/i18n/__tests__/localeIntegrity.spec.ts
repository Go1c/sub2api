import { describe, expect, it } from 'vitest'
import en from '../locales/en'
import zh from '../locales/zh'
import zhHant from '../locales/zh-Hant'

const locales = {
  en,
  zh,
  'zh-Hant': zhHant,
} as const

function collectStrings(value: unknown, prefix = ''): Array<{ path: string; value: string }> {
  if (typeof value === 'string') {
    return [{ path: prefix, value }]
  }

  if (!value || typeof value !== 'object') {
    return []
  }

  return Object.entries(value).flatMap(([key, child]) => {
    const path = prefix ? `${prefix}.${key}` : key
    return collectStrings(child, path)
  })
}

describe('locale integrity', () => {
  it('escapes literal at signs so vue-i18n does not parse them as linked messages', () => {
    for (const [locale, messages] of Object.entries(locales)) {
      const rawAtSigns = collectStrings(messages).filter(({ value }) =>
        value.replaceAll("{'@'}", '').includes('@'),
      )

      expect(rawAtSigns, `${locale} raw @ paths: ${rawAtSigns.map(({ path }) => path).join(', ')}`).toEqual([])
    }
  })

  it('does not ship question-mark mojibake in Chinese user request monitor copy', () => {
    for (const [locale, messages] of [
      ['zh', zh],
      ['zh-Hant', zhHant],
    ] as const) {
      const strings = collectStrings(messages.admin.ops.userRequestMonitor)
      const mojibake = strings.filter(({ value }) => /\?{2,}/.test(value))

      expect(mojibake, `${locale} mojibake paths: ${mojibake.map(({ path }) => path).join(', ')}`).toEqual([])
    }
  })

  it('keeps the simplified Chinese exhausted-subscription prompt actionable', () => {
    expect(zh.userSubscriptions.exhaustedAwaitingExpiry).toBe(
      '{count} 个已耗尽订阅等待到期，您可以前往购买一个新的订阅',
    )
    expect(en.userSubscriptions.exhaustedAwaitingExpiry).toBe('{count} exhausted subscription(s) waiting to expire')
    expect(zhHant.userSubscriptions.exhaustedAwaitingExpiry).toBe('{count} 個已耗盡訂閱等待到期')
  })
})
