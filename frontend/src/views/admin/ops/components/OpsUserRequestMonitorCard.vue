<template>
  <section class="rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
    <div class="flex flex-col gap-4 border-b border-gray-100 p-5 dark:border-dark-800 lg:flex-row lg:items-start lg:justify-between">
      <div>
        <div class="flex items-center gap-2">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-dark-50">
            {{ t('admin.ops.userRequestMonitor.title') }}
          </h2>
          <span class="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-semibold text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">
            {{ t('admin.ops.userRequestMonitor.adminOnly') }}
          </span>
        </div>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">
          {{ t('admin.ops.userRequestMonitor.description') }}
        </p>
      </div>
      <button
        type="button"
        class="rounded-lg border border-gray-200 px-3 py-2 text-sm font-medium text-gray-700 transition hover:bg-gray-50 dark:border-dark-700 dark:text-dark-100 dark:hover:bg-dark-800"
        :disabled="loadingMonitors"
        @click="loadMonitors"
      >
        {{ t('admin.ops.userRequestMonitor.refresh') }}
      </button>
    </div>

    <div class="grid gap-5 p-5 xl:grid-cols-[360px,1fr]">
      <form
        class="space-y-4 rounded-2xl border border-amber-200 bg-amber-50/70 p-4 dark:border-amber-900/50 dark:bg-amber-950/20"
        data-test="monitor-create"
        @submit.prevent="createMonitor"
      >
        <div>
          <h3 class="text-sm font-semibold text-amber-900 dark:text-amber-100">
            {{ t('admin.ops.userRequestMonitor.createTitle') }}
          </h3>
          <p class="mt-1 text-xs leading-5 text-amber-800 dark:text-amber-200">
            {{ t('admin.ops.userRequestMonitor.rawWarning') }}
          </p>
        </div>

        <label class="block">
          <span class="text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('admin.ops.userRequestMonitor.userId') }}</span>
          <input
            v-model="form.userId"
            data-test="monitor-user-id"
            type="number"
            min="1"
            class="mt-1 w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none transition focus:border-primary-500 dark:border-dark-700 dark:bg-dark-950 dark:text-dark-50"
            :placeholder="t('admin.ops.userRequestMonitor.userIdPlaceholder')"
          >
        </label>

        <div class="grid grid-cols-2 gap-3">
          <label class="block">
            <span class="text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('admin.ops.userRequestMonitor.durationMinutes') }}</span>
            <input
              v-model="form.durationMinutes"
              data-test="monitor-duration-minutes"
              type="number"
              min="1"
              max="1440"
              class="mt-1 w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none transition focus:border-primary-500 dark:border-dark-700 dark:bg-dark-950 dark:text-dark-50"
            >
          </label>
          <label class="block">
            <span class="text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('admin.ops.userRequestMonitor.retentionDays') }}</span>
            <input
              v-model="form.retentionDays"
              data-test="monitor-retention-days"
              type="number"
              min="1"
              max="30"
              class="mt-1 w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none transition focus:border-primary-500 dark:border-dark-700 dark:bg-dark-950 dark:text-dark-50"
            >
          </label>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <label class="block">
            <span class="text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('admin.ops.userRequestMonitor.rateLimit') }}</span>
            <input
              v-model="form.rateLimit"
              data-test="monitor-rate-limit"
              type="number"
              min="1"
              max="120"
              class="mt-1 w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none transition focus:border-primary-500 dark:border-dark-700 dark:bg-dark-950 dark:text-dark-50"
            >
          </label>
          <label class="block">
            <span class="text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('admin.ops.userRequestMonitor.sampleRate') }}</span>
            <input
              v-model="form.sampleRate"
              data-test="monitor-sample-rate"
              type="number"
              min="1"
              max="100"
              class="mt-1 w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none transition focus:border-primary-500 dark:border-dark-700 dark:bg-dark-950 dark:text-dark-50"
            >
          </label>
        </div>

        <button
          type="submit"
          class="w-full rounded-lg bg-amber-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-amber-700 disabled:cursor-not-allowed disabled:opacity-60"
          :disabled="creating"
        >
          {{ creating ? t('admin.ops.userRequestMonitor.creating') : t('admin.ops.userRequestMonitor.create') }}
        </button>
      </form>

      <div class="space-y-4">
        <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div class="flex flex-wrap gap-2">
            <button
              v-for="option in statusOptions"
              :key="option.value"
              type="button"
              class="rounded-full px-3 py-1 text-xs font-semibold transition"
              :class="statusFilter === option.value ? 'bg-primary-600 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-800 dark:text-dark-200 dark:hover:bg-dark-700'"
              @click="statusFilter = option.value"
            >
              {{ option.label }}
            </button>
          </div>
          <input
            v-model="userQuery"
            type="search"
            class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none transition focus:border-primary-500 dark:border-dark-700 dark:bg-dark-950 dark:text-dark-50 md:w-72"
            :placeholder="t('admin.ops.userRequestMonitor.searchPlaceholder')"
          >
        </div>

        <div v-if="monitorError" class="rounded-xl bg-red-50 p-3 text-sm text-red-600 dark:bg-red-950/30 dark:text-red-300">
          {{ monitorError }}
        </div>

        <div class="overflow-hidden rounded-2xl border border-gray-100 dark:border-dark-800">
          <div class="max-h-[360px] overflow-auto">
            <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-800">
              <thead class="sticky top-0 bg-gray-50 text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-950 dark:text-dark-400">
                <tr>
                  <th class="px-4 py-3 text-left">{{ t('admin.ops.userRequestMonitor.table.user') }}</th>
                  <th class="px-4 py-3 text-left">{{ t('admin.ops.userRequestMonitor.table.status') }}</th>
                  <th class="px-4 py-3 text-right">{{ t('admin.ops.userRequestMonitor.table.captures') }}</th>
                  <th class="px-4 py-3 text-left">{{ t('admin.ops.userRequestMonitor.table.limits') }}</th>
                  <th class="px-4 py-3 text-left">{{ t('admin.ops.userRequestMonitor.table.endsAt') }}</th>
                  <th class="px-4 py-3 text-right">{{ t('admin.ops.userRequestMonitor.table.actions') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-800 dark:bg-dark-900">
                <tr v-if="loadingMonitors">
                  <td colspan="6" class="px-4 py-8 text-center text-gray-500">{{ t('admin.ops.loadingText') }}</td>
                </tr>
                <tr v-else-if="monitors.length === 0">
                  <td colspan="6" class="px-4 py-8 text-center text-gray-500">{{ t('admin.ops.userRequestMonitor.empty') }}</td>
                </tr>
                <template v-else>
                  <tr v-for="monitor in monitors" :key="monitor.id" class="hover:bg-gray-50 dark:hover:bg-dark-800/60">
                    <td class="px-4 py-3">
                      <div class="font-medium text-gray-900 dark:text-dark-50">{{ monitor.target_email }}</div>
                      <div class="text-xs text-gray-500">ID {{ monitor.user_id }}</div>
                    </td>
                    <td class="px-4 py-3">
                      <span class="rounded-full px-2 py-1 text-xs font-semibold" :class="statusClass(monitor.status)">
                        {{ statusText(monitor.status) }}
                      </span>
                    </td>
                    <td class="px-4 py-3 text-right font-mono text-gray-800 dark:text-dark-100">
                      {{ monitor.capture_count }}
                    </td>
                    <td class="px-4 py-3 text-gray-600 dark:text-dark-300">
                      {{ monitor.max_captures_per_minute }}/min · {{ monitor.sample_rate_percent }}%
                    </td>
                    <td class="px-4 py-3 text-gray-600 dark:text-dark-300">
                      {{ formatDate(monitor.ends_at) }}
                    </td>
                    <td class="px-4 py-3 text-right">
                      <div class="flex flex-wrap justify-end gap-2">
                        <button type="button" class="whitespace-nowrap text-primary-600 hover:text-primary-700" @click="selectMonitor(monitor)">
                          {{ t('admin.ops.userRequestMonitor.viewCaptures') }}
                        </button>
                        <button
                          type="button"
                          class="whitespace-nowrap text-blue-600 hover:text-blue-700 disabled:cursor-not-allowed disabled:opacity-60"
                          :disabled="downloadingMonitorId === monitor.id"
                          :data-test="`monitor-download-${monitor.id}`"
                          @click="downloadMonitor(monitor)"
                        >
                          {{ t('admin.ops.userRequestMonitor.download') }}
                        </button>
                        <button
                          type="button"
                          class="whitespace-nowrap text-red-600 hover:text-red-700 disabled:cursor-not-allowed disabled:opacity-60"
                          :disabled="deletingMonitorId === monitor.id"
                          :data-test="`monitor-delete-${monitor.id}`"
                          @click="deleteMonitor(monitor)"
                        >
                          {{ t('admin.ops.userRequestMonitor.deleteMonitor') }}
                        </button>
                        <button
                          v-if="monitor.status === 'active'"
                          type="button"
                          class="whitespace-nowrap text-red-600 hover:text-red-700"
                          @click="stopMonitor(monitor)"
                        >
                          {{ t('admin.ops.userRequestMonitor.stop') }}
                        </button>
                      </div>
                    </td>
                  </tr>
                </template>
              </tbody>
            </table>
          </div>
        </div>

        <div v-if="selectedMonitor" class="rounded-2xl border border-gray-100 p-4 dark:border-dark-800">
          <div class="flex items-center justify-between gap-3">
            <div>
              <h3 class="text-sm font-semibold text-gray-900 dark:text-dark-50">
                {{ t('admin.ops.userRequestMonitor.capturesTitle') }}
              </h3>
              <p class="text-xs text-gray-500">{{ selectedMonitor.target_email }}</p>
            </div>
            <button type="button" class="text-sm text-primary-600" @click="loadCaptures(selectedMonitor.id)">
              {{ t('admin.ops.userRequestMonitor.refresh') }}
            </button>
          </div>

          <div class="mt-3 max-h-[300px] overflow-auto rounded-xl border border-gray-100 dark:border-dark-800">
            <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-800">
              <thead class="bg-gray-50 text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-950 dark:text-dark-400">
                <tr>
                  <th class="px-3 py-2 text-left">{{ t('admin.ops.userRequestMonitor.capture.time') }}</th>
                  <th class="px-3 py-2 text-left">{{ t('admin.ops.userRequestMonitor.capture.requestId') }}</th>
                  <th class="px-3 py-2 text-left">{{ t('admin.ops.userRequestMonitor.capture.model') }}</th>
                  <th class="px-3 py-2 text-right">{{ t('admin.ops.userRequestMonitor.capture.bytes') }}</th>
                  <th class="px-3 py-2 text-right">{{ t('admin.ops.userRequestMonitor.capture.actions') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-800">
                <tr v-if="loadingCaptures">
                  <td colspan="5" class="px-3 py-6 text-center text-gray-500">{{ t('admin.ops.loadingText') }}</td>
                </tr>
                <tr v-else-if="captures.length === 0">
                  <td colspan="5" class="px-3 py-6 text-center text-gray-500">{{ t('admin.ops.userRequestMonitor.noCaptures') }}</td>
                </tr>
                <template v-else>
                  <tr v-for="capture in captures" :key="capture.id">
                    <td class="px-3 py-2 text-gray-600 dark:text-dark-300">{{ formatDate(capture.created_at) }}</td>
                    <td class="px-3 py-2 font-mono text-xs text-gray-700 dark:text-dark-200">{{ capture.request_id || '-' }}</td>
                    <td class="px-3 py-2 text-gray-700 dark:text-dark-200">{{ capture.model || '-' }}</td>
                    <td class="px-3 py-2 text-right font-mono">
                      {{ capture.body_bytes }}<span v-if="capture.body_truncated" class="ml-1 text-amber-600">{{ t('admin.ops.userRequestMonitor.truncated') }}</span>
                    </td>
                    <td class="px-3 py-2 text-right">
                      <button type="button" class="text-primary-600 hover:text-primary-700" @click="openCapture(capture.id)">
                        {{ t('admin.ops.userRequestMonitor.detail') }}
                      </button>
                    </td>
                  </tr>
                </template>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <div v-if="detail" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="detail = null">
      <div class="max-h-[86vh] w-full max-w-5xl overflow-hidden rounded-2xl bg-white shadow-2xl dark:bg-dark-900">
        <div class="flex items-center justify-between border-b border-gray-100 p-4 dark:border-dark-800">
          <div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-dark-50">{{ t('admin.ops.userRequestMonitor.detailTitle') }}</h3>
            <p class="text-xs text-gray-500">{{ detail.request_id || '-' }} · {{ detail.model || '-' }}</p>
          </div>
          <button type="button" class="rounded-lg px-3 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-800" @click="detail = null">
            {{ t('admin.ops.userRequestMonitor.close') }}
          </button>
        </div>
        <div class="space-y-3 overflow-auto p-4">
          <div class="grid gap-2 text-xs text-gray-500 md:grid-cols-4">
            <div>{{ t('admin.ops.userRequestMonitor.capture.endpoint') }}: {{ detail.inbound_endpoint || '-' }}</div>
            <div>{{ t('admin.ops.userRequestMonitor.capture.contentType') }}: {{ detail.content_type || '-' }}</div>
            <div>{{ t('admin.ops.userRequestMonitor.capture.bytes') }}: {{ detail.body_bytes }}</div>
            <div>{{ t('admin.ops.userRequestMonitor.capture.expiresAt') }}: {{ formatDate(detail.expires_at) }}</div>
          </div>
          <pre class="max-h-[56vh] overflow-auto rounded-xl bg-gray-950 p-4 text-xs leading-5 text-gray-100"><code>{{ detail.body || '' }}</code></pre>
          <div class="flex justify-end gap-2">
            <button type="button" class="rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-700" @click="copyBody">
              {{ t('admin.ops.userRequestMonitor.copy') }}
            </button>
            <button type="button" class="rounded-lg bg-red-600 px-3 py-2 text-sm font-semibold text-white" @click="deleteCapture(detail)">
              {{ t('admin.ops.userRequestMonitor.deleteCapture') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { opsAPI, type OpsUserRequestCapture, type OpsUserRequestMonitor } from '@/api/admin/ops'
import { useAppStore } from '@/stores'

const { t } = useI18n()
const appStore = useAppStore()

const form = reactive({
  userId: '',
  durationMinutes: '30',
  rateLimit: '10',
  sampleRate: '100',
  retentionDays: '7'
})

const monitors = ref<OpsUserRequestMonitor[]>([])
const captures = ref<OpsUserRequestCapture[]>([])
const detail = ref<OpsUserRequestCapture | null>(null)
const selectedMonitor = ref<OpsUserRequestMonitor | null>(null)
const loadingMonitors = ref(false)
const loadingCaptures = ref(false)
const downloadingMonitorId = ref<number | null>(null)
const deletingMonitorId = ref<number | null>(null)
const creating = ref(false)
const monitorError = ref('')
const statusFilter = ref('')
const userQuery = ref('')

const statusOptions = computed(() => [
  { value: '', label: t('admin.ops.userRequestMonitor.status.all') },
  { value: 'active', label: t('admin.ops.userRequestMonitor.status.active') },
  { value: 'expired', label: t('admin.ops.userRequestMonitor.status.expired') },
  { value: 'stopped', label: t('admin.ops.userRequestMonitor.status.stopped') }
])

async function loadMonitors() {
  loadingMonitors.value = true
  monitorError.value = ''
  try {
    const data = await opsAPI.listUserRequestMonitors({
      status: statusFilter.value || undefined,
      user_query: userQuery.value || undefined,
      page: 1,
      page_size: 20
    })
    monitors.value = data.items || []
  } catch (err: any) {
    monitorError.value = err?.message || t('admin.ops.userRequestMonitor.loadFailed')
  } finally {
    loadingMonitors.value = false
  }
}

async function createMonitor() {
  const userId = Number.parseInt(form.userId, 10)
  const durationMinutes = Number.parseInt(form.durationMinutes, 10)
  const rateLimit = Number.parseInt(form.rateLimit, 10)
  const sampleRate = Number.parseInt(form.sampleRate, 10)
  const retentionDays = Number.parseInt(form.retentionDays || '7', 10)
  if (!Number.isFinite(userId) || userId <= 0 || !Number.isFinite(durationMinutes) || durationMinutes <= 0) {
    appStore.showError(t('admin.ops.userRequestMonitor.invalidForm'))
    return
  }
  creating.value = true
  try {
    await opsAPI.createUserRequestMonitor({
      user_id: userId,
      duration_seconds: durationMinutes * 60,
      max_captures_per_minute: rateLimit,
      sample_rate_percent: sampleRate,
      retention_days: retentionDays || 7
    })
    appStore.showSuccess(t('admin.ops.userRequestMonitor.created'))
    form.userId = ''
    await loadMonitors()
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.ops.userRequestMonitor.createFailed'))
  } finally {
    creating.value = false
  }
}

async function stopMonitor(monitor: OpsUserRequestMonitor) {
  try {
    await opsAPI.stopUserRequestMonitor(monitor.id)
    appStore.showSuccess(t('admin.ops.userRequestMonitor.stopped'))
    await loadMonitors()
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.ops.userRequestMonitor.stopFailed'))
  }
}

async function downloadMonitor(monitor: OpsUserRequestMonitor) {
  downloadingMonitorId.value = monitor.id
  try {
    const blob = await opsAPI.downloadUserRequestMonitor(monitor.id)
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = monitorExportFilename(monitor)
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)
    appStore.showSuccess(t('admin.ops.userRequestMonitor.downloaded'))
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.ops.userRequestMonitor.downloadFailed'))
  } finally {
    downloadingMonitorId.value = null
  }
}

async function deleteMonitor(monitor: OpsUserRequestMonitor) {
  if (!window.confirm(t('admin.ops.userRequestMonitor.deleteConfirm', { email: monitor.target_email || monitor.user_id }))) {
    return
  }
  deletingMonitorId.value = monitor.id
  try {
    await opsAPI.deleteUserRequestMonitor(monitor.id)
    if (selectedMonitor.value?.id === monitor.id) {
      selectedMonitor.value = null
      captures.value = []
      detail.value = null
    }
    await loadMonitors()
    appStore.showSuccess(t('admin.ops.userRequestMonitor.monitorDeleted'))
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.ops.userRequestMonitor.deleteMonitorFailed'))
  } finally {
    deletingMonitorId.value = null
  }
}

async function selectMonitor(monitor: OpsUserRequestMonitor) {
  selectedMonitor.value = monitor
  await loadCaptures(monitor.id)
}

async function loadCaptures(monitorId: number) {
  loadingCaptures.value = true
  try {
    const data = await opsAPI.listUserRequestCaptures(monitorId, { page: 1, page_size: 20 })
    captures.value = data.items || []
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.ops.userRequestMonitor.loadCapturesFailed'))
  } finally {
    loadingCaptures.value = false
  }
}

async function openCapture(captureId: number) {
  if (!selectedMonitor.value) return
  try {
    detail.value = await opsAPI.getUserRequestCapture(selectedMonitor.value.id, captureId)
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.ops.userRequestMonitor.loadCaptureFailed'))
  }
}

async function deleteCapture(capture: OpsUserRequestCapture) {
  if (!selectedMonitor.value) return
  try {
    await opsAPI.deleteUserRequestCapture(selectedMonitor.value.id, capture.id)
    detail.value = null
    await loadCaptures(selectedMonitor.value.id)
    appStore.showSuccess(t('admin.ops.userRequestMonitor.deleted'))
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.ops.userRequestMonitor.deleteFailed'))
  }
}

async function copyBody() {
  if (!detail.value?.body) return
  try {
    await navigator.clipboard?.writeText(detail.value.body)
    appStore.showSuccess(t('admin.ops.userRequestMonitor.copied'))
  } catch {
    appStore.showError(t('admin.ops.userRequestMonitor.copyFailed'))
  }
}

function statusText(status: string) {
  if (status === 'active') return t('admin.ops.userRequestMonitor.status.active')
  if (status === 'expired') return t('admin.ops.userRequestMonitor.status.expired')
  if (status === 'stopped') return t('admin.ops.userRequestMonitor.status.stopped')
  return status || '-'
}

function statusClass(status: string) {
  if (status === 'active') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (status === 'stopped') return 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-300'
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
}

function formatDate(value?: string | null) {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleString()
}

function monitorExportFilename(monitor: OpsUserRequestMonitor) {
  const rawTarget = monitor.target_email || `user-${monitor.user_id}`
  const safeTarget = rawTarget.replace(/[^a-zA-Z0-9._-]+/g, '_').slice(0, 80)
  return `user-request-monitor-${monitor.id}-${safeTarget || monitor.user_id}.jsonl`
}

watch([statusFilter, userQuery], () => {
  loadMonitors()
})

onMounted(() => {
  loadMonitors()
})
</script>
