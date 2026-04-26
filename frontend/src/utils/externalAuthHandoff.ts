import type { LocationQuery } from 'vue-router'

const HANDOFF_QUERY_KEY = 'handoff'
const RETURN_TO_QUERY_KEY = 'return_to'
const TOKEN_HASH_KEY = 'token'
const SENSITIVE_TOKEN_KEYS = [
  'token',
  'access_token',
  'refresh_token',
  'expires_in',
  'token_type',
]

const DEFAULT_ALLOWED_RETURN_ORIGINS = [
  'http://localhost:3000',
  'http://127.0.0.1:3000',
]

export interface ExternalAuthHandoffResult {
  active: boolean
  valid: boolean
  url: string
  reason?: 'inactive' | 'missing_return_to' | 'missing_token' | 'invalid_return_to' | 'origin_not_allowed'
}

function firstQueryValue(value: LocationQuery[string] | string | string[] | null | undefined): string {
  if (Array.isArray(value)) {
    return typeof value[0] === 'string' ? value[0] : ''
  }
  return typeof value === 'string' ? value : ''
}

function configuredAllowedReturnOrigins(): string[] {
  const raw = String(import.meta.env.VITE_EXTERNAL_AUTH_RETURN_ORIGINS || '')
  const configured = raw
    .split(',')
    .map((origin) => origin.trim())
    .filter(Boolean)

  return Array.from(new Set([...DEFAULT_ALLOWED_RETURN_ORIGINS, ...configured]))
}

function normalizeOrigin(origin: string): string {
  return origin.replace(/\/$/, '')
}

function isAllowedReturnOrigin(origin: string): boolean {
  const normalized = normalizeOrigin(origin)
  return configuredAllowedReturnOrigins().some((allowedOrigin) => normalizeOrigin(allowedOrigin) === normalized)
}

function removeSensitiveParams(params: URLSearchParams): void {
  for (const key of SENSITIVE_TOKEN_KEYS) {
    params.delete(key)
  }
}

function sanitizeHashFragment(hash: string): string {
  const fragment = hash.startsWith('#') ? hash.slice(1) : hash
  if (!fragment) {
    return ''
  }

  if (fragment.startsWith('/') || fragment.includes('?')) {
    const questionIndex = fragment.indexOf('?')
    if (questionIndex === -1) {
      return fragment
    }

    const path = fragment.slice(0, questionIndex)
    const query = fragment.slice(questionIndex + 1)
    const params = new URLSearchParams(query)
    removeSensitiveParams(params)
    const sanitizedQuery = params.toString()
    return sanitizedQuery ? `${path}?${sanitizedQuery}` : path
  }

  if (fragment.includes('=') || fragment.includes('&')) {
    const params = new URLSearchParams(fragment)
    removeSensitiveParams(params)
    return params.toString()
  }

  return fragment
}

function appendTokenToHash(fragment: string, token: string): string {
  const tokenParam = `${TOKEN_HASH_KEY}=${encodeURIComponent(token)}`
  if (!fragment) {
    return tokenParam
  }

  if (fragment.startsWith('/') && fragment.includes('?')) {
    const separator = fragment.endsWith('?') || fragment.endsWith('&') ? '' : '&'
    return `${fragment}${separator}${tokenParam}`
  }

  if (fragment.startsWith('/')) {
    return `${fragment}?${tokenParam}`
  }

  const separator = fragment.endsWith('&') ? '' : '&'
  return `${fragment}${separator}${tokenParam}`
}

export function buildExternalAuthHandoffUrl(returnTo: string, accessToken: string): string {
  const url = new URL(returnTo)
  removeSensitiveParams(url.searchParams)

  const sanitizedHash = sanitizeHashFragment(url.hash)
  url.hash = appendTokenToHash(sanitizedHash, accessToken)

  return url.toString()
}

export function resolveExternalAuthHandoff(
  query: LocationQuery | Record<string, unknown>,
  accessToken?: string | null,
): ExternalAuthHandoffResult {
  const handoff = firstQueryValue(query[HANDOFF_QUERY_KEY] as LocationQuery[string])
  if (handoff !== '1') {
    return { active: false, valid: false, url: '', reason: 'inactive' }
  }

  const returnTo = firstQueryValue(query[RETURN_TO_QUERY_KEY] as LocationQuery[string])
  if (!returnTo) {
    return { active: true, valid: false, url: '', reason: 'missing_return_to' }
  }

  if (!accessToken) {
    return { active: true, valid: false, url: '', reason: 'missing_token' }
  }

  let parsed: URL
  try {
    parsed = new URL(returnTo)
  } catch {
    return { active: true, valid: false, url: '', reason: 'invalid_return_to' }
  }

  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    return { active: true, valid: false, url: '', reason: 'invalid_return_to' }
  }

  if (!isAllowedReturnOrigin(parsed.origin)) {
    return { active: true, valid: false, url: '', reason: 'origin_not_allowed' }
  }

  return {
    active: true,
    valid: true,
    url: buildExternalAuthHandoffUrl(returnTo, accessToken),
  }
}

export function performExternalAuthHandoff(url: string): void {
  window.location.replace(url)
}
