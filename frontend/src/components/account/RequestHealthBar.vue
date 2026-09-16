<script setup lang="ts">
import { computed, ref } from 'vue'
import { sliceWindow, type RequestFailError, type RequestOutcome } from './requestHealth'
import RequestHealthErrorDialog from './RequestHealthErrorDialog.vue'

const props = defineProps<{
  outcomes: RequestOutcome[]
  windowSize: number
  sourceLabel?: string
}>()

const slots = computed(() => sliceWindow(props.outcomes, props.windowSize))
const showError = ref(false)
const selectedError = ref<RequestFailError | null>(null)

function openError(error?: RequestFailError) {
  if (!error) return
  selectedError.value = error
  showError.value = true
}

function closeError() {
  showError.value = false
}
</script>

<template>
  <div
    class="flex items-center gap-[2.5px]"
    :title="$t('admin.requestHealth.barTitle', { n: windowSize })"
  >
    <template v-for="(item, index) in slots" :key="index">
      <button
        v-if="item.slot === 'fail'"
        type="button"
        class="relative inline-flex h-3 w-[5px] flex-shrink-0 cursor-pointer items-center justify-center rounded-full bg-[#f43f5e] hover:bg-red-400"
        :title="$t('admin.requestHealth.errorHint')"
        @click.stop="openError(item.error)"
      >
        <span class="absolute -inset-1.5" aria-hidden="true" />
        <span class="sr-only">{{ $t('admin.requestHealth.viewError') }}</span>
      </button>
      <span
        v-else
        class="inline-block h-3 w-[5px] rounded-full"
        :class="item.slot === 'ok' ? 'bg-[#3dd68c]' : 'bg-gray-300 dark:bg-[#3a4558]'"
      />
    </template>
  </div>

  <RequestHealthErrorDialog
    :show="showError"
    :error="selectedError"
    :source-label="sourceLabel"
    @close="closeError"
  />
</template>
