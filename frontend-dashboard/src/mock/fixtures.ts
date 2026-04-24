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
      id: '18032145',
      email: 'demo@lumioapi.com',
      name: 'Demo User',
      plan: 'User',
      joinedAt: '2026-01-10',
      balance: 0
    }
  },
  user: {
    id: '18032145',
    email: 'demo@lumioapi.com',
    name: 'Demo User',
    plan: 'User',
    joinedAt: '2026-01-10',
    balance: 0
  },
  dashboardStats: {
    balance: 0,
    todayRequests: 0,
    todayTokens: 0,
    todayTokensIn: 0,
    todayTokensOut: 0,
    totalTokens: 0,
    totalTokensIn: 0,
    totalTokensOut: 0,
    todaySpend: 0,
    totalSpend: 0,
    activeKeys: 0,
    rpm: 0,
    tpm: 0,
    latencyMs: 0
  },
  dashboardInvite: {
    invited: 0,
    totalBonus: 0,
    monthBonus: 0,
    code: 'YY62PX4K'
  },
  dashboardTrend: {
    labels: [6, 5, 4, 3, 2, 1, 0].map(days),
    requests: [0, 0, 0, 0, 0, 0, 0],
    tokens: [0, 0, 0, 0, 0, 0, 0]
  },
  dashboardRecent: [] as Array<{ id: string; time: string; model: string; tokens: number; status: string }>,
  keys: [] as Array<{ id: string; name: string; prefix: string; created: string; status: string }>,
  usage: {
    labels: Array.from({ length: 30 }, (_, i) => days(29 - i)),
    requests: Array.from({ length: 30 }, () => 0),
    tokens: Array.from({ length: 30 }, () => 0),
    totals: { requests: 0, tokens: 0, cost: 0 }
  },
  groups: [] as Array<{ id: string; name: string; capacity: number; members: number; usage: string }>,

  // Public pricing — keep in sync with backend DefaultPublicModelPricingConfig.
  publicPricing: {
    currency: 'CNY',
    unit: '百万 tokens',
    rateNote: '价格以人民币（¥）计价，单位为百万 tokens；折扣展示可在管理员后台调整。',
    rows: [
      { model: 'Claude Opus 4.7', group: '企业稳定版', multiplier: '3.3%', inputPrice: 3.50, outputPrice: 17.50, officialInput: 15, officialOutput: 75, discount: '3.3%', openClaw: false, enabled: true },
      { model: 'Claude Sonnet 4.6', group: '企业稳定版', multiplier: '3.3%', inputPrice: 0.70, outputPrice: 3.50, officialInput: 3, officialOutput: 15, discount: '3.3%', openClaw: false, enabled: true },
      { model: 'GPT5.5', group: 'OpenAI', multiplier: '1.0%', inputPrice: 0.18, outputPrice: 0.70, officialInput: 2.5, officialOutput: 10, discount: '1.0%', openClaw: false, enabled: true },
      { model: 'GPT5.4', group: 'OpenAI', multiplier: '1.0%', inputPrice: 0.18, outputPrice: 1.05, officialInput: 2.5, officialOutput: 15, discount: '1.0%', openClaw: false, enabled: true }
    ]
  }
}
