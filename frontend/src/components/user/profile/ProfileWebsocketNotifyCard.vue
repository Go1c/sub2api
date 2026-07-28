<template>
  <div class="card" data-testid="profile-websocket-notify-card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t('profile.websocketNotify.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('profile.websocketNotify.description') }}
      </p>
    </div>
    <div class="space-y-6 px-6 py-6">
      <div class="flex items-center justify-between">
        <label class="input-label mb-0">{{ t('profile.websocketNotify.enabled') }}</label>
        <label class="relative inline-flex cursor-pointer items-center">
          <input
            v-model="enabled"
            type="checkbox"
            class="peer sr-only"
            @change="handleToggleMaster"
          />
          <div
            class="peer h-6 w-11 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-primary-600 peer-checked:after:translate-x-full peer-checked:after:border-white peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 dark:border-gray-600 dark:bg-gray-700 dark:peer-focus:ring-primary-800"
          />
        </label>
      </div>

      <template v-if="enabled">
        <div class="space-y-4 rounded-lg border border-gray-100 p-4 dark:border-dark-700">
          <div class="flex items-center justify-between gap-4">
            <div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">
                {{ t('profile.websocketNotify.balanceAlert') }}
              </div>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('profile.websocketNotify.balanceAlertHint') }}
              </p>
            </div>
            <label class="relative inline-flex shrink-0 cursor-pointer items-center">
              <input
                v-model="balanceAlertEnabled"
                type="checkbox"
                class="peer sr-only"
                @change="savePartial({ websocket_balance_alert_enabled: balanceAlertEnabled })"
              />
              <div
                class="peer h-5 w-9 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-4 after:w-4 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-primary-600 peer-checked:after:translate-x-full peer-checked:after:border-white dark:bg-gray-600"
              />
            </label>
          </div>
          <div v-if="balanceAlertEnabled">
            <label class="input-label">
              {{ t('profile.websocketNotify.balanceThreshold') }}
              <span class="ml-2 text-xs text-gray-400">{{
                t('profile.websocketNotify.balanceThresholdHint')
              }}</span>
            </label>
            <div class="flex items-center gap-2">
              <span class="text-gray-500">$</span>
              <input
                v-model.number="balanceThreshold"
                type="number"
                min="0"
                step="0.01"
                class="input flex-1"
                :placeholder="t('profile.websocketNotify.balanceThresholdPlaceholder')"
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
              {{ t('profile.websocketNotify.siteMessage') }}
            </div>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('profile.websocketNotify.siteMessageHint') }}
            </p>
          </div>
          <label class="relative inline-flex shrink-0 cursor-pointer items-center">
            <input
              v-model="siteMessageEnabled"
              type="checkbox"
              class="peer sr-only"
              @change="
                savePartial({ websocket_site_message_notify_enabled: siteMessageEnabled })
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
              {{ t('profile.websocketNotify.announcement') }}
            </div>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('profile.websocketNotify.announcementHint') }}
            </p>
          </div>
          <label class="relative inline-flex shrink-0 cursor-pointer items-center">
            <input
              v-model="announcementEnabled"
              type="checkbox"
              class="peer sr-only"
              @change="
                savePartial({ websocket_announcement_notify_enabled: announcementEnabled })
              "
            />
            <div
              class="peer h-5 w-9 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-4 after:w-4 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-primary-600 peer-checked:after:translate-x-full peer-checked:after:border-white dark:bg-gray-600"
            />
          </label>
        </div>

        <div class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('profile.websocketNotify.testHint') }}
          </p>
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="testing"
            @click="handleTest"
          >
            {{ testing ? t('profile.websocketNotify.testing') : t('profile.websocketNotify.test') }}
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

const enabled = ref(!!props.user.websocket_notify_enabled)
const balanceAlertEnabled = ref(props.user.websocket_balance_alert_enabled ?? true)
const balanceThreshold = ref<number | null>(props.user.websocket_balance_alert_threshold ?? null)
const siteMessageEnabled = ref(props.user.websocket_site_message_notify_enabled ?? true)
const announcementEnabled = ref(props.user.websocket_announcement_notify_enabled ?? true)
const savingThreshold = ref(false)
const testing = ref(false)

watch(
  () => props.user,
  (u) => {
    enabled.value = !!u.websocket_notify_enabled
    balanceAlertEnabled.value = u.websocket_balance_alert_enabled ?? true
    balanceThreshold.value = u.websocket_balance_alert_threshold ?? null
    siteMessageEnabled.value = u.websocket_site_message_notify_enabled ?? true
    announcementEnabled.value = u.websocket_announcement_notify_enabled ?? true
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
    appStore.showSuccess(t('profile.websocketNotify.saved'))
  } catch (e) {
    console.error(e)
    appStore.showError(t('profile.websocketNotify.saveFailed'))
  }
}

async function handleToggleMaster() {
  await savePartial({ websocket_notify_enabled: enabled.value })
}

async function handleSaveThreshold() {
  savingThreshold.value = true
  try {
    const threshold =
      balanceThreshold.value && balanceThreshold.value > 0 ? balanceThreshold.value : 0
    const updated = await userAPI.updateProfile({
      websocket_balance_alert_threshold: threshold
    })
    await applyUser(updated)
    balanceThreshold.value = updated.websocket_balance_alert_threshold ?? null
    appStore.showSuccess(t('profile.websocketNotify.saved'))
  } catch (e) {
    console.error(e)
    appStore.showError(t('profile.websocketNotify.saveFailed'))
  } finally {
    savingThreshold.value = false
  }
}

async function handleTest() {
  testing.value = true
  try {
    const res = await userAPI.sendWebsocketNotifyTest()
    appStore.showSuccess(res.message || t('profile.websocketNotify.testSent'))
  } catch (e: unknown) {
    console.error(e)
    const msg =
      (e as { response?: { data?: { message?: string } } })?.response?.data?.message ||
      t('profile.websocketNotify.testFailed')
    appStore.showError(msg)
  } finally {
    testing.value = false
  }
}
</script>
