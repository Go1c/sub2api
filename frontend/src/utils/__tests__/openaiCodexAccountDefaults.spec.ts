import { describe, expect, it } from 'vitest'
import {
  KIN_DEFAULT_CODEX_CLI_ONLY,
  KIN_DEFAULT_CODEX_CONCURRENCY,
  KIN_DEFAULT_CODEX_FINGERPRINT_CONVERGENCE,
  KIN_DEFAULT_CODEX_FINGERPRINT_MODE,
  KIN_DEFAULT_CODEX_GROUP_NAME,
  KIN_DEFAULT_OPENAI_WS_MODE,
  resolveCodexCLIOnlyFromExtra,
  resolveCodexDefaultGroupIds,
  resolveCodexFingerprintConvergenceFromExtra,
  resolveCodexFingerprintModeFromExtra
} from '@/utils/openaiCodexAccountDefaults'
import { OPENAI_WS_MODE_CTX_POOL } from '@/utils/openaiWsMode'

describe('openaiCodexAccountDefaults', () => {
  it('uses Kin defaults when extra keys are missing', () => {
    expect(KIN_DEFAULT_OPENAI_WS_MODE).toBe(OPENAI_WS_MODE_CTX_POOL)
    expect(KIN_DEFAULT_CODEX_CLI_ONLY).toBe(false)
    expect(KIN_DEFAULT_CODEX_FINGERPRINT_MODE).toBe('device')
    expect(KIN_DEFAULT_CODEX_FINGERPRINT_CONVERGENCE).toBe(true)
    expect(KIN_DEFAULT_CODEX_CONCURRENCY).toBe(10)
    expect(KIN_DEFAULT_CODEX_GROUP_NAME).toBe('Codex')
    expect(resolveCodexCLIOnlyFromExtra(undefined)).toBe(false)
    expect(resolveCodexCLIOnlyFromExtra({})).toBe(false)
    expect(resolveCodexFingerprintModeFromExtra(undefined)).toBe('device')
    expect(resolveCodexFingerprintModeFromExtra({})).toBe('device')
    expect(resolveCodexFingerprintConvergenceFromExtra(undefined)).toBe(true)
    expect(resolveCodexFingerprintConvergenceFromExtra({})).toBe(true)
  })

  it('honors explicit off/false values', () => {
    expect(resolveCodexCLIOnlyFromExtra({ codex_cli_only: false })).toBe(false)
    expect(resolveCodexCLIOnlyFromExtra({ codex_cli_only: true })).toBe(true)
    expect(resolveCodexFingerprintModeFromExtra({ codex_fingerprint_mode: 'off' })).toBe('off')
    expect(resolveCodexFingerprintModeFromExtra({ codex_fingerprint_mode: 'session' })).toBe('session')
    expect(resolveCodexFingerprintConvergenceFromExtra({
      codex_experimental_fingerprint_convergence: false
    })).toBe(false)
  })

  it('prefers the Codex-named OpenAI group, then a single compatible group', () => {
    expect(resolveCodexDefaultGroupIds([
      { id: 1, name: 'Claude', platform: 'anthropic' },
      { id: 6, name: 'Codex', platform: 'openai' },
      { id: 7, name: 'GPT', platform: 'openai' }
    ])).toEqual([6])
    expect(resolveCodexDefaultGroupIds([
      { id: 8, name: 'codex', platform: 'openai' }
    ])).toEqual([8])
    expect(resolveCodexDefaultGroupIds([
      { id: 2, name: 'Default', platform: 'openai' }
    ])).toEqual([2])
    expect(resolveCodexDefaultGroupIds([
      { id: 3, name: 'GPT', platform: 'openai' },
      { id: 4, name: 'API', platform: 'openai' }
    ])).toEqual([])
    expect(resolveCodexDefaultGroupIds([
      { id: 9, name: 'Shared', platform: 'composite' }
    ])).toEqual([9])
    expect(resolveCodexDefaultGroupIds(undefined)).toEqual([])
  })
})
