import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import LoginView from '../LoginView.vue'

const routerPush = vi.fn()
const locationReplace = vi.fn()
const showSuccess = vi.fn()
const showError = vi.fn()
const showWarning = vi.fn()
const login = vi.fn()
const login2FA = vi.fn()
const getPublicSettings = vi.fn()

let routeQuery: Record<string, string> = {}

vi.mock('vue-router', () => ({
  useRouter: () => ({
    currentRoute: {
      value: {
        query: routeQuery,
      },
    },
    push: routerPush,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/components/layout', () => ({
  AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' },
}))

vi.mock('@/components/auth/LinuxDoOAuthSection.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('@/components/auth/OidcOAuthSection.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('@/components/auth/WechatOAuthSection.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: { template: '<span />' },
}))

vi.mock('@/components/TurnstileWidget.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('@/components/auth/TotpLoginModal.vue', () => ({
  default: {
    template: '<button data-testid="totp-verify" @click="$emit(\'verify\', \'123456\')">verify</button>',
    methods: {
      setVerifying: vi.fn(),
      setError: vi.fn(),
    },
  },
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    login,
    login2FA,
    token: 'store-token',
  }),
  useAppStore: () => ({
    showSuccess,
    showError,
    showWarning,
  }),
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: (...args: any[]) => getPublicSettings(...args),
  isTotp2FARequired: (response: any) => response?.requires_2fa === true,
  isWeChatWebOAuthEnabled: () => false,
}))

function mountLoginView() {
  return mount(LoginView, {
    global: {
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })
}

async function submitLogin(wrapper: ReturnType<typeof mountLoginView>) {
  await wrapper.find('#email').setValue('test@example.com')
  await wrapper.find('#password').setValue('password123')
  await wrapper.find('form').trigger('submit')
  await flushPromises()
}

describe('LoginView external auth handoff', () => {
  const originalLocation = window.location

  beforeEach(() => {
    routeQuery = {}
    routerPush.mockReset()
    locationReplace.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
    showWarning.mockReset()
    login.mockReset()
    login2FA.mockReset()
    getPublicSettings.mockReset()
    getPublicSettings.mockResolvedValue({
      turnstile_enabled: false,
      turnstile_site_key: '',
      linuxdo_oauth_enabled: false,
      backend_mode_enabled: false,
      oidc_oauth_enabled: false,
      oidc_oauth_provider_name: 'OIDC',
      password_reset_enabled: false,
    })
    Object.defineProperty(window, 'location', {
      value: {
        replace: locationReplace,
      },
      writable: true,
      configurable: true,
    })
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
      configurable: true,
    })
    vi.restoreAllMocks()
  })

  it('replaces the page with the external return URL after password login succeeds with handoff', async () => {
    routeQuery = {
      handoff: '1',
      return_to: 'http://localhost:3000/#view=studio',
    }
    login.mockResolvedValue({
      access_token: 'login-token',
      token_type: 'Bearer',
      user: {
        id: 1,
        username: 'test',
        email: 'test@example.com',
        role: 'user',
        balance: 0,
        concurrency: 5,
        status: 'active',
        allowed_groups: null,
        created_at: '',
        updated_at: '',
      },
    })

    const wrapper = mountLoginView()
    await submitLogin(wrapper)

    expect(locationReplace).toHaveBeenCalledWith('http://localhost:3000/#view=studio&token=login-token')
    expect(routerPush).not.toHaveBeenCalled()
  })

  it('keeps normal dashboard redirect when password login has no handoff', async () => {
    login.mockResolvedValue({
      access_token: 'login-token',
      token_type: 'Bearer',
      user: {
        id: 1,
        username: 'test',
        email: 'test@example.com',
        role: 'user',
        balance: 0,
        concurrency: 5,
        status: 'active',
        allowed_groups: null,
        created_at: '',
        updated_at: '',
      },
    })

    const wrapper = mountLoginView()
    await submitLogin(wrapper)

    expect(routerPush).toHaveBeenCalledWith('/dashboard')
    expect(locationReplace).not.toHaveBeenCalled()
  })

  it('does not send a token externally when return_to is not allowed', async () => {
    routeQuery = {
      handoff: '1',
      return_to: 'https://evil.example/',
    }
    login.mockResolvedValue({
      access_token: 'login-token',
      token_type: 'Bearer',
      user: {
        id: 1,
        username: 'test',
        email: 'test@example.com',
        role: 'user',
        balance: 0,
        concurrency: 5,
        status: 'active',
        allowed_groups: null,
        created_at: '',
        updated_at: '',
      },
    })

    const wrapper = mountLoginView()
    await submitLogin(wrapper)

    expect(locationReplace).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('auth.externalHandoffInvalid')
    expect(routerPush).toHaveBeenCalledWith('/dashboard')
  })

  it('replaces the page with the external return URL after 2FA succeeds with handoff', async () => {
    routeQuery = {
      handoff: '1',
      return_to: 'http://localhost:3000/',
    }
    login.mockResolvedValue({
      requires_2fa: true,
      temp_token: 'temp-token',
    })
    login2FA.mockResolvedValue({
      id: 1,
      username: 'test',
      email: 'test@example.com',
      role: 'user',
      balance: 0,
      concurrency: 5,
      status: 'active',
      allowed_groups: null,
      created_at: '',
      updated_at: '',
    })

    const wrapper = mountLoginView()
    await submitLogin(wrapper)
    await wrapper.find('[data-testid="totp-verify"]').trigger('click')
    await flushPromises()

    expect(locationReplace).toHaveBeenCalledWith('http://localhost:3000/#token=store-token')
    expect(routerPush).not.toHaveBeenCalled()
  })
})
