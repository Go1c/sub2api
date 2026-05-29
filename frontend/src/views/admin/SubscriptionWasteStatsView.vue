<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('admin.subscriptions.wasteStats.title') }}</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.subscriptions.wasteStats.description') }}</p>
          </div>
          <button class="btn btn-secondary" :disabled="loading" @click="loadStats">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            {{ t('common.refresh') }}
          </button>
        </div>

        <div class="grid gap-3 md:grid-cols-5">
          <input v-model="filters.start" type="date" class="input" :title="t('admin.subscriptions.wasteStats.start')" />
          <input v-model="filters.end" type="date" class="input" :title="t('admin.subscriptions.wasteStats.end')" />
          <input v-model="filters.plan_id" type="number" min="1" class="input" :placeholder="t('admin.subscriptions.planId')" />
          <input v-model="filters.user_id" type="number" min="1" class="input" :placeholder="t('admin.subscriptions.wasteStats.userId')" />
          <Select v-model="filters.window" :options="windowOptions" />
        </div>
        <div class="mt-4 flex justify-end">
          <button class="btn btn-primary" :disabled="loading" @click="loadStats">{{ t('common.search') }}</button>
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-4">
        <StatCard :label="t('admin.subscriptions.wasteStats.totalQuota')" :value="formatUsd(totalPurchasedUsd)" tone="blue" />
        <StatCard :label="t('admin.subscriptions.wasteStats.totalUsed')" :value="formatUsd(totalConsumedUsd)" tone="emerald" />
        <StatCard :label="t('admin.subscriptions.wasteStats.totalWaste')" :value="formatUsd(stats?.total_wasted_usd)" tone="amber" />
        <StatCard :label="t('admin.subscriptions.wasteStats.wasteRate')" :value="formatPercent(totalWasteRatio)" tone="rose" />
      </div>

      <div class="grid gap-6 xl:grid-cols-2">
        <PeriodCard :title="t('admin.subscriptions.wasteStats.dailyBreakdown')" :rows="dailyRows" />
        <PeriodCard :title="t('admin.subscriptions.wasteStats.weeklyBreakdown')" :rows="weeklyRows" />
      </div>

      <div class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <h2 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.subscriptions.wasteStats.byPlan') }}</h2>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead>
              <tr class="text-left text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                <th class="px-3 py-2">{{ t('admin.subscriptions.wasteStats.plan') }}</th>
                <th class="px-3 py-2">{{ t('admin.subscriptions.wasteStats.subscriptions') }}</th>
                <th class="px-3 py-2">{{ t('admin.subscriptions.wasteStats.totalQuota') }}</th>
                <th class="px-3 py-2">{{ t('admin.subscriptions.wasteStats.totalUsed') }}</th>
                <th class="px-3 py-2">{{ t('admin.subscriptions.wasteStats.totalWaste') }}</th>
                <th class="px-3 py-2">{{ t('admin.subscriptions.wasteStats.wasteRate') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="plan in byPlanRows" :key="`${plan.plan_id ?? 'none'}-${plan.plan_name ?? ''}`" class="text-gray-700 dark:text-gray-200">
                <td class="px-3 py-3">{{ plan.plan_name || t('admin.subscriptions.wasteStats.unknownPlan') }}</td>
                <td class="px-3 py-3">{{ plan.purchase_count ?? 0 }}</td>
                <td class="px-3 py-3">{{ formatUsd(plan.purchased_usd) }}</td>
                <td class="px-3 py-3">{{ formatUsd(plan.consumed_usd) }}</td>
                <td class="px-3 py-3">{{ formatUsd(plan.total_wasted_usd) }}</td>
                <td class="px-3 py-3">{{ formatPercent(plan.total_wasted_ratio ?? plan.waste_ratio) }}</td>
              </tr>
              <tr v-if="!byPlanRows.length">
                <td colspan="6" class="px-3 py-8 text-center text-gray-500 dark:text-gray-400">{{ t('payment.admin.noData') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <PeriodCard :title="t('admin.subscriptions.wasteStats.trend')" :rows="trendRows" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type {
  WasteStatsByPlan,
  WasteStatsFilters,
  WasteStatsResult,
  WasteStatsTimeBucket
} from '@/api/admin/subscriptions'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const stats = ref<WasteStatsResult | null>(null)

type WasteStatsWindow = NonNullable<WasteStatsFilters['window']>

interface WasteStatsDisplayRow {
  period: string
  reset_count?: number
  total_wasted_usd?: number
  average_waste_ratio?: number
}

const filters = reactive({
  start: defaultStartDate(),
  end: new Date().toISOString().slice(0, 10),
  plan_id: '',
  user_id: '',
  window: 'daily' as WasteStatsWindow
})

const windowOptions = computed(() => [
  { value: 'daily', label: t('admin.subscriptions.daily') },
  { value: 'weekly', label: t('admin.subscriptions.weekly') },
  { value: 'total', label: t('common.total') },
  { value: 'all', label: t('common.all') }
])

const totalPurchasedUsd = computed(() => stats.value?.total_quota_purchased_usd ?? stats.value?.purchased_usd)
const totalConsumedUsd = computed(() => stats.value?.total_quota_consumed_usd ?? stats.value?.consumed_usd)
const totalWasteRatio = computed(() => stats.value?.total_wasted_ratio ?? stats.value?.waste_ratio)
const dailyRows = computed<WasteStatsDisplayRow[]>(() => stats.value
  ? [{
      period: t('admin.subscriptions.daily'),
      reset_count: stats.value.daily_reset_count,
      total_wasted_usd: stats.value.daily_total_wasted_usd,
      average_waste_ratio: stats.value.daily_average_waste_ratio
    }]
  : []
)
const weeklyRows = computed<WasteStatsDisplayRow[]>(() => stats.value
  ? [{
      period: t('admin.subscriptions.weekly'),
      reset_count: stats.value.weekly_reset_count,
      total_wasted_usd: stats.value.weekly_total_wasted_usd,
      average_waste_ratio: stats.value.weekly_average_waste_ratio
    }]
  : []
)
const trendRows = computed<WasteStatsDisplayRow[]>(() => (stats.value?.time_series ?? []).map(formatTimeBucketRow))
const byPlanRows = computed<WasteStatsByPlan[]>(() => stats.value?.by_plan ?? [])

function defaultStartDate() {
  const date = new Date()
  date.setDate(date.getDate() - 30)
  return date.toISOString().slice(0, 10)
}

function formatUsd(value: number | null | undefined): string {
  return `$${(value ?? 0).toFixed(2)}`
}

function formatPercent(value: number | null | undefined): string {
  const raw = value ?? 0
  return `${(raw > 1 ? raw : raw * 100).toFixed(1)}%`
}

function formatTimeBucketRow(row: WasteStatsTimeBucket): WasteStatsDisplayRow {
  return {
    period: formatPeriod(row),
    reset_count: row.reset_count,
    total_wasted_usd: row.total_wasted_usd,
    average_waste_ratio: row.average_waste_ratio
  }
}

function formatPeriod(row: WasteStatsTimeBucket): string {
  const start = formatDate(row.bucket_start)
  const end = formatDate(row.bucket_end)
  return start === end ? start : `${start} - ${end}`
}

function formatDate(value: string): string {
  return value ? value.slice(0, 10) : '-'
}

async function loadStats() {
  loading.value = true
  try {
    stats.value = await adminAPI.subscriptions.getWasteStats({
      start: filters.start || undefined,
      end: filters.end || undefined,
      plan_id: filters.plan_id ? Number(filters.plan_id) : undefined,
      user_id: filters.user_id ? Number(filters.user_id) : undefined,
      window: filters.window
    })
  } catch (error) {
    console.error('Failed to load subscription waste stats:', error)
    appStore.showError(t('admin.subscriptions.wasteStats.failedToLoad'))
  } finally {
    loading.value = false
  }
}

const StatCard = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true },
    tone: { type: String, required: true }
  },
  setup(props) {
    const tones: Record<string, string> = {
      blue: 'from-blue-50 to-sky-50 text-blue-700 dark:from-blue-950/40 dark:to-sky-950/20 dark:text-blue-200',
      emerald: 'from-emerald-50 to-teal-50 text-emerald-700 dark:from-emerald-950/40 dark:to-teal-950/20 dark:text-emerald-200',
      amber: 'from-amber-50 to-orange-50 text-amber-700 dark:from-amber-950/40 dark:to-orange-950/20 dark:text-amber-200',
      rose: 'from-rose-50 to-red-50 text-rose-700 dark:from-rose-950/40 dark:to-red-950/20 dark:text-rose-200'
    }
    return () => h('div', { class: `rounded-2xl bg-gradient-to-br p-5 shadow-sm ${tones[props.tone]}` }, [
      h('div', { class: 'text-sm opacity-75' }, props.label),
      h('div', { class: 'mt-2 text-2xl font-semibold tabular-nums' }, props.value)
    ])
  }
})

