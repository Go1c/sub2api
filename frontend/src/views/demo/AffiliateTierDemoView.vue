<template>
  <div class="min-h-screen bg-[#07111f] text-white">
    <header class="border-b border-white/10 bg-[#111c2d]/90">
      <div class="mx-auto flex max-w-[112rem] items-center justify-between px-6 py-4">
        <div>
          <h1 class="text-2xl font-semibold">邀请返利</h1>
          <p class="mt-1 text-sm text-slate-400">阶梯式邀请返利制度 · 后台配置样例</p>
        </div>
        <div class="rounded-full border border-cyan-400/30 bg-cyan-400/10 px-4 py-2 text-sm font-semibold text-cyan-200">
          Admin Config Demo
        </div>
      </div>
    </header>

    <main class="mx-auto max-w-[112rem] space-y-8 px-6 py-10">
      <section class="grid gap-5 md:grid-cols-2 xl:grid-cols-4">
        <div
          v-for="card in statCards"
          :key="card.label"
          class="rounded-2xl border border-white/10 bg-[#131f31] p-7 shadow-[0_20px_60px_rgba(0,0,0,0.22)]"
        >
          <p class="text-base font-medium text-slate-400">{{ card.label }}</p>
          <p :class="['mt-5 text-4xl font-semibold', card.accent ? 'text-teal-300' : 'text-white']">
            {{ card.value }}
          </p>
          <p v-if="card.hint" class="mt-4 text-sm leading-6 text-slate-500">{{ card.hint }}</p>
        </div>
      </section>

      <section class="rounded-2xl border border-white/10 bg-[#121d2f] p-7">
        <div class="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 class="text-2xl font-semibold">邀请返利</h2>
            <p class="mt-2 text-base text-slate-400">邀请新用户注册，并将返利额度转入账户余额</p>
          </div>
          <button class="rounded-xl bg-teal-400 px-6 py-3 text-sm font-semibold text-slate-950 shadow-lg shadow-teal-500/20">
            转入余额
          </button>
        </div>

        <div class="mt-7 grid gap-5 lg:grid-cols-2">
          <div>
            <p class="text-sm font-semibold text-slate-300">我的邀请码</p>
            <div class="mt-3 flex items-center justify-between rounded-xl border border-white/10 bg-[#0d1928] px-5 py-4">
              <code class="text-lg font-semibold text-white">RMSV7D76XM23</code>
              <span class="rounded-lg border border-white/10 px-3 py-2 text-sm text-slate-300">复制邀请码</span>
            </div>
          </div>
          <div>
            <p class="text-sm font-semibold text-slate-300">邀请链接</p>
            <div class="mt-3 flex items-center justify-between gap-4 rounded-xl border border-white/10 bg-[#0d1928] px-5 py-4">
              <code class="truncate text-lg text-slate-300">https://api.lumio.games/register?aff=RMSV7D76XM23</code>
              <span class="shrink-0 rounded-lg border border-white/10 px-3 py-2 text-sm text-slate-300">复制链接</span>
            </div>
          </div>
        </div>
      </section>

      <section class="overflow-hidden rounded-2xl border border-white/10 bg-[#121d2f] shadow-[0_22px_70px_rgba(0,0,0,0.24)]">
        <div class="border-b border-white/10 px-7 py-6">
          <div class="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <p class="text-sm font-semibold uppercase text-teal-300">Tiered Affiliate Rebate</p>
              <h2 class="mt-2 text-2xl font-semibold">我的邀请等级</h2>
              <p class="mt-2 max-w-3xl text-base leading-7 text-slate-400">
                邀请人数与邀请充值总额双维度同时达标后自动晋升，等级越高分享比例越高。
              </p>
              <p class="mt-2 text-sm text-slate-500">
                等级固定为 L1-L4；每档邀请人数、邀请充值总额和返利比例由管理员后台配置。
              </p>
            </div>
            <div class="rounded-2xl border border-teal-300/30 bg-teal-300/10 px-6 py-4">
              <p class="text-sm text-teal-200">当前等级</p>
              <div class="mt-1 flex items-end gap-3">
                <span class="text-5xl font-semibold text-teal-200">{{ currentTier.level }}</span>
                <span class="pb-1 text-xl font-semibold text-teal-300">{{ currentTier.rate }}%</span>
              </div>
            </div>
          </div>
        </div>

        <div class="grid gap-6 p-7 xl:grid-cols-[26rem_minmax(0,1fr)]">
          <div class="rounded-2xl border border-white/10 bg-[#0d1928] p-6">
            <p class="text-sm text-slate-400">下一等级进度</p>
            <p class="mt-2 text-xl font-semibold">{{ currentTier.level }} → {{ nextTier.level }}</p>

            <div class="mt-6 space-y-5">
              <div>
                <div class="mb-2 flex items-center justify-between text-sm">
                  <span class="text-slate-300">邀请人数</span>
                  <span class="text-slate-400">{{ inviteCount }} / {{ nextTier.minInvitees }}</span>
                </div>
                <div class="h-3 overflow-hidden rounded-full bg-white/10">
                  <div class="h-full rounded-full bg-cyan-400" :style="{ width: `${inviteProgress}%` }"></div>
                </div>
              </div>

              <div>
                <div class="mb-2 flex items-center justify-between text-sm">
                  <span class="text-slate-300">邀请充值总额</span>
                  <span class="text-slate-400">¥{{ formatNumber(rechargeTotal) }} / ¥{{ formatNumber(nextTier.minRecharge) }}</span>
                </div>
                <div class="h-3 overflow-hidden rounded-full bg-white/10">
                  <div class="h-full rounded-full bg-teal-300" :style="{ width: `${rechargeProgress}%` }"></div>
                </div>
              </div>
            </div>

            <div class="mt-6 rounded-xl bg-white/[0.04] p-4 text-sm leading-6 text-slate-300">
              距离 {{ nextTier.level }} 还差
              <span class="font-semibold text-cyan-200">{{ Math.max(0, nextTier.minInvitees - inviteCount) }}</span>
              人和
              <span class="font-semibold text-teal-200">¥{{ formatNumber(Math.max(0, nextTier.minRecharge - rechargeTotal)) }}</span>
              邀请充值。
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2 2xl:grid-cols-5">
            <div
              v-for="tier in tiers"
              :key="tier.level"
              :class="[
                'rounded-2xl border p-5',
                tier.level === currentTier.level
                  ? 'border-teal-300/60 bg-teal-300/10'
                  : isTierReached(tier)
                    ? 'border-cyan-300/30 bg-cyan-300/[0.08]'
                    : 'border-white/10 bg-[#0d1928]'
              ]"
            >
              <div class="flex items-center justify-between">
                <span
                  :class="[
                    'flex h-11 w-11 items-center justify-center rounded-full text-base font-semibold',
                    tier.level === currentTier.level
                      ? 'bg-teal-300 text-slate-950'
                      : isTierReached(tier)
                        ? 'bg-cyan-300 text-slate-950'
                        : 'bg-white/10 text-slate-300'
                  ]"
                >
                  {{ tier.level }}
                </span>
                <span v-if="tier.level === currentTier.level" class="rounded-full bg-teal-300/15 px-3 py-1 text-xs font-semibold text-teal-200">
                  当前
                </span>
              </div>

              <dl class="mt-6 space-y-3 text-sm">
                <div class="flex items-center justify-between gap-3">
                  <dt class="text-slate-400">邀请人数 ≥</dt>
                  <dd class="font-semibold text-white">{{ tier.minInvitees }} 人</dd>
                </div>
                <div class="flex items-center justify-between gap-3">
                  <dt class="text-slate-400">邀请充值 ≥</dt>
                  <dd class="font-semibold text-white">¥{{ formatNumber(tier.minRecharge) }}</dd>
                </div>
                <div class="border-t border-white/10 pt-3">
                  <dt class="text-slate-400">返利比例</dt>
                  <dd class="mt-1 text-3xl font-semibold text-teal-300">{{ tier.rate }}%</dd>
                </div>
              </dl>
            </div>
          </div>
        </div>
      </section>

      <section class="rounded-2xl border border-white/10 bg-[#121d2f] p-7">
        <div>
          <h2 class="text-2xl font-semibold">后台配置样式</h2>
          <p class="mt-2 text-base text-slate-400">实际落地时管理员只配置 L1-L4 四行，每行包含邀请人数、邀请充值总额和返利比例。</p>
        </div>

        <div class="mt-6 overflow-x-auto">
          <table class="w-full min-w-[720px] text-left text-sm">
            <thead>
              <tr class="border-b border-white/10 text-slate-400">
                <th class="px-4 py-3 font-medium">等级</th>
                <th class="px-4 py-3 font-medium">邀请人数 ≥</th>
                <th class="px-4 py-3 font-medium">邀请充值总额 ≥</th>
                <th class="px-4 py-3 font-medium">返利比例</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="tier in tiers" :key="`config-${tier.level}`" class="border-b border-white/10 last:border-0">
                <td class="px-4 py-4 font-semibold text-white">{{ tier.level }}</td>
                <td class="px-4 py-4">
                  <div class="rounded-xl border border-white/10 bg-[#0d1928] px-4 py-3 text-white">{{ tier.minInvitees }}</div>
                </td>
                <td class="px-4 py-4">
                  <div class="rounded-xl border border-white/10 bg-[#0d1928] px-4 py-3 text-white">¥{{ formatNumber(tier.minRecharge) }}</div>
                </td>
                <td class="px-4 py-4">
                  <div class="rounded-xl border border-white/10 bg-[#0d1928] px-4 py-3 font-semibold text-teal-300">{{ tier.rate }}%</div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface TierRow {
  level: string
  minInvitees: number
  minRecharge: number
  rate: number
}

