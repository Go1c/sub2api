import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    post
  }
}))

import { resetWeeklyLimit } from '@/api/subscriptions'

describe('user subscriptions api', () => {
  beforeEach(() => {
    post.mockReset()
    post.mockResolvedValue({ data: {} })
  })

  it('resets weekly limit via the user-facing endpoint (not admin)', async () => {
    await resetWeeklyLimit(42)

    expect(post).toHaveBeenCalledTimes(1)
    expect(post).toHaveBeenCalledWith('/subscriptions/42/reset-weekly-limit')
  })
})
