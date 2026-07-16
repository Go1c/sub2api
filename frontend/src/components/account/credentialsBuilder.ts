export const HEADER_OVERRIDE_ENABLED_CREDENTIAL_KEY = 'header_override_enabled'
export const HEADER_OVERRIDES_CREDENTIAL_KEY = 'header_overrides'

export interface HeaderOverrideRow {
  name: string
  value: string
}

const HEADER_OVERRIDE_PLATFORMS = new Set(['anthropic', 'openai'])

const PLAN_TYPE_CANONICAL = ['plus', 'pro', 'free'] as const

export function applyInterceptWarmup(
  credentials: Record<string, unknown>,
  enabled: boolean,
  mode: 'create' | 'edit'
): void {
  if (enabled) {
    credentials.intercept_warmup_requests = true
  } else if (mode === 'edit') {
    delete credentials.intercept_warmup_requests
  }
}

export function isHeaderOverridePlatform(platform: string): boolean {
  return HEADER_OVERRIDE_PLATFORMS.has(String(platform || '').toLowerCase())
}

/** Template header names for the fill-template button (values left empty). */
export function getHeaderOverrideTemplate(platform: string): HeaderOverrideRow[] {
  const p = String(platform || '').toLowerCase()
  if (p === 'anthropic') {
    return [
      { name: 'user-agent', value: '' },
      { name: 'anthropic-version', value: '' },
      { name: 'anthropic-beta', value: '' },
    ]
  }
  if (p === 'openai') {
    return [
      { name: 'user-agent', value: '' },
      { name: 'openai-organization', value: '' },
      { name: 'openai-project', value: '' },
    ]
  }
  return []
}

export function splitHeaderOverridesObject(raw: unknown): HeaderOverrideRow[] {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    return []
  }
  return Object.entries(raw as Record<string, unknown>).map(([name, value]) => ({
    name: String(name ?? ''),
    value: value == null ? '' : String(value),
  }))
}

export function validateHeaderOverrideRows(rows: HeaderOverrideRow[]): string | null {
  if (rows.length > 64) {
    return 'tooManyEntries'
  }
  const seen = new Set<string>()
  const nameRe = /^[!#$%&'*+\-.0-9A-Z^_`a-z|~]+$/
  const blocked = new Set([
    'authorization',
    'x-api-key',
    'proxy-authorization',
    'connection',
    'content-length',
    'transfer-encoding',
    'host',
  ])
  for (const row of rows) {
    const name = row.name.trim()
    const value = row.value
    if (!name && !value.trim()) {
      continue
    }
    if (!name || !nameRe.test(name)) {
      return 'invalidName'
    }
    const lower = name.toLowerCase()
    if (blocked.has(lower)) {
      return 'blockedName'
    }
    if (seen.has(lower)) {
      return 'duplicateName'
    }
    seen.add(lower)
    if (value.length > 8192 || /[\r\n\0]/.test(value)) {
      return 'invalidValue'
    }
  }
  return null
}

export function applyHeaderOverride(
  credentials: Record<string, unknown>,
  enabled: boolean,
  rows: HeaderOverrideRow[],
  mode: 'create' | 'edit'
): void {
  if (!enabled) {
    if (mode === 'edit') {
      delete credentials[HEADER_OVERRIDE_ENABLED_CREDENTIAL_KEY]
      delete credentials[HEADER_OVERRIDES_CREDENTIAL_KEY]
    }
    return
  }

  const overrides: Record<string, string> = {}
  for (const row of rows) {
    const name = row.name.trim()
    const value = row.value
    // Empty value = placeholder only; do not override
    if (!name || !value.trim()) {
      continue
    }
    overrides[name] = value
  }

  credentials[HEADER_OVERRIDE_ENABLED_CREDENTIAL_KEY] = true
  credentials[HEADER_OVERRIDES_CREDENTIAL_KEY] = overrides
}

export function applyAntigravityProjectID(
  credentials: Record<string, unknown>,
  projectId: string,
  mode: 'create' | 'edit'
): void {
  const trimmed = String(projectId ?? '').trim()
  if (trimmed) {
    credentials.project_id = trimmed
    return
  }
  if (mode === 'edit') {
    delete credentials.project_id
  }
}

export function readPlanType(credentials: Record<string, unknown> | undefined | null): string {
  if (!credentials) return ''
  const raw = credentials.plan_type
  return typeof raw === 'string' ? raw.trim() : ''
}

export function applyPlanType(
  credentials: Record<string, unknown>,
  planType: string
): Record<string, unknown> {
  const next = { ...credentials }
  const trimmed = String(planType ?? '').trim()
  if (!trimmed) {
    delete next.plan_type
  } else {
    next.plan_type = trimmed
  }
  return next
}

export function buildPlanTypeOptions(
  current: string,
  clearLabel: string
): Array<{ value: string; label: string }> {
  const options: Array<{ value: string; label: string }> = [
    { value: '', label: clearLabel },
    { value: 'plus', label: 'Plus' },
    { value: 'pro', label: 'Pro' },
    { value: 'free', label: 'Free' },
  ]
  const trimmed = String(current ?? '').trim()
  if (trimmed) {
    const lower = trimmed.toLowerCase()
    const known = PLAN_TYPE_CANONICAL.includes(lower as (typeof PLAN_TYPE_CANONICAL)[number])
    if (!known || trimmed !== lower) {
      // Preserve non-canonical / custom display while keeping the stored value
      if (!options.some((o) => o.value === trimmed)) {
        options.push({ value: trimmed, label: trimmed })
      }
    }
  }
  return options
}
