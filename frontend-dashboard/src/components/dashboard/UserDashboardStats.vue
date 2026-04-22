<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  balance: number
  todayRequests: number
  todayTokens: number
  activeKeys: number
}>()

const { t } = useI18n()

function formatNumber(n: number) {
  return n.toLocaleString()
}
</script>

<template>
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
    <div class="card">
      <div class="text-xs text-ink-500 ui-sans">{{ t('dashboard.stats.balance') }}</div>
      <div class="mt-2 brand-serif text-2xl font-semibold text-ink-900">¥ {{ balance.toFixed(2) }}</div>
    </div>
    <div class="card">
      <div class="text-xs text-ink-500 ui-sans">{{ t('dashboard.stats.requests') }}</div>
      <div class="mt-2 brand-serif text-2xl font-semibold text-ink-900">{{ formatNumber(todayRequests) }}</div>
    </div>
    <div class="card">
      <div class="text-xs text-ink-500 ui-sans">{{ t('dashboard.stats.tokens') }}</div>
      <div class="mt-2 brand-serif text-2xl font-semibold text-ink-900">{{ formatNumber(todayTokens) }}</div>
    </div>
    <div class="card">
      <div class="text-xs text-ink-500 ui-sans">{{ t('dashboard.stats.keys') }}</div>
      <div class="mt-2 brand-serif text-2xl font-semibold text-ink-900">{{ activeKeys }}</div>
    </div>
  </div>
</template>
