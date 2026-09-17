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
          <label class="input-label" for="pool-auto-inspect-threshold">{{ t('admin.accounts.poolAutoInspect.threshold') }}</label>
          <input
            id="pool-auto-inspect-threshold"
            v-model.number="form.success_rate_threshold"
            type="number"
            min="1"
            max="100"
            class="input"
            data-testid="pool-auto-inspect-threshold"
          />
        </div>
        <div>
          <label class="input-label" for="pool-auto-inspect-min-samples">{{ t('admin.accounts.poolAutoInspect.minSamples') }}</label>
          <input
            id="pool-auto-inspect-min-samples"
            v-model.number="form.min_samples"
            type="number"
            min="1"
            max="20"
            class="input"
          />
        </div>
      </div>

      <div>
        <label class="input-label">
          {{ t('admin.accounts.poolAutoInspect.addGroups') }}
          <span class="font-normal text-gray-400">{{ t('common.selectedCount', { count: form.add_group_ids.length }) }}</span>
        </label>
        <p class="mb-2 text-xs text-gray-500">{{ t('admin.accounts.poolAutoInspect.addGroupsHint') }}</p>
        <div class="grid max-h-40 grid-cols-1 gap-1 overflow-y-auto rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-dark-600 dark:bg-dark-800 md:grid-cols-2">
          <label
            v-for="group in groups"
            :key="group.id"
            class="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 hover:bg-white dark:hover:bg-dark-700"
          >
            <input
              type="checkbox"
              class="h-3.5 w-3.5 rounded border-gray-300 text-primary-500"
              :checked="form.add_group_ids.includes(group.id)"
              :data-testid="`pool-auto-inspect-group-${group.id}`"
              @change="toggleGroup(group.id, ($event.target as HTMLInputElement).checked)"
            />
            <GroupBadge
              :name="group.name"
              :platform="group.platform"
              :subscription-type="group.subscription_type || undefined"
              :rate-multiplier="group.rate_multiplier == null ? undefined : group.rate_multiplier"
              class="min-w-0 flex-1"
            />
          </label>
          <div v-if="groups.length === 0" class="col-span-2 py-2 text-center text-sm text-gray-500">
            {{ t('common.noGroupsAvailable') }}
          </div>
        </div>
      </div>

      <div>
        <label class="input-label" for="pool-auto-inspect-models">{{ t('admin.accounts.poolAutoInspect.removeModels') }}</label>
        <p class="mb-2 text-xs text-gray-500">{{ t('admin.accounts.poolAutoInspect.removeModelsHint') }}</p>
        <div class="flex gap-2">
          <input
            id="pool-auto-inspect-models"
            v-model="modelInput"
            type="text"
            class="input"
            :placeholder="t('admin.accounts.poolAutoInspect.removeModelsPlaceholder')"
            data-testid="pool-auto-inspect-model-input"
            @keydown.enter.prevent="addModel"
          />
          <button type="button" class="btn btn-secondary" @click="addModel">{{ t('common.add') }}</button>
        </div>
        <div class="mt-2 flex flex-wrap gap-2">
          <button
            v-for="model in form.remove_models"
            :key="model"
            type="button"
            class="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2.5 py-1 text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200"
            @click="removeModel(model)"
          >
            {{ model }}
            <Icon name="x" size="xs" />
          </button>
        </div>
      </div>

      <div class="flex items-center justify-between gap-3">
        <div>
          <div class="font-medium text-gray-900 dark:text-white">{{ t('admin.accounts.poolAutoInspect.close429Exemption') }}</div>
          <p class="text-xs text-gray-500">{{ t('admin.accounts.poolAutoInspect.close429ExemptionHint') }}</p>
        </div>
        <Toggle v-model="form.close_429_exemption_on_degrade" data-testid="pool-auto-inspect-toggle-close429" />
      </div>

      <div class="space-y-3 rounded-xl border border-gray-200 p-4 dark:border-dark-600">
        <div class="flex items-center justify-between gap-3">
          <div>
            <div class="font-medium text-gray-900 dark:text-white">{{ t('admin.accounts.poolAutoInspect.notify401') }}</div>
            <p class="text-xs text-gray-500">{{ t('admin.accounts.poolAutoInspect.notify401Hint') }}</p>
          </div>
          <Toggle v-model="form.notify_oauth_401" data-testid="pool-auto-inspect-toggle-notify401" />
        </div>
        <div v-if="form.notify_oauth_401" class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.accounts.poolAutoInspect.telegramBotToken') }}</label>
            <input v-model.trim="form.telegram_bot_token" type="password" autocomplete="off" class="input" placeholder="123456:ABC..." />
          </div>
          <div>
            <label class="input-label">{{ t('admin.accounts.poolAutoInspect.telegramChatId') }}</label>
            <input v-model.trim="form.telegram_chat_id" type="text" class="input" placeholder="-1001234567890" />
          </div>
          <div>
            <label class="input-label" for="pool-auto-inspect-cooldown">{{ t('admin.accounts.poolAutoInspect.cooldown') }}</label>
            <input
              id="pool-auto-inspect-cooldown"
              v-model.number="form.oauth_401_cooldown_minutes"
              type="number"
              min="1"
              max="1440"
              class="input"
              data-testid="pool-auto-inspect-cooldown"
            />
            <p class="mt-1 text-xs text-gray-500">{{ t('admin.accounts.poolAutoInspect.cooldownHint') }}</p>
          </div>
          <p class="text-xs text-gray-500 md:col-span-2">{{ t('admin.accounts.poolAutoInspect.telegramFallbackHint') }}</p>
        </div>
      </div>

      <pre
        v-if="statusText"
        class="whitespace-pre-wrap rounded-xl border border-gray-200 bg-gray-50 p-3 text-xs text-gray-600 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300"
        data-testid="pool-auto-inspect-status"
      >{{ statusText }}</pre>
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
import GroupBadge from '@/components/common/GroupBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { AdminGroup } from '@/types'
import type { PoolAutoInspectConfig, PoolAutoInspectStatus } from '@/api/admin/accounts'

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
const saving = ref(false)
const running = ref(false)
const modelInput = ref('')
const lastStatus = ref<PoolAutoInspectStatus | null>(null)

