import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    post
  }
}))

import { resetWeeklyLimit } from '@/api/admin/subscriptions'

describe('admin subscriptions api', () => {
  beforeEach(() => {
    post.mockReset()
    post.mockResolvedValue({ data: {} })
  })

  it('resets only the selected subscription weekly usage window', async () => {
    await resetWeeklyLimit(73)

    expect(post).toHaveBeenCalledTimes(1)
    expect(post).toHaveBeenCalledWith('/admin/subscriptions/73/reset-quota', {
      daily: false,
      weekly: true,
      monthly: false
    })
  })
})
