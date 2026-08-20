<template>
  <AppLayout>
    <div class="mb-5 space-y-4">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <h1 class="font-serif text-2xl font-semibold text-gray-900 dark:text-white">{{ t('checkin.admin.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('checkin.admin.description') }}</p>
          <p v-if="statsRangeLabel" class="mt-1 ui-mono text-xs text-gray-400">{{ statsRangeLabel }}</p>
        </div>
        <div class="inline-flex self-start rounded-lg border border-gray-200 bg-white p-1 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <button
            v-for="period in statsPeriods"
            :key="period"
            type="button"
            class="rounded-md px-3 py-1.5 text-sm font-semibold transition-colors"
            :class="statsPeriod === period
              ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200'
              : 'text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-white'"
            :data-test="`stats-period-${period}`"
            @click="selectStatsPeriod(period)"
          >
            {{ t(`checkin.admin.stats.${period}`) }}
          </button>
        </div>
      </div>
      <section data-test="checkin-stats" class="grid grid-cols-2 gap-3 md:grid-cols-4 xl:grid-cols-7">
        <div v-for="item in statsCards" :key="item.key" class="rounded-lg border border-gray-100 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800/60">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ item.label }}</p>
          <p class="mt-1 truncate text-lg font-semibold text-gray-900 dark:text-white" :class="item.mono ? 'ui-mono' : ''" :data-test="`stats-${item.key}`">{{ item.value }}</p>
        </div>
      </section>
      <p v-if="statsError" class="text-sm text-red-600 dark:text-red-400">{{ t('checkin.admin.stats.loadFailed') }}</p>
    </div>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="relative w-full md:w-64"><Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" /><input v-model="filters.search" type="search" class="input pl-10" :placeholder="t('checkin.admin.search')" @input="debounceLoad" /></div>
          <input v-model="filters.userId" data-test="user-id-filter" type="number" min="1" class="input w-full sm:w-32 ui-mono" :placeholder="t('checkin.admin.userId')" @change="reloadFromFirstPage" />
          <input v-model="filters.businessDate" data-test="date-filter" type="date" class="input w-full sm:w-44 ui-mono" :title="t('checkin.admin.date')" @change="reloadFromFirstPage" />
          <select v-model="filters.status" data-test="status-filter" class="input w-full sm:w-44" @change="reloadFromFirstPage"><option value="">{{ t('checkin.admin.allStatuses') }}</option><option value="awarded">{{ t('checkin.history.awarded') }}</option><option value="budget_exhausted">{{ t('checkin.history.exhausted') }}</option></select>
          <button type="button" class="btn btn-secondary h-10 w-10 p-0" :disabled="loading || statsLoading" :title="t('common.refresh')" @click="refreshAll"><Icon name="refresh" size="md" :class="(loading || statsLoading) ? 'animate-spin' : ''" /></button>
        </div>
        <p v-if="loadError" class="mt-2 text-sm text-red-600 dark:text-red-400">{{ t('checkin.admin.loadFailed') }}</p>
      </template>
      <template #table>
        <DataTable :columns="columns" :data="records" :loading="loading" :server-side-sort="true" default-sort-key="checked_at" default-sort-order="desc" sort-storage-key="admin-checkin-records-sort" @sort="handleSort">
          <template #cell-user="{ row }"><div class="max-w-64 space-y-0.5"><div class="ui-mono text-xs text-gray-500">#{{ row.user_id || '-' }}</div><div class="truncate text-sm font-medium" :title="row.user_email">{{ row.user_email || '-' }}</div><div class="truncate text-xs text-gray-500" :title="row.username">{{ row.username || '-' }}</div></div></template>
          <template #cell-business_date="{ row }"><span class="ui-mono">{{ row.business_date }}</span></template>
          <template #cell-checked_at="{ row }"><div class="whitespace-nowrap">{{ formatCheckedAt(row.checked_at) }}</div><div class="mt-0.5 text-xs text-gray-400">{{ row.timezone }}</div></template>
          <template #cell-streak_days="{ row }"><span class="ui-mono">{{ row.streak_days }} / {{ row.cycle_day }}</span></template>
          <template #cell-base_reward="{ row }"><MoneyCell :value="row.base_reward" /></template>
          <template #cell-milestone_bonus="{ row }"><div class="text-right"><MoneyCell :value="row.milestone_bonus" /><div v-if="row.milestone_day" class="text-xs text-gray-400">Day {{ row.milestone_day }}</div></div></template>
          <template #cell-actual_reward="{ row }"><MoneyCell :value="row.actual_reward" :strong="row.status === 'awarded'" /></template>
          <template #cell-balance_after="{ row }"><MoneyCell :value="row.balance_after" /></template>
          <template #cell-status="{ row }"><span class="inline-flex rounded-full px-2 py-1 text-xs font-medium" :class="row.status === 'awarded' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300' : 'bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300'">{{ row.status === 'awarded' ? t('checkin.history.awarded') : t('checkin.history.exhausted') }}</span></template>
        </DataTable>
      </template>
      <template #pagination><Pagination v-if="pagination.total > 0" :page="pagination.page" :total="pagination.total" :page-size="pagination.pageSize" @update:page="handlePageChange" @update:pageSize="handlePageSizeChange" /></template>
    </TablePageLayout>
  </AppLayout>
</template>
<script setup lang="ts">
import { computed, defineComponent, h, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'; import TablePageLayout from '@/components/layout/TablePageLayout.vue'; import DataTable from '@/components/common/DataTable.vue'; import Pagination from '@/components/common/Pagination.vue'; import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { formatDateTimeToMinute } from '@/utils/format'
import { checkinAPI } from './api'; import type { AdminCheckinListParams, AdminCheckinStats, AdminCheckinStatsPeriod, CheckinRecord, CheckinRecordStatus } from './types'

const { t } = useI18n(); const loading = ref(false), loadError = ref(false), records = ref<CheckinRecord[]>([])
const statsLoading = ref(false), statsError = ref(false), statsPeriod = ref<AdminCheckinStatsPeriod>('day')
const stats = ref<AdminCheckinStats | null>(null)
const statsPeriods: AdminCheckinStatsPeriod[] = ['day', 'week', 'month', 'all']
const filters = reactive({ search: '', userId: '', businessDate: '', status: '' }); const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const sort = reactive<{ by: NonNullable<AdminCheckinListParams['sort_by']>; order: 'asc' | 'desc' }>({ by: 'checked_at', order: 'desc' }); let debounceTimer: ReturnType<typeof setTimeout> | null = null
const columns = computed<Column[]>(() => [
  { key: 'user', label: t('checkin.admin.columns.user') }, { key: 'business_date', label: t('checkin.admin.columns.businessDate'), sortable: true }, { key: 'checked_at', label: t('checkin.admin.columns.checkedAt'), sortable: true },
  { key: 'streak_days', label: t('checkin.admin.columns.streak'), sortable: true }, { key: 'base_reward', label: t('checkin.admin.columns.base'), class: 'text-right' }, { key: 'milestone_bonus', label: t('checkin.admin.columns.milestone'), class: 'text-right' },
  { key: 'actual_reward', label: t('checkin.admin.columns.actual'), sortable: true, class: 'text-right' }, { key: 'balance_after', label: t('checkin.admin.columns.balance'), sortable: true, class: 'text-right' }, { key: 'status', label: t('checkin.admin.columns.status') }
])
function selectedUserId(): number | undefined { const id = Number(filters.userId); return Number.isInteger(id) && id > 0 ? id : undefined }
function buildParams(): AdminCheckinListParams { return { page: pagination.page, page_size: pagination.pageSize, user_id: selectedUserId(), search: filters.search.trim() || undefined, business_date: filters.businessDate || undefined, status: (filters.status || undefined) as CheckinRecordStatus | undefined, sort_by: sort.by, sort_order: sort.order } }
async function loadRecords() { loading.value = true; loadError.value = false; try { const response = await checkinAPI.listAdminRecords(buildParams()); records.value = response.items ?? []; pagination.total = response.total ?? 0 } catch { loadError.value = true } finally { loading.value = false } }
async function loadStats() {
  statsLoading.value = true
  statsError.value = false
  try {
    stats.value = await checkinAPI.getAdminStats({
      period: statsPeriod.value,
      user_id: selectedUserId(),
      search: filters.search.trim() || undefined,
      status: (filters.status || undefined) as CheckinRecordStatus | undefined,
    })
  } catch {
    statsError.value = true
  } finally {
    statsLoading.value = false
  }
}
function reloadFromFirstPage() { pagination.page = 1; void loadRecords(); void loadStats() }
function selectStatsPeriod(period: AdminCheckinStatsPeriod) { statsPeriod.value = period; void loadStats() }
const statsRangeLabel = computed(() => {
  if (!stats.value) return ''
  if (stats.value.from && stats.value.to) {
    return `${t('checkin.admin.stats.range', { from: stats.value.from, to: stats.value.to })} · ${stats.value.timezone}`
  }
  return stats.value.timezone
})
const statsCards = computed(() => {
  const current = stats.value
  const money = (value?: string) => `$${value || '0.0000'}`
  return [
    { key: 'unique-users', label: t('checkin.admin.stats.uniqueUsers'), value: String(current?.unique_users ?? 0), mono: true },
    { key: 'checkins', label: t('checkin.admin.stats.checkins'), value: String(current?.checkin_count ?? 0), mono: true },
    { key: 'total', label: t('checkin.admin.stats.total'), value: money(current?.total_amount), mono: true },
    { key: 'avg', label: t('checkin.admin.stats.avg'), value: money(current?.avg_amount), mono: true },
    { key: 'p50', label: t('checkin.admin.stats.p50'), value: money(current?.p50_amount), mono: true },
    { key: 'p90', label: t('checkin.admin.stats.p90'), value: money(current?.p90_amount), mono: true },
    { key: 'max', label: t('checkin.admin.stats.max'), value: money(current?.max_amount), mono: true },
  ]
})
function debounceLoad() { if (debounceTimer) clearTimeout(debounceTimer); debounceTimer = setTimeout(reloadFromFirstPage, 300) }
function handlePageChange(page: number) { pagination.page = page; void loadRecords() }
function handlePageSizeChange(size: number) { pagination.pageSize = size; reloadFromFirstPage() }
function handleSort(key: string, order: 'asc' | 'desc') { if (['business_date', 'checked_at', 'streak_days', 'actual_reward', 'balance_after'].includes(key)) { sort.by = key as typeof sort.by; sort.order = order; reloadFromFirstPage() } }
function formatCheckedAt(value: string) { return formatDateTimeToMinute(value) || '-' }
function refreshAll() { void loadRecords(); void loadStats() }
const MoneyCell = defineComponent({ props: { value: { type: String, default: '0.0000' }, strong: { type: Boolean, default: false } }, setup(props) { return () => h('span', { class: props.strong ? 'ui-mono text-sm font-semibold text-emerald-600 dark:text-emerald-400' : 'ui-mono text-sm' }, `$${props.value}`) } })
onMounted(() => { void loadRecords(); void loadStats() }); onBeforeUnmount(() => { if (debounceTimer) clearTimeout(debounceTimer) })
</script>
