import { createI18n } from 'vue-i18n'
import zhCN from './zh-CN'
import zhHant from './zh-Hant'
import enUS from './en-US'

export const supportedLocales = ['zh-CN', 'zh-Hant', 'en-US'] as const
export type SupportedLocale = (typeof supportedLocales)[number]

const saved = readSavedLocale()
const fallback = detectFallbackLocale()

const i18n = createI18n({
  legacy: false,
  locale: saved || fallback,
  fallbackLocale: 'en-US',
  messages: {
    'zh-CN': zhCN,
    'zh-Hant': zhHant,
    'en-US': enUS
  }
})

export default i18n

export function nextLocale(current: string): SupportedLocale {
  const index = supportedLocales.indexOf(current as SupportedLocale)
  return supportedLocales[(index + 1) % supportedLocales.length]
}

function readSavedLocale(): SupportedLocale | null {
  if (typeof localStorage === 'undefined' || typeof localStorage.getItem !== 'function') {
    return null
  }
  const value = localStorage.getItem('locale')
  return supportedLocales.includes(value as SupportedLocale) ? (value as SupportedLocale) : null
}

function detectFallbackLocale(): SupportedLocale {
  const language = typeof navigator !== 'undefined' ? navigator.language : ''
  if (/^zh-(tw|hk|mo|hant)/i.test(language)) return 'zh-Hant'
  if (/^zh/i.test(language)) return 'zh-CN'
  return 'en-US'
}
