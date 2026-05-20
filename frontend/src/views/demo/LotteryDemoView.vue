<template>
  <div class="relative min-h-screen overflow-hidden bg-[#0a0a14] text-slate-100">
    <!-- Background ambient glow -->
    <div class="pointer-events-none absolute inset-0 -z-0">
      <div class="absolute -top-40 left-1/4 h-[42rem] w-[42rem] rounded-full bg-blue-600/20 blur-[120px]"></div>
      <div class="absolute -top-20 right-1/4 h-[36rem] w-[36rem] rounded-full bg-fuchsia-600/20 blur-[120px]"></div>
      <div class="absolute inset-0 bg-[radial-gradient(ellipse_at_center,transparent_0%,#0a0a14_70%)]"></div>
    </div>

    <div class="relative z-10 mx-auto max-w-6xl px-6 py-12">
      <!-- Header / hero -->
      <header class="mb-10 text-center">
        <div class="inline-flex items-center gap-2 rounded-full bg-white/5 px-4 py-1.5 text-xs font-medium text-slate-200 ring-1 ring-white/10 backdrop-blur">
          <span class="inline-block h-1.5 w-1.5 rounded-full bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]"></span>
          <span>DEMO 演示模式 · 纯前端 Mock</span>
        </div>
        <h1 class="mt-5 text-4xl font-extrabold tracking-tight text-white sm:text-5xl">
          抽奖功能
          <span class="block bg-gradient-to-r from-blue-400 via-violet-400 to-fuchsia-400 bg-clip-text text-transparent">
            幸运转盘体验
          </span>
        </h1>
        <p class="mx-auto mt-4 max-w-2xl text-sm text-slate-400 sm:text-base">
          管理员创建抽奖活动 → 用户登录时弹窗转盘抽奖 → 中奖直接显示兑换码。
        </p>
      </header>

      <!-- Tabs (capsule) -->
      <div class="mb-8 flex justify-center">
        <div class="inline-flex rounded-full bg-white/5 p-1 ring-1 ring-white/10 backdrop-blur">
          <button
            v-for="t in tabs"
            :key="t.key"
            class="rounded-full px-5 py-1.5 text-sm font-medium transition-all"
            :class="activeTab === t.key
              ? 'bg-gradient-to-r from-blue-500 to-violet-600 text-white shadow-[0_4px_16px_-4px_rgba(99,102,241,0.6)]'
              : 'text-slate-300 hover:text-white'"
            @click="activeTab = t.key"
          >
            {{ t.label }}
          </button>
        </div>
      </div>

      <!-- ============ USER VIEW ============ -->
      <section v-if="activeTab === 'user'" class="grid gap-6 md:grid-cols-[1.1fr_1fr]">
        <div class="relative overflow-hidden rounded-3xl bg-white/[0.03] p-6 ring-1 ring-white/10 backdrop-blur">
          <div class="pointer-events-none absolute -right-20 -top-20 h-48 w-48 rounded-full bg-blue-500/20 blur-3xl"></div>
          <h2 class="relative text-lg font-bold text-white">模拟登录场景</h2>
          <p class="relative mt-1 text-sm text-slate-400">
            点击下方按钮模拟用户登录后弹出抽奖窗口（实际项目里会在登录成功后自动检测有效活动并弹出）。
          </p>

          <div class="relative mt-5 space-y-3">
            <div class="rounded-2xl bg-black/30 p-4 ring-1 ring-white/10">
              <div class="text-xs uppercase tracking-wider text-slate-500">当前活动</div>
              <div class="mt-1.5 flex flex-wrap items-baseline gap-2">
                <span class="text-base font-semibold text-white">
                  {{ activeCampaign?.name || '暂无进行中的活动' }}
                </span>
                <span v-if="activeCampaign" class="font-mono text-xs text-slate-400">
                  {{ activeCampaign.joined }}/{{ activeCampaign.maxParticipants }} 已参与 ·
                  {{ activeCampaign.prizeCount }} 个奖品
                </span>
              </div>
            </div>

            <button
              type="button"
              :disabled="!activeCampaign || userAlreadyDrew"
              class="group relative w-full overflow-hidden rounded-full bg-gradient-to-r from-blue-500 via-violet-500 to-fuchsia-500 px-5 py-3 text-sm font-semibold text-white shadow-[0_10px_30px_-6px_rgba(139,92,246,0.5)] transition-transform hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:translate-y-0"
              @click="openDialog = true"
            >
              <span class="relative">
                {{ userAlreadyDrew ? '你已经抽过这场活动了' : '模拟登录，弹出抽奖' }}
              </span>
            </button>

            <button
              v-if="userAlreadyDrew"
              type="button"
              class="w-full rounded-full bg-white/5 px-5 py-2.5 text-sm font-medium text-slate-200 ring-1 ring-white/10 hover:bg-white/10"
              @click="resetUserDraw"
            >
              重置我的抽奖记录（仅演示用）
            </button>
          </div>

          <div
            v-if="lastUserResult"
            class="relative mt-6 overflow-hidden rounded-2xl bg-white/[0.03] p-4 ring-1 ring-white/10"
          >
            <div class="pointer-events-none absolute inset-0 bg-gradient-to-br from-blue-500/10 via-violet-500/10 to-fuchsia-500/10"></div>
            <div class="relative">
              <div class="text-xs text-blue-300">上次结果</div>
              <div class="mt-1 text-base font-semibold text-white">
                {{ lastUserResult.won ? `🎉 中奖：${lastUserResult.label}` : '未中奖' }}
              </div>
              <div v-if="lastUserResult.code" class="mt-1 font-mono text-sm text-fuchsia-300">
                {{ lastUserResult.code }}
              </div>
            </div>
          </div>
        </div>

        <div class="rounded-3xl bg-white/[0.03] p-6 ring-1 ring-white/10 backdrop-blur">
          <h2 class="text-lg font-bold text-white">交互流程</h2>
          <ul class="mt-3 space-y-3 text-sm text-slate-300">
            <li class="flex gap-3">
              <span class="mt-0.5 inline-flex h-5 w-5 flex-none items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-violet-600 text-[10px] font-bold text-white">1</span>
              <span>登录后检测是否有进行中的抽奖活动且本人未参与</span>
            </li>
            <li class="flex gap-3">
              <span class="mt-0.5 inline-flex h-5 w-5 flex-none items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-violet-600 text-[10px] font-bold text-white">2</span>
              <span>弹窗显示转盘 + 实时统计（奖品数 / 已参与 / 剩余）</span>
            </li>
            <li class="flex gap-3">
              <span class="mt-0.5 inline-flex h-5 w-5 flex-none items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-violet-600 text-[10px] font-bold text-white">3</span>
              <span>点击中央"开始抽奖"，转盘旋转 6 圈后停在结果扇区</span>
            </li>
            <li class="flex gap-3">
              <span class="mt-0.5 inline-flex h-5 w-5 flex-none items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-violet-600 text-[10px] font-bold text-white">4</span>
              <span>中奖：显示兑换码 + 一键复制；未中奖：友好提示</span>
            </li>
            <li class="flex gap-3">
              <span class="mt-0.5 inline-flex h-5 w-5 flex-none items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-violet-600 text-[10px] font-bold text-white">5</span>
              <span>抽奖过程不可关闭，避免误操作</span>
            </li>
          </ul>
        </div>
      </section>

      <!-- ============ ADMIN VIEW ============ -->
      <section v-else-if="activeTab === 'admin'" class="space-y-6">
        <!-- Sub tabs (capsule, smaller) -->
        <div class="flex justify-center">
          <div class="inline-flex rounded-full bg-white/5 p-1 ring-1 ring-white/10 backdrop-blur">
            <button
              v-for="t in adminSubTabs"
              :key="t.key"
              class="rounded-full px-4 py-1.5 text-sm font-medium transition-all"
              :class="adminSubTab === t.key
                ? 'bg-white/10 text-white ring-1 ring-white/15'
                : 'text-slate-400 hover:text-white'"
              @click="adminSubTab = t.key"
            >
              {{ t.label }}
            </button>
          </div>
        </div>

        <!-- Create form -->
        <div v-if="adminSubTab === 'create'" class="rounded-3xl bg-white/[0.03] p-6 ring-1 ring-white/10 backdrop-blur sm:p-8">
          <h2 class="text-xl font-bold text-white">新建抽奖</h2>
          <p class="mt-1 text-sm text-slate-400">填入活动信息和兑换码，发布后用户登录即可弹窗抽奖。</p>

          <div class="mt-6 grid gap-5 md:grid-cols-2">
            <label class="block">
              <span class="text-xs font-medium uppercase tracking-wider text-slate-400">活动名称</span>
              <input
                v-model="form.name"
                type="text"
                placeholder="例如：五月幸运转盘"
                class="mt-2 w-full rounded-xl bg-white/5 px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 ring-1 ring-white/10 transition-all focus:bg-white/[0.07] focus:outline-none focus:ring-2 focus:ring-blue-500/60"
              />
            </label>
            <label class="block">
              <span class="text-xs font-medium uppercase tracking-wider text-slate-400">活动副标题</span>
              <input
                v-model="form.subtitle"
                type="text"
                placeholder="登录就有机会，转一转赢取兑换码"
                class="mt-2 w-full rounded-xl bg-white/5 px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 ring-1 ring-white/10 transition-all focus:bg-white/[0.07] focus:outline-none focus:ring-2 focus:ring-blue-500/60"
              />
            </label>

            <label class="block">
              <span class="text-xs font-medium uppercase tracking-wider text-slate-400">中奖数量</span>
              <input
                v-model.number="form.prizeCount"
                type="number"
                min="1"
                class="mt-2 w-full rounded-xl bg-white/5 px-3.5 py-2.5 font-mono text-sm text-white ring-1 ring-white/10 transition-all focus:bg-white/[0.07] focus:outline-none focus:ring-2 focus:ring-blue-500/60"
              />
            </label>
            <label class="block">
              <span class="text-xs font-medium uppercase tracking-wider text-slate-400">最大参与人数</span>
              <input
                v-model.number="form.maxParticipants"
                type="number"
                min="1"
                class="mt-2 w-full rounded-xl bg-white/5 px-3.5 py-2.5 font-mono text-sm text-white ring-1 ring-white/10 transition-all focus:bg-white/[0.07] focus:outline-none focus:ring-2 focus:ring-blue-500/60"
              />
            </label>
          </div>

          <div class="mt-5">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <span class="text-xs font-medium uppercase tracking-wider text-slate-400">
                兑换码 · 每行一个
              </span>
              <span
                class="font-mono text-xs"
                :class="codeLines.length >= form.prizeCount ? 'text-emerald-400' : 'text-amber-300'"
              >
                已填 {{ codeLines.length }} / 需要 {{ form.prizeCount }}
              </span>
            </div>
            <textarea
              v-model="form.codesRaw"
              rows="6"
              placeholder="LUCK-001-ABCD&#10;LUCK-002-EFGH&#10;LUCK-003-IJKL"
              class="mt-2 w-full rounded-xl bg-white/5 px-3.5 py-2.5 font-mono text-sm text-white placeholder:text-slate-500 ring-1 ring-white/10 transition-all focus:bg-white/[0.07] focus:outline-none focus:ring-2 focus:ring-blue-500/60"
            ></textarea>
          </div>

          <div class="mt-6 flex flex-wrap items-center gap-3">
            <button
              type="button"
              class="rounded-full bg-gradient-to-r from-blue-500 via-violet-500 to-fuchsia-500 px-6 py-2.5 text-sm font-semibold text-white shadow-[0_10px_30px_-6px_rgba(139,92,246,0.5)] transition-transform hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:translate-y-0"
              :disabled="!canSubmit"
              @click="submitCampaign"
            >
              发布抽奖
            </button>
            <button
              type="button"
              class="rounded-full bg-white/5 px-5 py-2.5 text-sm font-medium text-slate-200 ring-1 ring-white/10 hover:bg-white/10"
              @click="resetForm"
            >
              重置
            </button>
            <span v-if="formError" class="text-xs text-rose-300">{{ formError }}</span>
            <span v-if="formSaved" class="text-xs text-emerald-300">✓ 已发布到下方"历史抽奖记录"</span>
          </div>
        </div>

        <!-- History -->
        <div v-else-if="adminSubTab === 'history'" class="overflow-hidden rounded-3xl bg-white/[0.03] ring-1 ring-white/10 backdrop-blur">
          <div class="border-b border-white/10 px-6 py-5">
            <h2 class="text-xl font-bold text-white">历史抽奖记录</h2>
            <p class="mt-1 text-sm text-slate-400">所有历史活动、参与人数、中奖情况一览。</p>
          </div>

          <div v-if="campaigns.length === 0" class="px-6 py-16 text-center">
            <div class="text-sm text-slate-400">暂无抽奖活动，先去"新建抽奖"创建一个吧。</div>
          </div>

          <ul v-else class="divide-y divide-white/5">
            <li v-for="c in campaigns" :key="c.id" class="px-6 py-5">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <div class="flex items-center gap-2">
                    <span class="text-base font-semibold text-white">{{ c.name }}</span>
                    <span
                      class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ring-1"
                      :class="c.status === 'active'
                        ? 'bg-emerald-500/10 text-emerald-300 ring-emerald-500/30'
                        : 'bg-white/5 text-slate-400 ring-white/10'"
                    >
                      <span
                        v-if="c.status === 'active'"
                        class="inline-block h-1.5 w-1.5 rounded-full bg-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.8)]"
                      ></span>
                      {{ statusLabel(c) }}
                    </span>
                  </div>
                  <div class="mt-1 text-xs text-slate-500">{{ formatDate(c.createdAt) }}</div>
                </div>
                <div class="flex flex-wrap items-center gap-3 text-right">
                  <div class="font-mono text-xs text-slate-400">
                    <span class="font-bold text-white">{{ c.joined }}</span>/{{ c.maxParticipants }} 参与
                  </div>
                  <div class="font-mono text-xs text-slate-400">
                    <span class="font-bold text-fuchsia-300">{{ c.winners.length }}</span>/{{ c.prizeCount }} 中奖
                  </div>
                  <button
                    type="button"
                    class="rounded-full bg-white/5 px-3 py-1.5 text-xs font-medium text-slate-200 ring-1 ring-white/10 hover:bg-white/10"
                    @click="toggleExpand(c.id)"
                  >
                    {{ expanded === c.id ? '收起' : '详情' }}
                  </button>
                </div>
              </div>

              <div v-if="expanded === c.id" class="mt-4 grid gap-4 rounded-2xl bg-black/30 p-4 ring-1 ring-white/5 md:grid-cols-2">
                <div>
                  <div class="text-xs font-semibold uppercase tracking-wider text-slate-400">中奖名单</div>
                  <ul class="mt-2 space-y-1.5 text-xs">
                    <li
                      v-for="(w, i) in c.winners"
                      :key="i"
                      class="flex items-center justify-between gap-3 rounded-lg bg-white/5 px-3 py-2 ring-1 ring-white/10"
                    >
                      <span class="text-slate-200">{{ w.userName }}</span>
                      <span class="font-mono text-fuchsia-300">{{ w.code }}</span>
                    </li>
                    <li v-if="c.winners.length === 0" class="text-slate-500">尚无中奖</li>
                  </ul>
                </div>
                <div>
                  <div class="text-xs font-semibold uppercase tracking-wider text-slate-400">未派发兑换码</div>
                  <ul class="mt-2 space-y-1.5 text-xs">
                    <li
                      v-for="(code, i) in unclaimedCodes(c)"
                      :key="i"
                      class="rounded-lg bg-white/5 px-3 py-2 font-mono text-slate-200 ring-1 ring-white/10"
                    >
                      {{ code }}
                    </li>
                    <li v-if="unclaimedCodes(c).length === 0" class="text-slate-500">兑换码已全部派发</li>
                  </ul>
                </div>
              </div>
            </li>
          </ul>
        </div>
      </section>
    </div>

    <!-- User-side lottery dialog -->
    <LotteryDialog
      v-if="activeCampaign"
      :open="openDialog"
      :campaign-title="activeCampaign.name"
      :subtitle="activeCampaign.subtitle"
      :prize-count="activeCampaign.prizeCount"
      :max-participants="activeCampaign.maxParticipants"
      :joined="activeCampaign.joined"
      :segments="activeCampaign.segments"
      :draw-fn="drawForCurrentCampaign"
      @close="openDialog = false"
      @drawn="onDrawn"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import LotteryDialog, { type DrawResult } from '@/components/lottery/LotteryDialog.vue'
