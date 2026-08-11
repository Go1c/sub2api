import { beforeEach, describe, expect, it, vi } from 'vitest'

const post = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: { post },
  buildApiUrl: (path: string) => `/api/v1${path}`,
}))

const cookieUser = {
  id: 42,
  username: 'desktop-user',
  email: 'desktop@example.com',
  role: 'user' as const,
  balance: 10,
  concurrency: 5,
  status: 'active' as const,
  allowed_groups: null,
  created_at: '2026-08-11T00:00:00Z',
  updated_at: '2026-08-11T00:00:00Z',
  run_mode: 'standard' as const,
}

describe('cookie session auth API', () => {
  beforeEach(() => {
    localStorage.clear()
    post.mockReset()
    post.mockResolvedValue({ data: {} })
    vi.unstubAllGlobals()
  })

  it('probes the current user with credentials without persisting a token', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      code: 0,
      message: 'success',
      data: cookieUser,
    }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    }))
    vi.stubGlobal('fetch', fetchMock)
    const { probeCookieSession } = await import('@/api/auth')

    const got = await probeCookieSession()

    expect(got.id).toBe(cookieUser.id)
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/me', {
      method: 'GET',
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    expect(localStorage.getItem('auth_token')).toBeNull()
    expect(localStorage.getItem('refresh_token')).toBeNull()
  })

  it('returns a structured failure without invoking the global Axios redirect interceptor', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      code: 401,
      message: 'invalid token',
    }), {
      status: 401,
      headers: { 'Content-Type': 'application/json' },
    }))
    vi.stubGlobal('fetch', fetchMock)
    const { probeCookieSession } = await import('@/api/auth')

    await expect(probeCookieSession()).rejects.toMatchObject({
      status: 401,
      message: 'invalid token',
    })
    expect(post).not.toHaveBeenCalled()
  })

  it('always calls backend logout for a cookie-only session', async () => {
    const { logout } = await import('@/api/auth')

    await logout()

    expect(post).toHaveBeenCalledWith('/auth/logout', {})
  })
})
