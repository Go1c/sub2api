import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { defineComponent } from 'vue'

import LotteryPromptManager from '../LotteryPromptManager.vue'
import { useAuthStore } from '@/stores/auth'
import { lotteryAPI } from '@/api/lottery'
import type { LotteryActiveCampaign } from '@/types'

vi.mock('@/api/lottery', () => ({
  lotteryAPI: {
    getActive: vi.fn(),
    draw: vi.fn(),
  },
}))

const LotteryDialogStub = defineComponent({
  props: {
    open: {
      type: Boolean,
      default: false,
    },
    campaignTitle: {
      type: String,
      default: '',
    },
    drawFn: {
      type: Function,
      required: true,
    },
    promoText: {
      type: String,
      default: '',
    },
    promoImageUrl: {
      type: String,
      default: '',
    },
  },
  template: `
    <div v-if="open" data-test="lottery-dialog">
      {{ campaignTitle }}
      <span data-test="promo-image-url">{{ promoImageUrl }}</span>
      <span data-test="promo-text">{{ promoText }}</span>
      <button data-test="draw" @click="drawFn()">draw</button>
      <button data-test="close" @click="$emit('close')">close</button>
    </div>
  `,
})

const activeCampaign: LotteryActiveCampaign = {
  id: 1,
  name: '五月幸运转盘',
  subtitle: '登录就有机会，转一转赢取兑换码',
  prize_count: 1,
  max_participants: 3,
  joined_count: 0,
  segments: [
    { label: '奖品 1', is_prize: true },
    { label: '谢谢参与', is_prize: false },
  ],
}

const mountedWrappers: VueWrapper[] = []

function mountPromptManager(pinia = createPinia()) {
  const wrapper = mount(LotteryPromptManager, {
    global: {
      plugins: [pinia],
      stubs: {
        LotteryDialog: LotteryDialogStub,
      },
    },
  })
  mountedWrappers.push(wrapper)
  return wrapper
}

function loginUser() {
  const authStore = useAuthStore()
  authStore.token = 'token'
  authStore.user = {
    id: 7,
    username: 'alice',
    email: 'alice@example.com',
    role: 'user',
    balance: 0,
    concurrency: 1,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: false,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-05-21T00:00:00Z',
    updated_at: '2026-05-21T00:00:00Z',
  } as any
}

function logoutUser() {
  const authStore = useAuthStore()
  authStore.token = null
  authStore.user = null
}

describe('LotteryPromptManager', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    sessionStorage.clear()
    vi.mocked(lotteryAPI.getActive).mockReset()
    vi.mocked(lotteryAPI.draw).mockReset()
  })

  afterEach(() => {
    while (mountedWrappers.length > 0) {
      mountedWrappers.pop()?.unmount()
    }
  })

  it('fetches the active campaign after login', async () => {
    vi.mocked(lotteryAPI.getActive).mockResolvedValue({ campaign: activeCampaign })
    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mountPromptManager(pinia)

    loginUser()
    await flushPromises()

    expect(lotteryAPI.getActive).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[data-test="lottery-dialog"]').exists()).toBe(true)
  })

  it('passes WeChat promo assets from the active campaign into the dialog', async () => {
    vi.mocked(lotteryAPI.getActive).mockResolvedValue({
      campaign: {
        ...activeCampaign,
        promo_text: '关注公众号领福利',
        promo_image_url: 'https://cdn.example.com/qr.png',
      },
    })
    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mountPromptManager(pinia)

    loginUser()
    await flushPromises()

    expect(wrapper.get('[data-test="promo-text"]').text()).toBe('关注公众号领福利')
    expect(wrapper.get('[data-test="promo-image-url"]').text()).toBe('https://cdn.example.com/qr.png')
  })

  it('does not reopen during the same login session after dismissal without draw', async () => {
    vi.mocked(lotteryAPI.getActive).mockResolvedValue({ campaign: activeCampaign })
    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mountPromptManager(pinia)

    loginUser()
    await flushPromises()
    await wrapper.get('[data-test="close"]').trigger('click')
    await flushPromises()

    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      value: 'visible',
    })
    document.dispatchEvent(new Event('visibilitychange'))
    await flushPromises()

    expect(wrapper.find('[data-test="lottery-dialog"]').exists()).toBe(false)
  })

  it('opens again on the next login when the previous dismissal had no draw', async () => {
    vi.mocked(lotteryAPI.getActive).mockResolvedValue({ campaign: activeCampaign })
    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mountPromptManager(pinia)

    loginUser()
    await flushPromises()
    await wrapper.get('[data-test="close"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-test="lottery-dialog"]').exists()).toBe(false)

    logoutUser()
    await flushPromises()
    loginUser()
    await flushPromises()

    expect(lotteryAPI.getActive).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-test="lottery-dialog"]').exists()).toBe(true)
  })

  it('refetches and opens when an active campaign appears for an already logged-in user', async () => {
    vi.mocked(lotteryAPI.getActive)
      .mockResolvedValueOnce({ campaign: null })
      .mockResolvedValueOnce({ campaign: activeCampaign })
    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mountPromptManager(pinia)

    loginUser()
    await flushPromises()
    expect(wrapper.find('[data-test="lottery-dialog"]').exists()).toBe(false)

    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      value: 'visible',
    })
    document.dispatchEvent(new Event('visibilitychange'))
    await flushPromises()

    expect(lotteryAPI.getActive).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-test="lottery-dialog"]').exists()).toBe(true)
  })

  it('keeps the dialog open after draw resolves so the result can be shown', async () => {
    vi.mocked(lotteryAPI.getActive).mockResolvedValue({ campaign: activeCampaign })
    vi.mocked(lotteryAPI.draw).mockResolvedValue({
      won: false,
      index: 1,
      label: '谢谢参与',
      message: '很遗憾，这次没有中奖。',
    })
    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mountPromptManager(pinia)

    loginUser()
    await flushPromises()
    await wrapper.get('[data-test="draw"]').trigger('click')
    await flushPromises()

    expect(lotteryAPI.draw).toHaveBeenCalledWith(activeCampaign.id)
    expect(wrapper.find('[data-test="lottery-dialog"]').exists()).toBe(true)
  })
})
