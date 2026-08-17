import { beforeEach, describe, expect, it, vi } from 'vitest'

const fetchMock = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    post: vi.fn(),
  },
  buildApiUrl: (path: string) => `/api/v1${path}`,
}))

describe('parseAuthBridgeHash', () => {
  it('reads t and r from the hash fragment', async () => {
    const { parseAuthBridgeHash } = await import('../authBridge')

    expect(parseAuthBridgeHash('#t=access-jwt&r=/purchase')).toEqual({
      token: 'access-jwt',
      redirect: '/purchase',
    })
    expect(parseAuthBridgeHash('t=access-jwt&r=/dashboard')).toEqual({
      token: 'access-jwt',
      redirect: '/dashboard',
    })
  })

  it('defaults r to /purchase when missing or unsafe after decode', async () => {
    const { parseAuthBridgeHash } = await import('../authBridge')

    expect(parseAuthBridgeHash('#t=access-jwt').redirect).toBe('/purchase')
    expect(parseAuthBridgeHash('#t=access-jwt&r=').redirect).toBe('/purchase')
    expect(parseAuthBridgeHash('#t=access-jwt&r=purchase').redirect).toBe('/purchase')
    expect(parseAuthBridgeHash('#t=access-jwt&r=https://evil.example/phish').redirect).toBe('/purchase')
    expect(parseAuthBridgeHash('#t=access-jwt&r=//evil.example').redirect).toBe('/purchase')
    expect(parseAuthBridgeHash('#t=access-jwt&r=%2F%2Fevil.example').redirect).toBe('/purchase')
    expect(parseAuthBridgeHash('#t=access-jwt&r=%2Fdashboard').redirect).toBe('/dashboard')
  })

  it('returns an empty token when t is missing', async () => {
    const { parseAuthBridgeHash } = await import('../authBridge')

    expect(parseAuthBridgeHash('#r=/purchase')).toEqual({
      token: '',
      redirect: '/purchase',
    })
    expect(parseAuthBridgeHash('')).toEqual({
      token: '',
      redirect: '/purchase',
    })
  })
})

describe('exchangeAuthBridge', () => {
  beforeEach(() => {
    fetchMock.mockReset()
    vi.unstubAllGlobals()
    vi.stubGlobal('fetch', fetchMock)
  })

  it('posts the inbound token with an explicit Authorization header and unwraps the envelope', async () => {
    fetchMock.mockResolvedValue(new Response(JSON.stringify({
      code: 0,
      message: 'success',
      data: {
        access_token: 'new-access',
        refresh_token: 'rt_new',
        expires_in: 3600,
        token_type: 'Bearer',
        user: { id: 7, email: 'bridge@example.com' },
      },
    }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    }))

    const { apiClient } = await import('@/api/client')
    const { exchangeAuthBridge } = await import('../authBridge')
    const data = await exchangeAuthBridge('inbound-jwt')

    expect(data.access_token).toBe('new-access')
    expect(data.refresh_token).toBe('rt_new')
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/bridge', {
      method: 'POST',
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        Authorization: 'Bearer inbound-jwt',
      },
    })
    expect(apiClient.post).not.toHaveBeenCalled()
  })

  it('rejects failed envelopes without echoing the inbound token', async () => {
    fetchMock.mockResolvedValue(new Response(JSON.stringify({
      code: 401,
      message: 'invalid token',
    }), {
      status: 401,
      headers: { 'Content-Type': 'application/json' },
    }))

    const { exchangeAuthBridge } = await import('../authBridge')
    const error = await exchangeAuthBridge('secret-inbound').then(
      () => null,
      (err: unknown) => err,
    )

    expect(error).toMatchObject({ status: 401 })
    expect(JSON.stringify(error)).not.toContain('secret-inbound')
  })
})
