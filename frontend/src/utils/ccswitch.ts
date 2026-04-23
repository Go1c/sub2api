import type { GroupPlatform } from '@/types'

export type CcswitchClientType = 'claude' | 'gemini'

interface BuildCcswitchImportUrlOptions {
  apiKey: string
  baseUrl: string
  providerName?: string
  platform?: GroupPlatform | null
  clientType?: CcswitchClientType
  usageEnabled?: boolean
  defaultModels?: {
    anthropic?: string
    openai?: string
    gemini?: string
    antigravity?: string
    antigravityGemini?: string
  }
}

const DEFAULT_PROVIDER_NAME = 'sub2api'
const DEFAULT_OPENAI_MODEL = 'gpt-5.4'

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '')
}

function stripKnownApiSuffix(value: string): string {
  return trimTrailingSlash(value).replace(/\/v1(?:beta)?$/i, '')
}

function buildUsageScript(): string {
  return `({
    request: {
      url: "{{baseUrl}}/v1/usage",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
      const unit = response?.unit ?? response?.quota?.unit ?? "USD";
      return {
        isValid: response?.is_active ?? response?.isValid ?? true,
        remaining,
        unit
      };
    }
  })`
}

export function buildCcswitchImportUrl(options: BuildCcswitchImportUrlOptions): string {
  const normalizedBaseUrl = trimTrailingSlash(options.baseUrl)
  const baseRoot = stripKnownApiSuffix(normalizedBaseUrl)
  const platform = options.platform || 'anthropic'
  const clientType = options.clientType || 'claude'
  const providerName = (options.providerName || DEFAULT_PROVIDER_NAME).trim() || DEFAULT_PROVIDER_NAME
  const usageEnabled = options.usageEnabled === true
  const defaultModels = options.defaultModels || {}

  let app = 'claude'
  let endpoint = normalizedBaseUrl
  let usageBaseUrl = baseRoot
  let model = ''

  if (platform === 'antigravity') {
    app = clientType === 'gemini' ? 'gemini' : 'claude'
    endpoint = `${baseRoot}/antigravity`
    usageBaseUrl = endpoint
    model = clientType === 'gemini'
      ? (defaultModels.antigravityGemini || defaultModels.gemini || '')
      : (defaultModels.antigravity || defaultModels.anthropic || '')
  } else if (platform === 'openai') {
    app = 'codex'
    model = defaultModels.openai || DEFAULT_OPENAI_MODEL
  } else if (platform === 'gemini') {
    app = 'gemini'
    model = defaultModels.gemini || ''
  } else {
    model = defaultModels.anthropic || ''
  }

  const params = new URLSearchParams({
    resource: 'provider',
    app,
    name: providerName,
    homepage: baseRoot,
    endpoint,
    apiKey: options.apiKey,
    configFormat: 'json',
    usageEnabled: String(usageEnabled)
  })

  if (model) {
    params.set('model', model)
  }

  if (usageEnabled) {
    params.set('usageBaseUrl', usageBaseUrl)
    params.set('usageScript', btoa(buildUsageScript()))
    params.set('usageAutoInterval', '30')
  }

  return `ccswitch://v1/import?${params.toString()}`
}
