import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const getById = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())
const attemptImportNavigation = vi.hoisted(() => vi.fn())

vi.mock('@/api/keys', () => ({
  keysAPI: { getById },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError }),
}))

vi.mock('@/utils/providerImport', async () => {
  const actual = await vi.importActual<typeof import('@/utils/providerImport')>('@/utils/providerImport')
  return { ...actual, attemptImportNavigation }
})

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) =>
      ({
        'keys.importToBestCodex': '导入到 BestCodex',
        'keys.importToCcSwitch': '导入到 CCS',
        'keys.downloadBestCodex': '下载 BestCodex',
        'keys.downloadBestCodexPitch': 'BestCodex 比 CC Switch 更好：它会帮你自动导入、自动配置、自动下载，同时也包含 CC Switch 的功能。',
        'keys.importFollowUpHint': '浏览器可能会询问是否打开应用。如果没有打开，再点一次下面的链接。',
        'keys.importNotOpenedCcSwitch': '未打开？使用 CC Switch',
        'keys.openBestCodex': '打开 BestCodex',
        'keys.openCcSwitch': '打开 CC Switch',
        'keys.importEndpointInvalid': '服务地址无效',
        'keys.importPrepareFailed': '暂时无法准备导入配置',
        'keys.ccsClientSelect.title': '选择应用',
        'keys.ccsClientSelect.description': '请选择要导入的应用类型：',
        'keys.ccsClientSelect.claudeCode': 'Claude Code',
        'keys.ccsClientSelect.claudeCodeDesc': '导入为 Claude',
        'keys.ccsClientSelect.geminiCli': 'Gemini CLI',
        'keys.ccsClientSelect.geminiCliDesc': '导入为 Gemini',
      })[key] ?? key,
  }),
}))

import KeyImportMenu from '../KeyImportMenu.vue'

const secret = 'sk-plus+and&eq=中文'

function detail(platform = 'openai') {
  return {
    id: 7,
    key: secret,
    name: '我的密钥',
    group: { platform, name: '默认分组' },
  }
}

function mountMenu(props: Record<string, unknown> = {}) {
  return mount(KeyImportMenu, {
    attachTo: document.body,
    props: {
      keyId: 7,
      platform: 'openai',
      keyName: '我的密钥',
      groupName: '默认分组',
      apiBaseUrl: 'https://api.lumio.games',
      siteName: 'LumioAPI',
      ...props,
    },
  })
}

describe('KeyImportMenu', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
    getById.mockReset()
    showError.mockReset()
    attemptImportNavigation.mockReset()
    getById.mockResolvedValue(detail())
  })

  afterEach(() => {
    document.body.innerHTML = ''
    vi.useRealTimers()
  })

  it('keeps the original CCS button and does not fetch until it is clicked', () => {
    const wrapper = mountMenu()
    expect(wrapper.get('[data-testid="import-primary"]').text()).toContain('导入到 CCS')
    expect(wrapper.find('[data-testid="import-menu"]').exists()).toBe(false)
    expect(getById).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('fetches the key on click, opens BestCodex once, and keeps CC Switch for a later click', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'setInterval'] })
    const windowEvents = vi.spyOn(window, 'addEventListener')
    const documentEvents = vi.spyOn(document, 'addEventListener')
    const wrapper = mountMenu()

    await wrapper.get('[data-testid="import-primary"]').trigger('click')
    await flushPromises()

    expect(getById).toHaveBeenCalledTimes(1)
    expect(getById).toHaveBeenCalledWith(7)
    expect(attemptImportNavigation).toHaveBeenCalledTimes(1)
    const url = String(attemptImportNavigation.mock.calls[0][0])
    const params = new URL(url).searchParams
    expect(url.startsWith('bestcodex://v1/import?')).toBe(true)
    expect(params.get('app')).toBe('codex')
    expect(params.get('endpoint')).toBe('https://api.lumio.games/v1')
    expect(params.get('apiKey')).toBe(secret)
    expect(params.get('name')).toBe('我的密钥 · 默认分组')
    expect(params.get('model')).toBeNull()
    expect(params.get('enabled')).toBeNull()

    const openLink = document.querySelector('[data-testid="import-open-link"]')
    const fallback = document.querySelector('[data-testid="import-ccswitch-fallback"]')
    expect(openLink?.getAttribute('href')).toBe(url)
    expect(openLink?.textContent).toContain('打开 BestCodex')
    expect(fallback?.textContent).toContain('未打开？使用 CC Switch')
    expect(fallback?.getAttribute('href')?.startsWith('ccswitch://v1/import?')).toBe(true)
    expect(new URL(fallback?.getAttribute('href') || '').searchParams.get('apiKey')).toBe(secret)
    expect(document.body.textContent).toContain('自动导入、自动配置、自动下载')
    expect(document.querySelector('[data-testid="import-follow-up-download"]')?.getAttribute('href')).toBe('https://bestcodex.app/')
    expect(document.body.textContent).not.toContain('已成功导入')
    expect(document.body.textContent).not.toContain('已检测未安装')

    await vi.advanceTimersByTimeAsync(5000)
    expect(attemptImportNavigation).toHaveBeenCalledTimes(1)
    expect(windowEvents.mock.calls.map((call) => call[0])).not.toEqual(expect.arrayContaining(['blur', 'visibilitychange']))
    expect(documentEvents.mock.calls.map((call) => call[0])).not.toEqual(expect.arrayContaining(['blur', 'visibilitychange']))

    const stored = `${JSON.stringify(localStorage)} ${JSON.stringify(sessionStorage)}`
    expect(stored).not.toContain(secret)
    expect(stored).not.toContain('bestcodex://')
    expect(stored).not.toContain('ccswitch://')
    expect(localStorage.getItem('lumio.providerImport.client')).toBeNull()

    document.querySelector<HTMLButtonElement>('[aria-label="Close modal"]')?.click()
    await flushPromises()
    expect(document.querySelector('[data-testid="import-open-link"]')).toBeNull()
    expect(document.body.innerHTML).not.toContain('apiKey')
    expect(document.body.innerHTML).not.toContain('%2B')

    windowEvents.mockRestore()
    documentEvents.mockRestore()
    wrapper.unmount()
  })

  it('waits for an Antigravity app choice and does not force Codex', async () => {
    getById.mockResolvedValue(detail('antigravity'))
    const wrapper = mountMenu({ platform: 'antigravity' })
    await wrapper.get('[data-testid="import-primary"]').trigger('click')
    await flushPromises()

    expect(getById).not.toHaveBeenCalled()
    expect(attemptImportNavigation).not.toHaveBeenCalled()

    document.querySelector<HTMLButtonElement>('[data-testid="import-app-gemini"]')?.click()
    await flushPromises()

    const url = String(attemptImportNavigation.mock.calls[0][0])
    const params = new URL(url).searchParams
    expect(url.startsWith('bestcodex://')).toBe(true)
    expect(params.get('app')).toBe('gemini')
    expect(params.get('endpoint')).toBe('https://api.lumio.games/antigravity')
    wrapper.unmount()
  })

  it('does not navigate when the service address is not https', async () => {
    const wrapper = mountMenu({ apiBaseUrl: 'http://api.lumio.games' })
    await wrapper.get('[data-testid="import-primary"]').trigger('click')
    await flushPromises()

    expect(attemptImportNavigation).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('服务地址无效')
    expect(document.querySelector('[data-testid="import-open-link"]')).toBeNull()
    expect(document.body.innerHTML).not.toContain(secret)
    wrapper.unmount()
  })
})
