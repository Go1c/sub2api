<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="min-w-0 flex-1">
            <p class="text-sm text-gray-700 dark:text-gray-200">
              {{ t('admin.channelTurnState.includedCount', { n: accounts.length }) }}
            </p>
            <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
              {{
                policy.enabled
                  ? t('admin.channelTurnState.policyOn')
                  : t('admin.channelTurnState.policyOff')
              }}
            </p>
          </div>
          <div class="flex flex-wrap items-center justify-end gap-2">
            <button
              type="button"
              class="btn btn-secondary inline-flex items-center gap-2"
              data-testid="turn-state-settings"
              @click="openSettings"
            >
              <Icon name="cog" size="sm" />
              {{ t('admin.channelTurnState.settings') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="accounts" :loading="loading" row-key="account_id">
          <template #empty>
            <EmptyState
              :title="t('admin.channelTurnState.emptyTitle')"
              :description="t('admin.channelTurnState.emptyDescription')"
              :action-text="t('admin.channelTurnState.settings')"
              :action-icon="false"
              @action="openSettings"
            />
          </template>

          <template #cell-status="{ row }">
            <span
              class="inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold"
              :class="statusClass(row.status)"
            >
              {{ statusLabel(row.status) }}
            </span>
          </template>

          <template #cell-state_hash="{ row }">
            <code
              v-if="row.state_hash"
              class="ui-mono text-xs text-gray-800 dark:text-gray-200"
              :data-testid="`turn-state-hash-${row.account_id}`"
            >
              {{ row.state_hash }}
            </code>
            <span v-else class="text-xs text-gray-400 dark:text-gray-500">
              {{ t('admin.channelTurnState.noHash') }}
            </span>
          </template>

          <template #cell-state_length="{ row }">
            <span
              class="ui-mono text-sm"
              :data-testid="`turn-state-length-${row.account_id}`"
            >
              {{ row.state_length ? row.state_length : '—' }}
            </span>
          </template>

          <template #cell-model="{ row }">
            <code class="ui-mono text-xs text-gray-800 dark:text-gray-200">{{ row.model || '—' }}</code>
          </template>

          <template #cell-sticky_proxy_id="{ row }">
            <span v-if="row.sticky_proxy_id">#{{ row.sticky_proxy_id }}<br />{{ row.sticky_until ? formatDateTime(row.sticky_until) : '—' }}</span>
            <span v-else>—</span>
          </template>
          <template #cell-recheck_at="{ row }">
            <span class="text-xs text-gray-600 dark:text-gray-300">
              {{ row.recheck_at ? formatDateTime(row.recheck_at) : '—' }}
            </span>
            <p
              v-if="(row.status === 'failed' || row.status === 'skipped' || row.status === 'cooldown') && row.last_error"
              class="mt-1 max-w-xs truncate text-xs text-red-500"
              :title="row.last_error"
            >
              {{ row.last_error }}
            </p>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center justify-end gap-2">
              <button
                type="button"
                class="btn btn-secondary btn-sm inline-flex items-center gap-1"
                :data-testid="`turn-state-run-one-${row.account_id}`"
                :disabled="starting || row.status === 'running'"
                @click="handleRunOne(row.account_id)"
              >
                <Icon name="play" size="sm" />
                {{ t('admin.channelTurnState.runOne') }}
              </button>
              <button
                type="button"
                class="btn btn-ghost btn-sm inline-flex items-center gap-1 text-red-600 hover:bg-red-50 dark:text-red-300 dark:hover:bg-red-950/30"
                :data-testid="`turn-state-clear-${row.account_id}`"
                :disabled="starting"
                @click="askClear(row)"
              >
                <Icon name="trash" size="sm" />
                {{ t('admin.channelTurnState.clear') }}
              </button>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showSettings"
      :title="t('admin.channelTurnState.settingsTitle')"
      width="wide"
      @close="showSettings = false"
    >
      <div class="space-y-5">
        <div class="flex items-center justify-between gap-4">
          <div>
            <p class="text-sm font-medium text-gray-800 dark:text-gray-200">
              {{ t('admin.channelTurnState.policyEnabled') }}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.channelTurnState.policyHint') }}
            </p>
          </div>
          <Toggle v-model="draft.enabled" data-testid="turn-state-policy-enabled" />
        </div>

        <div>
          <label class="input-label">{{ t('admin.channelTurnState.exits') }}</label>
          <p class="mb-2 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.channelTurnState.exitsHint') }}
          </p>
          <p class="mb-2 text-xs font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.channelTurnState.existingProxies') }}
          </p>
          <div
            class="max-h-56 space-y-2 overflow-y-auto rounded-xl border border-gray-200 p-3 text-left dark:border-dark-600"
          >
            <label
              v-for="proxy in proxies"
              :key="proxy.id"
              class="flex w-full cursor-pointer items-center justify-start gap-2.5 text-left text-sm text-gray-800 dark:text-gray-200"
            >
              <input
                v-model="draft.proxyIds"
                type="checkbox"
                class="h-4 w-4 shrink-0 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :value="proxy.id"
                :data-testid="`turn-state-proxy-${proxy.id}`"
              />
              <span class="min-w-0 truncate">{{ proxy.name }}</span>
              <span class="shrink-0 text-xs text-gray-400">{{ proxy.host }}:{{ proxy.port }}</span>
            </label>
            <p v-if="proxies.length === 0" class="text-sm text-gray-500">{{ t('common.noData') }}</p>
          </div>
        </div>

        <div>
          <p class="input-label">{{ t('admin.channelTurnState.dynamicExit') }}</p>
          <div class="mt-2 grid grid-cols-1 gap-3 md:grid-cols-2">
            <div class="md:col-span-2">
              <label class="input-label" for="turn-state-dynamic-host">
                {{ t('admin.channelTurnState.dynamicHost') }}
              </label>
              <input
                id="turn-state-dynamic-host"
                v-model="draft.host"
                type="text"
                class="input"
                data-testid="turn-state-dynamic-host"
                :placeholder="t('admin.channelTurnState.dynamicHostPlaceholder')"
              />
            </div>
            <div>
              <label class="input-label" for="turn-state-dynamic-username">
                {{ t('admin.channelTurnState.dynamicUsername') }}
              </label>
              <input
                id="turn-state-dynamic-username"
                v-model="draft.username"
                type="text"
                class="input"
                data-testid="turn-state-dynamic-username"
                autocomplete="off"
              />
            </div>
            <div>
              <label class="input-label" for="turn-state-dynamic-password">
                {{ t('admin.channelTurnState.dynamicPassword') }}
              </label>
              <input
                id="turn-state-dynamic-password"
                v-model="draft.password"
                type="password"
                class="input"
                data-testid="turn-state-dynamic-password"
                autocomplete="new-password"
                :placeholder="t('admin.channelTurnState.dynamicPasswordKeep')"
              />
              <p
                v-if="policy.dynamic.password_set && !draft.password"
                class="mt-1 text-xs text-gray-500 dark:text-gray-400"
                data-testid="turn-state-password-set"
              >
                {{ t('admin.channelTurnState.passwordSet') }}
              </p>
            </div>
            <div>
              <label class="input-label" for="turn-state-dynamic-region">
                {{ t('admin.channelTurnState.dynamicRegion') }}
              </label>
              <input
                id="turn-state-dynamic-region"
                v-model="draft.region"
                type="text"
                class="input"
                data-testid="turn-state-dynamic-region"
              />
            </div>
            <div>
              <label class="input-label" for="turn-state-dynamic-session">
                {{ t('admin.channelTurnState.dynamicSessionMinutes') }}
              </label>
              <input
                id="turn-state-dynamic-session"
                v-model.number="draft.sessionMinutes"
                type="number"
                min="1"
                max="60"
                class="input"
                data-testid="turn-state-dynamic-session"
              />
            </div>
          </div>
        </div>

        <div>
          <label class="input-label" for="turn-state-model">{{ t('admin.channelTurnState.model') }}</label>
          <input
            id="turn-state-model"
            v-model="draft.model"
            type="text"
            class="input"
            data-testid="turn-state-model"
          />
        </div>

        <div class="flex items-center justify-between gap-4">
          <div>
            <p class="text-sm font-medium text-gray-800 dark:text-gray-200">
              {{ t('admin.channelTurnState.lengthFilter') }}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.channelTurnState.lengthFilterHint') }}
            </p>
          </div>
          <Toggle v-model="draft.lengthFilterEnabled" data-testid="turn-state-length-filter" />
        </div>

        <div>
          <label class="input-label" for="turn-state-min-length">
            {{ t('admin.channelTurnState.minStateLength') }}
          </label>
          <input
            id="turn-state-min-length"
            v-model.number="draft.minStateLength"
            type="number"
            min="1"
            max="4096"
            class="input"
            data-testid="turn-state-min-length"
          />
        </div>

        <div>
          <label class="input-label" for="turn-state-question">{{ t('admin.channelTurnState.question') }}</label>
          <textarea
            id="turn-state-question"
            v-model="draft.question"
            rows="4"
            class="input"
            data-testid="turn-state-question"
          />
        </div>

        <div>
          <label class="input-label" for="turn-state-answer">{{ t('admin.channelTurnState.answer') }}</label>
          <input
            id="turn-state-answer"
            v-model="draft.answer"
            type="text"
            class="input"
            data-testid="turn-state-answer"
          />
        </div>

        <div class="flex items-center justify-between gap-4">
          <div>
            <p class="text-sm font-medium text-gray-800 dark:text-gray-200">
              {{ t('admin.channelTurnState.fuzzyMatch') }}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.channelTurnState.fuzzyMatchHint') }}
            </p>
          </div>
          <Toggle v-model="draft.fuzzyMatch" data-testid="turn-state-fuzzy" />
        </div>

        <div>
          <label class="input-label" for="turn-state-recheck">{{ t('admin.channelTurnState.recheck') }}</label>
          <select
            id="turn-state-recheck"
            v-model.number="draft.recheckMinutes"
            class="input"
            data-testid="turn-state-recheck"
            disabled
          >
            <option v-for="option in recheckOptions" :key="option" :value="option">
              {{ t('admin.channelTurnState.recheckMinutes', { n: option }) }}
            </option>
          </select>
        </div>

        <div>
          <label class="input-label" for="turn-state-overload-threshold">{{ t('admin.channelTurnState.overloadThreshold') }}</label>
          <input id="turn-state-overload-threshold" v-model.number="draft.overloadThreshold" type="number" min="0" max="20" step="1" class="input" data-testid="turn-state-overload-threshold" />
          <p class="mt-1 text-xs text-gray-500">{{ t('admin.channelTurnState.overloadThresholdHint') }}</p>
        </div>

        <div>
          <label class="input-label" for="turn-state-rpm">{{ t('admin.channelTurnState.rpm') }}</label>
          <input
            id="turn-state-rpm"
            v-model.number="draft.rpm"
            type="number"
            min="1"
            max="60"
            class="input"
            data-testid="turn-state-rpm"
          />
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
            data-testid="turn-state-save-policy"
            :disabled="saving"
            @click="savePolicy"
          >
            {{ t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="pendingClear !== null"
      :title="t('admin.channelTurnState.clearTitle')"
      :message="t('admin.channelTurnState.clearConfirm', { name: pendingClear?.name || '' })"
      :confirm-text="t('admin.channelTurnState.clear')"
      :cancel-text="t('common.cancel')"
      danger
      @confirm="confirmClear"
      @cancel="pendingClear = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import {
  defaultTurnStateProbePolicy,
  type TurnStateProbeAccountItem,
  type TurnStateProbePolicy,
  type TurnStateProbePolicyPayload
} from '@/api/admin/turnStateProbe'
import type { Proxy } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorCode, extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

const POLL_MS = 2500
const RECHECK_PRESETS = [20]

const { t, te } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const starting = ref(false)
const saving = ref(false)
const showSettings = ref(false)
const accounts = ref<TurnStateProbeAccountItem[]>([])
const pendingClear = ref<TurnStateProbeAccountItem | null>(null)
const policy = ref<TurnStateProbePolicy>(defaultTurnStateProbePolicy())
const proxies = ref<Proxy[]>([])
const draft = reactive({
  enabled: false,
  proxyIds: [] as number[],
  host: '',
  username: '',
  password: '',
  region: 'Random',
  sessionMinutes: 5,
  model: 'gpt-6-astra',
  lengthFilterEnabled: true,
  minStateLength: 160,
  question: '',
  answer: '21',
  fuzzyMatch: true,
  recheckMinutes: 20,
  overloadThreshold: 3,
  rpm: 6
})

let pollTimer: ReturnType<typeof setTimeout> | null = null
let loadCtrl: AbortController | null = null

const recheckOptions = computed(() => {
  const current = draft.recheckMinutes
  if (RECHECK_PRESETS.includes(current)) return RECHECK_PRESETS
  return [...RECHECK_PRESETS, current].sort((a, b) => a - b)
})

const columns = computed(() => [
  { key: 'name', label: t('admin.channelTurnState.columns.name'), sortable: false },
  { key: 'status', label: t('admin.channelTurnState.columns.status'), sortable: false },
  { key: 'state_hash', label: t('admin.channelTurnState.columns.hash'), sortable: false },
  { key: 'state_length', label: t('admin.channelTurnState.columns.length'), sortable: false },
  { key: 'model', label: t('admin.channelTurnState.columns.model'), sortable: false },
  { key: 'sticky_proxy_id', label: t('admin.accounts.proxyModeSticky'), sortable: false },
  { key: 'recheck_at', label: t('admin.channelTurnState.columns.recheck'), sortable: false },
  { key: 'actions', label: t('admin.channelTurnState.columns.actions'), sortable: false }
])

function statusLabel(status: string) {
  const key = `admin.channelTurnState.status.${status}`
  return te(key) ? t(key) : status || t('admin.channelTurnState.status.idle')
}

function statusClass(status: string) {
  switch (status) {
    case 'running':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-300'
    case 'holding':
      return 'bg-green-100 text-green-700 dark:bg-green-500/20 dark:text-green-400'
    case 'cooldown':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-300'
    case 'skipped':
      return 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-300'
    case 'failed':
      return 'bg-red-100 text-red-700 dark:bg-red-500/20 dark:text-red-400'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-300'
  }
}

function isBusy(err: unknown) {
  return (
    extractApiErrorCode(err) === 'TURN_STATE_PROBE_BUSY' ||
    (typeof err === 'object' && err !== null && (err as { status?: number }).status === 409)
  )
}

function isDisabled(err: unknown) {
  return extractApiErrorCode(err) === 'TURN_STATE_PROBE_DISABLED'
}

function stopPoll() {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

function schedulePoll() {
  stopPoll()
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
    const out = await adminAPI.turnStateProbe.getOverview({ signal: ctrl.signal })
    policy.value = { ...defaultTurnStateProbePolicy(), ...out.policy, dynamic: { ...defaultTurnStateProbePolicy().dynamic, ...out.policy.dynamic } }
    accounts.value = out.accounts || []
  } catch (err) {
    if ((err as { code?: string }).code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(err, t('admin.channelTurnState.loadError')))
  } finally {
    if (!quiet) loading.value = false
  }
}

async function loadProxies() {
  try {
    proxies.value = await adminAPI.proxies.getAll()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.channelTurnState.loadError')))
  }
}

function openSettings() {
  const current = policy.value
  draft.enabled = current.enabled
  draft.proxyIds = [...(current.proxy_ids || [])]
  draft.host = current.dynamic.host || ''
  draft.username = current.dynamic.username || ''
  draft.password = ''
  draft.region = current.dynamic.region || 'Random'
  draft.sessionMinutes = current.dynamic.session_minutes || 5
  draft.model = current.model || 'gpt-6-astra'
  draft.lengthFilterEnabled = current.length_filter_enabled
  draft.minStateLength = current.min_state_length || 160
  draft.question = current.question || ''
  draft.answer = current.answer || '21'
  draft.fuzzyMatch = current.fuzzy_match
  draft.recheckMinutes = 20
  draft.overloadThreshold = current.overload_threshold ?? 3
  draft.rpm = current.rpm || 6
  showSettings.value = true
}

function buildPolicyPayload(): TurnStateProbePolicyPayload {
  const dynamic: TurnStateProbePolicyPayload['dynamic'] = {
    host: draft.host.trim(),
    username: draft.username.trim(),
    region: draft.region.trim() || 'Random',
    session_minutes: Number(draft.sessionMinutes) || 5
  }
  const password = draft.password
  if (password) {
    dynamic.password = password
  }
  return {
    enabled: draft.enabled,
    proxy_ids: draft.proxyIds.map((id) => Number(id)).filter((id) => Number.isFinite(id) && id > 0),
    dynamic,
    model: draft.model.trim() || 'gpt-6-astra',
    length_filter_enabled: draft.lengthFilterEnabled,
    min_state_length: Number(draft.minStateLength) || 160,
    question: draft.question.trim(),
    answer: draft.answer.trim() || '21',
    fuzzy_match: draft.fuzzyMatch,
    recheck_minutes: 20,
    overload_threshold: Number(draft.overloadThreshold),
    rpm: Number(draft.rpm) || 6
  }
}

async function savePolicy() {
  saving.value = true
  try {
    await adminAPI.turnStateProbe.updatePolicy(buildPolicyPayload())
    appStore.showSuccess(t('admin.channelTurnState.saveSuccess'))
    showSettings.value = false
    await loadOverview()
    schedulePoll()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.channelTurnState.saveFailed')))
  } finally {
    saving.value = false
  }
}

async function handleRunOne(accountId: number) {
  starting.value = true
  try {
    await adminAPI.turnStateProbe.runOne(accountId)
    appStore.showSuccess(t('admin.channelTurnState.started'))
    await loadOverview(true)
    schedulePoll()
  } catch (err) {
    if (isBusy(err)) {
      appStore.showError(t('admin.channelTurnState.busy'))
    } else if (isDisabled(err)) {
      appStore.showError(t('admin.channelTurnState.disabled'))
    } else {
      appStore.showError(extractApiErrorMessage(err, t('admin.channelTurnState.runFailed')))
    }
  } finally {
    starting.value = false
  }
}

function askClear(row: TurnStateProbeAccountItem) {
  pendingClear.value = row
}

async function confirmClear() {
  const row = pendingClear.value
  pendingClear.value = null
  if (!row) return
  starting.value = true
  try {
    await adminAPI.turnStateProbe.clearTicket(row.account_id)
    appStore.showSuccess(t('admin.channelTurnState.clearSuccess'))
    await loadOverview(true)
    schedulePoll()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.channelTurnState.clearFailed')))
  } finally {
    starting.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadOverview(), loadProxies()])
  schedulePoll()
})

onUnmounted(() => {
  stopPoll()
  loadCtrl?.abort()
})
</script>
