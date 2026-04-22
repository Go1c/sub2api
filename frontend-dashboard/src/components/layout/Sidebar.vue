<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { computed, ref } from 'vue'
import Icon, { type IconName } from '@/components/common/Icon.vue'

const { t } = useI18n()
const route = useRoute()

const collapsed = ref(false)
const darkMode = ref(false)

interface Item {
  to: string
  label: string
  icon: IconName
}

const items = computed<Item[]>(() => [
  { to: '/dashboard', label: t('nav.dashboard'), icon: 'dashboard' },
  { to: '/keys', label: t('nav.keys'), icon: 'key' },
  { to: '/usage', label: t('nav.usage'), icon: 'chart' },
  { to: '/recharge', label: t('nav.recharge'), icon: 'bolt' },
  { to: '/redeem', label: t('nav.redeem'), icon: 'gift' },
  { to: '/docs', label: t('nav.docs'), icon: 'doc' },
  { to: '/profile', label: t('nav.profile'), icon: 'user' }
])

function isActive(to: string) {
  return route.path === to || route.path.startsWith(to + '/')
}

function toggleTheme() {
  darkMode.value = !darkMode.value
  document.documentElement.classList.toggle('dark', darkMode.value)
}
</script>

<template>
  <aside
    class="bg-white border-r border-ink-100 flex flex-col transition-[width] duration-200"
    :class="collapsed ? 'w-16' : 'w-56'"
  >
    <div class="h-14 flex items-center gap-2 px-4 border-b border-ink-100">
      <div class="w-7 h-7 rounded-md bg-gradient-to-br from-brand-500 to-brand-900 flex items-center justify-center text-white text-sm font-semibold shrink-0">
        D
      </div>
      <span v-if="!collapsed" class="text-sm font-semibold text-ink-900">Dragon Code</span>
    </div>

    <nav class="flex-1 px-2 py-3 space-y-0.5">
      <RouterLink
        v-for="item in items"
        :key="item.to"
        :to="item.to"
        class="flex items-center gap-3 rounded-md text-sm transition-colors"
        :class="[
          collapsed ? 'justify-center px-0 py-2.5' : 'px-3 py-2',
          isActive(item.to)
            ? 'bg-brand-50 text-brand-600 font-medium'
            : 'text-ink-600 hover:bg-ink-50 hover:text-ink-900'
        ]"
      >
        <Icon :name="item.icon" class="w-4 h-4 shrink-0" />
        <span v-if="!collapsed">{{ item.label }}</span>
      </RouterLink>
    </nav>

    <div class="border-t border-ink-100 p-2 space-y-0.5">
      <button
        class="w-full flex items-center gap-3 rounded-md px-3 py-2 text-sm text-ink-600 hover:bg-ink-50"
        :class="collapsed ? 'justify-center px-0' : ''"
        @click="toggleTheme"
      >
        <Icon :name="darkMode ? 'sun' : 'moon'" class="w-4 h-4 shrink-0" />
        <span v-if="!collapsed">{{ darkMode ? t('nav.lightMode') : t('nav.darkMode') }}</span>
      </button>
      <button
        class="w-full flex items-center gap-3 rounded-md px-3 py-2 text-sm text-ink-600 hover:bg-ink-50"
        :class="collapsed ? 'justify-center px-0' : ''"
        @click="collapsed = !collapsed"
      >
        <Icon name="chevron-left" :class="['w-4 h-4 shrink-0 transition-transform', collapsed ? 'rotate-180' : '']" />
        <span v-if="!collapsed">{{ t('nav.collapse') }}</span>
      </button>
    </div>
  </aside>
</template>
