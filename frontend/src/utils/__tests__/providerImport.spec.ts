import { afterEach, describe, expect, it } from 'vitest'

import {
  IMPORT_CLIENT_STORAGE_KEY,
  providerImportUrl,
  readImportClient,
  resolveProviderImport,
  writeImportClient,
} from '../providerImport'

function paramsOf(url: string): URLSearchParams {
  const parsed = new URL(url)
  return parsed.searchParams
}

describe('providerImportUrl', () => {
  it('switches only the scheme and keeps the CC Switch v1 provider params', () => {
    const input = {
      app: 'codex' as const,
      name: 'Lumio',
      endpoint: 'https://api.lumio.games/v1',
      apiKey: 'sk-test',
    }

    const bestcodex = paramsOf(providerImportUrl('bestcodex', input))
    const ccswitch = paramsOf(providerImportUrl('ccswitch', input))

    expect(providerImportUrl('bestcodex', input).startsWith('bestcodex://v1/import?')).toBe(true)
    expect(providerImportUrl('ccswitch', input).startsWith('ccswitch://v1/import?')).toBe(true)
    expect(bestcodex.toString()).toBe(ccswitch.toString())
    expect([...bestcodex.keys()]).toEqual(['resource', 'app', 'name', 'endpoint', 'apiKey', 'homepage'])
    expect(bestcodex.get('resource')).toBe('provider')
    expect(bestcodex.get('app')).toBe('codex')
    expect(bestcodex.get('homepage')).toBe('https://api.lumio.games/')
    expect(bestcodex.get('model')).toBeNull()
    expect(bestcodex.get('enabled')).toBeNull()
    expect(bestcodex.get('usageScript')).toBeNull()
    expect(bestcodex.get('config')).toBeNull()
    expect(bestcodex.get('configUrl')).toBeNull()
    expect(bestcodex.get('ticket')).toBeNull()
  })

  it('round-trips names and keys that contain plus, ampersand, equals, or Chinese', () => {
    const name = '密钥 & 名称=测试'
    const apiKey = 'sk-plus+and&eq=中文'
    const url = providerImportUrl('bestcodex', {
      app: 'claude',
      name,
      endpoint: 'https://api.lumio.games',
      apiKey,
    })
    const params = paramsOf(url)

    expect(params.get('name')).toBe(name)
    expect(params.get('apiKey')).toBe(apiKey)
    expect(params.get('endpoint')).toBe('https://api.lumio.games/')
    expect(url).toContain('%2B')
    expect(url).toContain('%26')
    expect(url).toContain('%3D')
    expect(url).not.toContain('sk-plus+and')
    expect(url).not.toContain('密钥')
  })

  it.each([
    ['client', () => providerImportUrl('ccs' as 'bestcodex', { app: 'codex', name: 'n', endpoint: 'https://api.lumio.games/v1', apiKey: 'sk' }), '客户端无效'],
    ['app', () => providerImportUrl('bestcodex', { app: 'grok' as 'codex', name: 'n', endpoint: 'https://api.lumio.games/v1', apiKey: 'sk' }), '应用类型无效'],
    ['http', () => providerImportUrl('bestcodex', { app: 'codex', name: 'n', endpoint: 'http://api.lumio.games/v1', apiKey: 'sk' }), '服务地址无效'],
    ['userinfo', () => providerImportUrl('bestcodex', { app: 'codex', name: 'n', endpoint: 'https://user:secret@api.lumio.games/v1', apiKey: 'sk' }), '服务地址无效'],
    ['empty key', () => providerImportUrl('bestcodex', { app: 'codex', name: 'n', endpoint: 'https://api.lumio.games/v1', apiKey: '' }), 'Key 无效'],
    ['control key', () => providerImportUrl('bestcodex', { app: 'codex', name: 'n', endpoint: 'https://api.lumio.games/v1', apiKey: 'sk\nleak' }), 'Key 无效'],
  ])('rejects an invalid %s', (_label, build, message) => {
    expect(build).toThrow(message)
  })

  it('rejects an import url longer than 8000 characters', () => {
    expect(() => providerImportUrl('bestcodex', {
      app: 'codex',
      name: 'n',
      endpoint: 'https://api.lumio.games/v1',
      apiKey: `sk-${'a'.repeat(7900)}`,
    })).toThrow('配置过长，请使用客户端手动导入')
  })
})

describe('resolveProviderImport', () => {
  const base = {
    apiBaseUrl: 'https://api.lumio.games',
    keyName: '我的密钥',
    groupName: '默认分组',
  }

  it('keeps Codex on the Lumio Responses /v1 address without doubling it', () => {
    for (const apiBaseUrl of ['https://api.lumio.games', 'https://api.lumio.games/v1', 'https://api.lumio.games/v1/']) {
      const resolved = resolveProviderImport({ ...base, platform: 'openai', apiBaseUrl })
      expect(resolved).toMatchObject({
        app: 'codex',
        endpoint: 'https://api.lumio.games/v1',
        name: '我的密钥 · 默认分组',
      })
    }
  })

  it('keeps Claude and Gemini on the published base instead of appending /v1', () => {
    expect(resolveProviderImport({ ...base, platform: 'anthropic' })).toMatchObject({
      app: 'claude',
      endpoint: 'https://api.lumio.games',
    })
    expect(resolveProviderImport({ ...base, platform: 'gemini' })).toMatchObject({
      app: 'gemini',
      endpoint: 'https://api.lumio.games',
    })
    expect(resolveProviderImport({
      ...base,
      platform: 'anthropic',
      apiBaseUrl: 'https://api.lumio.games/v1',
    })).toMatchObject({
      app: 'claude',
      endpoint: 'https://api.lumio.games/v1',
    })
  })

  it('does not force other platforms onto Codex', () => {
    expect(resolveProviderImport({ ...base, platform: 'grok' })).toMatchObject({ app: 'claude' })
    expect(resolveProviderImport({ ...base, platform: null })).toMatchObject({ app: 'claude' })
  })

  it('asks for an app on Antigravity and then uses the /antigravity endpoint', () => {
    expect(resolveProviderImport({ ...base, platform: 'antigravity' })).toEqual({ needsAppChoice: true })
    expect(resolveProviderImport({
      ...base,
      platform: 'antigravity',
      antigravityApp: 'gemini',
      apiBaseUrl: 'https://api.lumio.games/v1/',
    })).toMatchObject({
      app: 'gemini',
      endpoint: 'https://api.lumio.games/antigravity',
    })
  })
})

describe('import client preference', () => {
  afterEach(() => {
    localStorage.clear()
  })

  it('defaults to BestCodex and stores only the client id', () => {
    expect(readImportClient()).toBe('bestcodex')
    localStorage.setItem(IMPORT_CLIENT_STORAGE_KEY, 'bestcodex://v1/import?apiKey=sk-secret')
    expect(readImportClient()).toBe('bestcodex')

    writeImportClient('ccswitch')
    expect(localStorage.getItem(IMPORT_CLIENT_STORAGE_KEY)).toBe('ccswitch')
    expect(JSON.stringify(localStorage)).not.toContain('sk-')
    expect(JSON.stringify(localStorage)).not.toContain('://')
  })
})
