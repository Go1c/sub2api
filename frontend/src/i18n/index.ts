import { reactive } from 'vue'
import { createI18n } from 'vue-i18n'

export type LocaleCode = 'en' | 'zh' | 'zh-Hant'

type LocaleMessages = Record<string, any>
type LocaleOption = { code: LocaleCode; name: string; flag: string; shortLabel: string }

const LOCALE_KEY = 'sub2api_locale'
const DEFAULT_LOCALE: LocaleCode = 'en'
const DEFAULT_AVAILABLE_LOCALES: LocaleCode[] = ['en', 'zh', 'zh-Hant']

const supportedLocales: LocaleOption[] = [
  { code: 'en', name: 'English', flag: '🇺🇸', shortLabel: 'EN' },
  { code: 'zh', name: '简体中文', flag: '🇨🇳', shortLabel: '简' },
  { code: 'zh-Hant', name: '繁體中文', flag: '🇹🇼', shortLabel: '繁' }
]

let enabledLocaleCodes = new Set<LocaleCode>(DEFAULT_AVAILABLE_LOCALES)

const localeLoaders: Record<LocaleCode, () => Promise<{ default: LocaleMessages }>> = {
  en: () => import('./locales/en'),
  zh: () => import('./locales/zh'),
  'zh-Hant': () => import('./locales/zh-Hant')
}

function normalizeLocaleCode(value: string): LocaleCode | null {
  const normalized = value.trim().toLowerCase()
  if (normalized === 'en' || normalized === 'en-us' || normalized === 'en-gb') {
    return 'en'
  }
  if (normalized === 'zh' || normalized === 'zh-cn' || normalized === 'zh-hans' || normalized === 'zh-sg') {
    return 'zh'
  }
  if (
    normalized === 'zh-hant' ||
    normalized === 'zh-tw' ||
    normalized === 'zh-hk' ||
    normalized === 'zh-mo'
  ) {
    return 'zh-Hant'
  }
  return null
}

export function normalizeAvailableLocaleCodes(values: readonly string[] | null | undefined): LocaleCode[] {
  const result: LocaleCode[] = []
  const source = Array.isArray(values) && values.length > 0 ? values : DEFAULT_AVAILABLE_LOCALES

  for (const value of source) {
    const code = normalizeLocaleCode(value)
    if (code && !result.includes(code)) {
      result.push(code)
    }
  }

  return result.length > 0 ? result : [...DEFAULT_AVAILABLE_LOCALES]
}

function browserPreferredLocale(): LocaleCode {
  const browserLang = navigator.language.toLowerCase()
  if (browserLang === 'zh-tw' || browserLang === 'zh-hk' || browserLang === 'zh-mo' || browserLang.startsWith('zh-hant')) {
    return 'zh-Hant'
  }
  if (browserLang.startsWith('zh')) {
    return 'zh'
  }
  return DEFAULT_LOCALE
}

export function pickLocaleForAvailability(current: string, available: readonly string[]): LocaleCode {
  const configured = normalizeAvailableLocaleCodes(available)
  const normalizedCurrent = normalizeLocaleCode(current)
  if (normalizedCurrent && configured.includes(normalizedCurrent)) {
    return normalizedCurrent
  }
  return configured[0] ?? DEFAULT_LOCALE
}

function getDefaultLocale(): LocaleCode {
  const saved = localStorage.getItem(LOCALE_KEY)
  if (saved) {
    const normalizedSaved = normalizeLocaleCode(saved)
    if (normalizedSaved && enabledLocaleCodes.has(normalizedSaved)) {
      return normalizedSaved
    }
  }

  const browserLocale = browserPreferredLocale()
  if (enabledLocaleCodes.has(browserLocale)) {
    return browserLocale
  }

  return [...enabledLocaleCodes][0] ?? DEFAULT_LOCALE
}

export const i18n = createI18n({
  legacy: false,
  locale: getDefaultLocale(),
  fallbackLocale: DEFAULT_LOCALE,
  messages: {},
  // 禁用 HTML 消息警告 - 引导步骤使用富文本内容（driver.js 支持 HTML）
  // 这些内容是内部定义的，不存在 XSS 风险
  warnHtmlMessage: false
})

const loadedLocales = new Set<LocaleCode>()

export async function loadLocaleMessages(locale: LocaleCode): Promise<void> {
  if (loadedLocales.has(locale)) {
    return
  }

  const loader = localeLoaders[locale]
  const module = await loader()
  i18n.global.setLocaleMessage(locale, module.default)
  loadedLocales.add(locale)
}

export async function initI18n(): Promise<void> {
  const current = getLocale()
  await loadLocaleMessages(current)
  document.documentElement.setAttribute('lang', current)
}

export async function setLocale(locale: string): Promise<void> {
  const normalized = normalizeLocaleCode(locale)
  if (!normalized || !enabledLocaleCodes.has(normalized)) {
    return
  }

  await loadLocaleMessages(normalized)
  i18n.global.locale.value = normalized
  localStorage.setItem(LOCALE_KEY, normalized)
  document.documentElement.setAttribute('lang', normalized)

  // 同步更新浏览器页签标题，使其跟随语言切换
  const { resolveDocumentTitle } = await import('@/router/title')
  const { default: router } = await import('@/router')
  const { useAppStore } = await import('@/stores/app')
  const route = router.currentRoute.value
  const appStore = useAppStore()
  document.title = resolveDocumentTitle(route.meta.title, appStore.siteName, route.meta.titleKey as string)
}

export function getLocale(): LocaleCode {
  const current = i18n.global.locale.value
  const normalized = normalizeLocaleCode(current)
  return normalized && enabledLocaleCodes.has(normalized) ? normalized : getDefaultLocale()
}

export const availableLocales = reactive<LocaleOption[]>([...supportedLocales])

export function configureAvailableLocales(values: readonly string[] | null | undefined): void {
  const normalized = normalizeAvailableLocaleCodes(values)
  enabledLocaleCodes = new Set(normalized)
  availableLocales.splice(
    0,
    availableLocales.length,
    ...supportedLocales.filter((locale) => enabledLocaleCodes.has(locale.code))
  )

  const selected = pickLocaleForAvailability(i18n.global.locale.value, normalized)
  if (selected !== i18n.global.locale.value) {
    i18n.global.locale.value = selected
    localStorage.setItem(LOCALE_KEY, selected)
    document.documentElement.setAttribute('lang', selected)
    void loadLocaleMessages(selected)
  }
}

export default i18n
