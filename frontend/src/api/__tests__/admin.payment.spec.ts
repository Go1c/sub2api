import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    post,
  },
}))

import { adminPaymentAPI, type PlanLimitSyncPreview, type PlanLimitSyncResult } from '@/api/admin/payment'

type Assert<T extends true> = T
type IsExact<T, U> = (
  (<G>() => G extends T ? 1 : 2) extends (<G>() => G extends U ? 1 : 2)
    ? ((<G>() => G extends U ? 1 : 2) extends (<G>() => G extends T ? 1 : 2) ? true : false)
    : false
)

type ExpectedPlanLimitSyncPreview = {
  plan_id: number
  daily_limit_usd: number | null
  weekly_limit_usd: number | null
  matched_count: number
  changed_count: number
}

type ExpectedPlanLimitSyncResult = ExpectedPlanLimitSyncPreview & {
  updated_count: number
}

const previewContractExact: Assert<IsExact<PlanLimitSyncPreview, ExpectedPlanLimitSyncPreview>> = true
const resultContractExact: Assert<IsExact<PlanLimitSyncResult, ExpectedPlanLimitSyncResult>> = true

describe('admin payment plan limit sync api', () => {
  beforeEach(() => {
    post.mockReset()
    post.mockResolvedValue({ data: {} })
  })

  it('posts to preview endpoint for plan limit sync', async () => {
    await adminPaymentAPI.previewPlanLimitSync(42)

    expect(post).toHaveBeenCalledWith('/admin/payment/plans/42/sync-limits/preview')
  })

  it('posts to execute endpoint for plan limit sync', async () => {
    await adminPaymentAPI.syncPlanLimits(42)

    expect(post).toHaveBeenCalledWith('/admin/payment/plans/42/sync-limits')
  })

  it('keeps plan limit sync response types aligned with the backend contract', () => {
    expect(previewContractExact).toBe(true)
    expect(resultContractExact).toBe(true)
  })
})
