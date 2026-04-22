<script setup lang="ts">
import Icon, { type IconName } from '@/components/common/Icon.vue'

defineProps<{
  icon: IconName
  tone: 'green' | 'purple' | 'blue' | 'orange' | 'pink' | 'gray' | 'amber'
  label: string
  value: string
  hint?: string
  action?: string
}>()

defineEmits<{ (e: 'action'): void }>()

const toneMap: Record<string, string> = {
  green: 'bg-emerald-50 text-emerald-600',
  purple: 'bg-violet-50 text-violet-600',
  blue: 'bg-brand-50 text-brand-600',
  orange: 'bg-orange-50 text-orange-600',
  pink: 'bg-pink-50 text-pink-600',
  gray: 'bg-ink-100 text-ink-600',
  amber: 'bg-amber-50 text-amber-600'
}
</script>

<template>
  <div class="card card-hover flex items-start gap-3">
    <div class="icon-box" :class="toneMap[tone]">
      <Icon :name="icon" class="w-5 h-5" />
    </div>
    <div class="flex-1 min-w-0">
      <div class="text-xs text-ink-500 flex items-center gap-2">
        <span>{{ label }}</span>
        <button
          v-if="action"
          class="text-brand-600 hover:underline text-xs"
          @click="$emit('action')"
        >
          {{ action }}
        </button>
      </div>
      <div class="mt-1 text-xl font-semibold text-ink-900 ui-mono truncate">{{ value }}</div>
      <div v-if="hint" class="text-xs text-ink-400 mt-0.5 truncate">{{ hint }}</div>
    </div>
  </div>
</template>
