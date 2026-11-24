import {
  BUILTIN_PLATFORM_CATALOG,
  getProviderMode,
  getProviderProfile,
  type ProviderProtocolRule,
} from '@/constants/platformCatalog'

export const HEADER_OVERRIDE_ENABLED_CREDENTIAL_KEY = 'header_override_enabled'
export const HEADER_OVERRIDES_CREDENTIAL_KEY = 'header_overrides'

export interface HeaderOverrideRow {
  name: string
  value: string
}

const HEADER_OVERRIDE_PLATFORMS = new Set(['anthropic', 'openai', 'grok', 'kimi', 'zhipu', 'deepseek'])

const PLAN_TYPE_CANONICAL = ['plus', 'pro', 'free'] as const

/** 禁止覆写的请求头（与后端 headerOverrideBlockedNames 保持一致） */
const HEADER_OVERRIDE_BLOCKED_NAMES = new Set([
  'host',
  'content-length',
  'content-type',
  'transfer-encoding',
  'connection',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'proxy-connection',
  'te',
  'trailer',
  'upgrade',
  'authorization',
  'x-api-key',
  'x-goog-api-key',
  'cookie',
  'accept-encoding',
  'sec-websocket-key',
  'sec-websocket-version',
  'sec-websocket-extensions',
  'sec-websocket-protocol',
  'sec-websocket-accept',
  'session_id',
  'conversation_id',
  'x-codex-turn-state',
  'x-codex-turn-metadata',
  'chatgpt-account-id',
  'x-claude-code-session-id',
  'x-client-request-id',
  'x-grok-conv-id'
])

