import { OPENAI_WS_MODE_CTX_POOL, type OpenAIWSMode } from './openaiWsMode'

export type CodexFingerprintMode = 'off' | 'device' | 'session' | 'full'

export const KIN_DEFAULT_OPENAI_WS_MODE: OpenAIWSMode = OPENAI_WS_MODE_CTX_POOL
export const KIN_DEFAULT_CODEX_CLI_ONLY = false
export const KIN_DEFAULT_CODEX_FINGERPRINT_MODE: CodexFingerprintMode = 'device'
export const KIN_DEFAULT_CODEX_FINGERPRINT_CONVERGENCE = true
export const KIN_DEFAULT_CODEX_CONCURRENCY = 100
export const KIN_DEFAULT_OPENAI_LONG_CONTEXT_BILLING = true
export const KIN_DEFAULT_CODEX_GROUP_NAME = 'Codex'

export type CodexDefaultGroup = {
  id: number
  name?: string | null
  platform?: string | null
}

export const resolveCodexDefaultGroupIds = (
  groups: CodexDefaultGroup[] | null | undefined
): number[] => {
  const compatible = (groups ?? []).filter(
    (group) => group.platform === 'openai' || group.platform === 'composite'
  )
  const wantedName = KIN_DEFAULT_CODEX_GROUP_NAME.toLowerCase()
  const named = compatible.find(
    (group) => (group.name ?? '').trim().toLowerCase() === wantedName
  )
  if (named) return [named.id]

  const openaiGroups = compatible.filter((group) => group.platform === 'openai')
  if (openaiGroups.length === 1) return [openaiGroups[0].id]
  if (compatible.length === 1) return [compatible[0].id]
  return []
}

export const resolveKinCodexImportGroupIds = (
  selected: number[] | null | undefined,
  snapshot: number[] | null | undefined,
  groups: CodexDefaultGroup[] | null | undefined
): number[] => {
  if (selected && selected.length > 0) return [...selected]
  if (snapshot && snapshot.length > 0) return [...snapshot]
  return resolveCodexDefaultGroupIds(groups)
}

export const resolveKinCodexImportConcurrency = (
  selected: number | null | undefined,
  snapshot?: number | null
): number => {
  if (typeof selected === 'number' && Number.isFinite(selected) && selected > 0) {
    return selected
  }
  if (typeof snapshot === 'number' && Number.isFinite(snapshot) && snapshot > 0) {
    return snapshot
  }
  return KIN_DEFAULT_CODEX_CONCURRENCY
}

const CODEX_FINGERPRINT_MODES = new Set<CodexFingerprintMode>([
  'off',
  'device',
  'session',
  'full'
])

export const resolveCodexCLIOnlyFromExtra = (
  extra: Record<string, unknown> | null | undefined
): boolean => extra?.codex_cli_only === true

export const resolveCodexFingerprintModeFromExtra = (
  extra: Record<string, unknown> | null | undefined
): CodexFingerprintMode => {
  const raw = extra?.codex_fingerprint_mode
  if (typeof raw === 'string' && CODEX_FINGERPRINT_MODES.has(raw as CodexFingerprintMode)) {
    return raw as CodexFingerprintMode
  }
  return KIN_DEFAULT_CODEX_FINGERPRINT_MODE
}

export const resolveCodexFingerprintConvergenceFromExtra = (
  extra: Record<string, unknown> | null | undefined
): boolean => extra?.codex_experimental_fingerprint_convergence !== false
