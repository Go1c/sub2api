import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UserOrdersView from '../UserOrdersView.vue'
import type { PaymentOrder } from '@/types/payment'

const getMyOrders = vi.hoisted(() => vi.fn())
const getRefundEligibleProviders = vi.hoisted(() => vi.fn())
const getMyWalletTransactions = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())
const showSuccess = vi.hoisted(() => vi.fn())
const routerPush = vi.hoisted(() => vi.fn())

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getMyOrders,
    getRefundEligibleProviders,
    cancelOrder: vi.fn(),
    requestRefund: vi.fn(),
  },
}))

vi.mock('@/api/user', () => ({
  userAPI: {
    getMyWalletTransactions,
  },
  getMyWalletTransactions,
}))

vi.mock('@/stores', () => ({
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
      t: (key: string) => key,
    }),
  }
})

const walletTxn = {
  txn_id: 'txn-wallet-hidden-xyz',
  client_id: 'client-uuid-hidden',
  client_name: 'CCHaven Control',
  amount: 19.9,
  balance_after: 8801.23,
  currency: 'CNY',
  purpose: 'cchaven.control.wallet_debit.purpose',
  ref: 'ref-hidden',
  created_at: '2026-08-19T07:49:08Z',
}

function paymentOrder(overrides: Partial<PaymentOrder> = {}): PaymentOrder {
  return {
    id: 41,
    user_id: 9,
    amount: 10,
    pay_amount: 10,
    fee_rate: 0,
    payment_type: 'alipay',
    out_trade_no: 'PAY-ORDER-41',
    status: 'COMPLETED',
    order_type: 'balance',
    created_at: '2026-08-18T00:00:00Z',
    expires_at: '2026-08-18T00:30:00Z',
    refund_amount: 0,
    ...overrides,
  }
}

function stubDesktopMatchMedia() {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: true,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
}

function mountView() {
  return mount(UserOrdersView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: true,
        Select: true,
        BaseDialog: {
          props: ['show', 'title'],
          template: '<div v-if="show" data-testid="base-dialog"><slot /><slot name="footer" /></div>',
        },
        OrderTable: {
          name: 'OrderTable',
          props: ['orders', 'loading'],
          template: '<div data-testid="order-table"></div>',
        },
      },
    },
  })
}

describe('UserOrdersView wallet debits', () => {
  beforeEach(() => {
    stubDesktopMatchMedia()
    getMyOrders.mockReset()
    getRefundEligibleProviders.mockReset()
    getMyWalletTransactions.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    routerPush.mockReset()

    getMyOrders.mockResolvedValue({
      data: { items: [paymentOrder()], total: 1, page: 1, page_size: 20, pages: 1 },
    })
    getRefundEligibleProviders.mockResolvedValue({ data: { provider_instance_ids: [] } })
    getMyWalletTransactions.mockResolvedValue({
      items: [walletTxn],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
  })

  it('loads payment orders and wallet debits together and keeps them in separate tables', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(getMyOrders).toHaveBeenCalledWith({
      page: 1,
      page_size: 20,
      status: undefined,
    })
    expect(getMyWalletTransactions).toHaveBeenCalledWith({
      page: 1,
      page_size: 20,
    })

    const text = wrapper.text()
    expect(text).toContain('payment.walletDebits.title')
    expect(text).toContain('CCHaven Control')
    expect(text).toContain('-$19.90')
    expect(text).toContain(new Date('2026-08-19T07:49:08Z').toLocaleString())
    expect(text).not.toContain('cchaven.control.wallet_debit.purpose')
    expect(text).not.toContain('txn-wallet-hidden-xyz')
    expect(text).not.toContain('8801.23')

    const orderTable = wrapper.findComponent({ name: 'OrderTable' })
    expect(orderTable.props('orders')).toEqual([expect.objectContaining({ id: 41, out_trade_no: 'PAY-ORDER-41' })])
    expect(orderTable.props('orders')).toHaveLength(1)
    expect(orderTable.props('orders')[0]).not.toEqual(expect.objectContaining({ client_name: 'CCHaven Control' }))
  })

  it('refreshes wallet debits together with orders', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(getMyWalletTransactions).toHaveBeenCalledTimes(1)

    await wrapper.get('button[title="common.refresh"]').trigger('click')
    await flushPromises()

    expect(getMyWalletTransactions).toHaveBeenCalledTimes(2)
    expect(getMyOrders).toHaveBeenCalledTimes(2)
  })
})
