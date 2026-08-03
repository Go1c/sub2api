import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import PaymentQRDialog from '../PaymentQRDialog.vue'

const pollOrderStatus = vi.hoisted(() => vi.fn())
const cancelOrder = vi.hoisted(() => vi.fn())
const verifyOrder = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())
const toCanvas = vi.hoisted(() => vi.fn())

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
    pollOrderStatus,
  }),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError,
  }),
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    cancelOrder,
    verifyOrder,
  },
}))

vi.mock('qrcode', () => ({
  default: {
    toCanvas,
  },
}))

const orderFactory = (status: string, paymentType = 'wxpay') => ({
  id: 42,
  user_id: 9,
  amount: 100,
  pay_amount: 100,
  fee_rate: 0,
  payment_type: paymentType,
  out_trade_no: 'sub2_202606250001',
  status,
  order_type: 'balance',
  created_at: '2026-06-25T10:00:00Z',
  expires_at: '2099-01-01T10:30:00Z',
  refund_amount: 0,
})

describe('PaymentQRDialog', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    pollOrderStatus.mockReset()
    cancelOrder.mockReset()
    verifyOrder.mockReset()
    showError.mockReset()
    toCanvas.mockReset().mockResolvedValue(undefined)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('recovers pending WeChat orders via verifyOrder during poll', async () => {
    pollOrderStatus.mockResolvedValue(orderFactory('PENDING'))
    verifyOrder.mockResolvedValue({
      data: orderFactory('COMPLETED'),
    })

    const wrapper = mount(PaymentQRDialog, {
      props: {
        show: false,
        orderId: 42,
        qrCode: 'weixin://wxpay/bizpayurl?pr=test',
        expiresAt: '2099-01-01T10:30:00Z',
        paymentType: 'wxpay',
      },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
          Icon: true,
        },
      },
    })

    await wrapper.setProps({ show: true })
    await flushPromises()
    await vi.advanceTimersByTimeAsync(3000)
    await flushPromises()

    expect(pollOrderStatus).toHaveBeenCalledWith(42)
    expect(verifyOrder).toHaveBeenCalledWith('sub2_202606250001')
    expect(wrapper.text()).toContain('payment.result.success')
    expect(wrapper.emitted('success')).toHaveLength(1)
  })

  it('does not call verifyOrder for non-WeChat pending orders', async () => {
    pollOrderStatus.mockResolvedValue(orderFactory('PENDING', 'alipay'))

    const wrapper = mount(PaymentQRDialog, {
      props: {
        show: false,
        orderId: 42,
        qrCode: 'https://qr.alipay.com/test',
        expiresAt: '2099-01-01T10:30:00Z',
        paymentType: 'alipay',
      },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
          Icon: true,
        },
      },
    })

    await wrapper.setProps({ show: true })
    await flushPromises()
    await vi.advanceTimersByTimeAsync(3000)
    await flushPromises()

    expect(pollOrderStatus).toHaveBeenCalledWith(42)
    expect(verifyOrder).not.toHaveBeenCalled()
  })
})
