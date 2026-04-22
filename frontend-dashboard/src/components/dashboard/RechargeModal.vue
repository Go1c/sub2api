<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

defineProps<{ open: boolean }>()
const emit = defineEmits<{ (e: 'close'): void }>()
const { t } = useI18n()

const amount = ref(50)
const options = [20, 50, 100, 200, 500]

function confirm() {
  window.alert(`[demo] top up ¥${amount.value}`)
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <transition name="fade">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-center justify-center bg-ink-900/40 backdrop-blur-sm p-4"
        @click.self="emit('close')"
      >
        <div class="bg-white rounded-lg shadow-card-hover w-full max-w-md p-6">
          <h3 class="brand-serif text-lg font-semibold text-ink-900">
            {{ t('dashboard.recharge.title') }}
          </h3>
          <p class="mt-1 text-xs text-ink-500 ui-sans">{{ t('dashboard.recharge.note') }}</p>

          <div class="mt-5">
            <span class="block text-xs text-ink-600 ui-sans mb-2">{{ t('dashboard.recharge.amount') }}</span>
            <div class="grid grid-cols-5 gap-2">
              <button
                v-for="v in options"
                :key="v"
                class="rounded-md border px-3 py-2 text-sm brand-serif"
                :class="
                  amount === v
                    ? 'border-brand-500 bg-brand-50 text-brand-700'
                    : 'border-ink-200 text-ink-700 hover:border-brand-300'
                "
                @click="amount = v"
              >
                ¥{{ v }}
              </button>
            </div>
          </div>

          <div class="mt-6 flex justify-end gap-2">
            <button class="px-4 py-2 text-sm text-ink-600 hover:text-ink-900 ui-sans" @click="emit('close')">
              {{ t('dashboard.recharge.cancel') }}
            </button>
            <button class="btn-brand" @click="confirm">
              {{ t('dashboard.recharge.confirm') }}
            </button>
          </div>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
