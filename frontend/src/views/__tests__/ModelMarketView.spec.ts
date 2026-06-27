import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import ModelMarketView from '../ModelMarketView.vue'

vi.mock('@/api/modelMarket', () => ({
  default: {
    getPublicModelMarket: vi.fn().mockResolvedValue({
      enabled: true,
      auto_sync: true,
      title: '模型广场',
      description: '当前模型',
      models: [
        {
          key: 'openai:gpt-5.4',
          name: 'gpt-5.4',
          platform: 'openai',
          billing_mode: 'token',
          pricing: {
            billing_mode: 'token',
            input_price: 0.000000275,
            output_price: 0.00000165,
            cache_write_price: null,
            cache_read_price: null,
            image_output_price: null,
            per_request_price: null,
            intervals: [],
          },
          groups: [
            {
              id: 1,
              name: 'openai-public',
              platform: 'openai',
              subscription_type: 'standard',
              rate_multiplier: 0.35,
              is_exclusive: false,
            },
          ],
          channels: ['OpenAI'],
          sort_order: 0,
        },
      ],
    }),
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    cachedPublicSettings: {
      site_name: 'LumioAPI',
      site_logo: '',
    },
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
    siteName: 'Sub2API',
    siteLogo: '',
  }),
  useAuthStore: () => ({
    checkAuth: vi.fn(),
    isAuthenticated: false,
    isAdmin: false,
  }),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: {
      value: 'zh-CN',
    },
  }),
}))

describe('ModelMarketView', () => {
  it('renders the public model market from the public model market endpoint', async () => {
    const wrapper = mount(ModelMarketView, {
      global: {
        stubs: {
          Icon: true,
          PlatformIcon: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('模型广场')
    expect(wrapper.text()).toContain('全部供应商')
    expect(wrapper.text()).toContain('OpenAI')
    expect(wrapper.text()).toContain('gpt-5.4')
    expect(wrapper.text()).toContain('openai-public')
  })
})
