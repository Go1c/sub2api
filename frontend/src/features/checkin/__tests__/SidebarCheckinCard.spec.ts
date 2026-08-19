import { describe, expect, it, vi } from 'vitest'; import { mount } from '@vue/test-utils'; import { createPinia, setActivePinia } from 'pinia'
vi.mock('vue-i18n', async () => { const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n'); return { ...actual, useI18n: () => ({ t: (key: string) => key }) } }); import SidebarCheckinCard from '../SidebarCheckinCard.vue'; import { useCheckinStore } from '../store'
const base = { checked_in_today: false, total_checkins: 0, total_reward: '0.0000', current_streak: 0, cycle_day: 0, next_milestone: null, balance: '0.0000', today_record: null, recent_records: [] }
const RouterLinkStub = { props: ['to'], template: '<a :href="to"><slot /></a>' }
describe('SidebarCheckinCard', () => {
  it('hides when disabled', () => { setActivePinia(createPinia()); useCheckinStore().status = { ...base, enabled: false }; const wrapper = mount(SidebarCheckinCard, { props: { collapsed: false }, global: { stubs: { RouterLink: true } } }); expect(wrapper.find('[data-test="sidebar-checkin"]').exists()).toBe(false) })
  it('links to check-in in collapsed mode', () => { setActivePinia(createPinia()); useCheckinStore().status = { ...base, enabled: true }; const wrapper = mount(SidebarCheckinCard, { props: { collapsed: true }, global: { stubs: { RouterLink: RouterLinkStub } } }); const link = wrapper.get('[data-test="sidebar-checkin"]'); expect(link.attributes('href')).toBe('/checkin'); expect(link.attributes('title')).toBe('nav.checkin') })
  it('nudges when enabled and not checked in today', () => {
    setActivePinia(createPinia())
    useCheckinStore().status = { ...base, enabled: true, checked_in_today: false }
    const wrapper = mount(SidebarCheckinCard, { props: { collapsed: false }, global: { stubs: { RouterLink: RouterLinkStub } } })
    const nudge = wrapper.get('[data-test="sidebar-checkin-nudge"]')
    expect(nudge.classes()).toContain('checkin-nudge')
  })
  it('does not nudge after checking in', () => {
    setActivePinia(createPinia())
    useCheckinStore().status = { ...base, enabled: true, checked_in_today: true }
    const wrapper = mount(SidebarCheckinCard, { props: { collapsed: false }, global: { stubs: { RouterLink: RouterLinkStub } } })
    expect(wrapper.find('[data-test="sidebar-checkin-nudge"]').exists()).toBe(false)
    expect(wrapper.find('.checkin-nudge').exists()).toBe(false)
  })
  it('does not nudge when the pool is already recorded as exhausted', () => {
    setActivePinia(createPinia())
    useCheckinStore().status = { ...base, enabled: true, checked_in_today: true, today_record: { status: 'budget_exhausted' } as never }
    const wrapper = mount(SidebarCheckinCard, { props: { collapsed: false }, global: { stubs: { RouterLink: RouterLinkStub } } })
    expect(wrapper.find('[data-test="sidebar-checkin-nudge"]').exists()).toBe(false)
  })
})
