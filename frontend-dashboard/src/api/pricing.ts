import { http } from './http'

export interface ModelPricingRow {
  model: string
  group: string
  multiplier: string
  inputPrice: number
  outputPrice: number
  officialInput: number
  officialOutput: number
  discount: string
  openClaw: boolean
}

export interface PublicPricing {
  currency: string
  unit: string
  rateNote: string
  rows: ModelPricingRow[]
}

/**
 * Source of truth for public-facing model pricing.
 *
 * The backend MUST expose this endpoint in production so the marketing
 * page and dashboard pricing reference the same data as the billing
 * engine (channel_model_pricing + groups.input_price/output_price).
 *
 * During development the mock layer intercepts this call — see
 * src/mock/fixtures.ts -> publicPricing. Keep mock + backend in sync.
 */
export async function fetchPublicPricing(): Promise<PublicPricing> {
  const res = await http.get('/pricing/public')
  return res.data?.data ?? res.data
}
