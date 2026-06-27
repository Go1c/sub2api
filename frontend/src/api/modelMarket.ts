import { apiClient } from './client'
import type { BillingMode } from '@/constants/channel'

export interface ModelMarketPricingInterval {
  min_tokens: number
  max_tokens: number | null
  tier_label?: string
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_read_price: number | null
  per_request_price: number | null
}

export interface ModelMarketPricing {
  billing_mode: BillingMode
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_read_price: number | null
  image_output_price: number | null
  per_request_price: number | null
  intervals: ModelMarketPricingInterval[]
}

export interface ModelMarketGroup {
  id: number
  name: string
  platform: string
  subscription_type: string
  rate_multiplier: number
  is_exclusive: boolean
}

export interface ModelMarketModel {
  key: string
  name: string
  platform: string
  billing_mode: BillingMode | 'unknown'
  pricing: ModelMarketPricing | null
  groups: ModelMarketGroup[]
  channels: string[]
  sort_order: number
}

export interface ModelMarketConfig {
  enabled: boolean
  auto_sync: boolean
  title: string
  description: string
  selected_models: ModelMarketSelection[]
  custom_models: ModelMarketCustomModel[]
}

export interface ModelMarketSelection {
  key: string
  platform?: string
  model?: string
  enabled: boolean
  sort_order: number
  billing_mode?: BillingMode
  pricing?: ModelMarketPricing | null
}

export interface ModelMarketCustomModel {
  key: string
  platform: string
  model: string
  enabled: boolean
  sort_order: number
  billing_mode: BillingMode
  pricing: ModelMarketPricing | null
  groups: ModelMarketGroup[]
}

export interface PublicModelMarketResponse {
  enabled: boolean
  auto_sync: boolean
  title: string
  description: string
  models: ModelMarketModel[]
}

export interface AdminModelMarketResponse {
  config: ModelMarketConfig
  candidates: ModelMarketModel[]
  models: ModelMarketModel[]
}

export async function getPublicModelMarket(): Promise<PublicModelMarketResponse> {
  const { data } = await apiClient.get<PublicModelMarketResponse>('/model-market/public')
  return data
}

export const modelMarketAPI = { getPublicModelMarket }

export default modelMarketAPI
