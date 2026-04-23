import { describe, expect, it } from 'vitest'

import { buildCcswitchImportUrl } from '../ccswitch'

function parseImportUrl(url: string): URLSearchParams {
  const [, query = ''] = url.split('?')
  return new URLSearchParams(query)
}

describe('buildCcswitchImportUrl', () => {
  it('forces Codex imports to use gpt-5.4 and disables usage query by default', () => {
    const params = parseImportUrl(buildCcswitchImportUrl({
      apiKey: 'sk-test',
      baseUrl: 'https://api.example.com',
      providerName: 'Test Provider',
      platform: 'openai',
      defaultModels: { openai: 'gpt-5.4' }
    }))

    expect(params.get('app')).toBe('codex')
    expect(params.get('model')).toBe('gpt-5.4')
    expect(params.get('endpoint')).toBe('https://api.example.com')
    expect(params.get('homepage')).toBe('https://api.example.com')
    expect(params.get('usageEnabled')).toBe('false')
    expect(params.get('usageScript')).toBeNull()
  })

  it('uses usageBaseUrl to avoid duplicate /v1 segments when usage query is enabled', () => {
    const params = parseImportUrl(buildCcswitchImportUrl({
      apiKey: 'sk-test',
      baseUrl: 'https://api.example.com/v1',
      platform: 'openai',
      usageEnabled: true
    }))

    expect(params.get('endpoint')).toBe('https://api.example.com/v1')
    expect(params.get('usageBaseUrl')).toBe('https://api.example.com')
    expect(atob(params.get('usageScript') || '')).toContain('{{baseUrl}}/v1/usage')
  })

  it('builds antigravity imports from the API root', () => {
    const params = parseImportUrl(buildCcswitchImportUrl({
      apiKey: 'sk-test',
      baseUrl: 'https://api.example.com/v1',
      platform: 'antigravity',
      clientType: 'gemini',
      defaultModels: { antigravityGemini: 'gemini-2.5-pro' }
    }))

    expect(params.get('app')).toBe('gemini')
    expect(params.get('endpoint')).toBe('https://api.example.com/antigravity')
    expect(params.get('homepage')).toBe('https://api.example.com')
    expect(params.get('model')).toBe('gemini-2.5-pro')
  })

  it('falls back to shared Claude/Gemini defaults for antigravity imports', () => {
    const claudeParams = parseImportUrl(buildCcswitchImportUrl({
      apiKey: 'sk-test',
      baseUrl: 'https://api.example.com/v1',
      platform: 'antigravity',
      clientType: 'claude',
      defaultModels: { anthropic: 'claude-sonnet-4-5' }
    }))
    const geminiParams = parseImportUrl(buildCcswitchImportUrl({
      apiKey: 'sk-test',
      baseUrl: 'https://api.example.com/v1',
      platform: 'antigravity',
      clientType: 'gemini',
      defaultModels: { gemini: 'gemini-2.5-pro' }
    }))

    expect(claudeParams.get('model')).toBe('claude-sonnet-4-5')
    expect(geminiParams.get('model')).toBe('gemini-2.5-pro')
  })

  it('supports configurable Claude imports', () => {
    const params = parseImportUrl(buildCcswitchImportUrl({
      apiKey: 'sk-test',
      baseUrl: 'https://api.example.com',
      platform: 'anthropic',
      defaultModels: { anthropic: 'claude-sonnet-4-5' }
    }))

    expect(params.get('app')).toBe('claude')
    expect(params.get('model')).toBe('claude-sonnet-4-5')
  })
})
