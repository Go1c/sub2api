import { describe, expect, it } from 'vitest'
import {
  buildExternalAuthHandoffUrl,
  resolveExternalAuthHandoff,
} from '../externalAuthHandoff'

describe('externalAuthHandoff', () => {
  it('builds a hash token handoff URL for an allowed return_to', () => {
    const result = resolveExternalAuthHandoff(
      {
        handoff: '1',
        return_to: 'http://localhost:3000/',
      },
      'access-token-123',
    )

    expect(result.valid).toBe(true)
    expect(result.url).toBe('http://localhost:3000/#token=access-token-123')
  })

  it('appends token to an existing hash without replacing the hash state', () => {
    const url = buildExternalAuthHandoffUrl(
      'http://localhost:3000/#view=studio',
      'access-token-123',
    )

    expect(url).toBe('http://localhost:3000/#view=studio&token=access-token-123')
  })

  it('removes existing token values from query and hash before appending the new token', () => {
    const url = buildExternalAuthHandoffUrl(
      'http://localhost:3000/?token=old&access_token=old2&ok=1#view=studio&refresh_token=old3',
      'fresh-token',
    )

    expect(url).toBe('http://localhost:3000/?ok=1#view=studio&token=fresh-token')
  })

  it('rejects relative, non-http, and untrusted return_to URLs', () => {
    expect(resolveExternalAuthHandoff({ handoff: '1', return_to: '/dashboard' }, 'token').valid).toBe(false)
    expect(resolveExternalAuthHandoff({ handoff: '1', return_to: 'ftp://localhost:3000/' }, 'token').valid).toBe(false)
    expect(resolveExternalAuthHandoff({ handoff: '1', return_to: 'https://evil.example/' }, 'token').valid).toBe(false)
  })

  it('is not active without handoff=1 or without a token', () => {
    expect(resolveExternalAuthHandoff({ return_to: 'http://localhost:3000/' }, 'token').active).toBe(false)
    expect(resolveExternalAuthHandoff({ handoff: '1', return_to: 'http://localhost:3000/' }, '').valid).toBe(false)
  })
})
