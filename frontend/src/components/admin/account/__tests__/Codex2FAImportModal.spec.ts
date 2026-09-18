import { mount, flushPromises } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'
import Codex2FAImportModal from '../Codex2FAImportModal.vue'

const api = vi.hoisted(() => ({ jobs: vi.fn(), importAccounts: vi.fn(), retry: vi.fn() }))
vi.mock('@/api/admin/codexLogin', () => api)
vi.mock('@/api/admin/groups', () => ({ getAll: vi.fn().mockResolvedValue([{ id: 7, name: 'OpenAI' }]) }))
vi.mock('@/api/admin/proxies', () => ({ getAll: vi.fn().mockResolvedValue([]) }))
vi.mock('@/api/admin/proxyIpGroups', () => ({ list: vi.fn().mockResolvedValue([{ id: 3, name: '出口组' }]) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('Codex2FAImportModal', () => {
  it('shows the backend retry reason rather than claiming every error is cooldown', async () => {
    api.jobs.mockResolvedValue({ available: true, jobs: [{ id: 1, email: 'demo@example.com', status: 'failed', error_message: 'login failed', attempts: 1 }] })
    api.retry.mockRejectedValue({ message: '登录 Worker 未配置' })
    const wrapper = mount(Codex2FAImportModal, { props: { show: true }, global: { stubs: {
      BaseDialog: { template: '<div><slot/><slot name="footer"/></div>' }
    } } })
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === 'codexLogin.retry')!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('登录 Worker 未配置')
    expect(wrapper.text()).not.toContain('codexLogin.retryFailed')
    wrapper.unmount()
  })

  it('uploads five files with chosen account configuration and clears inputs', async () => {
    api.jobs.mockResolvedValue({ available: true, jobs: [] })
    api.importAccounts.mockResolvedValue({ job_ids: [1, 2, 3, 4, 5], errors: [] })
    const wrapper = mount(Codex2FAImportModal, { props: { show: true }, global: { stubs: {
      BaseDialog: { template: '<div><slot/><slot name="footer"/></div>' }
    } } })
    await flushPromises()
    const input = wrapper.find('input[type=file]')
    const files = Array.from({ length: 5 }, (_, i) => {
      const file = new File(['{}'], `${i}.json`)
      Object.defineProperty(file, 'text', { value: async () => JSON.stringify({ email: `test${i}@example.com`, password: 'test', totp_secret: 'JBSWY3DPEHPK3PXP' }) })
      return file
    })
    Object.defineProperty(input.element, 'files', { value: files, configurable: true })
    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(api.importAccounts).toHaveBeenCalledWith(expect.objectContaining({
      documents: expect.any(Array), group_ids: [7], proxy_ip_group_id: 3
    }))
    expect(api.importAccounts.mock.calls[0][0].documents).toHaveLength(5)
    expect(wrapper.text()).not.toContain('JBSWY3DPEHPK3PXP')
    wrapper.unmount()
  })
})
