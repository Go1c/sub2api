import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SubscriptionsView from '../SubscriptionsView.vue'
import type { UserSubscription } from '@/types'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'

const getMySubscriptions = vi.hoisted(() => vi.fn())
const resetWeeklyLimit = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())
const showSuccess = vi.hoisted(() => vi.fn())
const routerPush = vi.hoisted(() => vi.fn())

vi.mock('@/api/subscriptions', () => ({
  default: {
    getMySubscriptions,
    resetWeeklyLimit,
  },
  resetWeeklyLimit,
  subscriptionCreditErrorMessages: {
    SUBSCRIPTION_WEEKLY_LIMIT_RESET_EXHAUSTED: 'Weekly limit reset already used this period.',
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerPush,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (!params) return key
        return `${key}:${JSON.stringify(params)}`
      },
    }),
  }
})

function subscription(overrides: Partial<UserSubscription>): UserSubscription {
  return {
    id: 1,
    user_id: 9,
    group_id: null,
    plan_id: null,
    status: 'active',
    is_usable: true,
    quota_limit_usd: 100,
    quota_used_usd: 0,
    quota_remaining_usd: 100,
    daily_limit_usd: null,
    daily_usage_usd: 0,
    weekly_limit_usd: null,
    weekly_usage_usd: 0,
    monthly_usage_usd: 0,
    scope_type: 'all_available_groups',
    scope_config: {},
    daily_window_start: null,
    weekly_window_start: null,
    monthly_window_start: null,
    created_at: '2026-06-27T00:00:00Z',
    updated_at: '2026-06-27T00:00:00Z',
    expires_at: '2099-01-01T00:00:00Z',
    ...overrides,
  }
}

function mountView() {
  return mount(SubscriptionsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: true,
        BaseDialog: {
          props: ['show', 'title'],
          template: '<div v-if="show" data-testid="base-dialog"><slot /><slot name="footer" /></div>',
        },
      },
    },
  })
}

