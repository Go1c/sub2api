/**
 * Lottery Store (frontend-only mock with localStorage persistence)
 *
 * This is the first phase of the lottery feature: admin can create campaigns,
 * users see a popup wheel on login and draw exactly once per active campaign.
 * Backend API will replace the in-memory + localStorage layer in a follow-up.
 */

import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

export interface WheelSegment {
  label: string
  isPrize: boolean
}

export interface Winner {
  userId: number
  userName: string
  code: string
  drawnAt: string
}

export interface Campaign {
  id: string
  name: string
  subtitle: string
  prizeCount: number
  maxParticipants: number
  codes: string[]
  joined: number
  winners: Winner[]
  segments: WheelSegment[]
  status: 'active' | 'finished'
  createdAt: string
  /** user ids that already drew (for popup gating) */
  drawnUserIds: number[]
}

export interface DrawResult {
  won: boolean
  index: number
  label: string
  code?: string
}

const STORAGE_KEY = 'lottery_demo_v1'

function loadFromStorage(): Campaign[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed as Campaign[]
  } catch {
    return []
  }
}

function saveToStorage(campaigns: Campaign[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(campaigns))
  } catch (e) {
    console.warn('lottery: failed to persist', e)
  }
}

function buildSegments(prizeCount: number, totalSlots = 8): WheelSegment[] {
  const slots = Math.max(prizeCount + 2, totalSlots)
  const segs: WheelSegment[] = []
  for (let i = 0; i < slots; i++) {
    segs.push(
      i < prizeCount
        ? { label: `奖品 ${i + 1}`, isPrize: true }
        : { label: '谢谢参与', isPrize: false }
    )
  }
  const order = [...Array(slots).keys()]
  for (let i = order.length - 1; i > 0; i--) {
    const j = Math.floor(((i * 9301 + 49297) % 233280) / 233280 * (i + 1))
    ;[order[i], order[j]] = [order[j], order[i]]
  }
  return order.map(idx => segs[idx])
}

function nowIso(): string {
  return new Date().toISOString().slice(0, 16).replace('T', ' ')
}

export const useLotteryStore = defineStore('lottery', () => {
  const campaigns = ref<Campaign[]>(loadFromStorage())
  const initialized = ref(false)

  function seedIfEmpty() {
    if (initialized.value) return
    initialized.value = true
    if (campaigns.value.length === 0) {
      campaigns.value = [
        {
          id: 'seed-may',
          name: '五月幸运转盘',
          subtitle: '登录就有机会，转一转赢取兑换码',
          prizeCount: 5,
          maxParticipants: 20,
          codes: ['LUCK-MAY-A1B2', 'LUCK-MAY-C3D4', 'LUCK-MAY-E5F6', 'LUCK-MAY-G7H8', 'LUCK-MAY-I9J0'],
          joined: 0,
          winners: [],
          segments: buildSegments(5, 8),
          status: 'active',
          createdAt: nowIso(),
          drawnUserIds: []
        }
      ]
      saveToStorage(campaigns.value)
    }
  }

  const activeCampaign = computed(() => campaigns.value.find(c => c.status === 'active') || null)

  function getActiveForUser(userId: number | null | undefined): Campaign | null {
    if (!userId) return null
    const c = activeCampaign.value
    if (!c) return null
    if (c.drawnUserIds.includes(userId)) return null
    if (c.joined >= c.maxParticipants) return null
    return c
  }

  function hasDrawn(userId: number, campaignId: string): boolean {
    const c = campaigns.value.find(x => x.id === campaignId)
    return !!c?.drawnUserIds.includes(userId)
  }

  function createCampaign(input: {
    name: string
    subtitle: string
    prizeCount: number
    maxParticipants: number
    codes: string[]
  }): Campaign {
    // Archive any existing active campaign to maintain "1 active at a time" invariant.
    campaigns.value.forEach(c => {
      if (c.status === 'active') c.status = 'finished'
    })
    const c: Campaign = {
      id: `c-${Date.now()}`,
      name: input.name.trim(),
      subtitle: input.subtitle.trim() || '登录就有机会，转一转赢取兑换码',
      prizeCount: input.prizeCount,
      maxParticipants: input.maxParticipants,
      codes: input.codes.slice(0, input.prizeCount),
      joined: 0,
      winners: [],
      segments: buildSegments(input.prizeCount, 8),
      status: 'active',
      createdAt: nowIso(),
      drawnUserIds: []
    }
    campaigns.value.unshift(c)
    saveToStorage(campaigns.value)
    return c
  }

  async function draw(
    userId: number,
    userName: string,
    campaignId: string
  ): Promise<DrawResult> {
    const c = campaigns.value.find(x => x.id === campaignId)
    if (!c) throw new Error('campaign not found')
    if (c.status !== 'active') throw new Error('campaign finished')
    if (c.drawnUserIds.includes(userId)) throw new Error('already drew')
    if (c.joined >= c.maxParticipants) throw new Error('campaign full')

    // Server-side fairness simulation: P(win) = remainingPrizes / remainingSlots
    const remainingPrizes = c.prizeCount - c.winners.length
    const remainingSlots = c.maxParticipants - c.joined
    const winProb = remainingSlots > 0 ? remainingPrizes / remainingSlots : 0
    const won = Math.random() < winProb

    let index: number
    let label: string
    let code: string | undefined

    if (won) {
      const prizeSegs = c.segments
        .map((s, i) => ({ s, i }))
        .filter(({ s }) => s.isPrize)
      const pick = prizeSegs[Math.floor(Math.random() * prizeSegs.length)]
      index = pick.i
      label = pick.s.label
      const claimed = new Set(c.winners.map(w => w.code))
      code = c.codes.find(x => !claimed.has(x)) || c.codes[0]
      c.winners.push({ userId, userName, code, drawnAt: nowIso() })
    } else {
      const blanks = c.segments
        .map((s, i) => ({ s, i }))
        .filter(({ s }) => !s.isPrize)
      const pick = blanks[Math.floor(Math.random() * blanks.length)]
      index = pick.i
      label = pick.s.label
    }

    c.drawnUserIds.push(userId)
    c.joined += 1
    if (c.winners.length >= c.prizeCount || c.joined >= c.maxParticipants) {
      c.status = 'finished'
    }
    saveToStorage(campaigns.value)

    // simulate network latency
    await new Promise(r => setTimeout(r, 300))
    return { won, index, label, code }
  }

  function unclaimedCodes(c: Campaign): string[] {
    const used = new Set(c.winners.map(w => w.code))
    return c.codes.filter(code => !used.has(code))
  }

  function resetAll() {
    campaigns.value = []
    initialized.value = false
    saveToStorage(campaigns.value)
  }

  return {
    campaigns,
    activeCampaign,
    seedIfEmpty,
    getActiveForUser,
    hasDrawn,
    createCampaign,
    draw,
    unclaimedCodes,
    resetAll
  }
})
