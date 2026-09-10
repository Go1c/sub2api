<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="min-w-0 flex-1">
            <p class="text-sm text-gray-700 dark:text-gray-200">
              {{ t('admin.channelIq.includedCount', { n: items.length }) }}
            </p>
            <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
              {{
                settings.auto_enabled
                  ? t('admin.channelIq.autoOn', { minutes: intervalMinutes })
                  : t('admin.channelIq.autoOff')
              }}
            </p>
          </div>
          <div class="flex flex-wrap items-center justify-end gap-2">
            <button
              type="button"
              class="btn btn-secondary inline-flex items-center gap-2"
              data-test="channel-iq-settings"
              @click="openSettings"
            >
              <Icon name="cog" size="sm" />
              {{ t('admin.channelIq.settings') }}
            </button>
            <button
              type="button"
              class="btn btn-primary inline-flex items-center gap-2"
              data-test="channel-iq-run-all"
              :disabled="running || starting || items.length === 0"
              @click="handleRunAll"
            >
              <Icon name="play" size="sm" :class="starting ? 'animate-pulse' : ''" />
              {{ t('admin.channelIq.runAll') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="items"
          :loading="loading"
          row-key="account_id"
        >
          <template #empty>
            <EmptyState
              :title="emptyTitle"
              :description="emptyDescription"
              :action-text="t('admin.channelIq.settings')"
              :action-icon="false"
              @action="openSettings"
            />
          </template>

          <template #cell-status="{ row }">
            <span
              class="inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold"
              :class="statusClass(row.status)"
            >
              {{ t(`admin.channelIq.status.${row.status}`) }}
            </span>
          </template>

          <template #cell-test_count="{ row }">
            <span class="ui-mono text-sm">{{ t('admin.channelIq.testCount', { n: row.test_count }) }}</span>
          </template>

          <template #cell-preview="{ row }">
            <button
              v-if="previewUrl(row.svg)"
              type="button"
              class="h-16 w-24 overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm transition hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800"
              @click="lightboxUrl = previewUrl(row.svg) || ''"
            >
              <img :src="previewUrl(row.svg) || ''" alt="" class="h-full w-full object-contain" />
            </button>
            <span v-else class="text-xs text-gray-400 dark:text-gray-500">
              {{ t('admin.channelIq.noPreview') }}
            </span>
          </template>

          <template #cell-model="{ row }">
            <code class="ui-mono text-xs text-gray-800 dark:text-gray-200">{{ row.model || '—' }}</code>
          </template>

          <template #cell-reasoning_effort="{ row }">
            <span class="ui-mono text-xs uppercase text-gray-600 dark:text-gray-300">{{
              row.reasoning_effort || 'low'
            }}</span>
          </template>

          <template #cell-stats="{ row }">
            <div v-if="row.duration_ms > 0 || row.total_tokens > 0" class="ui-mono text-sm">
              {{
                t('admin.channelIq.statsValue', {
                  duration: formatDuration(row.duration_ms),
                  tokens: formatTokens(row.total_tokens)
                })
              }}
            </div>
            <span v-else class="text-xs text-gray-400">—</span>
            <p v-if="row.status === 'failed' && row.error" class="mt-1 max-w-xs truncate text-xs text-red-500" :title="row.error">
              {{ row.error }}
            </p>
          </template>

          <template #cell-actions="{ row }">
            <button
              type="button"
              class="btn btn-secondary btn-sm inline-flex items-center gap-1"
              :data-test="`channel-iq-run-one-${row.account_id}`"
              :disabled="running || starting || row.status === 'running'"
              @click="handleRunOne(row.account_id)"
            >
              <Icon name="play" size="sm" />
              {{ t('admin.channelIq.runOne') }}
            </button>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showSettings"
      :title="t('admin.channelIq.settingsTitle')"
      width="wide"
      @close="showSettings = false"
    >
      <div class="space-y-5">
        <div>
          <label class="input-label">{{ t('admin.channelIq.groups') }}</label>
          <p class="mb-2 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.channelIq.groupsHint') }}</p>
          <div class="max-h-56 space-y-2 overflow-y-auto rounded-xl border border-gray-200 p-3 text-left dark:border-dark-600">
            <label
              v-for="group in groups"
              :key="group.id"
              class="flex w-full cursor-pointer items-center justify-start gap-2.5 text-left text-sm text-gray-800 dark:text-gray-200"
            >
              <input
                v-model="draftGroupIds"
                type="checkbox"
                class="h-4 w-4 shrink-0 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :value="group.id"
                :data-test="`channel-iq-group-${group.id}`"
              />
              <span class="min-w-0 truncate">{{ group.name }}</span>
              <span class="shrink-0 text-xs text-gray-400">{{ group.platform }}</span>
            </label>
            <p v-if="groups.length === 0" class="text-sm text-gray-500">{{ t('common.noData') }}</p>
          </div>
        </div>

        <div class="flex items-center justify-between gap-4">
          <div>
            <p class="text-sm font-medium text-gray-800 dark:text-gray-200">{{ t('admin.channelIq.autoEnabled') }}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.channelIq.autoHint') }}</p>
          </div>
          <Toggle v-model="draft.autoEnabled" />
        </div>

        <div>
          <label class="input-label" for="channel-iq-interval">{{ t('admin.channelIq.interval') }}</label>
          <select id="channel-iq-interval" v-model.number="draft.intervalMinutes" class="input">
            <option v-for="option in intervalOptions" :key="option" :value="option">
              {{ t('admin.channelIq.intervalMinutes', { n: option }) }}
            </option>
          </select>
        </div>

        <div>
          <label class="input-label" for="channel-iq-model">{{ t('admin.channelIq.model') }}</label>
          <input id="channel-iq-model" v-model="draft.model" type="text" class="input" />
        </div>

        <div>
          <label class="input-label" for="channel-iq-prompt">{{ t('admin.channelIq.prompt') }}</label>
          <textarea id="channel-iq-prompt" v-model="draft.prompt" rows="3" class="input" />
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="showSettings = false">
            {{ t('common.cancel') }}
          </button>
          <button
            type="button"
            class="btn btn-primary"
            data-test="channel-iq-save-settings"
            :disabled="saving"
            @click="saveSettings"
          >
            {{ t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <Teleport to="body">
      <Transition name="fade">
        <div
          v-if="lightboxUrl"
          class="fixed inset-0 z-[80] flex items-center justify-center bg-black/70 p-6"
          @click.self="lightboxUrl = ''"
        >
          <button
            type="button"
            class="absolute right-4 top-4 rounded-full bg-black/50 p-2 text-white hover:bg-black/70"
            @click="lightboxUrl = ''"
          >
            <Icon name="x" size="lg" :stroke-width="2" />
          </button>
          <img :src="lightboxUrl" alt="" class="max-h-[90vh] max-w-[90vw] rounded-lg object-contain shadow-2xl" />
        </div>
      </Transition>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { ChannelIQItem, ChannelIQSettings, ChannelIQStatus } from '@/api/admin/channelIq'
import type { AdminGroup } from '@/types'
import { iqSvgToImageUrl } from '@/components/admin/account/iqTest'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const POLL_MS = 2500
const intervalOptions = [5, 10, 15, 30, 60, 120]

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const starting = ref(false)
const saving = ref(false)
const showSettings = ref(false)
const lightboxUrl = ref('')
const items = ref<ChannelIQItem[]>([])
const settings = ref<ChannelIQSettings>({
  group_ids: [],
  auto_enabled: false,
  interval_seconds: 1800,
  prompt: '',
  model: ''
})
const groups = ref<AdminGroup[]>([])
const draftGroupIds = ref<number[]>([])
const draft = reactive({
  autoEnabled: false,
  intervalMinutes: 30,
  model: '',
  prompt: ''
})

let pollTimer: ReturnType<typeof setTimeout> | null = null
let loadCtrl: AbortController | null = null

const running = computed(() => items.value.some((item) => item.status === 'running'))
const intervalMinutes = computed(() => Math.max(5, Math.round((settings.value.interval_seconds || 1800) / 60)))
const hasGroups = computed(() => (settings.value.group_ids || []).length > 0)
const emptyTitle = computed(() =>
  hasGroups.value ? t('admin.channelIq.emptyAccountsTitle') : t('admin.channelIq.emptyGroupsTitle')
)
const emptyDescription = computed(() =>
  hasGroups.value ? t('admin.channelIq.emptyAccountsDescription') : t('admin.channelIq.emptyGroupsDescription')
)

const columns = computed(() => [
  { key: 'name', label: t('admin.channelIq.columns.name'), sortable: false },
  { key: 'status', label: t('admin.channelIq.columns.status'), sortable: false },
  { key: 'test_count', label: t('admin.channelIq.columns.testCount'), sortable: false },
  { key: 'preview', label: t('admin.channelIq.columns.preview'), sortable: false },
  { key: 'model', label: t('admin.channelIq.columns.model'), sortable: false },
  { key: 'reasoning_effort', label: t('admin.channelIq.columns.reasoning'), sortable: false },
  { key: 'stats', label: t('admin.channelIq.columns.stats'), sortable: false },
  { key: 'actions', label: t('admin.channelIq.columns.actions'), sortable: false }
])

function previewUrl(svg: string) {
  return iqSvgToImageUrl(svg)
}

function statusClass(status: ChannelIQStatus) {
  switch (status) {
    case 'running':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-300'
    case 'success':
      return 'bg-green-100 text-green-700 dark:bg-green-500/20 dark:text-green-400'
    case 'failed':
      return 'bg-red-100 text-red-700 dark:bg-red-500/20 dark:text-red-400'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-300'
  }
}

function formatDuration(ms: number) {
  if (ms < 1000) return `${ms} ms`
  return `${(ms / 1000).toFixed(1)} s`
}

function formatTokens(tokens: number) {
  return tokens.toLocaleString()
}

function isBusy(err: unknown) {
  return typeof err === 'object' && err !== null && (err as { status?: number }).status === 409
}

function stopPoll() {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

function schedulePoll() {
  stopPoll()
  if (!running.value) return
  pollTimer = setTimeout(async () => {
    await loadOverview(true)
    schedulePoll()
  }, POLL_MS)
}

async function loadOverview(quiet = false) {
  loadCtrl?.abort()
  const ctrl = new AbortController()
  loadCtrl = ctrl
  if (!quiet) loading.value = true
  try {
    const out = await adminAPI.channelIq.getOverview({ signal: ctrl.signal })
    settings.value = out.settings
    items.value = out.items || []
  } catch (err) {
    if ((err as { code?: string }).code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(err, t('admin.channelIq.loadError')))
  } finally {
    if (!quiet) loading.value = false
  }
}

async function loadGroups() {
  try {
    groups.value = await adminAPI.groups.getAll()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.channelIq.loadError')))
  }
}

function openSettings() {
  draftGroupIds.value = [...(settings.value.group_ids || [])]
  draft.autoEnabled = settings.value.auto_enabled
  draft.intervalMinutes = intervalMinutes.value
  if (!intervalOptions.includes(draft.intervalMinutes)) {
    draft.intervalMinutes = 30
  }
  draft.model = settings.value.model
  draft.prompt = settings.value.prompt
  showSettings.value = true
}

async function saveSettings() {
  saving.value = true
  try {
    await adminAPI.channelIq.updateSettings({
      group_ids: draftGroupIds.value,
      auto_enabled: draft.autoEnabled,
      interval_seconds: draft.intervalMinutes * 60,
      model: draft.model,
      prompt: draft.prompt
    })
    appStore.showSuccess(t('admin.channelIq.saveSuccess'))
    showSettings.value = false
    await loadOverview()
    schedulePoll()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.channelIq.saveFailed')))
  } finally {
    saving.value = false
  }
}

async function startRun(fn: () => Promise<unknown>) {
  if (items.value.length === 0) {
    appStore.showError(t('admin.channelIq.emptyRun'))
    return
  }
  starting.value = true
  try {
    await fn()
    appStore.showSuccess(t('admin.channelIq.started'))
    await loadOverview(true)
    schedulePoll()
  } catch (err) {
    if (isBusy(err)) {
      appStore.showError(t('admin.channelIq.busy'))
    } else {
      appStore.showError(extractApiErrorMessage(err, t('admin.channelIq.runFailed')))
    }
  } finally {
    starting.value = false
  }
}

function handleRunAll() {
  return startRun(() => adminAPI.channelIq.runAll())
}

function handleRunOne(accountId: number) {
  return startRun(() => adminAPI.channelIq.runOne(accountId))
}

onMounted(async () => {
  await Promise.all([loadOverview(), loadGroups()])
  schedulePoll()
})

onUnmounted(() => {
  stopPoll()
  loadCtrl?.abort()
})
</script>

<style>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

