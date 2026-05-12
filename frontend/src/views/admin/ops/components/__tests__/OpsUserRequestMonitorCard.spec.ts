import { describe, it, expect, beforeEach, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import OpsUserRequestMonitorCard from '../OpsUserRequestMonitorCard.vue'

const mockListUserRequestMonitors = vi.fn()
const mockCreateUserRequestMonitor = vi.fn()
const mockListUserRequestCaptures = vi.fn()
const mockDownloadUserRequestMonitor = vi.fn()
const mockDeleteUserRequestMonitor = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listUserRequestMonitors: (...args: any[]) => mockListUserRequestMonitors(...args),
    createUserRequestMonitor: (...args: any[]) => mockCreateUserRequestMonitor(...args),
    listUserRequestCaptures: (...args: any[]) => mockListUserRequestCaptures(...args),
    downloadUserRequestMonitor: (...args: any[]) => mockDownloadUserRequestMonitor(...args),
    deleteUserRequestMonitor: (...args: any[]) => mockDeleteUserRequestMonitor(...args),
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
    mockDownloadUserRequestMonitor.mockResolvedValue(new Blob(['{}\n'], { type: 'application/x-ndjson' }))
    mockDeleteUserRequestMonitor.mockResolvedValue(undefined)
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

  it('adds monitor-row download and delete actions', async () => {
    mockListUserRequestMonitors.mockResolvedValue({
      items: [{
        id: 7,
        user_id: 42,
        target_email: 'target@example.com',
        status: 'expired',
        duration_seconds: 60,
        max_captures_per_minute: 10,
        sample_rate_percent: 100,
        retention_days: 7,
        created_by: 1,
        created_at: '2026-05-12T01:00:00Z',
        starts_at: '2026-05-12T01:00:00Z',
        ends_at: '2026-05-12T01:01:00Z',
        capture_count: 2,
      }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
    const originalCreateObjectURL = window.URL.createObjectURL
    const originalRevokeObjectURL = window.URL.revokeObjectURL
    window.URL.createObjectURL = vi.fn(() => 'blob:monitor-export')
    window.URL.revokeObjectURL = vi.fn()

    try {
      const wrapper = mount(OpsUserRequestMonitorCard)
      await flushPromises()
      const realCreateElement = document.createElement.bind(document)
      const linkClickSpy = vi.fn()
      vi.spyOn(document, 'createElement').mockImplementation((tagName: string, options?: ElementCreationOptions) => {
        const element = realCreateElement(tagName, options)
        if (tagName.toLowerCase() === 'a') {
          element.click = linkClickSpy
        }
        return element
      })

      await wrapper.get('[data-test="monitor-download-7"]').trigger('click')
      await flushPromises()

      expect(mockDownloadUserRequestMonitor).toHaveBeenCalledWith(7)
      expect(window.URL.createObjectURL).toHaveBeenCalled()
      expect(linkClickSpy).toHaveBeenCalled()

      await wrapper.get('[data-test="monitor-delete-7"]').trigger('click')
      await flushPromises()

      expect(confirmSpy).toHaveBeenCalledWith('admin.ops.userRequestMonitor.deleteConfirm')
      expect(mockDeleteUserRequestMonitor).toHaveBeenCalledWith(7)
      expect(mockListUserRequestMonitors).toHaveBeenCalledTimes(2)
    } finally {
      window.URL.createObjectURL = originalCreateObjectURL
      window.URL.revokeObjectURL = originalRevokeObjectURL
      confirmSpy.mockRestore()
      vi.restoreAllMocks()
    }
  })
})