/** RFC 7230 token：合法的 HTTP header 名称字符集 */
const HEADER_NAME_PATTERN = /^[!#$%&'*+\-.^_`|~0-9A-Za-z]+$/

function isValidHeaderOverrideName(name: string): boolean {
  return HEADER_NAME_PATTERN.test(name)
}

/** 与后端 maxHeaderOverride* 常量保持一致 */
const HEADER_OVERRIDE_MAX_ENTRIES = 64
const HEADER_OVERRIDE_MAX_NAME_LENGTH = 200
const HEADER_OVERRIDE_MAX_VALUE_LENGTH = 8192

/** header value 不允许包含控制字符（与后端 httpguts.ValidHeaderFieldValue 对齐） */
// eslint-disable-next-line no-control-regex
const HEADER_VALUE_INVALID_PATTERN = /[\x00-\x08\x0a-\x1f\x7f]/

/** 长度限制按 UTF-8 字节计（与后端 Go len() 对齐，避免多字节值前端放行后端 400） */
const HEADER_TEXT_ENCODER = new TextEncoder()
function utf8ByteLength(value: string): number {
  return HEADER_TEXT_ENCODER.encode(value).length
}

// ---------- 请求头覆写变量（与后端 account_header_override_vars.go 保持一致） ----------

/** 可用变量名：转发时替换为本次请求的网关身份，仅 api_key 账号解析 */
export const HEADER_OVERRIDE_VARIABLES = ['user_id', 'api_key_id', 'group_id', 'request_id'] as const
export type HeaderOverrideVariable = (typeof HEADER_OVERRIDE_VARIABLES)[number]

const HEADER_OVERRIDE_VARIABLE_OPEN = '{{'
const HEADER_OVERRIDE_VARIABLE_CLOSE = '}}'
const HEADER_OVERRIDE_VARIABLE_SET: ReadonlySet<string> = new Set(HEADER_OVERRIDE_VARIABLES)

/** 变量在覆写值中的写法。花括号会被 vue-i18n 当作插值解析，因此只在代码里拼接。 */
export function headerOverrideVariableToken(name: HeaderOverrideVariable): string {
  return `${HEADER_OVERRIDE_VARIABLE_OPEN}${name}${HEADER_OVERRIDE_VARIABLE_CLOSE}`
}

/** 插入变量时建议的请求头名（值留给管理员按上游约定修改） */
export const HEADER_OVERRIDE_VARIABLE_SUGGESTED_NAMES: Record<HeaderOverrideVariable, string> = {
  user_id: 'x-gateway-user-id',
  api_key_id: 'x-gateway-api-key-id',
  group_id: 'x-gateway-group-id',
  request_id: 'x-gateway-request-id'
}

export function headerOverrideHasVariable(value: string): boolean {
  return value.includes(HEADER_OVERRIDE_VARIABLE_OPEN)
}

export type HeaderOverrideTemplatePart =
  | { kind: 'literal'; text: string }
  | { kind: 'variable'; name: HeaderOverrideVariable }

/**
 * 解析覆写值模板。任何 "{{" 都必须构成完整且已知的变量（名称两侧允许空白），
 * 否则返回 null —— 与后端保存校验同一规则，拼错的变量名不会被静默当成字面量。
 */
export function parseHeaderOverrideTemplate(value: string): HeaderOverrideTemplatePart[] | null {
  const parts: HeaderOverrideTemplatePart[] = []
  let rest = value
  for (;;) {
    const open = rest.indexOf(HEADER_OVERRIDE_VARIABLE_OPEN)
    if (open < 0) {
      if (rest) parts.push({ kind: 'literal', text: rest })
      return parts
    }
    if (open > 0) parts.push({ kind: 'literal', text: rest.slice(0, open) })
    rest = rest.slice(open + HEADER_OVERRIDE_VARIABLE_OPEN.length)
    const close = rest.indexOf(HEADER_OVERRIDE_VARIABLE_CLOSE)
    if (close < 0) return null
    const name = rest.slice(0, close).trim()
    if (!HEADER_OVERRIDE_VARIABLE_SET.has(name)) return null
    parts.push({ kind: 'variable', name: name as HeaderOverrideVariable })
    rest = rest.slice(close + HEADER_OVERRIDE_VARIABLE_CLOSE.length)
  }
}

export type HeaderOverrideRowError =
  | 'invalidName'
  | 'blockedName'
  | 'duplicateName'
  | 'invalidValue'
  | 'invalidVariable'
  | 'variableNotAllowed'

export interface HeaderOverrideValidationOptions {
  /** 账号是否允许使用变量（仅 api_key 账号；OAuth 账号不会把网关身份发给上游） */
  allowVariables?: boolean
}

/**
 * 逐行校验，返回与 rows 等长的错误数组（无错误为 null）。
 * 名称为空但值非空 / 名称非法 → invalidName；禁止覆写 → blockedName；
 * 大小写不敏感重名 → duplicateName（后出现的行）；值含控制字符或超长 → invalidValue；
 * 变量写法或变量名不对 → invalidVariable；账号类型不支持变量 → variableNotAllowed。
 */
export function getHeaderOverrideRowErrors(
  rows: HeaderOverrideRow[],
  options: HeaderOverrideValidationOptions = {}
): Array<HeaderOverrideRowError | null> {
  const allowVariables = options.allowVariables ?? true
  const seen = new Set<string>()
  return rows.map((row) => {
    const name = row.name.trim()
    const value = row.value.trim()
    if (!name) return value ? 'invalidName' : null
    if (!isValidHeaderOverrideName(name) || name.length > HEADER_OVERRIDE_MAX_NAME_LENGTH) {
      return 'invalidName'
    }
    const lower = name.toLowerCase()
    if (HEADER_OVERRIDE_BLOCKED_NAMES.has(lower)) return 'blockedName'
    if (seen.has(lower)) return 'duplicateName'
    seen.add(lower)
    if (
      HEADER_VALUE_INVALID_PATTERN.test(value) ||
      utf8ByteLength(value) > HEADER_OVERRIDE_MAX_VALUE_LENGTH
    ) {
      return 'invalidValue'
    }
    if (headerOverrideHasVariable(value)) {
      if (!allowVariables) return 'variableNotAllowed'
      if (parseHeaderOverrideTemplate(value) === null) return 'invalidVariable'
    }
    return null
  })
}

/** 校验请求头覆写行，返回首个错误的 i18n key（无错误返回 null）。 */
export function validateHeaderOverrideRows(
  rows: HeaderOverrideRow[],
  options: HeaderOverrideValidationOptions = {}
): HeaderOverrideRowError | 'tooManyEntries' | null {
  const firstError = getHeaderOverrideRowErrors(rows, options).find((error) => error !== null)
  if (firstError) return firstError
  const named = rows.filter((row) => row.name.trim()).length
  if (named > HEADER_OVERRIDE_MAX_ENTRIES) return 'tooManyEntries'
  return null
}

/** 行数组 → credentials 存储对象（名称小写化，丢弃空行） */
export function buildHeaderOverridesObject(rows: HeaderOverrideRow[]): Record<string, string> {
  const result: Record<string, string> = {}
  for (const row of rows) {
    const name = row.name.trim().toLowerCase()
    if (!name) continue
    result[name] = row.value.trim()
  }
  return result
}

/** credentials 存储对象 → 行数组（按名称排序保证稳定展示） */
export function splitHeaderOverridesObject(record: unknown): HeaderOverrideRow[] {
  if (!record || typeof record !== 'object' || Array.isArray(record)) return []
  return Object.entries(record as Record<string, unknown>)
    .filter(([, value]) => typeof value === 'string')
    .map(([name, value]) => ({ name, value: value as string }))
    .sort((a, b) => a.name.localeCompare(b.name))
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

// ========== 国产供应商（Kimi / Zhipu / DeepSeek）base_url 预设 ==========
// 与后端 service/domain_constants.go 的默认 base url 保持一致。
// 账号类型（payg 按量付费 / coding 编程套餐）决定额度监控方式；
// API 协议（chat_completions / anthropic / responses）决定转发端点与格式，
// 两者正交。同协议请求零转换直通，跨协议组合才走转换链。

export type CnAccountMode = 'payg' | 'coding'

/** 仅 deepseek 支持原生 responses；adaptive 会按入站协议选择原生端点。 */
export type CnApiProtocol = 'adaptive' | 'chat_completions' | 'anthropic' | 'responses'
export type CnNativeApiProtocol = Exclude<CnApiProtocol, 'adaptive'>

export interface CnBaseUrlPreset {
  mode: CnAccountMode
  protocol: CnApiProtocol
  /** 专有名词，不参与 i18n */
  label: string
  url: string
}

/** 各供应商按账号类型 × API 协议分档的快捷端点（点击快速填充，输入框仍可自由填写）。 */
export const CN_BASE_URL_PRESETS: Record<'kimi' | 'zhipu' | 'deepseek', CnBaseUrlPreset[]> = {
  kimi: [
    { mode: 'payg', protocol: 'chat_completions', label: 'Moonshot', url: 'https://api.moonshot.cn/v1' },
    { mode: 'payg', protocol: 'anthropic', label: 'Moonshot Anthropic', url: 'https://api.moonshot.cn/anthropic' },
    { mode: 'coding', protocol: 'chat_completions', label: 'Kimi For Coding', url: 'https://api.kimi.com/coding/v1' },
    { mode: 'coding', protocol: 'anthropic', label: 'Kimi Coding Anthropic', url: 'https://api.kimi.com/coding' }
  ],
  zhipu: [
    { mode: 'payg', protocol: 'chat_completions', label: 'GLM PaaS', url: 'https://open.bigmodel.cn/api/paas/v4' },
    { mode: 'payg', protocol: 'anthropic', label: 'GLM Anthropic', url: 'https://open.bigmodel.cn/api/anthropic' },
    { mode: 'coding', protocol: 'chat_completions', label: 'GLM Coding', url: 'https://open.bigmodel.cn/api/coding/paas/v4' },
    { mode: 'coding', protocol: 'anthropic', label: 'GLM Coding Anthropic', url: 'https://open.bigmodel.cn/api/anthropic' }
  ],
  deepseek: [
    { mode: 'payg', protocol: 'chat_completions', label: 'DeepSeek', url: 'https://api.deepseek.com' },
    { mode: 'payg', protocol: 'anthropic', label: 'DeepSeek Anthropic', url: 'https://api.deepseek.com/anthropic' },
    { mode: 'payg', protocol: 'responses', label: 'DeepSeek Responses', url: 'https://api.deepseek.com' }
  ]
}

/** 返回指定供应商 + 账号类型 + API 协议的默认 base url。 */
export function defaultCNBaseUrl(
  platform: string,
  mode: CnAccountMode,
  protocol: CnApiProtocol = 'chat_completions'
): string {
  if (protocol === 'anthropic') {
    switch (platform) {
      case 'kimi':
        return mode === 'coding' ? 'https://api.kimi.com/coding' : 'https://api.moonshot.cn/anthropic'
      case 'zhipu':
        return 'https://open.bigmodel.cn/api/anthropic'
      case 'deepseek':
        return 'https://api.deepseek.com/anthropic'
      default:
        return ''
    }
  }
  // responses 仅 deepseek：base 与 chat_completions 相同（端点路径差异由后端处理）。
  switch (platform) {
    case 'kimi':
      return mode === 'coding' ? 'https://api.kimi.com/coding/v1' : 'https://api.moonshot.cn/v1'
    case 'zhipu':
      return mode === 'coding'
        ? 'https://open.bigmodel.cn/api/coding/paas/v4'
        : 'https://open.bigmodel.cn/api/paas/v4'
    case 'deepseek':
      return 'https://api.deepseek.com'
    default:
      return ''
  }
}

/** 返回自适应模式下需要配置的原生协议及其默认端点。 */
export function defaultCNAdaptiveBaseUrls(
  platform: 'kimi' | 'zhipu' | 'deepseek',
  mode: CnAccountMode
): Record<CnNativeApiProtocol, string> {
  return {
    chat_completions: defaultCNBaseUrl(platform, mode, 'chat_completions'),
    anthropic: defaultCNBaseUrl(platform, mode, 'anthropic'),
    responses: platform === 'deepseek' ? defaultCNBaseUrl(platform, mode, 'responses') : ''
  }
}

// ===== 国产供应商用量单元格可见性（单一事实源） =====
// CNProviderQuotaCell / CNProviderBalanceCell 与 AccountUsageCell 的占位符判定
// 共用，避免多处复制条件后一处改另一处漏改。

export function cnQuotaCellVisible(platform: string, accountMode: string): boolean {
  return (platform === 'kimi' || platform === 'zhipu') && accountMode === 'coding'
}

export function cnBalanceCellVisible(platform: string, accountMode: string): boolean {
  return (platform === 'kimi' || platform === 'deepseek') && accountMode !== 'coding'
}

export function isCNCodingPlan(platform: string, accountMode: string): boolean {
  return cnQuotaCellVisible(platform, accountMode)
}

export function isCNPaygBalance(platform: string, accountMode: string): boolean {
  return cnBalanceCellVisible(platform, accountMode)
}

/**
 * 请求头覆写资格（与后端 IsHeaderOverrideEligible 对齐）：
 * anthropic/openai 仅 api_key；Grok 额外开放 oauth。
 */
export function isHeaderOverrideCapable(platform: string, type: string): boolean {
  const p = String(platform || '').toLowerCase()
  const t = String(type || '').toLowerCase()
  if (p === 'anthropic' || p === 'openai' || p === 'kimi' || p === 'zhipu' || p === 'deepseek') {
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

export type OpenCodeAccountMode = 'zen' | 'go'

const NATIVE_API_PROTOCOLS: CnNativeApiProtocol[] = ['chat_completions', 'anthropic', 'responses']

export function providerSupportsProtocol(platform: string, mode: string | undefined, protocol: CnNativeApiProtocol): boolean {
  return !!getProviderMode(platform, mode)?.base_urls[protocol]
}

export function cnSupportsNativeResponses(platform: string, mode?: string): boolean {
  return providerSupportsProtocol(platform, mode, 'responses')
}

export function providerNativeProtocols(platform: string, mode?: string): CnNativeApiProtocol[] {
  return NATIVE_API_PROTOCOLS.filter(protocol => providerSupportsProtocol(platform, mode, protocol))
}

export function providerHasModelCatalog(platform: string): boolean {
  return getProviderProfile(platform)?.model_catalog === true
}

export function providerRoutesByModel(platform: string): boolean {
  return getProviderProfile(platform)?.routing === 'by_model'
}

export interface OpenCodeGoProtocolRule {
  pattern: string
  protocol: CnNativeApiProtocol
  extraProtocols?: CnNativeApiProtocol[]
}

function isNativeOpenCodeGoProtocol(value: unknown): value is CnNativeApiProtocol {
  return value === 'chat_completions' || value === 'anthropic' || value === 'responses'
}

function extraRuleProtocols(protocol: CnNativeApiProtocol, protocols: unknown): CnNativeApiProtocol[] {
  if (!Array.isArray(protocols)) return []
  const extra: CnNativeApiProtocol[] = []
  for (const item of protocols) {
    if (isNativeOpenCodeGoProtocol(item) && item !== protocol && !extra.includes(item)) extra.push(item)
  }
  return extra
}

function buildRule(pattern: string, protocol: CnNativeApiProtocol, protocols?: unknown): OpenCodeGoProtocolRule {
  const extra = extraRuleProtocols(protocol, protocols)
  return extra.length > 0 ? { pattern, protocol, extraProtocols: extra } : { pattern, protocol }
}

function normalizeProfileRules(rules: ProviderProtocolRule[] | undefined): OpenCodeGoProtocolRule[] {
  return (rules ?? [])
    .filter(rule => rule.pattern && isNativeOpenCodeGoProtocol(rule.protocol))
    .map(rule => buildRule(rule.pattern, rule.protocol as CnNativeApiProtocol, rule.protocols))
}

function builtinProtocolRules(platform: string, mode: string): OpenCodeGoProtocolRule[] {
  const profile = BUILTIN_PLATFORM_CATALOG.platforms.find(spec => spec.id === platform)?.multi_protocol
  return normalizeProfileRules(profile?.modes.find(item => item.mode === mode)?.protocol_rules)
}

export const DEFAULT_OPENCODE_GO_PROTOCOL_RULES: OpenCodeGoProtocolRule[] = builtinProtocolRules('opencode_go', 'go')

export function defaultProviderProtocolRules(platform: string, mode?: string): OpenCodeGoProtocolRule[] {
  return normalizeProfileRules(getProviderMode(platform, mode)?.protocol_rules)
}

export function defaultOpenCodeProtocolRules(mode: string = 'go'): OpenCodeGoProtocolRule[] {
  return defaultProviderProtocolRules('opencode_go', mode === 'zen' ? 'zen' : 'go')
}

export function cloneOpenCodeGoProtocolRules(
  rules: OpenCodeGoProtocolRule[] = DEFAULT_OPENCODE_GO_PROTOCOL_RULES
): OpenCodeGoProtocolRule[] {
  return rules.map(rule => buildRule(rule.pattern, rule.protocol, rule.extraProtocols))
}

