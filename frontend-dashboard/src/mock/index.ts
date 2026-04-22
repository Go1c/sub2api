import { http } from '@/api/http'
import { fixtures } from './fixtures'

const shouldMock =
  import.meta.env.VITE_USE_MOCK !== 'false' && !import.meta.env.VITE_API_BASE_URL

type Handler = (url: string, body?: any) => any

const handlers: Array<{ method: 'GET' | 'POST'; match: RegExp; handler: Handler }> = [
  { method: 'POST', match: /\/auth\/login$/, handler: () => fixtures.loginResponse },
  { method: 'POST', match: /\/auth\/register$/, handler: () => fixtures.loginResponse },
  { method: 'GET', match: /\/user\/profile$/, handler: () => fixtures.user },
  { method: 'GET', match: /\/user\/dashboard\/stats$/, handler: () => fixtures.dashboardStats },
  { method: 'GET', match: /\/user\/dashboard\/invite$/, handler: () => fixtures.dashboardInvite },
  { method: 'GET', match: /\/user\/dashboard\/trend$/, handler: () => fixtures.dashboardTrend },
  { method: 'GET', match: /\/user\/dashboard\/recent$/, handler: () => fixtures.dashboardRecent },
  { method: 'GET', match: /\/user\/keys$/, handler: () => fixtures.keys },
  { method: 'GET', match: /\/user\/usage$/, handler: () => fixtures.usage },
  { method: 'GET', match: /\/user\/groups$/, handler: () => fixtures.groups }
]

function match(method: string, url: string) {
  return handlers.find((h) => h.method === method && h.match.test(url))
}

export function installMockIfNeeded() {
  if (!shouldMock) return
  // eslint-disable-next-line no-console
  console.info('[mock] interceptor enabled — set VITE_API_BASE_URL to use a real backend.')

  http.interceptors.request.use(async (config) => {
    const url = config.url || ''
    const method = (config.method || 'get').toUpperCase()
    const hit = match(method, url)
    if (!hit) return config

    // Simulate latency
    await new Promise((r) => setTimeout(r, 200))

    const data = hit.handler(url, config.data)

    // Short-circuit the request by rejecting with a fake adapter.
    // We do this via axios' adapter override instead, to return 2xx.
    config.adapter = async () => ({
      data: { code: 0, data, msg: 'ok' },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
      request: {}
    })
    return config
  })
}
