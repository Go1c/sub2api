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

  it('keeps the simplified Chinese no-active-subscription prompt pointing to purchase', () => {
    expect(zh.userSubscriptions.noActiveSubscriptionsDesc).toBe('您没有任何有效订阅.请前往充值页面订购')
  })

  it('labels check-in redeem history consistently across locales', () => {
    expect(en.redeem.balanceAddedCheckin).toBe('Balance Added (Check-in)')
    expect(zh.redeem.balanceAddedCheckin).toBe('余额充值（签到）')
    expect(zhHant.redeem.balanceAddedCheckin).toBe('餘額充值（簽到）')
    expect(en.admin.users.typeCheckinBalance).toBe('Balance (Check-in)')
    expect(zh.admin.users.typeCheckinBalance).toBe('余额（签到）')
    expect(zhHant.admin.users.typeCheckinBalance).toBe('餘額（簽到）')
  })
})
