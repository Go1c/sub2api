<template>
  <div class="card" data-testid="profile-webhook-balance-notify-card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t('profile.webhookBalanceNotify.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('profile.webhookBalanceNotify.description') }}
      </p>
    </div>
    <div class="space-y-6 px-6 py-6">
      <div class="flex items-center justify-between">
        <label class="input-label mb-0">{{ t('profile.webhookBalanceNotify.enabled') }}</label>
        <label class="relative inline-flex cursor-pointer items-center">
          <input v-model="enabled" type="checkbox" class="peer sr-only" @change="handleToggle" />
          <div
            class="peer h-6 w-11 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-primary-600 peer-checked:after:translate-x-full peer-checked:after:border-white peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 dark:border-gray-600 dark:bg-gray-700 dark:peer-focus:ring-primary-800"
          />
        </label>
      </div>

      <template v-if="enabled">
        <div>
          <label class="input-label">{{ t('profile.webhookBalanceNotify.webhookUrl') }}</label>
          <p class="mb-2 text-xs text-gray-500 dark:text-gray-400">
            {{ t('profile.webhookBalanceNotify.webhookUrlHint') }}
          </p>
          <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
            <input
              v-model="webhookUrl"
              type="url"
              class="input flex-1 font-mono text-sm"
              :placeholder="t('profile.webhookBalanceNotify.webhookUrlPlaceholder')"
            />
            <button
              type="button"
              class="btn btn-primary btn-sm whitespace-nowrap"
              :disabled="savingUrl"
              @click="handleSaveUrl"
            >
              {{ savingUrl ? t('common.saving') : t('common.save') }}
            </button>
          </div>
        </div>

        <div>
          <label class="input-label">
            {{ t('profile.webhookBalanceNotify.threshold') }}
            <span class="ml-2 text-xs text-gray-400">{{
              t('profile.webhookBalanceNotify.thresholdHint')
            }}</span>
          </label>
          <div class="flex items-center gap-2">
            <span class="text-gray-500">$</span>
            <input
              v-model.number="threshold"
              type="number"
              min="0"
              step="0.01"
              class="input flex-1"
              :placeholder="t('profile.webhookBalanceNotify.thresholdPlaceholder')"
            />
            <button
              type="button"
              class="btn btn-primary btn-sm whitespace-nowrap"
              :disabled="savingThreshold"
              @click="handleSaveThreshold"
            >
              {{ savingThreshold ? t('common.saving') : t('common.save') }}
            </button>
          </div>
        </div>

        <div
          class="flex flex-col gap-3 border-t border-gray-100 pt-4 sm:flex-row sm:items-center sm:justify-between dark:border-dark-700"
        >
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('profile.webhookBalanceNotify.testHint') }}
          </p>
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="testing || !webhookUrl"
            @click="handleTest"
          >
            {{ testing ? t('profile.webhookBalanceNotify.testing') : t('profile.webhookBalanceNotify.test') }}
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import * as userAPI from '@/api/user'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import type { User } from '@/types'

const props = defineProps<{
  user: User
}>()

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const enabled = ref(!!props.user.webhook_balance_notify_enabled)
const webhookUrl = ref(props.user.webhook_balance_notify_url || '')
const threshold = ref<number | null>(props.user.webhook_balance_notify_threshold ?? null)
const savingUrl = ref(false)
const savingThreshold = ref(false)
const testing = ref(false)

watch(
  () => props.user,
  (u) => {
    enabled.value = !!u.webhook_balance_notify_enabled
    webhookUrl.value = u.webhook_balance_notify_url || ''
    threshold.value = u.webhook_balance_notify_threshold ?? null
  },
  { deep: true }
)

async function applyUser(updated: User) {
  authStore.user = updated
}

async function handleToggle() {
  try {
    const updated = await userAPI.updateProfile({
      webhook_balance_notify_enabled: enabled.value
    })
    await applyUser(updated)
    appStore.showSuccess(t('profile.webhookBalanceNotify.saved'))
  } catch (e) {
    console.error(e)
    enabled.value = !enabled.value
    appStore.showError(t('profile.webhookBalanceNotify.saveFailed'))
  }
}

async function handleSaveUrl() {
  savingUrl.value = true
  try {
    const updated = await userAPI.updateProfile({
      webhook_balance_notify_url: webhookUrl.value.trim()
    })
    await applyUser(updated)
    webhookUrl.value = updated.webhook_balance_notify_url || ''
    appStore.showSuccess(t('profile.webhookBalanceNotify.saved'))
  } catch (e) {
    console.error(e)
    appStore.showError(t('profile.webhookBalanceNotify.saveFailed'))
  } finally {
    savingUrl.value = false
  }
}

async function handleSaveThreshold() {
  savingThreshold.value = true
  try {
    const value = threshold.value && threshold.value > 0 ? threshold.value : 0
    const updated = await userAPI.updateProfile({
      webhook_balance_notify_threshold: value
    })
    await applyUser(updated)
    threshold.value = updated.webhook_balance_notify_threshold ?? null
    appStore.showSuccess(t('profile.webhookBalanceNotify.saved'))
  } catch (e) {
    console.error(e)
    appStore.showError(t('profile.webhookBalanceNotify.saveFailed'))
  } finally {
    savingThreshold.value = false
  }
}

async function handleTest() {
  testing.value = true
  try {
    // ensure latest url is saved before test
    if ((props.user.webhook_balance_notify_url || '') !== webhookUrl.value.trim()) {
      await handleSaveUrl()
    }
    const res = await userAPI.sendWebhookBalanceNotifyTest()
    appStore.showSuccess(res.message || t('profile.webhookBalanceNotify.testSent'))
  } catch (e: unknown) {
    console.error(e)
    const msg =
      (e as { response?: { data?: { message?: string } } })?.response?.data?.message ||
      t('profile.webhookBalanceNotify.testFailed')
    appStore.showError(msg)
  } finally {
    testing.value = false
  }
}
</script>
