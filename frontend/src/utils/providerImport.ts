export type ImportClient = 'bestcodex' | 'ccswitch'
export type ImportApp = 'codex' | 'claude' | 'gemini'

export const IMPORT_CLIENT_STORAGE_KEY = 'lumio.providerImport.client'
export const PROVIDER_IMPORT_HOMEPAGE = 'https://api.lumio.games/'

const IMPORT_CLIENTS: readonly ImportClient[] = ['bestcodex', 'ccswitch']
const IMPORT_APPS: readonly ImportApp[] = ['codex', 'claude', 'gemini']
const MAX_IMPORT_URL_LENGTH = 8000

export interface ProviderImportInput {
  app: ImportApp
  name: string
  endpoint: string
  apiKey: string
}

export function providerImportUrl(client: ImportClient, input: ProviderImportInput): string {
  if (!IMPORT_CLIENTS.includes(client)) throw new Error('客户端无效')
  if (!IMPORT_APPS.includes(input.app)) throw new Error('应用类型无效')

  const endpoint = new URL(input.endpoint)
  if (endpoint.protocol !== 'https:' || endpoint.username || endpoint.password) {
    throw new Error('服务地址无效')
  }
  if (!input.apiKey || /[\u0000-\u001f\u007f]/.test(input.apiKey)) throw new Error('Key 无效')

  const params = new URLSearchParams({
    resource: 'provider',
    app: input.app,
    name: input.name,
    endpoint: endpoint.toString(),
    apiKey: input.apiKey,
    homepage: PROVIDER_IMPORT_HOMEPAGE,
  })
  const result = `${client}://v1/import?${params.toString()}`
  if (result.length > MAX_IMPORT_URL_LENGTH) throw new Error('配置过长，请使用客户端手动导入')
  return result
}

export function readImportClient(): ImportClient {
  try {
    return localStorage.getItem(IMPORT_CLIENT_STORAGE_KEY) === 'ccswitch' ? 'ccswitch' : 'bestcodex'
  } catch {
    return 'bestcodex'
  }
}

export function writeImportClient(client: ImportClient): void {
  if (!IMPORT_CLIENTS.includes(client)) return
  localStorage.setItem(IMPORT_CLIENT_STORAGE_KEY, client)
}

export function providerImportName(input: {
  keyName?: string | null
  groupName?: string | null
  siteName?: string | null
}): string {
  const keyName = input.keyName?.trim() || ''
  const groupName = input.groupName?.trim() || ''
  if (keyName && groupName && keyName !== groupName) return `${keyName} · ${groupName}`
  return keyName || groupName || input.siteName?.trim() || 'LumioAPI'
}

export type ResolvedProviderImport =
  | { needsAppChoice: true }
  | { needsAppChoice?: false; app: ImportApp; name: string; endpoint: string }

/**
 * Codex uses the Lumio Responses base, exactly one /v1.
 * Claude and Gemini keep the site's published base and do not gain another /v1.
 * Antigravity keeps the existing /antigravity path on that root.
 */
export function resolveProviderImport(input: {
  platform?: string | null
  antigravityApp?: 'claude' | 'gemini' | null
  apiBaseUrl?: string | null
  pageOrigin?: string | null
  keyName?: string | null
  groupName?: string | null
  siteName?: string | null
}): ResolvedProviderImport {
  let app: ImportApp
  if (input.platform === 'antigravity') {
    if (input.antigravityApp !== 'claude' && input.antigravityApp !== 'gemini') {
      return { needsAppChoice: true }
    }
    app = input.antigravityApp
  } else if (input.platform === 'openai') {
    app = 'codex'
  } else if (input.platform === 'gemini') {
    app = 'gemini'
  } else {
    app = 'claude'
  }

  return {
    app,
    name: providerImportName(input),
    endpoint: resolveImportEndpoint({
      app,
      platform: input.platform,
      apiBaseUrl: input.apiBaseUrl,
      pageOrigin: input.pageOrigin,
    }),
  }
}

export function resolveImportEndpoint(input: {
  app: ImportApp
  platform?: string | null
  apiBaseUrl?: string | null
  pageOrigin?: string | null
}): string {
  const configured = (input.apiBaseUrl || input.pageOrigin || '').trim()
  let parsed: URL
  try {
    parsed = new URL(configured)
  } catch {
    throw new Error('服务地址无效')
  }
  if (parsed.protocol !== 'https:' || parsed.username || parsed.password) {
    throw new Error('服务地址无效')
  }

  const path = parsed.pathname.replace(/\/+$/, '')
  const rootPath = path.replace(/\/v1(?:beta)?$/i, '')
  if (input.platform === 'antigravity') {
    return `${parsed.origin}${rootPath}/antigravity`
  }
  if (input.app === 'codex') {
    return `${parsed.origin}${rootPath}/v1`
  }
  return `${parsed.origin}${path}`
}

/** User-gesture continuation only. Callers must not invoke this from timers or visibility events. */
export function attemptImportNavigation(url: string): void {
  window.location.assign(url)
}
