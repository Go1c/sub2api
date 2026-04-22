<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

const { t, locale } = useI18n()
const auth = useAuthStore()
const router = useRouter()

function toggleLocale() {
  const next = locale.value === 'zh-CN' ? 'en-US' : 'zh-CN'
  locale.value = next
  localStorage.setItem('locale', next)
}

function logout() {
  auth.logout()
  router.replace('/login')
}
</script>

<template>
  <header class="h-14 bg-white border-b border-ink-200 flex items-center justify-end px-6 gap-4">
    <button
      class="text-xs text-ink-500 hover:text-brand-600 ui-sans"
      @click="toggleLocale"
    >
      {{ locale === 'zh-CN' ? 'EN' : '中' }}
    </button>
    <div class="flex items-center gap-2">
      <div class="w-8 h-8 rounded-full bg-brand-100 text-brand-700 flex items-center justify-center brand-serif font-semibold">
        {{ auth.user?.name?.[0] ?? 'D' }}
      </div>
      <div class="brand-serif text-sm leading-tight">
        <div class="font-medium">{{ auth.user?.name ?? 'Guest' }}</div>
        <div class="text-[11px] text-ink-500">{{ auth.user?.plan ?? '—' }}</div>
      </div>
    </div>
    <button
      class="text-xs text-ink-500 hover:text-rose-500 ui-sans"
      @click="logout"
    >
      {{ t('nav.logout') }}
    </button>
  </header>
</template>
