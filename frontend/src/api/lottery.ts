import { apiClient } from './client'
import type { LotteryActiveResponse, LotteryDrawResult } from '@/types'

async function getActive(): Promise<LotteryActiveResponse> {
  const { data } = await apiClient.get<LotteryActiveResponse>('/lottery/active')
  return data
}

async function draw(campaignId: number): Promise<LotteryDrawResult> {
  const { data } = await apiClient.post<LotteryDrawResult>(`/lottery/${campaignId}/draw`)
  return data
}

export const lotteryAPI = {
  getActive,
  draw,
}

export default lotteryAPI
