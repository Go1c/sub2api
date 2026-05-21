import { apiClient } from '../client'
import type {
  CreateLotteryCampaignRequest,
  LotteryCampaign,
  PaginatedResponse,
} from '@/types'

export interface AdminLotteryListParams {
  page?: number
  page_size?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

async function listCampaigns(
  params: AdminLotteryListParams = {},
): Promise<PaginatedResponse<LotteryCampaign>> {
  const { data } = await apiClient.get<PaginatedResponse<LotteryCampaign>>(
    '/admin/lottery/campaigns',
    {
      params: {
        page: params.page ?? 1,
        page_size: params.page_size ?? 20,
        sort_by: params.sort_by ?? 'created_at',
        sort_order: params.sort_order ?? 'desc',
      },
    },
  )
  return data
}

async function createCampaign(
  request: CreateLotteryCampaignRequest,
): Promise<LotteryCampaign> {
  const { data } = await apiClient.post<LotteryCampaign>(
    '/admin/lottery/campaigns',
    request,
  )
  return data
}

async function getCampaign(id: number): Promise<LotteryCampaign> {
  const { data } = await apiClient.get<LotteryCampaign>(
    `/admin/lottery/campaigns/${id}`,
  )
  return data
}

async function finishCampaign(id: number): Promise<LotteryCampaign> {
  const { data } = await apiClient.post<LotteryCampaign>(
    `/admin/lottery/campaigns/${id}/finish`,
  )
  return data
}

export const adminLotteryAPI = {
  listCampaigns,
  createCampaign,
  getCampaign,
  finishCampaign,
}

export default adminLotteryAPI
