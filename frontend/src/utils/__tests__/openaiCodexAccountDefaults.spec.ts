import { describe, expect, it } from 'vitest'
import {
  KIN_DEFAULT_CODEX_CLI_ONLY,
  KIN_DEFAULT_CODEX_FINGERPRINT_CONVERGENCE,
  KIN_DEFAULT_CODEX_FINGERPRINT_MODE,
  KIN_DEFAULT_OPENAI_WS_MODE,
  resolveCodexCLIOnlyFromExtra,
  resolveCodexFingerprintConvergenceFromExtra,
  resolveCodexFingerprintModeFromExtra
} from '@/utils/openaiCodexAccountDefaults'
import { OPENAI_WS_MODE_CTX_POOL } from '@/utils/openaiWsMode'

describe('openaiCodexAccountDefaults', () => {
  it('uses Kin defaults when extra keys are missing', () => {
    expect(KIN_DEFAULT_OPENAI_WS_MODE).toBe(OPENAI_WS_MODE_CTX_POOL)
    expect(KIN_DEFAULT_CODEX_CLI_ONLY).toBe(true)
    expect(KIN_DEFAULT_CODEX_FINGERPRINT_MODE).toBe('device')
    expect(KIN_DEFAULT_CODEX_FINGERPRINT_CONVERGENCE).toBe(true)
    expect(resolveCodexCLIOnlyFromExtra(undefined)).toBe(true)
    expect(resolveCodexCLIOnlyFromExtra({})).toBe(true)
    expect(resolveCodexFingerprintModeFromExtra(undefined)).toBe('device')
    expect(resolveCodexFingerprintModeFromExtra({})).toBe('device')
    expect(resolveCodexFingerprintConvergenceFromExtra(undefined)).toBe(true)
    expect(resolveCodexFingerprintConvergenceFromExtra({})).toBe(true)
  })

  it('honors explicit off/false values', () => {
    expect(resolveCodexCLIOnlyFromExtra({ codex_cli_only: false })).toBe(false)
    expect(resolveCodexFingerprintModeFromExtra({ codex_fingerprint_mode: 'off' })).toBe('off')
    expect(resolveCodexFingerprintModeFromExtra({ codex_fingerprint_mode: 'session' })).toBe('session')
    expect(resolveCodexFingerprintConvergenceFromExtra({
      codex_experimental_fingerprint_convergence: false
    })).toBe(false)
  })
})
