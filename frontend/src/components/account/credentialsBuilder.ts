export const HEADER_OVERRIDE_ENABLED_CREDENTIAL_KEY = 'header_override_enabled'
export const HEADER_OVERRIDES_CREDENTIAL_KEY = 'header_overrides'

export interface HeaderOverrideRow {
  name: string
  value: string
}

const HEADER_OVERRIDE_PLATFORMS = new Set(['anthropic', 'openai', 'grok'])

const PLAN_TYPE_CANONICAL = ['plus', 'pro', 'free'] as const

// ========== Grok 自定义转发地址（base_url 仅改写转发端点，凭证生命周期不受影响） ==========

/** OAuth 账号建号/刷新默认写入的 CLI 网关 host——只有它视同"未定制"。 */
const GROK_DEFAULT_GATEWAY_HOST = 'cli-chat-proxy.grok.com'

/**
 * 判断 Grok 账号存储的 base_url 是否为主动指定的上游端点。
 * 运营方可在官方 API / 区域 API / 第三方转发地址之间手动切换（应对单端点
 * 不可用），这些值都必须回显（开关开启 + 显示地址）。仅默认 CLI 网关
 * （建号/刷新自动写入）、空值与无法解析的值视为"未定制"（与后端
 * GetGrokBaseURL 的回落语义对齐），用于 OAuth 账号编辑时决定开关初始状态。
 */
export function isCustomGrokBaseUrl(value: unknown): boolean {
  if (typeof value !== 'string') return false
  const trimmed = value.trim()
  if (!trimmed) return false
  let parsed: URL
  try {
    parsed = new URL(trimmed)
  } catch {
    return false
  }
  return parsed.hostname.toLowerCase() !== GROK_DEFAULT_GATEWAY_HOST
}

export interface GrokBaseUrlPreset {
  /** i18n 子键：admin.accounts.grokCustomBaseUrl.presets.<labelKey> */
  labelKey?: 'cli' | 'official'
  /** 字面标签（如区域标识 us-east-1），专有名词不参与 i18n */
  label?: string
  url: string
}

/**
 * Grok 快捷端点（仅供快速填充，输入框仍可自由填写任意转发地址）。
 * 官方端点偶发不可用时，运营方靠这组预设在端点间手动切换。
 */
export const GROK_BASE_URL_PRESETS: GrokBaseUrlPreset[] = [
  { labelKey: 'cli', url: 'https://cli-chat-proxy.grok.com/v1' },
  { labelKey: 'official', url: 'https://api.x.ai/v1' },
  { label: 'us-east-1', url: 'https://us-east-1.api.x.ai/v1' },
  { label: 'us-west-2', url: 'https://us-west-2.api.x.ai/v1' },
  { label: 'eu-west-1', url: 'https://eu-west-1.api.x.ai/v1' }
]

/**
 * 请求头覆写资格（与后端 IsHeaderOverrideEligible 对齐）：
 * anthropic/openai 仅 api_key；Grok 额外开放 oauth。
 */
export function isHeaderOverrideCapable(platform: string, type: string): boolean {
  const p = String(platform || '').toLowerCase()
  const t = String(type || '').toLowerCase()
  if (p === 'anthropic' || p === 'openai') {
    return t === 'apikey'
  }
  if (p === 'grok') {
    return t === 'apikey' || t === 'oauth'
  }
  return false
}

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

/** 行数组 → credentials 存储对象（小写 header 名） */
export function buildHeaderOverridesObject(rows: HeaderOverrideRow[]): Record<string, string> {
  const result: Record<string, string> = {}
  for (const row of rows) {
    const name = row.name.trim().toLowerCase()
    if (!name) continue
    result[name] = row.value.trim()
  }
  return result
}

/**
 * 解析粘贴的 JSON 文本为请求头覆写行。
 * 仅接受扁平 JSON 对象；值允许 string/number/boolean（统一转字符串），
 * 其余类型或非对象输入返回 null 表示格式非法。键为空白的条目直接丢弃。
 */
export function parseHeaderOverridesJson(text: string): HeaderOverrideRow[] | null {
  let parsed: unknown
  try {
    parsed = JSON.parse(text)
  } catch {
    return null
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null
  const rows: HeaderOverrideRow[] = []
  for (const [rawName, rawValue] of Object.entries(parsed as Record<string, unknown>)) {
    const name = rawName.trim()
    if (!name) continue
    if (
      typeof rawValue !== 'string' &&
      typeof rawValue !== 'number' &&
      typeof rawValue !== 'boolean'
    ) {
      return null
    }
    rows.push({ name, value: String(rawValue).trim() })
  }
  return rows.sort((a, b) => a.name.localeCompare(b.name))
}

/** 请求头覆写行 → 便于迁移/备份的 JSON 文本（跳过名称为空的占位行） */
export function serializeHeaderOverrideRows(rows: HeaderOverrideRow[]): string {
  const record: Record<string, string> = {}
  for (const row of rows) {
    const name = row.name.trim()
    if (!name) continue
    record[name] = row.value.trim()
  }
  return JSON.stringify(record, null, 2)
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

  credentials[HEADER_OVERRIDE_ENABLED_CREDENTIAL_KEY] = true
  credentials[HEADER_OVERRIDES_CREDENTIAL_KEY] = buildHeaderOverridesObject(rows)
}

// Configured Antigravity OAuth project fallback (not Vertex onboard project_id).
export const ANTIGRAVITY_PROJECT_ID_CREDENTIAL_KEY = 'antigravity_project_id'

export function applyAntigravityProjectID(
  credentials: Record<string, unknown>,
  projectId: string,
  mode: 'create' | 'edit'
): void {
  const trimmed = String(projectId ?? '').trim()
  if (trimmed) {
    credentials[ANTIGRAVITY_PROJECT_ID_CREDENTIAL_KEY] = trimmed
    return
  }
  if (mode === 'edit') {
    delete credentials[ANTIGRAVITY_PROJECT_ID_CREDENTIAL_KEY]
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
