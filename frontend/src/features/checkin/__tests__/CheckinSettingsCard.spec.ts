import { beforeEach, describe, expect, it, vi } from 'vitest'; import { flushPromises, mount } from '@vue/test-utils'
vi.mock('vue-i18n', async () => { const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n'); return { ...actual, useI18n: () => ({ t: (key: string) => key }) } })
const api = vi.hoisted(() => ({ getSettings: vi.fn(), updateSettings: vi.fn() })); vi.mock('../api', () => ({ checkinAPI: api })); import CheckinSettingsCard from '../CheckinSettingsCard.vue'
const settings = { enabled: false, min_reward: '0.1000', max_reward: '0.5000', timezone: 'Asia/Shanghai', daily_cap: '0.4000', milestones: [{ day: 7, bonus: '1.0000' }], maximum_single_reward: '1.5000', updated_at: '2026-08-19T00:00:00Z' }
describe('CheckinSettingsCard', () => {
  beforeEach(() => { api.getSettings.mockReset().mockResolvedValue(settings); api.updateSettings.mockReset() })
  it('shows live maximum and budget risk', async () => { const wrapper = mount(CheckinSettingsCard); await flushPromises(); expect(wrapper.get('[data-test="maximum-reward"]').text()).toContain('$1.5000'); expect(wrapper.find('[data-test="daily-cap-warning"]').exists()).toBe(true) })
  it('saves only dedicated payload', async () => { api.updateSettings.mockResolvedValue({ ...settings, enabled: true }); const wrapper = mount(CheckinSettingsCard); await flushPromises(); await wrapper.get('[data-test="enabled-toggle"]').trigger('click'); await wrapper.get('[data-test="save-settings"]').trigger('click'); await flushPromises(); expect(api.updateSettings).toHaveBeenCalledWith({ enabled: true, min_reward: '0.1000', max_reward: '0.5000', timezone: 'Asia/Shanghai', daily_cap: '0.4000', milestones: [{ day: 7, bonus: '1.0000' }] }) })
})
