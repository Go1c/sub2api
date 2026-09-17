import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import type { UserMonitorView } from '@/api/channelMonitor'
import MonitorCard from '../monitor/MonitorCard.vue'

vi.mock('@/utils/featureFlags', () => ({
  isChannelMonitorQuotaVisible: () => false,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key, te: () => true }),
  }
})

function makeItem(overrides: Partial<UserMonitorView> = {}): UserMonitorView {
  return {
    id: 1,
    name: 'iq-main',
    provider: 'openai',
    group_name: '',
    primary_model: 'gpt-4o-mini',
    primary_status: 'operational',
    primary_latency_ms: 800,
    primary_ping_latency_ms: 20,
    availability_7d: 100,
    extra_models: [],
    timeline: [],
    ...overrides,
  }
}

describe('MonitorCard IQ dual timeline', () => {
  it('renders a second IQ timeline only for check_mode=iq', () => {
    const wrapper = mount(MonitorCard, {
      props: {
        item: makeItem({
          check_mode: 'iq',
          iq_timeline: [{ status: 'iq_ok', latency_ms: 800, ping_latency_ms: 20, checked_at: '2026-09-17T00:00:00Z' }],
        }),
        window: '7d',
        availabilityValue: 100,
        countdownSeconds: 12,
      },
      global: {
        stubs: {
          MonitorMetricPair: true,
          MonitorAvailabilityRow: true,
        },
      },
    })
    const timelines = wrapper.findAllComponents({ name: 'MonitorTimeline' })
    expect(timelines).toHaveLength(2)
    expect(timelines[0].props('kind')).toBe('server')
    expect(timelines[1].props('kind')).toBe('iq')
    expect(timelines[0].props('label')).toBe('monitorCommon.timelineServer')
    expect(timelines[1].props('showCountdown')).toBe(false)
    expect(wrapper.text()).not.toContain('monitorCommon.iqStatus.iq_down')
  })

  it('keeps a single timeline for probe monitors', () => {
    const wrapper = mount(MonitorCard, {
      props: {
        item: makeItem({ check_mode: 'probe' }),
        window: '7d',
        availabilityValue: 100,
        countdownSeconds: 12,
      },
      global: {
        stubs: {
          MonitorMetricPair: true,
          MonitorAvailabilityRow: true,
        },
      },
    })
    expect(wrapper.findAllComponents({ name: 'MonitorTimeline' })).toHaveLength(1)
  })
})
