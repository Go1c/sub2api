<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.errorHistory.title')"
    width="extra-wide"
    @close="handleClose"
  >
    <div class="space-y-4">
      <!-- Account Info Header -->
      <div
        v-if="account"
        class="flex items-center justify-between rounded-xl border border-primary-200 bg-gradient-to-r from-primary-50 to-primary-100 p-3 dark:border-primary-700/50 dark:from-primary-900/20 dark:to-primary-800/20"
      >
        <div class="flex items-center gap-3">
          <div
            class="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-primary-500 to-primary-600"
          >
            <Icon name="exclamationTriangle" size="md" class="text-white" :stroke-width="2" />
          </div>
          <div>
            <div class="font-semibold text-gray-900 dark:text-gray-100">{{ account.name }}</div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.errorHistory.subtitle') }}
            </div>
          </div>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <!-- Empty State -->
      <div
        v-else-if="items.length === 0"
        class="flex flex-col items-center justify-center py-12 text-gray-500 dark:text-gray-400"
      >
        <Icon name="checkCircle" size="xl" class="mb-4 h-12 w-12 text-green-400" :stroke-width="1.5" />
        <p class="text-sm">{{ t('admin.accounts.errorHistory.empty') }}</p>
      </div>

      <!-- Error History Table -->
      <DataTable
        v-else
        :columns="columns"
        :data="items"
        :loading="false"
        :sticky-first-column="false"
        :sticky-actions-column="false"
        row-key="id"
      >
        <!-- Time (local) with full timestamp on hover -->
        <template #cell-created_at="{ row }">
          <span :title="formatDateTime(row.created_at)" class="text-gray-500 dark:text-gray-400">
            {{ formatRelativeTime(row.created_at) }}
          </span>
        </template>

        <!-- User email -->
        <template #cell-user_email="{ row }">
          <span>{{ row.user_email || EMPTY }}</span>
        </template>

        <!-- Model -->
        <template #cell-model="{ row }">
          <span class="font-mono text-xs">{{ row.model || EMPTY }}</span>
        </template>

        <!-- Upstream status code -->
        <template #cell-upstream_status_code="{ row }">
          <span class="font-mono">{{ row.upstream_status_code ?? EMPTY }}</span>
        </template>

        <!-- Source badge -->
        <template #cell-source="{ row }">
          <span
            class="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300"
          >
            {{ sourceLabel(row.source) }}
          </span>
        </template>

        <!-- Message (truncated, full text on hover) -->
        <template #cell-message="{ row }">
          <span
            :title="row.message"
            class="block max-w-md truncate text-gray-700 dark:text-gray-300"
          >
            {{ row.message }}
          </span>
        </template>

        <!-- Duplicate count (×N when collapsed) -->
        <template #cell-dup_count="{ row }">
          <span
            v-if="row.dup_count > 1"
            :title="t('admin.accounts.errorHistory.dupHint', { count: row.dup_count })"
            class="font-mono text-xs font-semibold text-amber-600 dark:text-amber-400"
          >
            ×{{ row.dup_count }}
          </span>
          <span v-else class="text-gray-400 dark:text-gray-500">{{ EMPTY }}</span>
        </template>
      </DataTable>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button
          @click="handleClose"
          class="rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
        >
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { AccountErrorHistoryItem } from '@/api/admin/accounts'
import type { Column } from '@/components/common/types'
import { formatDateTime, formatRelativeTime } from '@/utils/format'
import type { Account } from '@/types'

const EMPTY = '—'

const { t } = useI18n()

const props = defineProps<{
  show: boolean
  account: Account | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const loading = ref(false)
const items = ref<AccountErrorHistoryItem[]>([])

const columns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.accounts.errorHistory.columns.time') },
  { key: 'user_email', label: t('admin.accounts.errorHistory.columns.userEmail') },
  { key: 'model', label: t('admin.accounts.errorHistory.columns.model') },
  { key: 'upstream_status_code', label: t('admin.accounts.errorHistory.columns.statusCode') },
  { key: 'source', label: t('admin.accounts.errorHistory.columns.source') },
  { key: 'message', label: t('admin.accounts.errorHistory.columns.message') },
  { key: 'dup_count', label: t('admin.accounts.errorHistory.columns.count') }
])

const knownSources = ['gateway', 'test', 'ratelimit', 'refresh', 'schedule', 'admin']

const sourceLabel = (source: string): string => {
  // Fall back to the raw value for any source the i18n layer does not yet know.
  return knownSources.includes(source)
    ? t(`admin.accounts.errorHistory.sources.${source}`)
    : source
}

// Lazy-load: only fetch when the modal is actually opened (diagnostic, non-critical).
watch(
  () => props.show,
  async (visible) => {
    if (visible && props.account) {
      await loadErrorHistory()
    } else {
      items.value = []
    }
  }
)

const loadErrorHistory = async () => {
  if (!props.account) return
  loading.value = true
  try {
    items.value = await adminAPI.accounts.getErrorHistory(props.account.id, 20)
  } catch (error) {
    console.error('Failed to load account error history:', error)
    items.value = []
  } finally {
    loading.value = false
  }
}

const handleClose = () => {
  emit('close')
}
</script>
