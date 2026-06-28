import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SubscriptionsView from '../SubscriptionsView.vue'
import type { UserSubscription } from '@/types'

const getMySubscriptions = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())
const routerPush = vi.hoisted(() => vi.fn())

vi.mock('@/api/subscriptions', () => ({
  default: {
    getMySubscriptions,
  },
  subscriptionCreditErrorMessages: {},
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
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

describe('SubscriptionsView', () => {
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
        created_at: '2026-06-28T00:00:00Z',
      }),
    ])

    const wrapper = mount(SubscriptionsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
        },
      },
    })

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
})
