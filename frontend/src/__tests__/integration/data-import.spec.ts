import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { getModelsByPlatform } from '@/composables/useModelWhitelist'
import ImportDataModal from '@/components/admin/account/ImportDataModal.vue'

const showError = vi.fn()
const showSuccess = vi.fn()
const showWarning = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showWarning
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: { getAll: vi.fn().mockResolvedValue([{ id: 5, platform: 'openai' }]) },
    proxyIpGroups: { list: vi.fn().mockResolvedValue([{ id: 91 }]) },
    accounts: {
      importData: vi.fn()
    }
  }
}))

vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({
    t: (key: string) => key
  })
}))

const mountModal = () =>
  mount(ImportDataModal, {
    props: { show: true },
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
      }
    }
  })

const makeJsonFile = (name: string, content: string, type = 'application/json') => {
  const file = new File([content], name, { type })
  Object.defineProperty(file, 'text', {
    value: () => Promise.resolve(content)
  })
  return file
}

const setInputFiles = (element: Element, files: File[]) => {
  Object.defineProperty(element, 'files', {
    value: files,
    configurable: true
  })
}

describe('ImportDataModal', () => {
  beforeEach(async () => {
    showError.mockReset()
    showSuccess.mockReset()
    showWarning.mockReset()
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockReset()
  })

  it('applies OpenAI JSON file import defaults to the actual data endpoint', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({ proxy_created: 0, proxy_reused: 0, proxy_failed: 0, account_created: 1, account_failed: 0 })
    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile('accounts.json', JSON.stringify({ proxies: [], accounts: [{ name: 'imported', platform: 'openai', type: 'oauth', credentials: { access_token: 'test', model_mapping: { custom: 'upstream' } }, extra: { keep: true }, concurrency: 10, priority: 1 }] }))])
    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    const account = vi.mocked(adminAPI.accounts.importData).mock.calls[0][0].data.accounts[0]
    expect(account).toMatchObject({ concurrency: 8, group_ids: [5], proxy_ip_group_id: 91, extra: { keep: true, codex_fingerprint_mode: 'device', error_alert: { enabled: false }, turn_state_probe: { enabled: true }, openai_oauth_responses_websockets_v2_mode: 'passthrough', openai_oauth_responses_websockets_v2_enabled: true, account_traffic_control: { strict_rpm_enabled: true, adaptive_enabled: true, rpm: 60, burst: 5, adaptive_mode: 'observe' } } })
    expect(account.credentials.model_mapping).toEqual({ ...Object.fromEntries(getModelsByPlatform('openai').map(model => [model, model])), custom: 'upstream' })
  })

  it('forces JSON import switches to alerts off and Turn-State on', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({ proxy_created: 0, proxy_reused: 0, proxy_failed: 0, account_created: 1, account_failed: 0 })
    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile('accounts.json', JSON.stringify({
      proxies: [],
      accounts: [{
        name: 'imported',
        platform: 'openai',
        type: 'oauth',
        credentials: { token: 'test' },
        extra: { error_alert: { enabled: true }, turn_state_probe: { enabled: false }, openai_oauth_responses_websockets_v2_mode: 'ctx_pool', account_traffic_control: { strict_rpm_enabled: false, adaptive_enabled: false } }
      }]
    }))])
    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    const account = vi.mocked(adminAPI.accounts.importData).mock.calls[0][0].data.accounts[0]
    expect(account.extra.error_alert).toEqual({ enabled: false })
    expect(account.extra.turn_state_probe).toEqual({ enabled: true })
    expect(account.extra.openai_oauth_responses_websockets_v2_mode).toBe('passthrough')
    expect(account.extra.account_traffic_control).toEqual(expect.objectContaining({ strict_rpm_enabled: true, adaptive_enabled: true }))
  })

  it('keeps explicit bindings and leaves other platforms unchanged', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({ proxy_created: 0, proxy_reused: 0, proxy_failed: 0, account_created: 2, account_failed: 0 })
    const other = { name: 'claude', platform: 'anthropic', type: 'oauth', credentials: { token: 'test' } }
    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile('accounts.json', JSON.stringify({ proxies: [], accounts: [
      { name: 'openai', platform: 'openai', type: 'oauth', credentials: { token: 'test' }, group_ids: [7], proxy_key: 'explicit-proxy' }, other
    ] }))])
    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    const accounts = vi.mocked(adminAPI.accounts.importData).mock.calls[0][0].data.accounts
    expect(accounts[0]).toMatchObject({ group_ids: [7], proxy_key: 'explicit-proxy' })
    expect(accounts[0].proxy_ip_group_id).toBeUndefined()
    expect(accounts[1]).toEqual(other)
  })

  it('does not silently import without defaults when the IP group lookup fails', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.proxyIpGroups.list).mockRejectedValueOnce(new Error('IP groups unavailable'))
    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile('accounts.json', JSON.stringify({ proxies: [], accounts: [{ name: 'openai', platform: 'openai', type: 'oauth', credentials: { token: 'test' } }] }))])
    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(adminAPI.accounts.importData).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('IP groups unavailable')
  })

  it('未选择文件时提示错误', async () => {
    const wrapper = mountModal()

    await wrapper.find('form').trigger('submit')
    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportSelectFile')
  })

  it('无效 JSON 时按文件名提示解析失败', async () => {
    const { adminAPI } = await import('@/api/admin')
    const wrapper = mountModal()

    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile('data.json', 'invalid json')])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportParseFailedFile')
    expect(adminAPI.accounts.importData).not.toHaveBeenCalled()
  })

  it('不是导出数据的 JSON 按文件名拒绝', async () => {
    const { adminAPI } = await import('@/api/admin')
    const wrapper = mountModal()

    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [makeJsonFile('random.json', JSON.stringify({ name: 'test' }))])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportInvalidFile')
    expect(adminAPI.accounts.importData).not.toHaveBeenCalled()
  })

  it('无有效 JSON 的选择不清空已有选择', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 0
    })

    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')

    const valid = makeJsonFile(
      'valid.json',
      JSON.stringify({ exported_at: '2026-07-05T00:00:00Z', proxies: [], accounts: [{ name: 'a' }] })
    )
    setInputFiles(input.element, [valid])
    await input.trigger('change')

    setInputFiles(input.element, [new File(['hello'], 'notes.txt', { type: 'text/plain' })])
    await input.trigger('change')
    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportSelectFile')

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(adminAPI.accounts.importData).toHaveBeenCalledWith({
      data: expect.objectContaining({
        accounts: [{ name: 'a' }]
      }),
      skip_default_group_bind: true
    })
  })

  it('merges multiple selected JSON files before importing', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 2,
      account_failed: 0
    })

    const wrapper = mountModal()

    const input = wrapper.find('input[type="file"]')
    const first = makeJsonFile(
      'first.json',
      JSON.stringify({ exported_at: '2026-07-05T00:00:00Z', proxies: [], accounts: [{ name: 'a' }] })
    )
    const second = makeJsonFile(
      'second.json',
      JSON.stringify({
        exported_at: '2026-07-05T00:00:01Z',
        proxies: [{ proxy_key: 'p' }],
        accounts: [{ name: 'b' }]
      })
    )
    setInputFiles(input.element, [first, second])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(adminAPI.accounts.importData).toHaveBeenCalledWith({
      data: expect.objectContaining({
        proxies: [{ proxy_key: 'p' }],
        accounts: [{ name: 'a' }, { name: 'b' }]
      }),
      skip_default_group_bind: true
    })
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.dataImportSuccess')
  })

  it('部分成功时关闭弹窗仍通知父组件刷新', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 1
    })

    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')
    setInputFiles(input.element, [
      makeJsonFile(
        'mixed.json',
        JSON.stringify({
          exported_at: '2026-07-05T00:00:00Z',
          proxies: [],
          accounts: [{ name: 'a' }, { name: 'b' }]
        })
      )
    ])

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportCompletedWithErrors')
    expect(wrapper.emitted('imported')).toBeUndefined()

    // 第二个 btn-secondary 是 footer 的取消按钮(第一个是选择文件)
    await wrapper.findAll('button.btn-secondary')[1]!.trigger('click')

    expect(wrapper.emitted('imported')).toHaveLength(1)
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
