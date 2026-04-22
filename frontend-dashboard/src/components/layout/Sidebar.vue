<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'

const { t } = useI18n()
const route = useRoute()

const items = computed(() => [
  { to: '/dashboard', label: t('nav.dashboard'), icon: '⊞' },
  { to: '/keys', label: t('nav.keys'), icon: '⚿' },
  { to: '/usage', label: t('nav.usage'), icon: '∿' },
  { to: '/groups', label: t('nav.groups'), icon: '◈' },
  { to: '/profile', label: t('nav.profile'), icon: '◉' }
])

function isActive(to: string) {
  return route.path === to || route.path.startsWith(to + '/')
}
</script>

<template>
  <aside class="w-60 shrink-0 bg-brand-gradient text-white flex flex-col">
    <div class="px-6 py-5 flex items-center gap-3 border-b border-white/10">
      <img src="/logo.svg" alt="logo" class="w-8 h-8 rounded-md" />
      <div class="brand-serif leading-tight">
        <div class="font-semibold text-lg">Dragon Code</div>
        <div class="text-[11px] text-white/60">{{ $t('site.tagline') }}</div>
      </div>
    </div>

    <nav class="flex-1 px-3 py-4 space-y-1">
      <RouterLink
        v-for="item in items"
        :key="item.to"
        :to="item.to"
        class="flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors"
        :class="
          isActive(item.to)
            ? 'bg-white/15 text-white shadow-inner'
            : 'text-white/70 hover:bg-white/10 hover:text-white'
        "
      >
        <span class="w-5 text-center text-base">{{ item.icon }}</span>
        <span class="brand-serif">{{ item.label }}</span>
      </RouterLink>
    </nav>

    <div class="px-6 py-4 text-[11px] text-white/50 border-t border-white/10 ui-sans">
      v0.1.0 · mock
    </div>
  </aside>
</template>