const inviteCount = 27
const rechargeTotal = 12800

const tiers: TierRow[] = [
  { level: 'L1', minInvitees: 0, minRecharge: 0, rate: 5 },
  { level: 'L2', minInvitees: 5, minRecharge: 500, rate: 15 },
  { level: 'L3', minInvitees: 20, minRecharge: 5000, rate: 25 },
  { level: 'L4', minInvitees: 50, minRecharge: 20000, rate: 35 },
]

const currentTier = computed(() => (
  [...tiers].reverse().find((tier) => isTierReached(tier)) ?? tiers[0]
))

const nextTier = computed(() => {
  const index = tiers.findIndex((tier) => tier.level === currentTier.value.level)
  return tiers[index + 1] ?? currentTier.value
})

const inviteProgress = computed(() => progressPercent(inviteCount, nextTier.value.minInvitees))
const rechargeProgress = computed(() => progressPercent(rechargeTotal, nextTier.value.minRecharge))

const statCards = computed(() => [
  { label: '我的返利比例', value: `${currentTier.value.rate}%`, accent: true, hint: '根据当前等级自动计算' },
  { label: '邀请人数', value: String(inviteCount) },
  { label: '邀请充值总额', value: `¥${formatNumber(rechargeTotal)}`, accent: true },
  { label: '历史返利额度', value: 'US$23.66' },
])

function isTierReached(tier: TierRow): boolean {
  return inviteCount >= tier.minInvitees && rechargeTotal >= tier.minRecharge
}

function progressPercent(current: number, target: number): number {
  if (target <= 0) return 100
  return Math.min(100, Math.round((current / target) * 100))
}

function formatNumber(value: number): string {
  return value.toLocaleString('zh-CN')
}
</script>
