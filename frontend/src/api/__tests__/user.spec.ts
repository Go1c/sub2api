import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    put: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
  },
}))

describe('user api oauth binding urls', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.com/api/v1')
  })

  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('builds third-party bind urls against the bind start endpoint', async () => {
    const { buildOAuthBindingStartURL } = await import('@/api/user')

    expect(buildOAuthBindingStartURL('linuxdo', { redirectTo: '/settings/profile' })).toBe(
      'https://api.example.com/api/v1/auth/oauth/linuxdo/bind/start?redirect=%2Fsettings%2Fprofile&intent=bind_current_user'
    )
    expect(
      buildOAuthBindingStartURL('wechat', {
        redirectTo: '/settings/profile',
        wechatOAuthSettings: {
          wechat_oauth_open_enabled: true,
          wechat_oauth_mp_enabled: false,
          wechat_oauth_mobile_enabled: false
        }
      })
    ).toBe(
      'https://api.example.com/api/v1/auth/oauth/wechat/bind/start?redirect=%2Fsettings%2Fprofile&intent=bind_current_user&mode=open'
    )
  })
})

describe('user api wallet transactions', () => {
  beforeEach(() => {
    get.mockReset()
    get.mockResolvedValue({
      data: { items: [], total: 0, page: 1, page_size: 20, pages: 1 },
    })
  })

  it('requests GET /user/balance/transactions with page and page_size', async () => {
    const { getMyWalletTransactions, userAPI } = await import('@/api/user')

    await userAPI.getMyWalletTransactions({ page: 2, page_size: 20 })

    expect(get).toHaveBeenCalledWith('/user/balance/transactions', {
      params: { page: 2, page_size: 20 },
    })

    await getMyWalletTransactions({ page: 1, page_size: 20 })

    expect(get).toHaveBeenNthCalledWith(2, '/user/balance/transactions', {
      params: { page: 1, page_size: 20 },
    })
  })
})
