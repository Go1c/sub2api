import { createI18n } from 'vue-i18n'
import zhCN from './zh-CN'
import enUS from './en-US'

const saved = typeof localStorage !== 'undefined' ? localStorage.getItem('locale') : null
const fallback = navigator.language.startsWith('zh') ? 'zh-CN' : 'en-US'

const i18n = createI18n({
  legacy: false,
  locale: saved || fallback,
  fallbackLocale: 'en-US',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS
  }
})

export default i18n
