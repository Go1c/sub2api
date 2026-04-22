<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { http } from '@/api/http'
import StatCard from '@/components/dashboard/StatCard.vue'
import InviteSection from '@/components/dashboard/InviteSection.vue'
import TimeRangeFilter from '@/components/dashboard/TimeRangeFilter.vue'
import ChartCard from '@/components/dashboard/ChartCard.vue'
import QuickActionsPanel from '@/components/dashboard/QuickActionsPanel.vue'
import RecentUsagePanel from '@/components/dashboard/RecentUsagePanel.vue'

const { t } = useI18n()
const router = useRouter()

interface Stats {
  balance: number
  todayRequests: number
  todayTokens: number
  activeKeys: number
  todayTokensIn?: number
  todayTokensOut?: number
  totalTokens?: number
  totalTokensIn?: number
  totalTokensOut?: number
  todaySpend?: number
  totalSpend?: number
  rpm?: number
  tpm?: number
  latencyMs?: number
}

const stats = ref<Stats>({ balance: 0, todayRequests: 0, todayTokens: 0, activeKeys: 0 })
const invite = ref({ invited: 0, totalBonus: 0, monthBonus: 0, code: '-' })
const recent = ref<Array<{ id: string; time: string; model: string; tokens: number; status: string }>>([])
const range = ref('7')
const granularity = ref('day')

async function load() {
  const [s, r] = await Promise.all([
    http.get('/user/dashboard/stats'),
    http.get('/user/dashboard/recent')
  ])
  stats.value = { ...stats.value, ...s.data.data }
  recent.value = r.data.data
  // invite info from fixtures if available; otherwise keep defaults
  try {
    const inv = await http.get('/user/dashboard/invite')
    invite.value = inv.data.data
  } catch {
    // noop
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-5 animate-fade-in">
    <!-- Row 1: 4 stat cards -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      <StatCard
        icon="dollar"
        tone="green"
        :label="t('dashboard.stats.balance')"
        :value="`$${stats.balance.toFixed(2)}`"
        :hint="t('dashboard.stats.balanceHint')"
        :action="t('dashboard.stats.balanceAction')"
        @action="router.push('/recharge')"
      />
      <StatCard
        icon="key"
        tone="purple"
        :label="t('dashboard.stats.keys')"
        :value="`${stats.activeKeys}`"
        :hint="`${stats.activeKeys} ${t('dashboard.stats.keysHint')}`"
      />
      <StatCard
        icon="chart"
        tone="gray"
        :label="t('dashboard.stats.requests')"
        :value="`${stats.todayRequests}`"
        :hint="t('dashboard.stats.requestsTotal', { v: stats.todayRequests })"
      />
      <StatCard
        icon="dollar"
        tone="orange"
        :label="t('dashboard.stats.spend')"
        :value="`$${(stats.todaySpend || 0).toFixed(4)}`"
        :hint="`${t('dashboard.stats.spendTotal')} $${(stats.totalSpend || 0).toFixed(4)}`"
      />
    </div>

    <!-- Row 2: 4 stat cards -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      <StatCard
        icon="cube"
        tone="blue"
        :label="t('dashboard.stats.tokenToday')"
        :value="`${stats.todayTokens.toLocaleString()}`"
        :hint="t('dashboard.stats.tokenIO', { i: stats.todayTokensIn || 0, o: stats.todayTokensOut || 0 })"
      />
      <StatCard
        icon="database"
        tone="gray"
        :label="t('dashboard.stats.tokenTotal')"
        :value="`${(stats.totalTokens || 0).toLocaleString()}`"
        :hint="t('dashboard.stats.tokenIO', { i: stats.totalTokensIn || 0, o: stats.totalTokensOut || 0 })"
      />
      <StatCard
        icon="gauge"
        tone="amber"
        :label="t('dashboard.stats.perf')"
        :value="`${stats.rpm || 0} ${t('dashboard.stats.perfRpm')}`"
        :hint="`${stats.tpm || 0} ${t('dashboard.stats.perfTpm')}`"
      />
      <StatCard
        icon="clock"
        tone="blue"
        :label="t('dashboard.stats.latency')"
        :value="`${stats.latencyMs || 0}ms`"
        :hint="t('dashboard.stats.latencyHint')"
      />
    </div>

    <!-- Invite -->
    <InviteSection
      :invited="invite.invited"
      :total-bonus="invite.totalBonus"
      :month-bonus="invite.monthBonus"
      :code="invite.code"
    />

    <!-- Time range filter -->
    <TimeRangeFilter
      :range="range"
      :granularity="granularity"
      @update:range="range = $event"
      @update:granularity="granularity = $event"
    />

    <!-- Charts row -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5">
      <ChartCard
        :title="t('dashboard.charts.model')"
        empty
        :empty-text="t('dashboard.charts.empty')"
        :empty-fields="t('dashboard.charts.emptyFields') as any"
      />
      <ChartCard :title="t('dashboard.charts.token')" empty :empty-text="t('dashboard.charts.empty')" />
    </div>

    <!-- Recent + Quick -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-5">
      <div class="lg:col-span-2">
        <RecentUsagePanel :items="[]" :range="t('dashboard.recent.range')" />
      </div>
      <div>
        <QuickActionsPanel />
      </div>
    </div>
  </div>
</template>
