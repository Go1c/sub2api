<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/common/Icon.vue'

defineProps<{
  invited: number
  totalBonus: number
  monthBonus: number
  code: string
}>()

const { t } = useI18n()
const copied = ref(false)

async function copy(code: string) {
  try {
    await navigator.clipboard.writeText(code)
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  } catch {
    // noop
  }
}
</script>

<template>
  <div class="card">
    <h3 class="text-sm font-semibold text-ink-900 mb-4">{{ t('dashboard.invite.title') }}</h3>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-5">
      <div class="bg-ink-50 rounded-lg p-4 text-center">
        <div class="text-xs text-ink-500">{{ t('dashboard.invite.invited') }}</div>
        <div class="mt-1 text-2xl font-semibold text-ink-900 ui-mono">{{ invited }}</div>
      </div>
      <div class="bg-ink-50 rounded-lg p-4 text-center">
        <div class="text-xs text-ink-500">{{ t('dashboard.invite.totalBonus') }}</div>
        <div class="mt-1 text-2xl font-semibold text-brand-600 ui-mono">${{ totalBonus.toFixed(4) }}</div>
      </div>
      <div class="bg-ink-50 rounded-lg p-4 text-center">
        <div class="text-xs text-ink-500">{{ t('dashboard.invite.monthBonus') }}</div>
        <div class="mt-1 text-2xl font-semibold text-brand-600 ui-mono">${{ monthBonus.toFixed(4) }}</div>
      </div>
    </div>

    <div class="flex items-center flex-wrap gap-3 mb-2">
      <span class="text-xs text-ink-500">{{ t('dashboard.invite.code') }}:</span>
      <code class="px-3 py-1 rounded bg-brand-50 text-brand-700 ui-mono text-sm tracking-wider">{{ code }}</code>
      <button
        class="inline-flex items-center gap-1 text-xs text-brand-600 hover:underline"
        @click="copy(code)"
      >
        <Icon :name="copied ? 'check' : 'copy'" class="w-3.5 h-3.5" />
        {{ copied ? t('dashboard.invite.copied') : t('dashboard.invite.copy') }}
      </button>
    </div>
    <p class="text-xs text-ink-400 leading-relaxed">{{ t('dashboard.invite.note') }}</p>
  </div>
</template>