describe('SubscriptionsView', () => {
  beforeEach(() => {
    getMySubscriptions.mockReset()
    resetWeeklyLimit.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    routerPush.mockReset()
  })

  it('prioritizes usable subscriptions, orders them by earliest purchase, and shows full usage details on every usable subscription', async () => {
    getMySubscriptions.mockResolvedValueOnce([
      subscription({
        id: 0,
        plan_id: 100,
        plan_name: '已耗尽套餐',
        is_usable: false,
        quota_limit_usd: 100,
        quota_used_usd: 100,
        quota_remaining_usd: 0,
        exhausted_at: '2026-06-26T01:00:00Z',
        created_at: '2026-06-26T00:00:00Z',
      }),
      subscription({
        id: 1,
        plan_id: 101,
        plan_name: '标准版',
        quota_limit_usd: 240,
        quota_used_usd: 4.15,
        quota_remaining_usd: 235.85,
        daily_limit_usd: 90,
        daily_usage_usd: 4.15,
        weekly_limit_usd: 90,
        weekly_usage_usd: 4.15,
        weekly_limit_reset_remaining: 1,
        created_at: '2026-06-27T00:00:00Z',
      }),
      subscription({
        id: 2,
        plan_id: 102,
        plan_name: '轻享版',
        quota_limit_usd: 93,
        quota_used_usd: 0,
        quota_remaining_usd: 93,
        daily_limit_usd: 35,
        daily_usage_usd: 1.25,
        weekly_limit_usd: 35,
        weekly_usage_usd: 2.5,
        weekly_limit_reset_remaining: 1,
        created_at: '2026-06-28T00:00:00Z',
      }),
    ])

    const wrapper = mountView()

    await flushPromises()

    const text = wrapper.text()

    expect(wrapper.findAll('h2').map(heading => heading.text())).toEqual(['标准版', '轻享版'])
    expect(text).toContain('已耗尽套餐')
    expect(text).toContain('userSubscriptions.exhaustedAwaitingExpiry')
    expect((text.match(/userSubscriptions\.currentlyUsable/g) || []).length).toBe(2)
    expect((text.match(/userSubscriptions\.totalCredit/g) || []).length).toBe(2)
    expect((text.match(/payment\.renewNow/g) || []).length).toBe(2)
    expect((text.match(/payment\.planCard\.scope/g) || []).length).toBe(2)
    expect(text).toContain('userSubscriptions.usageOf:{"used":"$1.25","limit":"$35"}')
    expect(text).toContain('userSubscriptions.usageOf:{"used":"$2.5","limit":"$35"}')
  })

  it('renders reset weekly limit button on usable cards with weekly limit, next to renew', async () => {
    getMySubscriptions.mockResolvedValueOnce([
      subscription({
        id: 10,
        plan_name: '有周限',
        is_usable: true,
        weekly_limit_usd: 50,
        weekly_limit_reset_remaining: 1,
      }),
      subscription({
        id: 11,
        plan_name: '无周限',
        is_usable: true,
        weekly_limit_usd: null,
      }),
    ])

    const wrapper = mountView()
    await flushPromises()

    const resetButtons = wrapper.findAll('[data-testid="reset-weekly-limit"]')
    expect(resetButtons).toHaveLength(1)
    expect(resetButtons[0].text()).toContain('userSubscriptions.resetWeeklyLimit')
    expect(resetButtons[0].classes().join(' ')).toMatch(/red/)
    expect(wrapper.text()).toContain('payment.renewNow')
  })

  it('does not render reset weekly limit on non-usable cards even with weekly limit', async () => {
    getMySubscriptions.mockResolvedValueOnce([
      subscription({
        id: 12,
        plan_name: '已耗尽有周限',
        is_usable: false,
        weekly_limit_usd: 50,
        weekly_limit_reset_remaining: 1,
        exhausted_at: '2026-06-26T01:00:00Z',
        created_at: '2026-06-26T00:00:00Z',
      }),
    ])

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.findAll('[data-testid="reset-weekly-limit"]')).toHaveLength(0)
  })

  it('disables reset weekly limit when remaining is 0 and does not open dialog', async () => {
    getMySubscriptions.mockResolvedValueOnce([
      subscription({
        id: 13,
        plan_name: '次数用尽',
        is_usable: true,
        weekly_limit_usd: 40,
        weekly_limit_reset_remaining: 0,
      }),
    ])

    const wrapper = mountView()
    await flushPromises()

    const button = wrapper.get('[data-testid="reset-weekly-limit"]')
    expect(button.attributes('disabled')).toBeDefined()
    await button.trigger('click')
    await flushPromises()

    expect(wrapper.findComponent(ConfirmDialog).props('show')).toBe(false)
    expect(resetWeeklyLimit).not.toHaveBeenCalled()
  })

  it('opens danger confirm dialog with remaining count and calls user reset API on confirm', async () => {
    getMySubscriptions
      .mockResolvedValueOnce([
        subscription({
          id: 14,
          plan_name: '可重置',
          is_usable: true,
          weekly_limit_usd: 60,
          weekly_usage_usd: 12,
          weekly_limit_reset_remaining: 1,
        }),
      ])
      .mockResolvedValueOnce([
        subscription({
          id: 14,
          plan_name: '可重置',
          is_usable: true,
          weekly_limit_usd: 60,
          weekly_usage_usd: 0,
          weekly_limit_reset_remaining: 0,
        }),
      ])
    resetWeeklyLimit.mockResolvedValueOnce({})

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="reset-weekly-limit"]').trigger('click')
    await flushPromises()

    const dialog = wrapper.findComponent(ConfirmDialog)
    expect(dialog.props('show')).toBe(true)
    expect(dialog.props('danger')).toBe(true)
    expect(dialog.props('message')).toContain('userSubscriptions.resetWeeklyLimitConfirm')
    expect(dialog.props('message')).toContain('"remaining":1')

    await dialog.vm.$emit('confirm')
    await flushPromises()

    expect(resetWeeklyLimit).toHaveBeenCalledTimes(1)
    expect(resetWeeklyLimit).toHaveBeenCalledWith(14)
    expect(showSuccess).toHaveBeenCalledWith('userSubscriptions.weeklyLimitResetSuccess')
    expect(getMySubscriptions).toHaveBeenCalledTimes(2)
    expect(dialog.props('show')).toBe(false)
  })
})
