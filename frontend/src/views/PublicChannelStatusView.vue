<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <div class="pointer-events-none fixed inset-0 bg-mesh-gradient"></div>

    <div class="relative min-h-screen">
      <AppHeader :show-mobile-menu="false" />

      <main class="p-4 md:p-6 lg:p-8">
        <MonitorHero
          :overall-status="overallStatus"
          :interval-seconds="DEFAULT_INTERVAL_SECONDS"
          :window="currentWindow"
          :loading="loading"
          :auto-refresh="autoRefresh"
          @update:window="handleWindowChange"
          @refresh="manualReload"
        />

        <MonitorCardGrid
          :items="items"
          :window="currentWindow"
          :countdown-seconds="countdown"
          :loading="loading"
          :detail-cache="detailCache"
          @card-click="openDetail"
        />

        <MonitorDetailDialog
          :show="showDetail"
          :monitor-id="detailTarget?.id ?? null"
          :title="detailTitle"
          :load-detail="loadPublicDetail"
          @close="closeDetail"
        />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  publicList as listChannelMonitorViews,
  publicStatus as fetchChannelMonitorDetail,
  type UserMonitorView,
  type UserMonitorDetail,
} from '@/api/channelMonitor'
import AppHeader from '@/components/layout/AppHeader.vue'
import MonitorHero, {
  type MonitorWindow,
  type OverallStatus,
} from '@/components/user/monitor/MonitorHero.vue'
import MonitorCardGrid from '@/components/user/monitor/MonitorCardGrid.vue'
import MonitorDetailDialog from '@/components/user/MonitorDetailDialog.vue'
import { DEFAULT_INTERVAL_SECONDS, STATUS_OPERATIONAL } from '@/constants/channelMonitor'
import { useAutoRefresh } from '@/composables/useAutoRefresh'

const { t } = useI18n()
const appStore = useAppStore()

const items = ref<UserMonitorView[]>([])
const loading = ref(false)
const currentWindow = ref<MonitorWindow>('7d')
const detailCache = reactive<Record<number, UserMonitorDetail>>({})
const showDetail = ref(false)
const detailTarget = ref<UserMonitorView | null>(null)

let abortController: AbortController | null = null

const autoRefresh = useAutoRefresh({
  storageKey: 'public-channel-status-auto-refresh',
  intervals: [30, 60, 120] as const,
  defaultInterval: DEFAULT_INTERVAL_SECONDS,
  onRefresh: () => reload(true),
  shouldPause: () => document.hidden || loading.value,
})
const countdown = autoRefresh.countdown

const overallStatus = computed<OverallStatus>(() => {
  if (items.value.length === 0) return 'operational'
  return items.value.some((item) => item.primary_status !== STATUS_OPERATIONAL)
    ? 'degraded'
    : 'operational'
})

const detailTitle = computed(() => {
  return detailTarget.value?.name || t('channelStatus.detailTitle')
})

async function reload(silent = false) {
  if (abortController) abortController.abort()
  const ctrl = new AbortController()
  abortController = ctrl
  if (!silent) loading.value = true
  try {
    const res = await listChannelMonitorViews({ signal: ctrl.signal })
    if (ctrl.signal.aborted || abortController !== ctrl) return
    items.value = res.items || []
  } catch (err: unknown) {
    const e = err as { name?: string; code?: string }
    if (e?.name === 'AbortError' || e?.code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(err, t('channelStatus.loadError')))
  } finally {
    if (abortController === ctrl) {
      if (!silent) loading.value = false
      countdown.value = DEFAULT_INTERVAL_SECONDS
      abortController = null
    }
  }
}

async function manualReload() {
  await reload(false)
  if (currentWindow.value !== '7d') {
    await Promise.all(items.value.map((item) => loadDetail(item.id, true)))
  }
}

async function loadPublicDetail(id: number): Promise<UserMonitorDetail> {
  return fetchChannelMonitorDetail(id)
}

async function loadDetail(id: number, force = false) {
  if (!force && detailCache[id]) return
  try {
    detailCache[id] = await loadPublicDetail(id)
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('channelStatus.detailLoadError')))
  }
}

async function ensureDetailsForWindow() {
  if (currentWindow.value === '7d') return
  await Promise.all(items.value.map((item) => loadDetail(item.id)))
}

async function handleWindowChange(value: MonitorWindow) {
  currentWindow.value = value
  await ensureDetailsForWindow()
}

function openDetail(row: UserMonitorView) {
  detailTarget.value = row
  showDetail.value = true
}

function closeDetail() {
  showDetail.value = false
  detailTarget.value = null
}

watch(items, () => {
  void ensureDetailsForWindow()
})

watch(
  () => appStore.cachedPublicSettings?.channel_monitor_enabled,
  (enabled) => {
    if (enabled === false) autoRefresh.stop()
    else if (autoRefresh.enabled.value) autoRefresh.start()
  },
)

onMounted(() => {
  if (!appStore.publicSettingsLoaded) {
    void appStore.fetchPublicSettings()
  }
  void reload(false)
  if (appStore.cachedPublicSettings?.channel_monitor_enabled !== false) {
    autoRefresh.setEnabled(true)
  }
})

onBeforeUnmount(() => {
  if (abortController) abortController.abort()
})
</script>
