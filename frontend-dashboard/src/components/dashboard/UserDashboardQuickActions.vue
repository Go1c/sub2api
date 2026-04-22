<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

const emit = defineEmits<{ (e: 'recharge'): void }>()
const { t } = useI18n()
const router = useRouter()

const actions = [
  { key: 'newKey', icon: '⚿', to: '/keys' },
  { key: 'recharge', icon: '✦', action: 'recharge' as const },
  { key: 'docs', icon: '❏', href: 'https://github.com/Wei-Shaw/sub2api' },
  { key: 'usage', icon: '∿', to: '/usage' }
]

function onClick(a: (typeof actions)[number]) {
  if (a.action === 'recharge') emit('recharge')
  else if (a.to) router.push(a.to)
  else if (a.href) window.open(a.href, '_blank', 'noopener,noreferrer')
}
</script>

<template>
  <div class="card">
    <h3 class="brand-serif text-base font-semibold text-ink-900 mb-4">
      {{ t('dashboard.quick.title') }}
    </h3>
    <div class="grid grid-cols-2 gap-3">
      <button
        v-for="a in actions"
        :key="a.key"
        class="group flex items-center gap-3 rounded-md border border-ink-200 px-3 py-3 hover:border-brand-500 hover:bg-brand-50 transition-colors text-left"
        @click="onClick(a)"
      >
        <span class="w-9 h-9 rounded-md bg-brand-100 text-brand-700 flex items-center justify-center">{{ a.icon }}</span>
        <span class="brand-serif text-sm text-ink-800 group-hover:text-brand-700">
          {{ t(`dashboard.quick.${a.key}`) }}
        </span>
      </button>
    </div>
  </div>
</template>
