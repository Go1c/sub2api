import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useLotteryStore } from '../lottery'
import { lotteryAPI } from '@/api/lottery'
import { adminLotteryAPI } from '@/api/admin/lottery'
import type {
  CreateLotteryCampaignRequest,
  LotteryActiveCampaign,
  LotteryCampaign,
  LotteryDrawResult,
  PaginatedResponse,
} from '@/types'

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

const createdCampaign: LotteryCampaign = {
  ...campaignSummary,
  codes: [
    {
      id: 8,
      campaign_id: 1,
      code: 'CODE-1',
      created_at: '2026-05-21T00:00:00Z',
      updated_at: '2026-05-21T00:00:00Z',
    },
  ],
  draws: [],
}

const listResponse: PaginatedResponse<LotteryCampaign> = {
  items: [campaignSummary],
  total: 1,
  page: 1,
  page_size: 20,
  pages: 1,
}

describe('useLotteryStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(lotteryAPI.getActive).mockReset()
    vi.mocked(lotteryAPI.draw).mockReset()
    vi.mocked(adminLotteryAPI.listCampaigns).mockReset()
    vi.mocked(adminLotteryAPI.createCampaign).mockReset()
    vi.mocked(adminLotteryAPI.getCampaign).mockReset()
    vi.mocked(adminLotteryAPI.finishCampaign).mockReset()
  })

  it('fetches the active campaign from the backend', async () => {
    const store = useLotteryStore()
    vi.mocked(lotteryAPI.getActive).mockResolvedValue({ campaign: activeCampaign })

    await store.fetchActive()

    expect(lotteryAPI.getActive).toHaveBeenCalledTimes(1)
    expect(store.activeCampaign).toEqual(activeCampaign)
  })

  it('stores the draw result and clears the active campaign after drawing', async () => {
    const store = useLotteryStore()
    const drawResult: LotteryDrawResult = {
      won: true,
      index: 0,
      label: '奖品 1',
      message: '恭喜你中奖！兑换码已通过站内信发放，请前往站内信领取。',
      site_message_id: 12,
    }
    vi.mocked(lotteryAPI.getActive).mockResolvedValue({ campaign: activeCampaign })
    vi.mocked(lotteryAPI.draw).mockResolvedValue(drawResult)

    await store.fetchActive()
    const result = await store.draw(activeCampaign.id)

    expect(lotteryAPI.draw).toHaveBeenCalledWith(activeCampaign.id)
    expect(result).toEqual(drawResult)
    expect(store.lastResult).toEqual(drawResult)
    expect(store.activeCampaign).toBeNull()
  })

  it('creates a campaign through the admin API and reloads campaign history', async () => {
    const store = useLotteryStore()
    const input: CreateLotteryCampaignRequest = {
      name: '五月幸运转盘',
      subtitle: '',
      prize_count: 1,
      max_participants: 3,
      codes: ['CODE-1'],
    }
    vi.mocked(adminLotteryAPI.createCampaign).mockResolvedValue(createdCampaign)
    vi.mocked(adminLotteryAPI.listCampaigns).mockResolvedValue(listResponse)

    const created = await store.createCampaign(input)

    expect(adminLotteryAPI.createCampaign).toHaveBeenCalledWith(input)
    expect(adminLotteryAPI.listCampaigns).toHaveBeenCalledTimes(1)
    expect(created.id).toBe(createdCampaign.id)
    expect(store.campaigns).toEqual(listResponse.items)
  })
})
