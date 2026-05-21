import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
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
  },
  template: `
    <div v-if="open" data-test="lottery-dialog">
      {{ campaignTitle }}
      <button data-test="draw" @click="drawFn()">draw</button>
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

describe('LotteryPromptManager', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    sessionStorage.clear()
    vi.mocked(lotteryAPI.getActive).mockReset()
    vi.mocked(lotteryAPI.draw).mockReset()
  })

  it('fetches the active campaign after login', async () => {
    vi.mocked(lotteryAPI.getActive).mockResolvedValue({ campaign: activeCampaign })
    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(LotteryPromptManager, {
      global: {
        plugins: [pinia],
        stubs: {
          LotteryDialog: LotteryDialogStub,
        },
      },
    })

    loginUser()
    await flushPromises()

    expect(lotteryAPI.getActive).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[data-test="lottery-dialog"]').exists()).toBe(true)
  })

  it('does not open when the campaign was dismissed in this session', async () => {
    sessionStorage.setItem('lottery_dismissed_v2', JSON.stringify([activeCampaign.id]))
    vi.mocked(lotteryAPI.getActive).mockResolvedValue({ campaign: activeCampaign })
    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(LotteryPromptManager, {
      global: {
        plugins: [pinia],
        stubs: {
          LotteryDialog: LotteryDialogStub,
        },
      },
    })

    loginUser()
    await flushPromises()

    expect(wrapper.find('[data-test="lottery-dialog"]').exists()).toBe(false)
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

    const wrapper = mount(LotteryPromptManager, {
      global: {
        plugins: [pinia],
        stubs: {
          LotteryDialog: LotteryDialogStub,
        },
      },
    })

    loginUser()
    await flushPromises()
    await wrapper.get('[data-test="draw"]').trigger('click')
    await flushPromises()

    expect(lotteryAPI.draw).toHaveBeenCalledWith(activeCampaign.id)
    expect(wrapper.find('[data-test="lottery-dialog"]').exists()).toBe(true)
  })
})