const emptyForm = (): PoolAutoInspectConfig => ({
  enabled: false,
  interval_minutes: 5,
  success_rate_threshold: 50,
  min_samples: 4,
  add_group_ids: [],
  remove_models: [],
  notify_oauth_401: false,
  oauth_401_cooldown_minutes: 60,
  close_429_exemption_on_degrade: true,
  telegram_bot_token: '',
  telegram_chat_id: ''
})

const form = reactive<PoolAutoInspectConfig>(emptyForm())

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

function applyConfig(cfg: PoolAutoInspectConfig | PoolAutoInspectStatus) {
  form.enabled = !!cfg.enabled
  form.interval_minutes = cfg.interval_minutes || 5
  form.success_rate_threshold = cfg.success_rate_threshold || 50
  form.min_samples = cfg.min_samples || 4
  form.add_group_ids = [...(cfg.add_group_ids || [])]
  form.remove_models = [...(cfg.remove_models || [])]
  form.notify_oauth_401 = !!cfg.notify_oauth_401
  form.oauth_401_cooldown_minutes = cfg.oauth_401_cooldown_minutes || 60
  form.close_429_exemption_on_degrade = cfg.close_429_exemption_on_degrade !== false
  form.telegram_bot_token = cfg.telegram_bot_token || ''
  form.telegram_chat_id = cfg.telegram_chat_id || ''
}

async function load() {
  loading.value = true
  try {
    const cfg = await adminAPI.accounts.getPoolAutoInspectConfig()
    lastStatus.value = cfg
    applyConfig(cfg)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.accounts.poolAutoInspect.loadFailed')))
  } finally {
    loading.value = false
  }
}

function toggleGroup(id: number, checked: boolean) {
  if (checked && !form.add_group_ids.includes(id)) {
    form.add_group_ids.push(id)
    return
  }
  form.add_group_ids = form.add_group_ids.filter((groupId) => groupId !== id)
}

function addModel() {
  const model = modelInput.value.trim()
  if (!model) return
  if (!form.remove_models.includes(model)) {
    form.remove_models.push(model)
  }
  modelInput.value = ''
}

function removeModel(model: string) {
  form.remove_models = form.remove_models.filter((item) => item !== model)
}

async function save() {
  saving.value = true
  try {
    const updated = await adminAPI.accounts.updatePoolAutoInspectConfig({ ...form })
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
    const updated = await adminAPI.accounts.updatePoolAutoInspectConfig({ ...form })
    applyConfig(updated)
    const status = await adminAPI.accounts.runPoolAutoInspect()
    lastStatus.value = status
    appStore.showSuccess(t('admin.accounts.poolAutoInspect.runDone'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.accounts.poolAutoInspect.runFailed')))
  } finally {
    running.value = false
  }
}
</script>
