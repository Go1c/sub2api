import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'

import LotteryView from '../LotteryView.vue'
import { adminLotteryAPI } from '@/api/admin/lottery'
import type { LotteryCampaign, PaginatedResponse } from '@/types'

vi.mock('@/api/lottery', () => ({
  lotteryAPI: {
    getActive: vi.fn(),
    draw: vi.fn(),
  },
}))

vi.mock('@/api/admin/lottery', () => {
  const api = {
    listCampaigns: vi.fn(),
    createCampaign: vi.fn(),
    getCampaign: vi.fn(),
    finishCampaign: vi.fn(),
  }

  return {
    adminLotteryAPI: api,
    default: api,
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

const campaignSummary: LotteryCampaign = {
  id: 1,
  name: '五月幸运转盘',
  subtitle: '登录就有机会，转一转赢取兑换码',
  status: 'active',
  prize_count: 1,
  max_participants: 3,
  joined_count: 0,
  winner_count: 0,
  created_by: 9,
  created_at: '2026-05-21T00:00:00Z',
  updated_at: '2026-05-21T00:00:00Z',
}

const listResponse: PaginatedResponse<LotteryCampaign> = {
  items: [campaignSummary],
  total: 1,
  page: 1,
  page_size: 20,
  pages: 1,
}

function mountView() {
  return mount(LotteryView, {
    global: {
      plugins: [createPinia()],
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        LotteryDialog: true,
        Teleport: true,
      },
    },
  })
}

async function fillForm(wrapper: ReturnType<typeof mountView>) {
  await wrapper.get('[data-test="lottery-name"]').setValue('五月幸运转盘')
  await wrapper.get('[data-test="lottery-subtitle"]').setValue('')
  await wrapper.get('[data-test="lottery-prize-count"]').setValue('1')
  await wrapper.get('[data-test="lottery-max-participants"]').setValue('3')
  await wrapper.get('[data-test="lottery-codes"]').setValue('CODE-1')
}

describe('admin LotteryView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(adminLotteryAPI.listCampaigns).mockReset()
    vi.mocked(adminLotteryAPI.createCampaign).mockReset()
    vi.mocked(adminLotteryAPI.getCampaign).mockReset()
    vi.mocked(adminLotteryAPI.finishCampaign).mockReset()
    vi.mocked(adminLotteryAPI.listCampaigns).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 1,
    })
  })

  it('creates a campaign through the admin lottery API', async () => {
    vi.mocked(adminLotteryAPI.createCampaign).mockResolvedValue(campaignSummary)

    const wrapper = mountView()
    await flushPromises()
    await fillForm(wrapper)
    await wrapper.get('[data-test="lottery-submit"]').trigger('click')
    await flushPromises()

    expect(adminLotteryAPI.createCampaign).toHaveBeenCalledWith(
      expect.objectContaining({
        name: '五月幸运转盘',
        prize_count: 1,
        max_participants: 3,
        codes: ['CODE-1'],
      }),
    )
  })

  it('reloads campaign history after create', async () => {
    vi.mocked(adminLotteryAPI.createCampaign).mockResolvedValue(campaignSummary)
    vi.mocked(adminLotteryAPI.listCampaigns)
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        page: 1,
        page_size: 20,
        pages: 1,
      })
      .mockResolvedValueOnce(listResponse)

    const wrapper = mountView()
    await flushPromises()
    await fillForm(wrapper)
    await wrapper.get('[data-test="lottery-submit"]').trigger('click')
    await flushPromises()

    expect(adminLotteryAPI.listCampaigns).toHaveBeenCalledTimes(2)
  })

  it('shows a clear error when site messages are disabled', async () => {
    vi.mocked(adminLotteryAPI.createCampaign).mockRejectedValue({
      reason: 'LOTTERY_SITE_MESSAGES_DISABLED',
      message: 'site messages must be enabled before starting a lottery campaign',
    })

    const wrapper = mountView()
    await flushPromises()
    await fillForm(wrapper)
    await wrapper.get('[data-test="lottery-submit"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('site messages must be enabled before starting a lottery campaign')
  })
})
