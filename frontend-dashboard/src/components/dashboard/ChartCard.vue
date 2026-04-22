<script setup lang="ts">
defineProps<{ title: string; empty?: boolean; emptyText?: string; emptyFields?: string[] }>()
</script>

<template>
  <div class="card h-72 flex flex-col">
    <h3 class="text-sm font-semibold text-ink-900 mb-3">{{ title }}</h3>
    <div class="flex-1 relative">
      <div v-if="empty" class="absolute inset-0 flex flex-col items-center justify-center gap-3 text-ink-300">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="w-10 h-10">
          <path d="M20 6L9 17l-5-5" stroke-opacity="0" />
          <rect x="3" y="3" width="18" height="18" rx="3" />
          <path d="M3 16l5-5 4 4 8-8" />
        </svg>
        <div class="text-xs text-ink-400">{{ emptyText }}</div>
        <div v-if="emptyFields" class="flex gap-3 text-[11px] text-ink-400">
          <span v-for="f in emptyFields" :key="f">{{ f }}</span>
        </div>
      </div>
      <slot v-else />
    </div>
  </div>
</template>
