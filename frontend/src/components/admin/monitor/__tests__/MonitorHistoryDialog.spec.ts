import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import MonitorHistoryDialog from '../MonitorHistoryDialog.vue'
import type { ChannelMonitor } from '@/api/admin/channelMonitor'

const { listHistory, showError } = vi.hoisted(() => ({
  listHistory: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    channelMonitor: {
      listHistory
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError
  })
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (_err: unknown, fallback: string) => fallback
}))

vi.mock('@/utils/format', () => ({
  formatDateTime: (value: string) => value
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'admin.channelMonitor.logs.title': 'Request logs - {name}',
    'admin.channelMonitor.logs.empty': 'No request logs',
    'admin.channelMonitor.logs.checkedAt': 'Checked At',
    'admin.channelMonitor.logs.model': 'Model',
    'admin.channelMonitor.logs.status': 'Status',
    'admin.channelMonitor.logs.latency': 'Latency',
    'admin.channelMonitor.logs.pingLatency': 'Ping',
    'admin.channelMonitor.logs.message': 'Message'
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        const template = messages[key] || key
        return template.replace('{name}', String(params?.name ?? ''))
      }
    })
  }
})

const monitor: ChannelMonitor = {
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

describe('MonitorHistoryDialog', () => {
  it('fetches the latest 100 history rows and renders them', async () => {
    listHistory.mockResolvedValue({
      items: [
        {
          id: 101,
          model: 'claude-opus-4-6',
          status: 'operational',
          latency_ms: 3398,
          ping_latency_ms: 91,
          message: 'ok',
          checked_at: '2026-05-02T10:00:00Z'
        }
      ]
    })

    const wrapper = mount(MonitorHistoryDialog, {
      props: {
        show: true,
        monitor
      },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show', 'title'],
            emits: ['close'],
            template: '<section v-if="show"><h2>{{ title }}</h2><slot /><slot name="footer" /></section>'
          },
          Icon: true
        }
      }
    })

    await flushPromises()

    expect(listHistory).toHaveBeenCalledWith(7, { limit: 100 })
    expect(wrapper.text()).toContain('Request logs - Claude-Kiro')
    expect(wrapper.text()).toContain('claude-opus-4-6')
    expect(wrapper.text()).toContain('3398 ms')
    expect(wrapper.text()).toContain('91 ms')
    expect(wrapper.text()).toContain('ok')
  })
})
