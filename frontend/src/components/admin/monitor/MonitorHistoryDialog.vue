<template>
  <BaseDialog
    :show="show"
    :title="dialogTitle"
    width="extra-wide"
    @close="$emit('close')"
  >
    <div v-if="loading" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('common.loading') }}
    </div>

    <div v-else-if="items.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('admin.channelMonitor.logs.empty') }}
    </div>

    <div v-else class="max-h-[60vh] overflow-auto">
      <table class="w-full min-w-[860px] text-left text-sm">
        <thead class="sticky top-0 border-b border-gray-200 bg-white text-xs text-gray-500 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-400">
          <tr>
            <th class="py-2 pr-4 font-medium">{{ t('admin.channelMonitor.logs.checkedAt') }}</th>
            <th class="py-2 pr-4 font-medium">{{ t('admin.channelMonitor.logs.model') }}</th>
            <th class="py-2 pr-4 font-medium">{{ t('admin.channelMonitor.logs.status') }}</th>
            <th class="py-2 pr-4 font-medium">{{ t('admin.channelMonitor.logs.latency') }}</th>
            <th class="py-2 pr-4 font-medium">{{ t('admin.channelMonitor.logs.pingLatency') }}</th>
            <th class="py-2 font-medium">{{ t('admin.channelMonitor.logs.message') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="item in items"
            :key="item.id"
            class="border-b border-gray-100 text-gray-700 last:border-0 dark:border-dark-800 dark:text-gray-300"
          >
            <td class="whitespace-nowrap py-2 pr-4 text-xs text-gray-500 dark:text-gray-400">
              {{ formatDateTime(item.checked_at) }}
            </td>
            <td class="py-2 pr-4 font-medium text-gray-900 dark:text-gray-100">
              {{ item.model }}
            </td>
            <td class="py-2 pr-4">
              <span
                class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px]"
                :class="statusBadgeClass(item.status)"
              >
                {{ statusLabel(item.status) }}
              </span>
            </td>
            <td class="whitespace-nowrap py-2 pr-4 text-xs">
              {{ formatLatencyWithUnit(item.latency_ms) }}
            </td>
            <td class="whitespace-nowrap py-2 pr-4 text-xs">
              {{ formatLatencyWithUnit(item.ping_latency_ms) }}
            </td>
            <td class="max-w-[360px] break-words py-2 text-xs">
              {{ item.message || '-' }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button @click="$emit('close')" class="btn btn-primary">
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { ChannelMonitor, HistoryItem } from '@/api/admin/channelMonitor'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'

const HISTORY_LIMIT = 100

const props = defineProps<{
  show: boolean
  monitor: ChannelMonitor | null
}>()

defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const { statusLabel, statusBadgeClass, formatLatency } = useChannelMonitorFormat()

const items = ref<HistoryItem[]>([])
const loading = ref(false)
let loadSeq = 0

const dialogTitle = computed(() => (
  t('admin.channelMonitor.logs.title', { name: props.monitor?.name || '' })
))

function formatLatencyWithUnit(ms: number | null): string {
  if (ms == null) return '-'
  return `${formatLatency(ms)} ms`
}

async function loadHistory(monitor: ChannelMonitor) {
  const seq = ++loadSeq
  loading.value = true
  items.value = []
  try {
    const res = await adminAPI.channelMonitor.listHistory(monitor.id, { limit: HISTORY_LIMIT })
    if (seq !== loadSeq) return
    items.value = res.items || []
  } catch (err: unknown) {
    if (seq !== loadSeq) return
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.logs.loadError')))
  } finally {
    if (seq === loadSeq) {
      loading.value = false
    }
  }
}

watch(
  () => [props.show, props.monitor?.id] as const,
  ([show]) => {
    if (!show || !props.monitor) {
      items.value = []
      return
    }
    void loadHistory(props.monitor)
  },
  { immediate: true },
)
</script>
