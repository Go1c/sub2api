<template>
  <BaseDialog :show="show" :title="t('admin.users.siteMessage.title')" width="wide" @close="emit('close')">
    <form v-if="user" id="admin-user-site-message-form" class="space-y-4" @submit.prevent="handleSubmit">
      <div class="flex items-center gap-3 rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/40">
          <span class="text-base font-semibold text-primary-700 dark:text-primary-300">{{ user.email.charAt(0).toUpperCase() }}</span>
        </div>
        <div class="min-w-0 flex-1">
          <p class="truncate font-medium text-gray-900 dark:text-white">{{ user.email }}</p>
          <p class="mt-0.5 truncate text-sm text-gray-500 dark:text-gray-400">{{ user.username || `#${user.id}` }}</p>
        </div>
      </div>

      <div>
        <label class="input-label">{{ t('admin.users.siteMessage.subject') }}</label>
        <input v-model="form.subject" type="text" maxlength="200" class="input" required />
      </div>

      <div>
        <label class="input-label">{{ t('admin.users.siteMessage.content') }}</label>
        <textarea v-model="form.content" rows="8" class="input" required></textarea>
      </div>

      <label class="flex items-start gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
        <input
          v-model="form.sendEmail"
          type="checkbox"
          class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
        />
        <span class="min-w-0">
          <span class="block text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.users.siteMessage.sendEmail') }}</span>
          <span class="mt-1 block text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('admin.users.siteMessage.sendEmailHint') }}</span>
        </span>
      </label>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="submit" form="admin-user-site-message-form" class="btn btn-primary" :disabled="submitting || !canSubmit">
          {{ submitting ? t('admin.users.siteMessage.sending') : t('admin.users.siteMessage.send') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAppStore } from '@/stores/app'
import type { AdminUser } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{
  show: boolean
  user: AdminUser | null
}>()

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'success'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const submitting = ref(false)
const form = reactive({
  subject: '',
  content: '',
  sendEmail: true,
})

const canSubmit = computed(() => form.subject.trim().length > 0 && form.content.trim().length > 0)

watch(
  () => props.show,
  (show) => {
    if (show) {
      form.subject = ''
      form.content = ''
      form.sendEmail = true
    }
  }
)

async function handleSubmit(): Promise<void> {
  if (!props.user || !canSubmit.value) return
  submitting.value = true
  try {
    await adminAPI.siteMessages.sendToUser(props.user.id, {
      subject: form.subject.trim(),
      content: form.content.trim(),
      send_email: form.sendEmail,
    })
    appStore.showSuccess(t('admin.users.siteMessage.sent'))
    emit('success')
    emit('close')
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.users.siteMessage.failed')))
  } finally {
    submitting.value = false
  }
}
</script>
