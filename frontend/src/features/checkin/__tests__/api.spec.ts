import { beforeEach, describe, expect, it, vi } from 'vitest'
const client = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: client }))
import { checkinAPI } from '../api'

describe('checkinAPI', () => {
  beforeEach(() => Object.values(client).forEach(mock => mock.mockReset()))
  it('uses independent user routes', async () => {
    client.get.mockResolvedValue({ data: {} }); client.post.mockResolvedValue({ data: {} })
    await checkinAPI.getUserStatus(); await checkinAPI.checkIn()
    expect(client.get).toHaveBeenCalledWith('/user/checkin'); expect(client.post).toHaveBeenCalledWith('/user/checkin')
  })
  it('passes admin filters through the independent namespace', async () => {
    client.get.mockResolvedValue({ data: { items: [] } })
    const params = { page: 2, page_size: 50, user_id: 17, search: 'user@example.com', business_date: '2026-08-19', status: 'budget_exhausted' as const, sort_by: 'actual_reward' as const, sort_order: 'asc' as const }
    await checkinAPI.listAdminRecords(params)
    expect(client.get).toHaveBeenCalledWith('/admin/affiliates/checkins', { params })
  })
  it('loads admin payout stats for a period', async () => {
    client.get.mockResolvedValue({ data: { unique_users: 3, total_amount: '2.5000' } })
    await checkinAPI.getAdminStats({ period: 'week', search: 'qq.com', status: 'awarded' })
    expect(client.get).toHaveBeenCalledWith('/admin/affiliates/checkins/stats', { params: { period: 'week', search: 'qq.com', status: 'awarded' } })
  })
  it('updates only dedicated settings', async () => {
    const settings = { enabled: false, min_reward: '0.1000', max_reward: '0.5000', timezone: 'Asia/Shanghai', daily_cap: '0.0000', milestones: [] }
    client.get.mockResolvedValue({ data: settings }); client.put.mockResolvedValue({ data: settings })
    await checkinAPI.getSettings(); await checkinAPI.updateSettings(settings)
    expect(client.get).toHaveBeenCalledWith('/admin/affiliates/checkins/settings'); expect(client.put).toHaveBeenCalledWith('/admin/affiliates/checkins/settings', settings)
  })
})