import type { WheelSegment } from '@/components/lottery/LotteryWheel.vue'

interface Winner {
  userName: string
  code: string
  drawnAt: string
}
interface Campaign {
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
}

const tabs = [
  { key: 'user', label: '用户视角' },
  { key: 'admin', label: '管理员视角' }
] as const
const adminSubTabs = [
  { key: 'create', label: '新建抽奖' },
  { key: 'history', label: '历史抽奖记录' }
] as const
const activeTab = ref<'user' | 'admin'>('user')
const adminSubTab = ref<'create' | 'history'>('create')

function buildSegments(prizeCount: number, totalSlots = 8): WheelSegment[] {
  const slots = Math.max(prizeCount + 2, totalSlots)
  const segs: WheelSegment[] = []
  for (let i = 0; i < slots; i++) {
    if (i < prizeCount) {
      segs.push({ label: `奖品 ${i + 1}`, isPrize: true })
    } else {
      segs.push({ label: '谢谢参与', isPrize: false })
    }
  }
  const order = [...Array(slots).keys()]
  for (let i = order.length - 1; i > 0; i--) {
    const j = Math.floor(((i * 9301 + 49297) % 233280) / 233280 * (i + 1))
    ;[order[i], order[j]] = [order[j], order[i]]
  }
  return order.map(idx => segs[idx])
}

