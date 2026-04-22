<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  items: Array<{ id: string; time: string; model: string; tokens: number; status: string }>
  range: string
}>()

const { t } = useI18n()
</script>

<template>
  <div class="card h-full">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-sm font-semibold text-ink-900">{{ t('dashboard.recent.title') }}</h3>
      <span class="text-xs text-ink-400">{{ range }}</span>
    </div>

    <div v-if="items.length === 0" class="min-h-[160px] flex flex-col items-center justify-center text-ink-300">
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="w-10 h-10">
        <rect x="3" y="4" width="18" height="16" rx="2" />
        <path d="M3 9h18" />
      </svg>
      <div class="mt-2 text-xs text-ink-400">{{ t('dashboard.recent.empty') }}</div>
    </div>

    <table v-else class="w-full text-sm">
      <thead>
        <tr class="text-xs text-ink-500">
          <th class="text-left pb-2 font-normal">{{ $t('dashboard.recent.title') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="i in items" :key="i.id" class="border-t border-ink-100">
          <td class="py-2 ui-mono text-xs">{{ i.time }}</td>
          <td class="py-2">{{ i.model }}</td>
          <td class="py-2 text-right ui-mono text-xs">{{ i.tokens }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
