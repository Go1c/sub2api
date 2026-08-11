import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

type NavigationGuard = (
  to: Record<string, any>,
  from: Record<string, any>,
  next: ReturnType<typeof vi.fn>
) => Promise<void>

const routerHarness = vi.hoisted(() => ({
  guard: null as NavigationGuard | null,
  routes: [] as Array<Record<string, any>>,
}))

const authStore = vi.hoisted(() => ({
  checkAuth: vi.fn(),
  restoreCookieSession: vi.fn(),
  isAuthenticated: true,
  isAdmin: false,
  isSimpleMode: false,
  hasPendingAuthSession: false,
  token: null as string | null,
  user: null as Record<string, unknown> | null,
}))

const appStore = vi.hoisted(() => ({
  siteName: 'Sub2API',
  backendModeEnabled: false,
  publicSettingsLoaded: true,
  cachedPublicSettings: {
    payment_enabled: true,
    risk_control_enabled: false,
    custom_menu_items: [],
  },
  fetchPublicSettings: vi.fn(),
}))

function resolvedFullPath(location: Record<string, any>): string {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(location.query || {})) {
    if (Array.isArray(value)) {
      value.forEach((item) => params.append(key, String(item)))
    } else if (value != null) {
      params.append(key, String(value))
    }
  }
  const query = params.toString()
  return location.path + (query ? '?' + query : '') + (location.hash || '')
}

vi.mock('vue-router', () => ({
  createWebHistory: vi.fn(() => ({})),
  createRouter: vi.fn((options: { routes: Array<Record<string, any>> }) => {
    routerHarness.routes = options.routes
    return {
      beforeEach: vi.fn((guard: NavigationGuard) => {
        routerHarness.guard = guard
      }),
      afterEach: vi.fn(),
      onError: vi.fn(),
      resolve: vi.fn((location: Record<string, any>) => ({
        fullPath: resolvedFullPath(location),
      })),
    }
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({ customMenuItems: [] }),
}))

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
  }),
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
  }),
}))

vi.mock('@/api/setup', () => ({
  getSetupStatus: vi.fn(),
}))

function runGuard(options: {
  path: string
  fullPath?: string
  query?: Record<string, unknown>
  hash?: string
  meta?: Record<string, unknown>
}) {
  if (!routerHarness.guard) {
    throw new Error('router guard was not registered')
  }
  const next = vi.fn()
  const navigation = routerHarness.guard({
    path: options.path,
    fullPath: options.fullPath || options.path,
    query: options.query || {},
    hash: options.hash || '',
    name: 'DesktopPaymentRoute',
    params: {},
    meta: { requiresAuth: true, ...options.meta },
  }, {}, next)
  return { navigation, next }
}

describe('desktop payment handoff routing', () => {
  beforeAll(async () => {
    await import('@/router')
  })

  beforeEach(() => {
    authStore.checkAuth.mockClear()
    authStore.restoreCookieSession.mockReset()
    authStore.isAuthenticated = true
    authStore.isAdmin = false
    authStore.isSimpleMode = false
    authStore.hasPendingAuthSession = false
    authStore.token = null
    authStore.user = null
    appStore.backendModeEnabled = false
    appStore.publicSettingsLoaded = true
    appStore.cachedPublicSettings.payment_enabled = true
    appStore.fetchPublicSettings.mockReset()
  })

  it('resolves /payment as an alias of the existing purchase view', () => {
    const purchase = routerHarness.routes.find((route) => route.name === 'PurchaseSubscription')

    expect(purchase?.path).toBe('/purchase')
    expect(purchase?.alias).toBe('/payment')
  })

  it('replaces stale local auth and removes only the desktop marker', async () => {
    authStore.restoreCookieSession.mockResolvedValue(true)
    const { navigation, next } = runGuard({
      path: '/payment',
      fullPath: '/payment?desktop_handoff=1&source=desktop#top',
      query: { desktop_handoff: '1', source: 'desktop' },
      hash: '#top',
      meta: { requiresPayment: true },
    })

    await navigation

    expect(authStore.checkAuth).not.toHaveBeenCalled()
    expect(authStore.restoreCookieSession).toHaveBeenCalledWith({
      replaceClientSession: true,
    })
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith({
      path: '/payment',
      query: { source: 'desktop' },
      hash: '#top',
      replace: true,
    })
  })

  it('restores a cookie-only session before rejecting a protected route', async () => {
    authStore.isAuthenticated = false
    authStore.restoreCookieSession.mockImplementation(async () => {
      authStore.isAuthenticated = true
      return true
    })
    const { navigation, next } = runGuard({ path: '/dashboard' })

    await navigation

    expect(authStore.restoreCookieSession).toHaveBeenCalledWith()
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith()
  })

  it('strips an invalid desktop marker before redirecting to login', async () => {
    authStore.restoreCookieSession.mockImplementation(async () => {
      authStore.isAuthenticated = false
      return false
    })
    const { navigation, next } = runGuard({
      path: '/payment',
      fullPath: '/payment?desktop_handoff=1&source=desktop#top',
      query: { desktop_handoff: '1', source: 'desktop' },
      hash: '#top',
      meta: { requiresPayment: true },
    })

    await navigation

    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith({
      path: '/login',
      query: { redirect: '/payment?source=desktop#top' },
    })
  })
})