const campaigns = ref<Campaign[]>([
  {
    id: 'demo-1',
    name: '五月幸运转盘',
    subtitle: '登录就有机会，转一转赢取兑换码',
    prizeCount: 5,
    maxParticipants: 20,
    codes: ['LUCK-MAY-A1B2', 'LUCK-MAY-C3D4', 'LUCK-MAY-E5F6', 'LUCK-MAY-G7H8', 'LUCK-MAY-I9J0'],
    joined: 7,
    winners: [
      { userName: 'alice', code: 'LUCK-MAY-A1B2', drawnAt: '2026-05-18 10:21' },
      { userName: 'bob', code: 'LUCK-MAY-C3D4', drawnAt: '2026-05-19 14:08' }
    ],
    segments: buildSegments(5, 8),
    status: 'active',
    createdAt: '2026-05-15 09:00'
  }
])

const activeCampaign = computed(() => campaigns.value.find(c => c.status === 'active') || null)

const openDialog = ref(false)
const userAlreadyDrew = ref(false)
const lastUserResult = ref<DrawResult | null>(null)

async function drawForCurrentCampaign(): Promise<DrawResult> {
  const c = activeCampaign.value!
  const remainingPrizes = c.prizeCount - c.winners.length
  const remainingSlots = c.maxParticipants - c.joined
  const winProb = remainingSlots > 0 ? remainingPrizes / remainingSlots : 0
  const won = Math.random() < winProb

  let index: number
  let label: string
  let code: string | undefined

  if (won) {
    const prizeSegs = c.segments.map((s, i) => ({ s, i })).filter(({ s }) => s.isPrize)
    const pick = prizeSegs[Math.floor(Math.random() * prizeSegs.length)]
    index = pick.i
    label = pick.s.label
    const claimed = new Set(c.winners.map(w => w.code))
    code = c.codes.find(x => !claimed.has(x)) || c.codes[0]
  } else {
    const blanks = c.segments.map((s, i) => ({ s, i })).filter(({ s }) => !s.isPrize)
    const pick = blanks[Math.floor(Math.random() * blanks.length)]
    index = pick.i
    label = pick.s.label
  }

  await new Promise(r => setTimeout(r, 350))
  return { won, index, label, code }
}

