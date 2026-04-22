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

  // Public pricing — mirrors screenshot-captured pricing from dragoncode.codes.
  // Keep in sync with backend channel_model_pricing once the /pricing/public
  // endpoint is implemented; until then, this file is the source of truth.
  publicPricing: {
    currency: 'CNY',
    unit: '百万 tokens',
    rateNote: '我们的价格以人民币（¥）计价 · 官方原价以美元（$）标注 · 汇率按 1:7 折算（单位：百万 tokens）',
    rows: [
      { model: 'Opus 4.6', group: '企业稳定版', multiplier: '2.5x', inputPrice: 12.50, outputPrice: 62.50, officialInput: 35, officialOutput: 175, discount: '3.6折', openClaw: false },
      { model: 'Sonnet 4.6', group: '企业稳定版', multiplier: '2.5x', inputPrice: 7.50, outputPrice: 37.50, officialInput: 21, officialOutput: 105, discount: '3.6折', openClaw: false },
      { model: 'Opus 4.6', group: 'kiro 逆向', multiplier: '0.65x', inputPrice: 3.25, outputPrice: 16.25, officialInput: 35, officialOutput: 175, discount: '0.9折', openClaw: true },
      { model: 'Sonnet 4.6', group: 'kiro 逆向', multiplier: '0.65x', inputPrice: 1.95, outputPrice: 9.75, officialInput: 21, officialOutput: 105, discount: '0.9折', openClaw: true },
      { model: 'Opus 4.6', group: '反重力逆向', multiplier: '0.85x', inputPrice: 4.25, outputPrice: 21.25, officialInput: 35, officialOutput: 175, discount: '1.2折', openClaw: true },
      { model: 'Sonnet 4.6', group: '反重力逆向', multiplier: '0.85x', inputPrice: 2.55, outputPrice: 12.75, officialInput: 21, officialOutput: 105, discount: '1.2折', openClaw: true },
      { model: '5.4', group: 'codex', multiplier: '0.5x', inputPrice: 1.25, outputPrice: 7.50, officialInput: 21, officialOutput: 105, discount: '0.7折', openClaw: true },
      { model: '5.3 Codex', group: 'codex', multiplier: '0.5x', inputPrice: 0.87, outputPrice: 7.00, officialInput: 12.25, officialOutput: 98, discount: '0.7折', openClaw: true }
    ]
  }
}
