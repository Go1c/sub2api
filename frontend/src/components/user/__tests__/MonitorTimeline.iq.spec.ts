import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import MonitorTimeline from '../monitor/MonitorTimeline.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key, te: () => true }),
  }
})

describe('MonitorTimeline IQ vs server', () => {
  it('shows timeouts in yellow with a timeout label, not IQ down or test error', () => {
    const wrapper = mount(MonitorTimeline, {
      props: {
        kind: 'iq', countdownSeconds: 10, length: 1,
        buckets: [{ status: 'test_timeout', latency_ms: 180000, checked_at: '2026-09-18T00:00:00Z' }],
      },
    })
    const bar = wrapper.find('.flex-1.min-w-\\[3px\\]')
    expect(bar.classes()).toContain('bg-amber-500')
    expect(bar.attributes('title')).toContain('monitorCommon.iqStatus.test_timeout')
  })

  it('server bars follow probe status and ignore iq_status', () => {
    const wrapper = mount(MonitorTimeline, {
      props: {
        kind: 'server',
        countdownSeconds: 10,
        length: 4,
        buckets: [{
          status: 'operational',
          iq_status: 'monitor_network',
          latency_ms: 800,
          ping_latency_ms: 20,
          checked_at: '2026-09-17T00:00:00Z',
        }],
      },
    })
    const bars = wrapper.findAll('.flex-1.min-w-\\[3px\\]')
    expect(bars.at(-1)?.classes()).toContain('bg-emerald-500')
    expect(bars.at(-1)?.classes().join(' ')).not.toContain('bg-gray-300')
  })

  it('IQ bars follow iq timeline status', () => {
    const wrapper = mount(MonitorTimeline, {
      props: {
        kind: 'iq',
        countdownSeconds: 10,
        length: 4,
        showCountdown: false,
        buckets: [{
          status: 'iq_ok',
          iq_status: 'iq_ok',
          latency_ms: 800,
          ping_latency_ms: 20,
          checked_at: '2026-09-17T00:00:00Z',
        }],
      },
    })
    const bars = wrapper.findAll('.flex-1.min-w-\\[3px\\]')
    expect(bars.at(-1)?.classes()).toContain('bg-emerald-500')
  })
})
