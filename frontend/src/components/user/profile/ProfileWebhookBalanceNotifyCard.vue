<template>
  <div class="card" data-testid="profile-webhook-notify-card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t('profile.webhookNotify.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('profile.webhookNotify.description') }}
      </p>
    </div>
    <div class="space-y-6 px-6 py-6">
      <div class="flex items-center justify-between">
        <label class="input-label mb-0">{{ t('profile.webhookNotify.enabled') }}</label>
        <label class="relative inline-flex cursor-pointer items-center">
          <input v-model="enabled" type="checkbox" class="peer sr-only" @change="handleToggle" />
          <div
            class="peer h-6 w-11 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-primary-600 peer-checked:after:translate-x-full peer-checked:after:border-white peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 dark:border-gray-600 dark:bg-gray-700 dark:peer-focus:ring-primary-800"
          />
        </label>
      </div>

      <template v-if="enabled">
        <div>
          <label class="input-label">{{ t('profile.webhookNotify.webhookUrl') }}</label>
          <p class="mb-2 text-xs text-gray-500 dark:text-gray-400">
            {{ t('profile.webhookNotify.webhookUrlHint') }}
          </p>
          <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
            <input
              v-model="webhookUrl"
              type="url"
              class="input flex-1 font-mono text-sm"
              :placeholder="t('profile.webhookNotify.webhookUrlPlaceholder')"
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

        <div class="space-y-4 rounded-lg border border-gray-100 p-4 dark:border-dark-700">
          <div class="flex items-center justify-between gap-4">
            <div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">
                {{ t('profile.webhookNotify.balanceAlert') }}
              </div>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('profile.webhookNotify.balanceAlertHint') }}
              </p>
            </div>
            <span class="text-xs text-gray-400">{{ t('profile.webhookNotify.alwaysOnWithMaster') }}</span>
          </div>
          <div>
            <label class="input-label">
              {{ t('profile.webhookNotify.threshold') }}
              <span class="ml-2 text-xs text-gray-400">{{
                t('profile.webhookNotify.thresholdHint')
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
                :placeholder="t('profile.webhookNotify.thresholdPlaceholder')"
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
        </div>

        <div class="flex items-center justify-between gap-4">
          <div>
            <div class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t('profile.webhookNotify.siteMessage') }}
            </div>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('profile.webhookNotify.siteMessageHint') }}
            </p>
          </div>
          <label class="relative inline-flex shrink-0 cursor-pointer items-center">
            <input
              v-model="siteMessageEnabled"
              type="checkbox"
              class="peer sr-only"
              @change="
                savePartial({ webhook_site_message_notify_enabled: siteMessageEnabled })
              "
            />
            <div
              class="peer h-5 w-9 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-4 after:w-4 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-primary-600 peer-checked:after:translate-x-full peer-checked:after:border-white dark:bg-gray-600"
            />
          </label>
        </div>

        <div class="flex items-center justify-between gap-4">
          <div>
            <div class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t('profile.webhookNotify.announcement') }}
            </div>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('profile.webhookNotify.announcementHint') }}
            </p>
          </div>
          <label class="relative inline-flex shrink-0 cursor-pointer items-center">
            <input
              v-model="announcementEnabled"
              type="checkbox"
              class="peer sr-only"
              @change="
                savePartial({ webhook_announcement_notify_enabled: announcementEnabled })
              "
            />
            <div
              class="peer h-5 w-9 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-4 after:w-4 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-primary-600 peer-checked:after:translate-x-full peer-checked:after:border-white dark:bg-gray-600"
            />
          </label>
        </div>

        <div
          class="flex flex-col gap-3 border-t border-gray-100 pt-4 sm:flex-row sm:items-center sm:justify-between dark:border-dark-700"
        >
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('profile.webhookNotify.testHint') }}
          </p>
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="testing || !webhookUrl"
            @click="handleTest"
          >
            {{ testing ? t('profile.webhookNotify.testing') : t('profile.webhookNotify.test') }}
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
const siteMessageEnabled = ref(props.user.webhook_site_message_notify_enabled ?? true)
const announcementEnabled = ref(props.user.webhook_announcement_notify_enabled ?? true)
const savingUrl = ref(false)
const savingThreshold = ref(false)
const testing = ref(false)

watch(
  () => props.user,
  (u) => {
    enabled.value = !!u.webhook_balance_notify_enabled
    webhookUrl.value = u.webhook_balance_notify_url || ''
    threshold.value = u.webhook_balance_notify_threshold ?? null
    siteMessageEnabled.value = u.webhook_site_message_notify_enabled ?? true
    announcementEnabled.value = u.webhook_announcement_notify_enabled ?? true
  },
  { deep: true }
)

async function applyUser(updated: User) {
  authStore.user = updated
}

async function savePartial(patch: Parameters<typeof userAPI.updateProfile>[0]) {
  try {
    const updated = await userAPI.updateProfile(patch)
    await applyUser(updated)
    appStore.showSuccess(t('profile.webhookNotify.saved'))
  } catch (e) {
    console.error(e)
    appStore.showError(t('profile.webhookNotify.saveFailed'))
  }
}

async function handleToggle() {
  await savePartial({ webhook_balance_notify_enabled: enabled.value })
}

async function handleSaveUrl() {
  savingUrl.value = true
  try {
    const updated = await userAPI.updateProfile({
      webhook_balance_notify_url: webhookUrl.value.trim()
    })
    await applyUser(updated)
    webhookUrl.value = updated.webhook_balance_notify_url || ''
    appStore.showSuccess(t('profile.webhookNotify.saved'))
  } catch (e) {
    console.error(e)
    appStore.showError(t('profile.webhookNotify.saveFailed'))
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
    appStore.showSuccess(t('profile.webhookNotify.saved'))
  } catch (e) {
    console.error(e)
    appStore.showError(t('profile.webhookNotify.saveFailed'))
  } finally {
    savingThreshold.value = false
  }
}

async function handleTest() {
  testing.value = true
  try {
    if ((props.user.webhook_balance_notify_url || '') !== webhookUrl.value.trim()) {
      await handleSaveUrl()
    }
    const res = await userAPI.sendWebhookBalanceNotifyTest()
    appStore.showSuccess(res.message || t('profile.webhookNotify.testSent'))
  } catch (e: unknown) {
    console.error(e)
    const msg =
      (e as { response?: { data?: { message?: string } } })?.response?.data?.message ||
      t('profile.webhookNotify.testFailed')
    appStore.showError(msg)
  } finally {
    testing.value = false
  }
}
</script>
