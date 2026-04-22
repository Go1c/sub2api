<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useRouter, useRoute } from 'vue-router'
import { computed } from 'vue'
import Icon from '@/components/common/Icon.vue'

const { t, locale } = useI18n()
const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const title = computed(() => {
  const map: Record<string, { title: string; subtitle: string }> = {
    '/dashboard': { title: t('dashboard.title'), subtitle: t('dashboard.subtitle') },
    '/keys': { title: t('keys.title'), subtitle: '' },
    '/usage': { title: t('usage.title'), subtitle: t('usage.subtitle') },
    '/groups': { title: t('groups.title'), subtitle: '' },
    '/profile': { title: t('profile.title'), subtitle: '' }
  }
  return map[route.path] || { title: '', subtitle: '' }
})

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
  <header class="h-14 bg-white border-b border-ink-100 flex items-center px-6 gap-4">
    <div class="flex-1 min-w-0">
      <h1 class="text-lg font-semibold text-ink-900 leading-tight truncate">{{ title.title }}</h1>
      <p v-if="title.subtitle" class="text-xs text-ink-500 leading-tight truncate">{{ title.subtitle }}</p>
    </div>

    <button class="w-8 h-8 rounded-full hover:bg-ink-50 text-ink-500 flex items-center justify-center" :title="'Announcements'">
      <Icon name="announce" class="w-4 h-4" />
    </button>
    <button class="w-8 h-8 rounded-full hover:bg-ink-50 text-ink-500 flex items-center justify-center" :title="'Notifications'">
      <Icon name="bell" class="w-4 h-4" />
    </button>

    <button class="flex items-center gap-1 text-xs text-ink-600 hover:text-brand-600 px-2 py-1 rounded-md hover:bg-ink-50" @click="toggleLocale">
      <Icon name="globe" class="w-3.5 h-3.5" />
      <span>{{ locale === 'zh-CN' ? 'CN ZH' : 'EN' }}</span>
    </button>

    <div class="chip bg-emerald-50 text-emerald-700 h-7 gap-1 px-3">
      <Icon name="dollar" class="w-3 h-3" />
      <span class="ui-mono">{{ (auth.user?.balance ?? 0).toFixed(2) }}</span>
    </div>

    <button class="flex items-center gap-2" @click="logout">
      <div class="w-8 h-8 rounded-full bg-gradient-to-br from-brand-500 to-brand-900 text-white text-xs flex items-center justify-center font-semibold">
        {{ auth.user?.name?.[0]?.toUpperCase() || 'U' }}
      </div>
      <div class="text-left leading-tight hidden sm:block">
        <div class="text-xs font-medium text-ink-900 ui-mono">{{ auth.user?.id || '—' }}</div>
        <div class="text-[11px] text-ink-500">{{ auth.user?.plan || 'User' }}</div>
      </div>
    </button>
  </header>
</template>
