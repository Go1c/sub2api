<template>
  <BaseDialog :show="show" :title="t('admin.accounts.poolAutoInspect.title')" width="wide" @close="emit('close')">
    <div v-if="loading" class="py-8 text-center text-sm text-gray-500">{{ t('common.loading') }}</div>
    <div v-else class="space-y-5">
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.accounts.poolAutoInspect.hint') }}</p>

      <div class="flex items-center justify-between gap-3">
        <div>
          <div class="font-medium text-gray-900 dark:text-white">{{ t('admin.accounts.poolAutoInspect.enable') }}</div>
          <p class="text-xs text-gray-500">{{ t('admin.accounts.poolAutoInspect.enableHint') }}</p>
        </div>
        <Toggle v-model="form.enabled" data-testid="pool-auto-inspect-toggle-enabled" />
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <div>
          <label class="input-label" for="pool-auto-inspect-interval">{{ t('admin.accounts.poolAutoInspect.interval') }}</label>
          <input
            id="pool-auto-inspect-interval"
            v-model.number="form.interval_minutes"
            type="number"
            min="1"
            max="1440"
            class="input"
            data-testid="pool-auto-inspect-interval"
          />
          <p class="mt-1 text-xs text-gray-500">{{ t('admin.accounts.poolAutoInspect.intervalHint') }}</p>
        </div>
        <div>
          <label class="input-label" for="pool-auto-inspect-jitter">{{ t('admin.accounts.poolAutoInspect.jitter') }}</label>
          <input
            id="pool-auto-inspect-jitter"
            v-model.number="form.jitter_seconds"
            type="number"
            min="0"
            :max="maxJitterSeconds"
            class="input"
            data-testid="pool-auto-inspect-jitter"
          />
          <p class="mt-1 text-xs text-gray-500">{{ t('admin.accounts.poolAutoInspect.jitterHint') }}</p>
        </div>
        <div>
          <label class="input-label" for="pool-auto-inspect-pause">{{ t('admin.accounts.poolAutoInspect.pause') }}</label>
          <input
            id="pool-auto-inspect-pause"
            v-model.number="form.pause_minutes"
            type="number"
            min="1"
            max="1440"
            class="input"
            data-testid="pool-auto-inspect-pause"
          />
          <p class="mt-1 text-xs text-gray-500">{{ t('admin.accounts.poolAutoInspect.pauseHint') }}</p>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div>
          <label class="input-label" for="pool-auto-inspect-model">{{ t('admin.accounts.poolAutoInspect.model') }}</label>
          <input
            id="pool-auto-inspect-model"
            v-model.trim="form.model"
            type="text"
            class="input"
            maxlength="80"
            data-testid="pool-auto-inspect-model"
          />
        </div>
        <div>
          <label class="input-label" for="pool-auto-inspect-answer">{{ t('admin.accounts.poolAutoInspect.answer') }}</label>
          <input
            id="pool-auto-inspect-answer"
            v-model.trim="form.answer"
            type="text"
            class="input"
            maxlength="200"
            data-testid="pool-auto-inspect-answer"
          />
          <p class="mt-1 text-xs text-gray-500">{{ t('admin.accounts.poolAutoInspect.answerHint') }}</p>
        </div>
      </div>

      <div>
        <label class="input-label" for="pool-auto-inspect-question">{{ t('admin.accounts.poolAutoInspect.question') }}</label>
        <textarea
          id="pool-auto-inspect-question"
          v-model="form.question"
          rows="4"
          maxlength="2000"
          class="input"
          data-testid="pool-auto-inspect-question"
        />
        <p class="mt-1 text-xs text-gray-500">{{ t('admin.accounts.poolAutoInspect.questionHint') }}</p>
      </div>

      <div class="flex items-center justify-between gap-3">
        <div>
          <div class="font-medium text-gray-900 dark:text-white">{{ t('admin.accounts.poolAutoInspect.fuzzyMatch') }}</div>
          <p class="text-xs text-gray-500">{{ t('admin.accounts.poolAutoInspect.fuzzyMatchHint') }}</p>
        </div>
        <Toggle v-model="form.fuzzy_match" data-testid="pool-auto-inspect-toggle-fuzzy" />
      </div>

      <div class="flex items-center justify-between gap-3">
        <div>
          <div class="font-medium text-gray-900 dark:text-white">{{ t('admin.accounts.poolAutoInspect.disableFirstImport') }}</div>
          <p class="text-xs text-gray-500">{{ t('admin.accounts.poolAutoInspect.disableFirstImportHint') }}</p>
        </div>
        <Toggle v-model="form.disable_first_import_on_incorrect" data-testid="pool-auto-inspect-toggle-first-import" />
      </div>

      <div>
        <div class="flex items-center justify-between gap-3">
          <div>
            <div class="font-medium text-gray-900 dark:text-white">{{ t('admin.accounts.poolAutoInspect.accounts') }}</div>
            <p class="text-xs text-gray-500">{{ t('admin.accounts.poolAutoInspect.accountsHint') }}</p>
          </div>
          <span class="shrink-0 text-xs text-gray-500" data-testid="pool-auto-inspect-account-count">
            {{ t('admin.accounts.poolAutoInspect.accountsCount', { selected: selectedAccountCount, total: inspectableAccounts.length }) }}
          </span>
        </div>
        <div class="mt-2 flex flex-wrap items-center gap-2">
          <input
            v-model.trim="accountSearch"
            type="search"
            class="input min-w-0 flex-1"
            :placeholder="t('admin.accounts.poolAutoInspect.accountsSearch')"
            data-testid="pool-auto-inspect-account-search"
          />
          <button type="button" class="btn btn-secondary btn-sm" data-testid="pool-auto-inspect-select-all" @click="selectAllAccounts">
            {{ t('admin.accounts.poolAutoInspect.selectAll') }}
          </button>
          <button type="button" class="btn btn-secondary btn-sm" data-testid="pool-auto-inspect-clear-all" @click="clearAllAccounts">
            {{ t('admin.accounts.poolAutoInspect.clearAll') }}
          </button>
        </div>
        <div
          class="mt-2 max-h-56 space-y-2 overflow-y-auto rounded-xl border border-gray-200 p-3 text-left dark:border-dark-600"
          data-testid="pool-auto-inspect-accounts"
        >
          <p v-if="accountsLoading" class="text-sm text-gray-500">{{ t('common.loading') }}</p>
          <p v-else-if="inspectableAccounts.length === 0" class="text-sm text-gray-500">
            {{ t('admin.accounts.poolAutoInspect.accountsEmpty') }}
          </p>
          <p v-else-if="filteredAccounts.length === 0" class="text-sm text-gray-500">{{ t('common.noData') }}</p>
          <label
            v-for="account in filteredAccounts"
            :key="account.id"
            class="flex w-full cursor-pointer items-center justify-start gap-2.5 text-left text-sm text-gray-800 dark:text-gray-200"
          >
            <input
              type="checkbox"
              class="h-4 w-4 shrink-0 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="isAccountIncluded(account.id)"
              :data-testid="`pool-auto-inspect-account-${account.id}`"
              @change="toggleAccount(account.id, ($event.target as HTMLInputElement).checked)"
            />
            <span class="min-w-0 truncate">{{ account.name || `#${account.id}` }}</span>
          </label>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div>
          <label class="input-label" for="pool-auto-inspect-correct-group">{{ t('admin.accounts.poolAutoInspect.correctGroup') }}</label>
          <select
            id="pool-auto-inspect-correct-group"
            v-model.number="form.correct_group_id"
            class="input"
            data-testid="pool-auto-inspect-correct-group"
          >
            <option :value="0">{{ t('admin.accounts.poolAutoInspect.groupPlaceholder') }}</option>
            <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
          </select>
        </div>
        <div>
          <label class="input-label" for="pool-auto-inspect-incorrect-group">{{ t('admin.accounts.poolAutoInspect.incorrectGroup') }}</label>
          <select
            id="pool-auto-inspect-incorrect-group"
            v-model.number="form.incorrect_group_id"
            class="input"
            data-testid="pool-auto-inspect-incorrect-group"
          >
            <option :value="0">{{ t('admin.accounts.poolAutoInspect.groupPlaceholder') }}</option>
            <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
          </select>
        </div>
      </div>

      <pre
        v-if="statusText"
        class="whitespace-pre-wrap rounded-xl border border-gray-200 bg-gray-50 p-3 text-xs text-gray-600 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300"
        data-testid="pool-auto-inspect-status"
      >{{ statusText }}</pre>

      <div>
        <div class="mb-2 font-medium text-gray-900 dark:text-white">{{ t('admin.accounts.poolAutoInspect.logTitle') }}</div>
        <p v-if="logEntries.length === 0" class="text-xs text-gray-500" data-testid="pool-auto-inspect-log-empty">
          {{ t('admin.accounts.poolAutoInspect.logEmpty') }}
        </p>
        <ul
          v-else
          class="max-h-48 space-y-1 overflow-y-auto rounded-xl border border-gray-200 bg-gray-50 p-3 text-xs text-gray-700 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300"
          data-testid="pool-auto-inspect-log"
        >
          <li v-for="(entry, index) in logEntries" :key="`${entry.at}-${entry.account_id}-${index}`">
            <span class="text-gray-400">{{ formatLogTime(entry.at) }}</span>
            {{ ' ' }}
            <span>{{ formatLogEntry(entry) }}</span>
          </li>
        </ul>
      </div>
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="saving || running" @click="emit('close')">{{ t('common.cancel') }}</button>
      <button
        type="button"
        class="btn btn-secondary"
        :disabled="saving || running || loading"
        data-testid="pool-auto-inspect-run"
        @click="runNow"
      >
        {{ running ? t('admin.accounts.poolAutoInspect.running') : t('admin.accounts.poolAutoInspect.runNow') }}
      </button>
      <button
        type="button"
        class="btn btn-primary"
        :disabled="saving || running || loading"
        data-testid="pool-auto-inspect-save"
        @click="save"
      >
        {{ saving ? t('common.saving') : t('common.save') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Toggle from '@/components/common/Toggle.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { AccountListItem, AdminGroup } from '@/types'
import type { PoolAutoInspectConfig, PoolAutoInspectLogEntry, PoolAutoInspectStatus } from '@/api/admin/accounts'

const props = defineProps<{
  show: boolean
  groups: AdminGroup[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const accountsLoading = ref(false)
const saving = ref(false)
const running = ref(false)
const lastStatus = ref<PoolAutoInspectStatus | null>(null)
const logEntries = ref<PoolAutoInspectLogEntry[]>([])
const inspectableAccounts = ref<AccountListItem[]>([])
const accountSearch = ref('')
const excludedAccountIds = ref<number[]>([])

const emptyForm = (): PoolAutoInspectConfig => ({
  enabled: false,
  interval_minutes: 10,
  jitter_seconds: 60,
  model: 'gpt-6-astra',
  question: '',
  answer: '21',
  fuzzy_match: true,
  correct_group_id: 0,
  incorrect_group_id: 0,
  pause_minutes: 1,
  disable_first_import_on_incorrect: false,
  excluded_account_ids: []
})

const form = reactive<PoolAutoInspectConfig>(emptyForm())

const filteredAccounts = computed(() => {
  const query = accountSearch.value.trim().toLowerCase()
  if (!query) return inspectableAccounts.value
  return inspectableAccounts.value.filter((account) => {
    const name = (account.name || '').toLowerCase()
    return name.includes(query) || String(account.id).includes(query)
  })
})

const selectedAccountCount = computed(
  () => inspectableAccounts.value.filter((account) => isAccountIncluded(account.id)).length
)

const maxJitterSeconds = computed(() => {
  const minutes = Number(form.interval_minutes) || 1
  const half = Math.floor((minutes * 60) / 2)
  const floor = minutes * 60 - 60
  return Math.max(0, Math.min(half, floor))
})

const statusText = computed(() => {
  const status = lastStatus.value
  if (!status?.last_run_at && !status?.last_result) return ''
  const parts = [t('admin.accounts.poolAutoInspect.lastRun')]
  if (status.last_run_at) parts.push(new Date(status.last_run_at).toLocaleString())
  if (status.last_result) parts.push(status.last_result)
  if (status.last_error) parts.push(status.last_error)
  return parts.join(' · ')
})

watch(
  () => props.show,
  async (show) => {
    if (!show) return
    await load()
  },
  { immediate: true }
)

function numberOr(value: number | undefined, fallback: number) {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

function isInspectableAccount(account: AccountListItem) {
  if (!account?.id || account.parent_account_id) return false
  if (account.platform !== 'openai') return false
  return account.type === 'oauth' || account.type === 'setup-token'
}

function excludedIdsFrom(ids: number[] | undefined) {
  const seen = new Set<number>()
  const out: number[] = []
  for (const id of ids || []) {
    if (!Number.isFinite(id) || id <= 0 || seen.has(id)) continue
    seen.add(id)
    out.push(id)
  }
  return out
}

function isAccountIncluded(id: number) {
  return !excludedAccountIds.value.includes(id)
}

function toggleAccount(id: number, included: boolean) {
  if (included) {
    excludedAccountIds.value = excludedAccountIds.value.filter((item) => item !== id)
    return
  }
  if (!excludedAccountIds.value.includes(id)) {
    excludedAccountIds.value = [...excludedAccountIds.value, id]
  }
}

function selectAllAccounts() {
  const visible = new Set(filteredAccounts.value.map((account) => account.id))
  excludedAccountIds.value = excludedAccountIds.value.filter((id) => !visible.has(id))
}

function clearAllAccounts() {
  const next = new Set(excludedAccountIds.value)
  for (const account of filteredAccounts.value) next.add(account.id)
  excludedAccountIds.value = [...next]
}

function payload(): PoolAutoInspectConfig {
  return {
    ...form,
    excluded_account_ids: excludedIdsFrom(excludedAccountIds.value)
  }
}

function applyConfig(cfg: PoolAutoInspectConfig | PoolAutoInspectStatus) {
  form.enabled = !!cfg.enabled
  form.interval_minutes = numberOr(cfg.interval_minutes, 10) || 10
  form.jitter_seconds = numberOr(cfg.jitter_seconds, 60)
  form.model = cfg.model || 'gpt-6-astra'
  form.question = cfg.question || ''
  form.answer = cfg.answer || '21'
  form.fuzzy_match = cfg.fuzzy_match !== false
  form.correct_group_id = numberOr(cfg.correct_group_id, 0)
  form.incorrect_group_id = numberOr(cfg.incorrect_group_id, 0)
  form.pause_minutes = numberOr(cfg.pause_minutes, 1) || 1
  form.disable_first_import_on_incorrect = !!cfg.disable_first_import_on_incorrect
  excludedAccountIds.value = excludedIdsFrom(cfg.excluded_account_ids)
  form.excluded_account_ids = excludedAccountIds.value
}

function groupLabel(name: string | undefined, id: number | undefined) {
  const trimmed = (name || '').trim()
  if (trimmed) return trimmed
  if (id && id > 0) return `#${id}`
  return t('admin.accounts.poolAutoInspect.logUnknownGroup')
}

function formatLogTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function formatLogEntry(entry: PoolAutoInspectLogEntry) {
  const name = entry.account_name || `#${entry.account_id}`
  if (entry.action === 'disabled') {
    return t('admin.accounts.poolAutoInspect.logDisabled', { name })
  }
  return t('admin.accounts.poolAutoInspect.logMoved', {
    name,
    from: groupLabel(entry.from_group_name, entry.from_group_id),
    to: groupLabel(entry.to_group_name, entry.to_group_id)
  })
}

async function loadLog() {
  try {
    logEntries.value = await adminAPI.accounts.getPoolAutoInspectLog()
  } catch {
    logEntries.value = []
  }
}

async function loadAccounts() {
  accountsLoading.value = true
  try {
    const pageSize = 100
    const collected: AccountListItem[] = []
    let page = 1
    let pages = 1
    do {
      const result = await adminAPI.accounts.list(page, pageSize, { platform: 'openai' })
      collected.push(...(result.items || []).filter(isInspectableAccount))
      pages = result.pages || 1
      page += 1
    } while (page <= pages && page <= 20)
    inspectableAccounts.value = collected
  } catch (error) {
    inspectableAccounts.value = []
    appStore.showError(extractApiErrorMessage(error, t('admin.accounts.poolAutoInspect.loadFailed')))
  } finally {
    accountsLoading.value = false
  }
}

async function load() {
  loading.value = true
  accountSearch.value = ''
  try {
    const cfg = await adminAPI.accounts.getPoolAutoInspectConfig()
    lastStatus.value = cfg
    applyConfig(cfg)
    await Promise.all([loadLog(), loadAccounts()])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.accounts.poolAutoInspect.loadFailed')))
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const updated = await adminAPI.accounts.updatePoolAutoInspectConfig(payload())
    applyConfig(updated)
    appStore.showSuccess(t('admin.accounts.poolAutoInspect.saved'))
    emit('close')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.accounts.poolAutoInspect.saveFailed')))
  } finally {
    saving.value = false
  }
}

async function runNow() {
  running.value = true
  try {
    const updated = await adminAPI.accounts.updatePoolAutoInspectConfig(payload())
    applyConfig(updated)
    const status = await adminAPI.accounts.runPoolAutoInspect()
    lastStatus.value = status
    await loadLog()
    appStore.showSuccess(t('admin.accounts.poolAutoInspect.runDone'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.accounts.poolAutoInspect.runFailed')))
  } finally {
    running.value = false
  }
}
</script>
