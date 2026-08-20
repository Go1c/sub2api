import { beforeEach, describe, expect, it, vi } from 'vitest'; import { flushPromises, mount } from '@vue/test-utils'
const api = vi.hoisted(() => ({ listAdminRecords: vi.fn(), getAdminStats: vi.fn() }))
vi.mock('../api', () => ({ checkinAPI: api }))
vi.mock('vue-i18n', async () => { const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n'); return { ...actual, useI18n: () => ({ t: (key: string) => key }) } })
import AdminCheckinView from '../AdminCheckinView.vue'
const Slot = { template: '<div><slot /><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' }
const emptyStats = {
  period: 'day' as const,
  timezone: 'Asia/Shanghai',
  from: '2026-08-20',
  to: '2026-08-20',
  unique_users: 0,
  checkin_count: 0,
  total_amount: '0.0000',
  avg_amount: '0.0000',
  p50_amount: '0.0000',
  p90_amount: '0.0000',
  max_amount: '0.0000',
}

describe('AdminCheckinView', () => {
  beforeEach(() => {
    api.listAdminRecords.mockReset().mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 1 })
    api.getAdminStats.mockReset().mockResolvedValue(emptyStats)
  })

  it('applies user date and payout filters', async () => {
    const wrapper = mount(AdminCheckinView, { global: { stubs: { AppLayout: Slot, TablePageLayout: Slot, DataTable: true, Pagination: true, Icon: true } } })
    await flushPromises()
    await wrapper.get('[data-test="user-id-filter"]').setValue('17')
    await wrapper.get('[data-test="date-filter"]').setValue('2026-08-19')
    await wrapper.get('[data-test="status-filter"]').setValue('budget_exhausted')
    await wrapper.get('[data-test="status-filter"]').trigger('change')
    await flushPromises()
    expect(api.listAdminRecords).toHaveBeenLastCalledWith(expect.objectContaining({ user_id: 17, business_date: '2026-08-19', status: 'budget_exhausted' }))
    expect(api.getAdminStats).toHaveBeenLastCalledWith(expect.objectContaining({ user_id: 17, status: 'budget_exhausted', period: 'day' }))
    expect(api.getAdminStats.mock.calls.at(-1)?.[0]).not.toHaveProperty('business_date')
  })

  it('loads period stats and shows user count plus payout distribution', async () => {
    api.getAdminStats.mockResolvedValue({
      ...emptyStats,
      period: 'month',
      from: '2026-08-01',
      to: '2026-08-20',
      unique_users: 3,
      checkin_count: 4,
      total_amount: '2.5000',
      avg_amount: '0.8333',
      p50_amount: '0.8000',
      p90_amount: '1.2000',
      max_amount: '1.5000',
    })
    const wrapper = mount(AdminCheckinView, { global: { stubs: { AppLayout: Slot, TablePageLayout: Slot, DataTable: true, Pagination: true, Icon: true } } })
    await flushPromises()
    await wrapper.get('[data-test="stats-period-month"]').trigger('click')
    await flushPromises()
    expect(api.getAdminStats).toHaveBeenLastCalledWith(expect.objectContaining({ period: 'month' }))
    expect(wrapper.get('[data-test="stats-unique-users"]').text()).toBe('3')
    expect(wrapper.get('[data-test="stats-checkins"]').text()).toBe('4')
    expect(wrapper.get('[data-test="stats-total"]').text()).toBe('$2.5000')
    expect(wrapper.get('[data-test="stats-avg"]').text()).toBe('$0.8333')
    expect(wrapper.get('[data-test="stats-p50"]').text()).toBe('$0.8000')
    expect(wrapper.get('[data-test="stats-p90"]').text()).toBe('$1.2000')
    expect(wrapper.get('[data-test="stats-max"]').text()).toBe('$1.5000')
  })
})
