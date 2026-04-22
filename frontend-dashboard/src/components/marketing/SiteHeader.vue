<script setup lang="ts">
import { RouterLink, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { computed } from 'vue'

const { t } = useI18n()
const auth = useAuthStore()
const router = useRouter()

const navItems = computed(() => [
  { key: 'home', to: '/', label: t('marketing.header.home') },
  { key: 'pricing', to: '#pricing', label: t('marketing.header.pricing') },
  { key: 'status', to: '#status', label: t('marketing.header.status') },
  { key: 'docs', href: 'https://github.com/Wei-Shaw/sub2api', label: t('marketing.header.docs') }
])

function gotoCta() {
  router.push(auth.isAuthed ? '/dashboard' : '/login')
}

function onNav(item: { to?: string; href?: string }) {
  if (item.href) {
    window.open(item.href, '_blank', 'noopener,noreferrer')
    return
  }
  if (!item.to) return
  if (item.to.startsWith('#')) {
    const el = document.querySelector(item.to)
    el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  } else {
    router.push(item.to)
  }
}
</script>

<template>
  <header class="sticky top-0 z-40 bg-white/85 backdrop-blur-md border-b border-ink-100">
    <div class="max-w-6xl mx-auto h-16 px-6 flex items-center">
      <RouterLink to="/" class="flex items-center gap-2">
        <div class="w-8 h-8 rounded-md bg-gradient-to-br from-brand-500 to-brand-900 flex items-center justify-center text-white text-sm font-semibold">
          D
        </div>
        <span class="text-base font-semibold text-ink-900">DragonCode</span>
      </RouterLink>

      <nav class="hidden md:flex items-center gap-8 ml-10">
        <button
          v-for="item in navItems"
          :key="item.key"
          class="text-sm text-ink-700 hover:text-brand-600 transition-colors"
          @click="onNav(item)"
        >
          {{ item.label }}
        </button>
      </nav>

      <div class="flex-1" />

      <button
        class="rounded-full border border-ink-300 px-5 py-1.5 text-sm font-medium text-ink-800 hover:border-brand-500 hover:text-brand-600 transition-colors"
        @click="gotoCta"
      >
        {{ auth.isAuthed ? t('marketing.header.ctaAuthed') : t('marketing.header.cta') }}
      </button>
    </div>
  </header>
</template>
