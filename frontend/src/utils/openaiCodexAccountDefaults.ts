import { OPENAI_WS_MODE_CTX_POOL, type OpenAIWSMode } from './openaiWsMode'

export type CodexFingerprintMode = 'off' | 'device' | 'session' | 'full'

export const KIN_DEFAULT_OPENAI_WS_MODE: OpenAIWSMode = OPENAI_WS_MODE_CTX_POOL
export const KIN_DEFAULT_CODEX_CLI_ONLY = true
export const KIN_DEFAULT_CODEX_FINGERPRINT_MODE: CodexFingerprintMode = 'device'
export const KIN_DEFAULT_CODEX_FINGERPRINT_CONVERGENCE = true
export const KIN_DEFAULT_CODEX_CONCURRENCY = 10
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

const CODEX_FINGERPRINT_MODES = new Set<CodexFingerprintMode>([
  'off',
  'device',
  'session',
  'full'
])

export const resolveCodexCLIOnlyFromExtra = (
  extra: Record<string, unknown> | null | undefined
): boolean => extra?.codex_cli_only !== false

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
