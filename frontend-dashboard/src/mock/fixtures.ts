const today = new Date()
const days = (n: number) => {
  const d = new Date(today)
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}

export const fixtures = {
  loginResponse: {
    token: 'mock-token-' + Math.random().toString(36).slice(2),
    user: {
      id: 'u_001',
      email: 'demo@dragoncode.codes',
      name: 'Demo User',
      plan: 'Pro',
      joinedAt: '2026-01-10',
      balance: 128.5
    }
  },
  user: {
    id: 'u_001',
    email: 'demo@dragoncode.codes',
    name: 'Demo User',
    plan: 'Pro',
    joinedAt: '2026-01-10',
    balance: 128.5
  },
  dashboardStats: {
    balance: 128.5,
    todayRequests: 1284,
    todayTokens: 356_912,
    activeKeys: 3
  },
  dashboardTrend: {
    labels: [6, 5, 4, 3, 2, 1, 0].map(days),
    requests: [820, 1020, 930, 1140, 1205, 1100, 1284],
    tokens: [210_000, 245_000, 231_000, 288_000, 301_200, 279_500, 356_912]
  },
  dashboardRecent: [
    { id: 'r1', time: '10:24', model: 'claude-sonnet-4-6', tokens: 1832, status: 'success' },
    { id: 'r2', time: '10:19', model: 'claude-opus-4-7', tokens: 4210, status: 'success' },
    { id: 'r3', time: '10:03', model: 'gemini-2.5-pro', tokens: 928, status: 'success' },
    { id: 'r4', time: '09:58', model: 'claude-haiku-4-5', tokens: 412, status: 'error' },
    { id: 'r5', time: '09:41', model: 'claude-sonnet-4-6', tokens: 2204, status: 'success' }
  ],
  keys: [
    { id: 'k1', name: '本地开发', prefix: 'sk-dc-ab12', created: '2026-03-12', status: 'active' },
    { id: 'k2', name: '服务端', prefix: 'sk-dc-cd34', created: '2026-02-20', status: 'active' },
    { id: 'k3', name: '已弃用', prefix: 'sk-dc-ef56', created: '2026-01-05', status: 'disabled' }
  ],
  usage: {
    labels: Array.from({ length: 30 }, (_, i) => days(29 - i)),
    requests: Array.from({ length: 30 }, () => Math.floor(600 + Math.random() * 900)),
    tokens: Array.from({ length: 30 }, () => Math.floor(150_000 + Math.random() * 250_000)),
    totals: { requests: 32_845, tokens: 8_912_450, cost: 42.18 }
  },
  groups: [
    { id: 'g1', name: 'Claude Code 主组', capacity: 80, members: 12, usage: '64%' },
    { id: 'g2', name: 'Gemini 备用组', capacity: 50, members: 8, usage: '38%' }
  ]
}
