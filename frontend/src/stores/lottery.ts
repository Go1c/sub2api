import { defineStore } from 'pinia'
import { ref } from 'vue'

import { lotteryAPI } from '@/api/lottery'
import {
  adminLotteryAPI,
  type AdminLotteryListParams,
} from '@/api/admin/lottery'
import type {
  CreateLotteryCampaignRequest,
  LotteryActiveCampaign,
  LotteryCampaign,
  LotteryCode,
  LotteryDrawResult,
} from '@/types'

const DEFAULT_ADMIN_PAGE_SIZE = 50

function upsertCampaignSummary(
  items: LotteryCampaign[],
  campaign: LotteryCampaign,
): LotteryCampaign[] {
  const next = items.slice()
  const index = next.findIndex((item) => item.id === campaign.id)
  if (index === -1) {
    next.unshift(campaign)
    return next
  }
  next[index] = {
    ...next[index],
    ...campaign,
  }
  return next
}

export const useLotteryStore = defineStore('lottery', () => {
  const activeCampaign = ref<LotteryActiveCampaign | null>(null)
  const loadingActive = ref(false)
  const drawing = ref(false)
  const lastResult = ref<LotteryDrawResult | null>(null)

  const campaigns = ref<LotteryCampaign[]>([])
  const loadingCampaigns = ref(false)
  const campaignDetails = ref<Record<number, LotteryCampaign>>({})

  function clearActive() {
    activeCampaign.value = null
  }

  function clearLastResult() {
    lastResult.value = null
  }

  async function fetchActive(): Promise<LotteryActiveCampaign | null> {
    loadingActive.value = true
    lastResult.value = null
    try {
      const response = await lotteryAPI.getActive()
      activeCampaign.value = response.campaign
      return response.campaign
    } finally {
      loadingActive.value = false
    }
  }

  async function draw(campaignId: number): Promise<LotteryDrawResult> {
    drawing.value = true
    lastResult.value = null
    try {
      const result = await lotteryAPI.draw(campaignId)
      lastResult.value = result
      activeCampaign.value = null
      return result
    } catch (error) {
      const code = String(
        (error as { reason?: string; code?: string | number })?.reason ??
          (error as { reason?: string; code?: string | number })?.code ??
          '',
      )
      if (code === 'LOTTERY_ALREADY_DRAWN' || code === 'LOTTERY_CAMPAIGN_CLOSED') {
        activeCampaign.value = null
      }
      throw error
    } finally {
      drawing.value = false
    }
  }

  async function loadCampaigns(
    params: AdminLotteryListParams = {},
  ): Promise<LotteryCampaign[]> {
    loadingCampaigns.value = true
    try {
      const response = await adminLotteryAPI.listCampaigns({
        page: params.page ?? 1,
        page_size: params.page_size ?? DEFAULT_ADMIN_PAGE_SIZE,
        sort_by: params.sort_by ?? 'created_at',
        sort_order: params.sort_order ?? 'desc',
      })
      campaigns.value = response.items
      return response.items
    } finally {
      loadingCampaigns.value = false
    }
  }

  function getCampaignDetail(id: number): LotteryCampaign | null {
    return campaignDetails.value[id] ?? campaigns.value.find((item) => item.id === id) ?? null
  }

  async function loadCampaign(id: number): Promise<LotteryCampaign> {
    const campaign = await adminLotteryAPI.getCampaign(id)
    campaignDetails.value = {
      ...campaignDetails.value,
      [id]: campaign,
    }
    campaigns.value = upsertCampaignSummary(campaigns.value, campaign)
    return campaign
  }

  async function createCampaign(
    input: CreateLotteryCampaignRequest,
  ): Promise<LotteryCampaign> {
    const campaign = await adminLotteryAPI.createCampaign(input)
    campaignDetails.value = {
      ...campaignDetails.value,
      [campaign.id]: campaign,
    }
    await loadCampaigns()
    return campaign
  }

  async function finishCampaign(id: number): Promise<LotteryCampaign> {
    const campaign = await adminLotteryAPI.finishCampaign(id)
    campaignDetails.value = {
      ...campaignDetails.value,
      [id]: campaign,
    }
    campaigns.value = upsertCampaignSummary(campaigns.value, campaign)
    if (activeCampaign.value?.id === id) {
      activeCampaign.value = null
    }
    return campaign
  }

  function unclaimedCodes(campaign: LotteryCampaign): LotteryCode[] {
    return (campaign.codes ?? []).filter((code) => !code.assigned_user_id)
  }

  return {
    activeCampaign,
    loadingActive,
    drawing,
    lastResult,
    campaigns,
    loadingCampaigns,
    campaignDetails,
    fetchActive,
    draw,
    clearActive,
    clearLastResult,
    loadCampaigns,
    getCampaignDetail,
    loadCampaign,
    createCampaign,
    finishCampaign,
    unclaimedCodes,
  }
})
