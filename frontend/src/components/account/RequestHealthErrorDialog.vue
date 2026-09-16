<script setup lang="ts">
import BaseDialog from '@/components/common/BaseDialog.vue'
import { formatDateTime } from '@/utils/format'
import type { RequestFailError } from './requestHealth'

defineProps<{
  show: boolean
  error: RequestFailError | null
  sourceLabel?: string
}>()

const emit = defineEmits<{
  close: []
}>()
</script>

<template>
  <BaseDialog
    :show="show"
    :title="$t('admin.requestHealth.errorTitle')"
    width="normal"
    :close-on-click-outside="true"
    :z-index="80"
    @close="emit('close')"
  >
    <div v-if="error" class="space-y-4 text-sm">
      <div class="grid grid-cols-2 gap-x-6 gap-y-3">
        <div>
          <span class="font-medium text-gray-500 dark:text-dark-400">{{ $t('admin.requestHealth.time') }}</span>
          <p class="mt-0.5 text-gray-900 dark:text-dark-100">{{ formatDateTime(error.occurredAt) }}</p>
        </div>
        <div>
          <span class="font-medium text-gray-500 dark:text-dark-400">{{ $t('admin.requestHealth.statusCode') }}</span>
          <p class="mt-0.5">
            <span class="badge badge-danger">{{ error.statusCode }}</span>
          </p>
        </div>
        <div v-if="sourceLabel">
          <span class="font-medium text-gray-500 dark:text-dark-400">{{ $t('admin.requestHealth.egress') }}</span>
          <p class="mt-0.5 font-mono text-gray-900 dark:text-dark-100">{{ sourceLabel }}</p>
        </div>
        <div v-if="error.model">
          <span class="font-medium text-gray-500 dark:text-dark-400">{{ $t('admin.requestHealth.model') }}</span>
          <p class="mt-0.5 text-gray-900 dark:text-dark-100">{{ error.model }}</p>
        </div>
        <div v-if="error.endpoint" class="col-span-2">
          <span class="font-medium text-gray-500 dark:text-dark-400">{{ $t('admin.requestHealth.endpoint') }}</span>
          <p class="mt-0.5 font-mono text-gray-900 dark:text-dark-100">{{ error.endpoint }}</p>
        </div>
      </div>
      <div>
        <span class="font-medium text-gray-500 dark:text-dark-400">{{ $t('admin.requestHealth.message') }}</span>
        <p class="mt-1 whitespace-pre-wrap break-all rounded-lg border border-gray-200 bg-gray-50 p-3 text-gray-800 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-200">
          {{ error.message }}
        </p>
      </div>
    </div>
    <template #footer>
      <button type="button" class="btn btn-secondary" @click="emit('close')">
        {{ $t('admin.requestHealth.close') }}
      </button>
    </template>
  </BaseDialog>
</template>
