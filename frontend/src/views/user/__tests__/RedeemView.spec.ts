import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import RedeemView from '../RedeemView.vue'

const {
  redeem,
  getHistory,
  getPublicSettings,
  refreshUser,
  fetchActiveSubscriptions,
  showError,
  showSuccess,
  showWarning,
} = vi.hoisted(() => ({
  redeem: vi.fn(),
  getHistory: vi.fn(),
  getPublicSettings: vi.fn(),
  refreshUser: vi.fn(),
  fetchActiveSubscriptions: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
}))

vi.mock('@/api', () => ({
  redeemAPI: {
    redeem,
    getHistory,
  },
  authAPI: {
    getPublicSettings,
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: {
      balance: 0,
      concurrency: 0,
    },
    refreshUser,
  }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showWarning,
  }),
}))

vi.mock('@/stores/subscriptions', () => ({
  useSubscriptionStore: () => ({
    fetchActiveSubscriptions,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) =>
        key === 'redeem.failedToRedeem' ? 'Unable to redeem. Please try again.' : key,
    }),
  }
})

async function submitRedeem(error: unknown) {
  redeem.mockRejectedValueOnce(error)

  const wrapper = mount(RedeemView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: true,
      },
    },
  })

  await flushPromises()
  await wrapper.get('#code').setValue('TEST-CODE')
  await wrapper.get('form').trigger('submit')
  await flushPromises()

  return wrapper
}

describe('RedeemView error display', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getHistory.mockResolvedValue([])
    getPublicSettings.mockResolvedValue({ contact_info: '' })
    refreshUser.mockResolvedValue(undefined)
    fetchActiveSubscriptions.mockResolvedValue(undefined)
  })

  it.each([
    {
      name: 'HTTP 404',
      error: {
        status: 404,
        reason: 'REDEEM_CODE_NOT_FOUND',
        message: 'redeem code not found',
      },
      expected: 'redeem code not found',
    },
    {
      name: 'HTTP 409',
      error: {
        status: 409,
        reason: 'REDEEM_CODE_DATA_CONFLICT',
        message: 'redeem code data conflict, please contact support',
      },
      expected: 'redeem code data conflict, please contact support',
    },
    {
      name: 'HTTP 429',
      error: {
        status: 429,
        reason: 'too many failed attempts',
      },
      expected: 'too many failed attempts',
    },
    {
      name: 'network error',
      error: {
        status: 0,
        message: 'Network error. Please check your connection.',
      },
      expected: 'Network error. Please check your connection.',
    },
  ])('shows the same concrete message for $name in the page and toast', async ({ error, expected }) => {
    const wrapper = await submitRedeem(error)

    expect(wrapper.text()).toContain(expected)
    expect(showError).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith(expected)
  })

  it('falls back to the localized generic message for an unknown unreadable error', async () => {
    const wrapper = await submitRedeem({ message: '   ', reason: { code: 'UNKNOWN' } })

    expect(wrapper.text()).toContain('Unable to redeem. Please try again.')
    expect(wrapper.text()).not.toContain('[object Object]')
    expect(showError).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith('Unable to redeem. Please try again.')
  })
})
