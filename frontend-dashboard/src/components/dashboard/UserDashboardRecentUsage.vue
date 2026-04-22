<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  items: Array<{
    id: string
    time: string
    model: string
    tokens: number
    status: 'success' | 'error' | string
  }>
}>()

const { t } = useI18n()
</script>

<template>
  <div class="card">
    <h3 class="brand-serif text-base font-semibold text-ink-900 mb-4">
      {{ t('dashboard.recent.title') }}
    </h3>
    <table class="w-full text-sm">
      <thead class="text-xs text-ink-500 ui-sans">
        <tr class="border-b border-ink-200">
          <th class="text-left pb-2 font-normal">{{ t('dashboard.recent.time') }}</th>
          <th class="text-left pb-2 font-normal">{{ t('dashboard.recent.model') }}</th>
          <th class="text-right pb-2 font-normal">{{ t('dashboard.recent.tokens') }}</th>
          <th class="text-right pb-2 font-normal">{{ t('dashboard.recent.status') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in items" :key="item.id" class="border-b border-ink-100 last:border-0">
          <td class="py-2.5 ui-mono text-xs text-ink-600">{{ item.time }}</td>
          <td class="py-2.5 brand-serif">{{ item.model }}</td>
          <td class="py-2.5 text-right ui-mono text-xs">{{ item.tokens.toLocaleString() }}</td>
          <td class="py-2.5 text-right">
            <span
              class="inline-block px-2 py-0.5 rounded-full text-[11px] ui-sans"
              :class="
                item.status === 'success'
                  ? 'bg-emerald-50 text-emerald-600'
                  : 'bg-rose-50 text-rose-500'
              "
            >
              {{ item.status }}
            </span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
