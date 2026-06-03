import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import StripePaymentView from '../StripePaymentView.vue'

const routeState = vi.hoisted(() => ({
  query: {
    order_id: '1762',
    client_secret: 'pi_secret_test',
  } as Record<string, unknown>,
}))
const routerPush = vi.hoisted(() => vi.fn())
const fetchConfig = vi.hoisted(() => vi.fn())
const paymentConfig = vi.hoisted(() => ({
  value: {
    stripe_publishable_key: '',
  } as { stripe_publishable_key?: string },
}))
const getOrder = vi.hoisted(() => vi.fn())

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => routeState,
    useRouter: () => ({
      push: routerPush,
    }),
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/stores/payment', () => ({
  usePaymentStore: () => ({
    fetchConfig,
    config: paymentConfig.value,
  }),
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getOrder,
  },
}))

vi.mock('@/utils/device', () => ({
  isMobileDevice: () => false,
}))

describe('StripePaymentView', () => {
  beforeEach(() => {
    routeState.query = {
      order_id: '1762',
      client_secret: 'pi_secret_test',
    }
    routerPush.mockReset()
    fetchConfig.mockReset().mockResolvedValue(paymentConfig.value)
    paymentConfig.value = {
      stripe_publishable_key: '',
    }
    getOrder.mockReset().mockResolvedValue({
      data: {
        id: 1762,
        pay_amount: 10,
      },
    })
  })

  it('forces a fresh payment config fetch before loading Stripe', async () => {
    shallowMount(StripePaymentView, {
      global: {
        stubs: {
          AppLayout: {
            template: '<div><slot /></div>',
          },
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(fetchConfig).toHaveBeenCalledWith(true)
  })
})
