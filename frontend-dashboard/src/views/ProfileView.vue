<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const auth = useAuthStore()
</script>

<template>
  <div class="space-y-5 animate-fade-in">
    <h1 class="brand-serif text-2xl font-semibold text-ink-900">{{ t('profile.title') }}</h1>

    <div class="card max-w-xl">
      <dl class="divide-y divide-ink-100">
        <div class="py-3 flex justify-between">
          <dt class="text-xs text-ink-500 ui-sans">{{ t('profile.email') }}</dt>
          <dd class="brand-serif text-ink-900">{{ auth.user?.email ?? '—' }}</dd>
        </div>
        <div class="py-3 flex justify-between">
          <dt class="text-xs text-ink-500 ui-sans">{{ t('profile.plan') }}</dt>
          <dd class="brand-serif text-ink-900">{{ auth.user?.plan ?? '—' }}</dd>
        </div>
        <div class="py-3 flex justify-between">
          <dt class="text-xs text-ink-500 ui-sans">{{ t('profile.joinedAt') }}</dt>
          <dd class="brand-serif text-ink-900">{{ auth.user?.joinedAt ?? '—' }}</dd>
        </div>
      </dl>
    </div>
  </div>
</template>
