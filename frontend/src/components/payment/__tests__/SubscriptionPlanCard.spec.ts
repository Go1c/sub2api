import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SubscriptionPlanCard from '../SubscriptionPlanCard.vue'
import type { SubscriptionPlan } from '@/types/payment'
import type { UserSubscription } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

function planFactory(overrides: Partial<SubscriptionPlan> = {}): SubscriptionPlan {
  return {
    id: 10,
    group_id: null,
    name: 'Pro',
    description: '',
    price: 12,
    validity_days: 30,
    validity_unit: 'day',
    features: [],
    for_sale: true,
    sort_order: 1,
    scope_type: 'all_available_groups',
    ...overrides,
  }
}

function subscriptionFactory(overrides: Partial<UserSubscription> = {}): UserSubscription {
  return {
    id: 99,
    user_id: 1,
    group_id: null,
    plan_id: 20,
    status: 'active',
    is_usable: true,
    daily_usage_usd: 0,
    weekly_usage_usd: 0,
    monthly_usage_usd: 0,
    daily_window_start: null,
    weekly_window_start: null,
    monthly_window_start: null,
    created_at: '2026-05-01T00:00:00Z',
    updated_at: '2026-05-01T00:00:00Z',
    expires_at: null,
    ...overrides,
  }
}

describe('SubscriptionPlanCard renewal state', () => {
  it('does not show renewal for an unrelated usable subscription', () => {
    const wrapper = mount(SubscriptionPlanCard, {
      props: {
        plan: planFactory({ id: 10 }),
        activeSubscriptions: [subscriptionFactory({ plan_id: 20 })],
      },
    })

    expect(wrapper.text()).toContain('payment.subscribeNow')
    expect(wrapper.text()).not.toContain('payment.renewNow')
  })

  it('shows renewal when a usable subscription belongs to the same plan', () => {
    const wrapper = mount(SubscriptionPlanCard, {
      props: {
        plan: planFactory({ id: 10 }),
        activeSubscriptions: [subscriptionFactory({ plan_id: 10 })],
      },
    })

    expect(wrapper.text()).toContain('payment.renewNow')
  })

  it('falls back to scope matching for legacy usable subscriptions without plan_id', () => {
    const wrapper = mount(SubscriptionPlanCard, {
      props: {
        plan: planFactory({ id: 10, scope_type: 'platforms' }),
        activeSubscriptions: [subscriptionFactory({ plan_id: null, scope_type: 'platforms' })],
      },
    })

    expect(wrapper.text()).toContain('payment.renewNow')
  })
})
