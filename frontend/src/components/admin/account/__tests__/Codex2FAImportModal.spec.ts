import { mount, flushPromises } from '@vue/test-utils'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import Codex2FAImportModal from '../Codex2FAImportModal.vue'
import * as ipGroupsAPI from '@/api/admin/proxyIpGroups'
import { getImportProxyDefault } from '@/api/admin/importProxy'

const api = vi.hoisted(() => ({ jobs: vi.fn(), importAccounts: vi.fn(), retry: vi.fn() }))
vi.mock('@/api/admin/codexLogin', () => api)
vi.mock('@/api/admin/importProxy', () => ({ getImportProxyDefault: vi.fn().mockResolvedValue({proxy_ip_group_id:3,proxy_id:null,mode:'group'}) }))
vi.mock('@/api/admin/groups', () => ({ getAll: vi.fn().mockResolvedValue([{ id: 7, name: 'OpenAI' }]) }))
vi.mock('@/api/admin/proxies', () => ({ getAll: vi.fn().mockResolvedValue([]) }))
vi.mock('@/api/admin/proxyIpGroups', () => ({ list: vi.fn().mockResolvedValue([{ id: 3, name: '出口组' }]) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('Codex2FAImportModal', () => {
  beforeEach(() => vi.clearAllMocks())
  it('requires an available configured exit and never offers direct egress', async () => {
    api.jobs.mockResolvedValue({ available: true, jobs: [] })
    api.importAccounts.mockClear()
    vi.mocked(ipGroupsAPI.list).mockResolvedValueOnce([])
    vi.mocked(getImportProxyDefault).mockResolvedValueOnce({proxy_id:null,proxy_ip_group_id:null,mode:'none'})
    const wrapper = mount(Codex2FAImportModal, { props: { show: true }, global: { stubs: {
      BaseDialog: { template: '<div><slot/><slot name="footer"/></div>' }
    } } })
    await flushPromises()
    await wrapper.find('form').trigger('submit')
    expect(api.importAccounts).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('codexLogin.requireIPGroup')
    expect(wrapper.text()).not.toContain('codexLogin.direct')
    expect(wrapper.find('option[value=""]').attributes()).toHaveProperty('disabled')
    wrapper.unmount()
  })

  it('uses the server-selected single proxy fallback', async () => {
    api.jobs.mockResolvedValue({ available: true, jobs: [] })
    api.importAccounts.mockClear()
    api.importAccounts.mockResolvedValue({job_ids:[1],errors:[]})
    vi.mocked(getImportProxyDefault).mockResolvedValueOnce({proxy_id:9,proxy_ip_group_id:null,mode:'single'})
    const proxies = await import('@/api/admin/proxies')
    vi.mocked(proxies.getAll).mockResolvedValueOnce([{id:9,name:'single'}] as any)
    const wrapper=mount(Codex2FAImportModal,{props:{show:true},global:{stubs:{BaseDialog:{template:'<div><slot/><slot name="footer"/></div>'}}}})
    await flushPromises()
    await wrapper.get('#codex-2fa-proxy').setValue('proxy:9')
    await wrapper.get('textarea').setValue('sample@example.com----password----secret')
    await wrapper.get('form').trigger('submit');await flushPromises()
    expect(api.importAccounts).toHaveBeenCalledWith({documents:['sample@example.com----password----secret'],group_ids:[7],proxy_id:9})
    wrapper.unmount()
  })

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
      documents: expect.any(Array), group_ids: [7]
    }))
    expect(api.importAccounts.mock.calls[0][0].documents).toHaveLength(5)
    expect(wrapper.text()).not.toContain('JBSWY3DPEHPK3PXP')
    wrapper.unmount()
  })
  it('explains unchanged retry and requires manual group selection for reimport', async () => {
    api.jobs.mockResolvedValue({ available: true, jobs: [{ id: 290, email: 'mock@example.com', status: 'failed', proxy_ip_group_id: 2, attempts: 1 }] })
    api.importAccounts.mockClear()
    api.importAccounts.mockResolvedValue({job_ids:[290], errors:[]})
    vi.mocked(ipGroupsAPI.list).mockResolvedValueOnce([{id:1,name:'916'}, {id:2,name:'921'}] as any)
    const wrapper = mount(Codex2FAImportModal, {props:{show:true},global:{stubs:{BaseDialog:{template:'<div><slot/><slot name="footer"/></div>'}}}})
    await flushPromises()
    expect(wrapper.text()).toContain('codexLogin.retrySameGroup')
    await wrapper.findAll('button').find(b => b.text() === 'codexLogin.changeGroup')!.trigger('click')
    await wrapper.get('textarea').setValue('mock@example.com----password----secret')
    await wrapper.get('form').trigger('submit')
    expect(api.importAccounts).not.toHaveBeenCalled()
    await wrapper.get('#codex-2fa-proxy').setValue('group:1')
    await wrapper.get('form').trigger('submit'); await flushPromises()
    expect(api.importAccounts).toHaveBeenCalledWith(expect.objectContaining({proxy_ip_group_id:1}))
    wrapper.unmount()
  })

  it('sends the displayed default group even without a dropdown change', async () => {
    api.jobs.mockResolvedValue({available:true,jobs:[]})
    api.importAccounts.mockClear()
    api.importAccounts.mockResolvedValue({job_ids:[1],errors:[]})
    const wrapper = mount(Codex2FAImportModal, {props:{show:true},global:{stubs:{BaseDialog:{template:'<div><slot/><slot name="footer"/></div>'}}}})
    await flushPromises()
    await wrapper.get('textarea').setValue('mock@example.com----password----secret')
    await wrapper.get('form').trigger('submit'); await flushPromises()
    expect(api.importAccounts).toHaveBeenCalledWith(expect.objectContaining({proxy_ip_group_id:3}))
    wrapper.unmount()
  })

})
