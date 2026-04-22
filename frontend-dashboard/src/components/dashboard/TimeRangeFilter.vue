<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/common/Icon.vue'

defineProps<{ range: string; granularity: string }>()
defineEmits<{
  (e: 'update:range', v: string): void
  (e: 'update:granularity', v: string): void
}>()

const { t } = useI18n()
</script>

<template>
  <div class="flex items-center justify-between flex-wrap gap-3 px-1">
    <label class="inline-flex items-center gap-2 text-xs text-ink-500">
      <span>{{ t('dashboard.filter.range') }}:</span>
      <div class="relative">
        <select
          :value="range"
          class="appearance-none rounded-md border border-ink-200 bg-white pl-3 pr-8 py-1.5 text-sm text-ink-800 hover:border-brand-300 focus:outline-none focus:border-brand-500"
          @change="(e) => $emit('update:range', (e.target as HTMLSelectElement).value)"
        >
          <option value="7">{{ t('dashboard.filter.last7') }}</option>
          <option value="30">{{ t('dashboard.filter.last30') }}</option>
          <option value="90">{{ t('dashboard.filter.last90') }}</option>
        </select>
        <Icon name="chevron-down" class="w-3 h-3 absolute right-2.5 top-1/2 -translate-y-1/2 text-ink-400 pointer-events-none" />
      </div>
    </label>

    <label class="inline-flex items-center gap-2 text-xs text-ink-500">
      <span>{{ t('dashboard.filter.granularity') }}:</span>
      <div class="relative">
        <select
          :value="granularity"
          class="appearance-none rounded-md border border-ink-200 bg-white pl-3 pr-8 py-1.5 text-sm text-ink-800 hover:border-brand-300 focus:outline-none focus:border-brand-500"
          @change="(e) => $emit('update:granularity', (e.target as HTMLSelectElement).value)"
        >
          <option value="day">{{ t('dashboard.filter.byDay') }}</option>
          <option value="hour">{{ t('dashboard.filter.byHour') }}</option>
        </select>
        <Icon name="chevron-down" class="w-3 h-3 absolute right-2.5 top-1/2 -translate-y-1/2 text-ink-400 pointer-events-none" />
      </div>
    </label>
  </div>
</template>
