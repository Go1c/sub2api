import { beforeEach, describe, expect, it, vi } from 'vitest'; import { flushPromises, mount } from '@vue/test-utils'; import { createPinia, setActivePinia } from 'pinia'; import type { CheckinStatus } from '../types'
const messages: Record<string, string> = { 'checkin.action.exhausted': "Today's reward pool is exhausted", 'checkin.action.ineligible': 'Spend threshold not met', 'checkin.today.awarded': 'Today you received ${amount}.', 'checkin.today.ineligible': 'Not eligible yet. Cumulative spend ${amount} to check in.', 'checkin.today.ineligibleCurrent': 'Current billed spend ${amount}' }
vi.mock('vue-i18n', async () => { const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n'); return { ...actual, useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => Object.entries(params ?? {}).reduce((text, [name, value]) => text.replace(`{${name}}`, String(value)), messages[key] ?? key) }) } })
const api = vi.hoisted(() => ({ getUserStatus: vi.fn(), checkIn: vi.fn() })); vi.mock('../api', () => ({ checkinAPI: api })); import CheckinView from '../CheckinView.vue'
const record = { id: 1, user_id: 4, user_email: 'user@example.com', username: 'user', business_date: '2026-08-19', checked_at: '2026-08-19T01:00:00Z', timezone: 'Asia/Shanghai', streak_days: 3, cycle_day: 3, base_reward: '0.4000', milestone_bonus: '0.0000', actual_reward: '0.0000', status: 'budget_exhausted' as const, balance_after: '10.0000' }
const status: CheckinStatus = { enabled: true, checked_in_today: true, total_checkins: 8, total_reward: '2.0000', current_streak: 3, cycle_day: 3, next_milestone: { day: 7, bonus: '1.0000', days_until: 4 }, balance: '10.0000', spend_eligible: true, spend_required: '0.0000', spend_total: '0.0000', today_record: record, recent_records: [] }
describe('CheckinView', () => {
  beforeEach(() => { setActivePinia(createPinia()); api.getUserStatus.mockReset(); api.checkIn.mockReset() })
  it('disables action when the pool is exhausted', async () => { api.getUserStatus.mockResolvedValue(status); const wrapper = mount(CheckinView, { global: { plugins: [createPinia()], stubs: { AppLayout: { template: '<main><slot /></main>' } } } }); await flushPromises(); const button = wrapper.get('[data-test="checkin-action"]'); expect(button.attributes('disabled')).toBeDefined(); expect(button.text()).toContain("Today's reward pool is exhausted"); expect(wrapper.get('[data-test="today-result"]').classes()).toEqual(expect.arrayContaining(['dark:bg-amber-950/40', 'dark:text-amber-300'])) })
  it('renders a zero-dollar awarded record as successful', async () => { api.getUserStatus.mockResolvedValue({ ...status, today_record: { ...record, base_reward: '0.0000', status: 'awarded' } }); const wrapper = mount(CheckinView, { global: { plugins: [createPinia()], stubs: { AppLayout: { template: '<main><slot /></main>' } } } }); await flushPromises(); expect(wrapper.get('[data-test="today-result"]').text()).toContain('$0.0000'); expect(wrapper.get('[data-test="today-result"]').classes()).toEqual(expect.arrayContaining(['dark:bg-emerald-950/40', 'dark:text-emerald-300'])) })
  it('applies dark tokens to the history table', async () => {
    api.getUserStatus.mockResolvedValue({ ...status, recent_records: [record] })
    const wrapper = mount(CheckinView, { global: { plugins: [createPinia()], stubs: { AppLayout: { template: '<main><slot /></main>' } } } })
    await flushPromises()
    expect(wrapper.get('table').classes()).toContain('dark:divide-dark-700')
    expect(wrapper.get('thead').classes()).toEqual(expect.arrayContaining(['dark:bg-dark-800', 'dark:text-gray-400']))
    expect(wrapper.get('tbody').classes()).toContain('dark:divide-dark-800')
  })
  it('disables action and shows the spend gate when historical spend is too low', async () => {
    api.getUserStatus.mockResolvedValue({ ...status, checked_in_today: false, today_record: null, spend_eligible: false, spend_required: '10.0000', spend_total: '3.2500' })
    const wrapper = mount(CheckinView, { global: { plugins: [createPinia()], stubs: { AppLayout: { template: '<main><slot /></main>' } } } })
    await flushPromises()
    const button = wrapper.get('[data-test="checkin-action"]')
    expect(button.attributes('disabled')).toBeDefined()
    expect(button.text()).toContain('Spend threshold not met')
    const gate = wrapper.get('[data-test="spend-gate"]')
    expect(gate.text()).toContain('Not eligible yet. Cumulative spend $10.0000 to check in.')
    expect(gate.text()).toContain('Current billed spend $3.2500')
  })
})
