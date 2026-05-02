import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import MonitorActionsCell from '../MonitorActionsCell.vue'
import type { ChannelMonitor } from '@/api/admin/channelMonitor'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const row: ChannelMonitor = {
  id: 7,
  name: 'Claude-Kiro',
  provider: 'anthropic',
  endpoint: 'https://api.example.com',
  api_key_masked: 'sk-***',
  primary_model: 'claude-opus-4-6',
  extra_models: [],
  group_name: '',
  enabled: true,
  interval_seconds: 60,
  last_checked_at: null,
  created_by: 1,
  created_at: '2026-05-02T00:00:00Z',
  updated_at: '2026-05-02T00:00:00Z',
  primary_status: 'operational',
  primary_latency_ms: 1200,
  availability_7d: 100,
  extra_models_status: [],
  template_id: null,
  extra_headers: {},
  body_override_mode: 'off',
  body_override: null,
  compatibility_probe_enabled: false
}

describe('MonitorActionsCell', () => {
  it('emits logs when clicking the log action', async () => {
    const wrapper = mount(MonitorActionsCell, {
      props: { row, running: false },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    const logButton = wrapper.findAll('button').find((button) => (
      button.text().includes('admin.channelMonitor.logs.action')
    ))
    expect(logButton).toBeTruthy()

    await logButton!.trigger('click')

    expect(wrapper.emitted('logs')).toEqual([[row]])
  })
})