function onDrawn(r: DrawResult) {
  const c = activeCampaign.value
  if (!c) return
  c.joined += 1
  if (r.won && r.code) {
    c.winners.push({
      userName: '你',
      code: r.code,
      drawnAt: new Date().toISOString().slice(0, 16).replace('T', ' ')
    })
  }
  userAlreadyDrew.value = true
  lastUserResult.value = r
  if (c.winners.length >= c.prizeCount || c.joined >= c.maxParticipants) {
    c.status = 'finished'
  }
}

function resetUserDraw() {
  userAlreadyDrew.value = false
  lastUserResult.value = null
}

const form = ref({
  name: '',
  subtitle: '登录就有机会，转一转赢取兑换码',
  prizeCount: 5,
  maxParticipants: 20,
  codesRaw: ''
})
const formError = ref('')
const formSaved = ref(false)

const codeLines = computed(() =>
  form.value.codesRaw.split(/\r?\n/).map(s => s.trim()).filter(Boolean)
)
const canSubmit = computed(
  () =>
    form.value.name.trim().length > 0 &&
    form.value.prizeCount > 0 &&
    form.value.maxParticipants >= form.value.prizeCount &&
    codeLines.value.length >= form.value.prizeCount
)

function submitCampaign() {
  formError.value = ''
  formSaved.value = false
  if (!canSubmit.value) {
    formError.value = '请检查必填项与兑换码数量'
    return
  }
  campaigns.value.forEach(c => {
    if (c.status === 'active') c.status = 'finished'
  })
  const codes = codeLines.value.slice(0, form.value.prizeCount)
  const newCampaign: Campaign = {
    id: `demo-${Date.now()}`,
    name: form.value.name.trim(),
    subtitle: form.value.subtitle.trim() || '登录就有机会，转一转赢取兑换码',
    prizeCount: form.value.prizeCount,
    maxParticipants: form.value.maxParticipants,
    codes,
    joined: 0,
    winners: [],
    segments: buildSegments(form.value.prizeCount, 8),
    status: 'active',
    createdAt: new Date().toISOString().slice(0, 16).replace('T', ' ')
  }
  campaigns.value.unshift(newCampaign)
  formSaved.value = true
  resetUserDraw()
}

function resetForm() {
  form.value = {
    name: '',
    subtitle: '登录就有机会，转一转赢取兑换码',
    prizeCount: 5,
    maxParticipants: 20,
    codesRaw: ''
  }
  formError.value = ''
  formSaved.value = false
}

const expanded = ref<string | null>(null)
function toggleExpand(id: string) {
  expanded.value = expanded.value === id ? null : id
}
function statusLabel(c: Campaign): string {
  return c.status === 'active' ? '进行中' : '已结束'
}
function formatDate(s: string): string {
  return s
}
function unclaimedCodes(c: Campaign): string[] {
  const used = new Set(c.winners.map(w => w.code))
  return c.codes.filter(code => !used.has(code))
}
</script>
