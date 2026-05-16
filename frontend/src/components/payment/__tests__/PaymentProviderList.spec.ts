import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'

import PaymentProviderList from '@/components/payment/PaymentProviderList.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, fallback?: string) => fallback ?? key,
    }),
  }
})

const DraggableStub = defineComponent({
  props: {
    modelValue: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['update:modelValue', 'end'],
  setup(_, { slots }) {
    return () => h('div', { class: 'draggable-stub' }, slots.default?.())
  },
})

const mapayProvider = {
  id: 11,
  provider_key: 'mapay',
  name: 'Mapay',
  config: {},
  supported_types: ['alipay', 'wxpay'],
  enabled: false,
  payment_mode: 'qrcode',
  refund_enabled: false,
  allow_user_refund: false,
  limits: '',
  sort_order: 0,
}

function mountList(enabledPaymentTypes: string[]) {
  return mount(PaymentProviderList, {
    props: {
      providers: [mapayProvider],
      loading: false,
      canCreate: true,
      enabledPaymentTypes,
      allPaymentTypes: [
        { value: 'alipay', label: 'Alipay' },
        { value: 'wxpay', label: 'WeChat Pay' },
        { value: 'stripe', label: 'Stripe' },
      ],
      redirectLabel: 'Redirect',
    },
    global: {
      stubs: {
        Icon: true,
        VueDraggable: DraggableStub,
      },
    },
  })
}

describe('PaymentProviderList', () => {
  it('keeps Mapay providers usable when a supported visible payment method is enabled', () => {
    const wrapper = mountList(['alipay'])

    const disabledTitles = wrapper
      .findAll('[title]')
      .map(node => node.attributes('title') || '')
      .filter(title => title.includes('admin.settings.payment.typeDisabled'))

    expect(disabledTitles).toEqual([])
  })
})
