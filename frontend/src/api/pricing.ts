import { apiClient } from './client'

export interface PublicPricingRow {
  model: string
  group: string
  multiplier: string
  inputPrice: number
  outputPrice: number
  officialInput: number
  officialOutput: number
  discount: string
  openClaw: boolean
  enabled: boolean
}

export interface PublicPricingConfig {
  currency: string
  unit: string
  rateNote: string
  rows: PublicPricingRow[]
}

export async function getPublicPricing(): Promise<PublicPricingConfig> {
  const { data } = await apiClient.get<PublicPricingConfig>('/pricing/public')
  return data
}
