import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useAppStore } from '@/stores/app'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

describe('FeatureFlags.siteMessages', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('is registered as an opt-in public setting', () => {
    const flag = (FeatureFlags as Record<string, any>).siteMessages

    expect(flag).toMatchObject({
      key: 'site_messages_enabled',
      mode: 'opt-in',
    })
  })

  it('is hidden until public settings explicitly enable it', () => {
    const appStore = useAppStore()
    const flag = (FeatureFlags as Record<string, any>).siteMessages

    expect(isFeatureFlagEnabled(flag)).toBe(false)

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

    expect(isFeatureFlagEnabled(flag)).toBe(true)
  })
})

describe('FeatureFlags.userSubscriptions', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('is registered as an opt-out public setting', () => {
    const flag = (FeatureFlags as Record<string, any>).userSubscriptions

    expect(flag).toMatchObject({
      key: 'user_subscriptions_visible',
      mode: 'opt-out',
    })
  })

  it('is hidden only when public settings explicitly disable it', () => {
    const appStore = useAppStore()
    const flag = (FeatureFlags as Record<string, any>).userSubscriptions

    expect(isFeatureFlagEnabled(flag)).toBe(true)

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
      payment_enabled: true,
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
      user_subscriptions_visible: false,
    }

    expect(isFeatureFlagEnabled(flag)).toBe(false)
  })
})
