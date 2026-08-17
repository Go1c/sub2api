import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const routerReplace = vi.fn()
const applyAuthResponse = vi.fn()
const apiClientPost = vi.fn()
const fetchMock = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({
    replace: routerReplace,
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    applyAuthResponse,
  }),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    post: (...args: unknown[]) => apiClientPost(...args),
  },
  buildApiUrl: (path: string) => `/api/v1${path}`,
}))

const authResponse = {
  access_token: 'new-access',
  refresh_token: 'rt_new',
  expires_in: 3600,
  token_type: 'Bearer',
  user: {
    id: 9,
    email: 'bridge@example.com',
    username: 'bridge',
    role: 'user',
    balance: 0,
    concurrency: 1,
    status: 'active',
    allowed_groups: null,
    created_at: '2026-08-17',
    updated_at: '2026-08-17',
  },
}

function mockBridgeSuccess() {
  fetchMock.mockResolvedValue(new Response(JSON.stringify({
    code: 0,
    message: 'success',
    data: authResponse,
  }), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  }))
}

function mockBridgeFailure() {
  fetchMock.mockResolvedValue(new Response(JSON.stringify({
    code: 401,
    message: 'invalid token',
  }), {
    status: 401,
    headers: { 'Content-Type': 'application/json' },
  }))
}

async function mountBridge() {
  const { default: BridgeView } = await import('../BridgeView.vue')
  const wrapper = mount(BridgeView)
  await flushPromises()
  return wrapper
}

describe('BridgeView', () => {
  const originalLocation = window.location

  beforeEach(() => {
    routerReplace.mockReset()
    applyAuthResponse.mockReset()
    apiClientPost.mockReset()
    fetchMock.mockReset()
    vi.unstubAllGlobals()
    vi.stubGlobal('fetch', fetchMock)
    window.history.replaceState(window.history.state, '', '/auth/bridge')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    window.history.replaceState(window.history.state, '', originalLocation.pathname + originalLocation.search)
  })

  it('renders a minimal 登录中… page', async () => {
    mockBridgeFailure()
    window.history.replaceState(window.history.state, '', '/auth/bridge#t=x')
    const wrapper = await mountBridge()
    expect(wrapper.text()).toContain('登录中…')
  })

  it('writes the console session, strips the hash, then replaces to r', async () => {
    mockBridgeSuccess()
    const replaceState = vi.spyOn(window.history, 'replaceState')
    window.history.replaceState(window.history.state, '', '/auth/bridge#t=inbound-jwt&r=/purchase')

    await mountBridge()

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/bridge', expect.objectContaining({
      method: 'POST',
      headers: expect.objectContaining({
        Authorization: 'Bearer inbound-jwt',
      }),
    }))
    expect(apiClientPost).not.toHaveBeenCalled()
    expect(applyAuthResponse).toHaveBeenCalledWith(authResponse)
    expect(replaceState).toHaveBeenCalled()
    expect(window.location.hash).toBe('')
    expect(routerReplace).toHaveBeenCalledWith('/purchase')

    const hashClearedAt = replaceState.mock.invocationCallOrder[replaceState.mock.calls.findIndex((call) => {
      const url = String(call[2] ?? '')
      return url.includes('/auth/bridge') && !url.includes('#')
    })]
    const navigatedAt = routerReplace.mock.invocationCallOrder[0]
    expect(hashClearedAt).toBeLessThan(navigatedAt)
  })

  it('sends forged tokens to /login?redirect=<r> after stripping the hash', async () => {
    mockBridgeFailure()
    window.history.replaceState(window.history.state, '', '/auth/bridge#t=forged&r=/dashboard')

    await mountBridge()

    expect(applyAuthResponse).not.toHaveBeenCalled()
    expect(window.location.hash).toBe('')
    expect(routerReplace).toHaveBeenCalledWith({
      path: '/login',
      query: { redirect: '/dashboard' },
    })
  })

  it('treats a missing token as failure and still strips the hash', async () => {
    window.history.replaceState(window.history.state, '', '/auth/bridge#r=/purchase')

    await mountBridge()

    expect(fetchMock).not.toHaveBeenCalled()
    expect(window.location.hash).toBe('')
    expect(routerReplace).toHaveBeenCalledWith({
      path: '/login',
      query: { redirect: '/purchase' },
    })
  })
})
