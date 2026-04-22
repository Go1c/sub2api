<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Icon, { type IconName } from '@/components/common/Icon.vue'

const { t } = useI18n()
const router = useRouter()

interface Action {
  key: string
  icon: IconName
  tone: string
  to?: string
  href?: string
}

const actions: Action[] = [
  { key: 'newKey', icon: 'key', tone: 'bg-brand-50 text-brand-600', to: '/keys' },
  { key: 'viewUsage', icon: 'chart', tone: 'bg-emerald-50 text-emerald-600', to: '/usage' },
  { key: 'recharge', icon: 'bolt', tone: 'bg-orange-50 text-orange-600', to: '/recharge' },
  { key: 'docs', icon: 'doc', tone: 'bg-violet-50 text-violet-600', href: 'https://github.com/Wei-Shaw/sub2api' }
]

function go(a: Action) {
  if (a.to) router.push(a.to)
  else if (a.href) window.open(a.href, '_blank', 'noopener,noreferrer')
}
</script>

<template>
  <div class="card h-full">
    <h3 class="text-sm font-semibold text-ink-900 mb-4">{{ t('dashboard.quick.title') }}</h3>
    <div class="space-y-2">
      <button
        v-for="a in actions"
        :key="a.key"
        class="w-full flex items-center gap-3 p-3 rounded-lg border border-ink-100 hover:border-brand-300 hover:bg-brand-50/40 transition-colors text-left"
        @click="go(a)"
      >
        <div class="icon-box" :class="a.tone">
          <Icon :name="a.icon" class="w-5 h-5" />
        </div>
        <div class="flex-1 min-w-0">
          <div class="text-sm font-medium text-ink-900">{{ t(`dashboard.quick.${a.key}`) }}</div>
          <div class="text-xs text-ink-500 truncate">{{ t(`dashboard.quick.${a.key}Hint`) }}</div>
        </div>
      </button>
    </div>
  </div>
</template>
