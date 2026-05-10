import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useSiteMessageStore } from '@/stores/siteMessages'
import { siteMessagesAPI } from '@/api/siteMessages'

vi.mock('@/api/siteMessages', () => ({
  siteMessagesAPI: {
    getUnreadCount: vi.fn(),
  },
}))

describe('useSiteMessageStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(siteMessagesAPI.getUnreadCount).mockReset()
  })

  it('does not request unread count when site messages are disabled', async () => {
    const appStore = useAppStore()
    const authStore = useAuthStore()
    const store = useSiteMessageStore()

    authStore.user = { id: 1, role: 'user' } as any
    authStore.token = 'token'
    appStore.cachedPublicSettings = {
      registration_enabled: false,
      email_verify_enabled: false,
      force_email_on_third_party_signup: false,
      registration_email_suffix_whitelist: [],
      promo_code_enabled: false,
      password_reset_enabled: false,
      invitation_code_enabled: false,
      turnstile_enabled: false,
      turnstile_site_key: '',
      site_name: 'Sub2API',
      site_logo: '',
      site_subtitle: '',
      api_base_url: '',
      contact_info: '',
      contact_channels: [],
      doc_url: '',
      site_pages: [],
      home_content: '',
      hide_ccs_import_button: false,
      payment_enabled: false,
      risk_control_enabled: false,
      table_default_page_size: 20,
      table_page_size_options: [10, 20, 50, 100],
      custom_menu_items: [],
      custom_endpoints: [],
      frontend_locales: ['en', 'zh', 'zh-Hant'],
      linuxdo_oauth_enabled: false,
      oidc_oauth_enabled: false,
      oidc_oauth_provider_name: 'OIDC',
      wechat_oauth_enabled: false,
      github_oauth_enabled: false,
      google_oauth_enabled: false,
      backend_mode_enabled: false,
      version: '',
      balance_low_notify_enabled: false,
      account_quota_notify_enabled: false,
      balance_low_notify_threshold: 0,
      channel_monitor_enabled: true,
      channel_monitor_default_interval_seconds: 60,
      available_channels_enabled: false,
      affiliate_enabled: false,
      site_messages_enabled: false,
    }

    await store.refreshUnreadCount()

    expect(siteMessagesAPI.getUnreadCount).not.toHaveBeenCalled()
    expect(store.unreadCount).toBe(0)
    expect(store.hasUnread).toBe(false)
  })

  it('updates unread count when the feature is enabled', async () => {
    const appStore = useAppStore()
    const authStore = useAuthStore()
    const store = useSiteMessageStore()

    authStore.user = { id: 1, role: 'user' } as any
    authStore.token = 'token'
    appStore.cachedPublicSettings = {
      registration_enabled: false,
      email_verify_enabled: false,
      force_email_on_third_party_signup: false,
      registration_email_suffix_whitelist: [],
      promo_code_enabled: false,
      password_reset_enabled: false,
      invitation_code_enabled: false,
      turnstile_enabled: false,
      turnstile_site_key: '',
      site_name: 'Sub2API',
      site_logo: '',
      site_subtitle: '',
      api_base_url: '',
      contact_info: '',
      contact_channels: [],
      doc_url: '',
      site_pages: [],
      home_content: '',
      hide_ccs_import_button: false,
      payment_enabled: false,
      risk_control_enabled: false,
      table_default_page_size: 20,
      table_page_size_options: [10, 20, 50, 100],
      custom_menu_items: [],
      custom_endpoints: [],
      frontend_locales: ['en', 'zh', 'zh-Hant'],
      linuxdo_oauth_enabled: false,
      oidc_oauth_enabled: false,
      oidc_oauth_provider_name: 'OIDC',
      wechat_oauth_enabled: false,
      github_oauth_enabled: false,
      google_oauth_enabled: false,
      backend_mode_enabled: false,
      version: '',
      balance_low_notify_enabled: false,
      account_quota_notify_enabled: false,
      balance_low_notify_threshold: 0,
      channel_monitor_enabled: true,
      channel_monitor_default_interval_seconds: 60,
      available_channels_enabled: false,
      affiliate_enabled: false,
      site_messages_enabled: true,
    }
    vi.mocked(siteMessagesAPI.getUnreadCount).mockResolvedValue({ count: 3 })

    await store.refreshUnreadCount()

    expect(siteMessagesAPI.getUnreadCount).toHaveBeenCalledTimes(1)
    expect(store.unreadCount).toBe(3)
    expect(store.hasUnread).toBe(true)
  })
})
