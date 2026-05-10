import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import RegisterView from '../RegisterView.vue'

const routerPush = vi.fn()
const showSuccess = vi.fn()
const showError = vi.fn()
const showWarning = vi.fn()
const register = vi.fn()
const getPublicSettings = vi.fn()
const validatePromoCode = vi.fn()
const validateInvitationCode = vi.fn()

let routeQuery: Record<string, string> = {}

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerPush
  }),
  useRoute: () => ({
    query: routeQuery
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: { value: 'zh' }
    })
  }
})

vi.mock('@/components/layout', () => ({
  AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' }
}))

vi.mock('@/components/auth/LinuxDoOAuthSection.vue', () => ({
  default: { template: '<div />' }
}))

vi.mock('@/components/auth/OidcOAuthSection.vue', () => ({
  default: { template: '<div />' }
}))

vi.mock('@/components/auth/WechatOAuthSection.vue', () => ({
  default: { template: '<div />' }
}))

vi.mock('@/components/auth/EmailOAuthButtons.vue', () => ({
  default: { template: '<div />' }
}))

vi.mock('@/components/auth/LoginAgreementPrompt.vue', () => ({
  default: { template: '<div />' }
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: { template: '<span />' }
}))

vi.mock('@/components/TurnstileWidget.vue', () => ({
  default: { template: '<div />' }
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    register
  }),
  useAppStore: () => ({
    showSuccess,
    showError,
    showWarning
  })
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: (...args: any[]) => getPublicSettings(...args),
  isWeChatWebOAuthEnabled: () => false,
  validatePromoCode: (...args: any[]) => validatePromoCode(...args),
  validateInvitationCode: (...args: any[]) => validateInvitationCode(...args)
}))

function defaultPublicSettings(overrides: Record<string, unknown> = {}) {
  return {
    registration_enabled: true,
    email_verify_enabled: false,
    promo_code_enabled: false,
    invitation_code_enabled: true,
    invitation_registration_mode: 'redeem_code',
    turnstile_enabled: false,
    turnstile_site_key: '',
    site_name: 'Sub2API',
    linuxdo_oauth_enabled: false,
    oidc_oauth_enabled: false,
    oidc_oauth_provider_name: 'OIDC',
    github_oauth_enabled: false,
    google_oauth_enabled: false,
    registration_email_suffix_whitelist: [],
    login_agreement_enabled: false,
    login_agreement_documents: [],
    ...overrides
  }
}

describe('RegisterView invitation registration modes', () => {
  beforeEach(() => {
    routeQuery = {}
    routerPush.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
    showWarning.mockReset()
    register.mockReset()
    getPublicSettings.mockReset()
    validatePromoCode.mockReset()
    validateInvitationCode.mockReset()
    validateInvitationCode.mockResolvedValue({ valid: true })
    localStorage.clear()
    sessionStorage.clear()
  })

  it('autofills the invitation field from an invite link when affiliate-link registration is enabled', async () => {
    routeQuery = { aff: 'RMSV7D76XM23' }
    getPublicSettings.mockResolvedValue(
      defaultPublicSettings({
        invitation_registration_mode: 'affiliate_link'
      })
    )

    const wrapper = mount(RegisterView, {
      global: {
        stubs: {
          RouterLink: { template: '<a><slot /></a>' },
          transition: false
        }
      }
    })

    await flushPromises()

    expect((wrapper.get('#invitation_code').element as HTMLInputElement).value).toBe('RMSV7D76XM23')
    expect(wrapper.text()).toContain('auth.invitationOnlyNotice')
    expect(validateInvitationCode).toHaveBeenCalledWith('RMSV7D76XM23')
  })

  it('keeps affiliate links out of the invitation field in redeem-code-only mode', async () => {
    routeQuery = { aff: 'RMSV7D76XM23' }
    getPublicSettings.mockResolvedValue(defaultPublicSettings())

    const wrapper = mount(RegisterView, {
      global: {
        stubs: {
          RouterLink: { template: '<a><slot /></a>' },
          transition: false
        }
      }
    })

    await flushPromises()

    expect((wrapper.get('#invitation_code').element as HTMLInputElement).value).toBe('')
    expect(wrapper.text()).toContain('auth.invitationOnlyNotice')
    expect(validateInvitationCode).not.toHaveBeenCalled()
  })
})
