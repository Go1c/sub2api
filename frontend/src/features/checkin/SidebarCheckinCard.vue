<template>
  <router-link v-if="store.enabled" to="/checkin" data-test="sidebar-checkin" class="hidden min-h-12 items-center border-y border-gray-100 bg-gray-50/80 text-gray-700 transition-colors hover:bg-primary-50 hover:text-primary-700 dark:border-dark-800 dark:bg-dark-900/60 dark:text-gray-300 dark:hover:bg-primary-950/30 dark:hover:text-primary-300 lg:flex" :class="collapsed ? 'mx-3 justify-center rounded-md border px-0' : 'px-4 py-3'" :title="collapsed ? t('nav.checkin') : undefined">
    <span class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-md bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300"><Icon :name="store.checkedInToday ? 'checkCircle' : 'calendar'" size="md" /></span>
    <span v-if="!collapsed" class="ml-3 min-w-0"><span class="block truncate text-sm font-semibold">{{ t('nav.checkin') }}</span><span class="mt-0.5 block truncate text-xs text-gray-500 dark:text-gray-400">{{ statusText }}</span></span>
  </router-link>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useCheckinStore } from './store'
defineProps<{ collapsed: boolean }>()
const { t } = useI18n(); const store = useCheckinStore()
const statusText = computed(() => store.status?.today_record?.status === 'budget_exhausted' ? t('checkin.sidebar.exhausted') : store.checkedInToday ? t('checkin.sidebar.checked') : t('checkin.sidebar.ready'))
</script>
