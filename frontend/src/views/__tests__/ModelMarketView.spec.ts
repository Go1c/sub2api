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
        {
          key: 'anthropic:claude-sonnet-4-6',
          name: 'claude-sonnet-4-6',
          platform: 'anthropic',
          billing_mode: 'token',
          pricing: {
            billing_mode: 'token',
            input_price: 0.000003,
            output_price: 0.000015,
            cache_write_price: null,
            cache_read_price: null,
            image_output_price: null,
            per_request_price: null,
            intervals: [],
          },
          groups: [
            {
              id: 2,
              name: 'anthropic-public',
              platform: 'anthropic',
              subscription_type: 'standard',
              rate_multiplier: 1,
              is_exclusive: false,
            },
          ],
          channels: ['Anthropic'],
          sort_order: 0,
        },
        {
          key: 'openai:gpt-image-2',
          name: 'gpt-image-2',
          platform: 'openai',
          billing_mode: 'image',
          pricing: {
            billing_mode: 'image',
            input_price: null,
            output_price: null,
            cache_write_price: null,
            cache_read_price: null,
            image_output_price: null,
            per_request_price: null,
            intervals: [
              {
                min_tokens: 0,
                max_tokens: null,
                tier_label: '1K',
                input_price: null,
                output_price: null,
                cache_write_price: null,
                cache_read_price: null,
                per_request_price: 0.05,
              },
              {
                min_tokens: 0,
                max_tokens: null,
                tier_label: '4K',
                input_price: null,
                output_price: null,
                cache_write_price: null,
                cache_read_price: null,
                per_request_price: 0.15,
              },
            ],
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

const routerPush = vi.hoisted(() => vi.fn())

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerPush,
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
    expect(wrapper.text()).toContain('输入价格（官方价）')
    expect(wrapper.text()).toContain('折扣倍率')
    expect(wrapper.text()).toContain('0.35x')
    expect(wrapper.text()).toContain('充值倍率')
    expect(wrapper.text()).toContain('1积分 = 1美元')
    expect(wrapper.text()).toContain('订阅倍率')
    expect(wrapper.text()).toContain('×')
    expect(wrapper.text()).toContain('最低')
    expect(wrapper.text()).toContain('0.75x')
    expect(wrapper.text()).not.toContain('开通订阅后可用订阅额度调用此模型')
    expect(wrapper.text()).toContain('去订阅')
    expect(wrapper.text()).not.toContain('官方价 × 0.35')
    expect(wrapper.text()).not.toContain('充值单位：积分')
  })

  it('renders configured image tiers without a default image price', async () => {
    const wrapper = mount(ModelMarketView, {
      global: {
        stubs: {
          Icon: true,
          PlatformIcon: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('gpt-image-2')
    expect(wrapper.text()).toContain('1K')
    expect(wrapper.text()).toContain('$0.05')
    expect(wrapper.text()).toContain('4K')
    expect(wrapper.text()).toContain('$0.15')

    const imageArticle = wrapper.findAll('article').find((node) => node.text().includes('gpt-image-2'))
    expect(imageArticle).toBeDefined()
    expect(imageArticle?.text()).not.toContain('折扣倍率')
    expect(imageArticle?.text()).not.toContain('0.35x')
    expect(imageArticle?.text()).not.toContain('订阅倍率')
    expect(imageArticle?.text()).not.toContain('去订阅')
  })

  it('resets the provider filter when selecting a group filter', async () => {
    const wrapper = mount(ModelMarketView, {
      global: {
        stubs: {
          Icon: true,
          PlatformIcon: true,
        },
      },
    })

    await flushPromises()

    const openaiButton = wrapper.findAll('button').find((node) => node.text().includes('OpenAI'))
    expect(openaiButton).toBeDefined()
    await openaiButton?.trigger('click')

    expect(wrapper.findAll('article').length).toBeGreaterThanOrEqual(1)
    expect(wrapper.text()).toContain('gpt-5.4')

    const anthropicGroupButton = wrapper.findAll('button').find((node) => node.text().includes('anthropic-public'))
    expect(anthropicGroupButton).toBeDefined()
    await anthropicGroupButton?.trigger('click')

    expect(wrapper.findAll('article')).toHaveLength(1)
    expect(wrapper.text()).toContain('claude-sonnet-4-6')
  })

  it('sends guests to login with a subscription purchase redirect', async () => {
    routerPush.mockReset()
    const wrapper = mount(ModelMarketView, {
      global: {
        stubs: {
          Icon: true,
          PlatformIcon: true,
        },
      },
    })

    await flushPromises()

    const subscribeButton = wrapper.findAll('button').find((node) => node.text().includes('去订阅'))
    expect(subscribeButton).toBeDefined()
    await subscribeButton?.trigger('click')

    expect(routerPush).toHaveBeenCalledWith({
      path: '/login',
      query: { redirect: '/purchase?tab=subscription' },
    })
  })
})
