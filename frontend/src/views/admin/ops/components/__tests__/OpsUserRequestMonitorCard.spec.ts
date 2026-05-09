import { describe, it, expect, beforeEach, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import OpsUserRequestMonitorCard from '../OpsUserRequestMonitorCard.vue'

const mockListUserRequestMonitors = vi.fn()
const mockCreateUserRequestMonitor = vi.fn()
const mockListUserRequestCaptures = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listUserRequestMonitors: (...args: any[]) => mockListUserRequestMonitors(...args),
    createUserRequestMonitor: (...args: any[]) => mockCreateUserRequestMonitor(...args),
    listUserRequestCaptures: (...args: any[]) => mockListUserRequestCaptures(...args),
    stopUserRequestMonitor: vi.fn(),
    getUserRequestCapture: vi.fn(),
    deleteUserRequestCapture: vi.fn(),
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showSuccess: vi.fn(),
    showError: vi.fn(),
  }),
}))

describe('OpsUserRequestMonitorCard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockListUserRequestMonitors.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    mockListUserRequestCaptures.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 1,
    })
  })

  it('shows raw-body warning and defaults retention to seven days when creating a monitor', async () => {
    mockCreateUserRequestMonitor.mockResolvedValue({ id: 1 })
    const wrapper = mount(OpsUserRequestMonitorCard)
    await flushPromises()

    expect(wrapper.text()).toContain('admin.ops.userRequestMonitor.rawWarning')

    await wrapper.get('[data-test="monitor-user-id"]').setValue('42')
    await wrapper.get('[data-test="monitor-duration-minutes"]').setValue('5')
    await wrapper.get('[data-test="monitor-rate-limit"]').setValue('3')
    await wrapper.get('[data-test="monitor-sample-rate"]').setValue('100')
    await wrapper.get('[data-test="monitor-create"]').trigger('submit')
    await flushPromises()

    expect(mockCreateUserRequestMonitor).toHaveBeenCalledWith({
      user_id: 42,
      duration_seconds: 300,
      max_captures_per_minute: 3,
      sample_rate_percent: 100,
      retention_days: 7,
    })
  })
})