const PeriodCard = defineComponent({
  props: {
    title: { type: String, required: true },
    rows: { type: Array as PropType<WasteStatsDisplayRow[]>, required: true }
  },
  setup(props) {
    return () => h('div', { class: 'rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800' }, [
      h('h2', { class: 'mb-4 text-base font-semibold text-gray-900 dark:text-white' }, props.title),
      h('div', { class: 'overflow-x-auto' }, [
        h('table', { class: 'min-w-full text-sm' }, [
          h('thead', [
            h('tr', { class: 'text-left text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400' }, [
              h('th', { class: 'px-3 py-2' }, t('admin.subscriptions.wasteStats.period')),
              h('th', { class: 'px-3 py-2' }, t('common.reset')),
              h('th', { class: 'px-3 py-2' }, t('admin.subscriptions.wasteStats.totalWaste')),
              h('th', { class: 'px-3 py-2' }, t('admin.subscriptions.wasteStats.wasteRate'))
            ])
          ]),
          h('tbody', props.rows.length
            ? props.rows.map(row => h('tr', { class: 'border-t border-gray-100 text-gray-700 dark:border-dark-700 dark:text-gray-200' }, [
              h('td', { class: 'px-3 py-3' }, row.period),
              h('td', { class: 'px-3 py-3' }, row.reset_count ?? 0),
              h('td', { class: 'px-3 py-3' }, formatUsd(row.total_wasted_usd)),
              h('td', { class: 'px-3 py-3' }, formatPercent(row.average_waste_ratio))
            ]))
            : [h('tr', [h('td', { class: 'px-3 py-8 text-center text-gray-500 dark:text-gray-400', colspan: 4 }, t('payment.admin.noData'))])]
          )
        ])
      ])
    ])
  }
})

onMounted(loadStats)
</script>
