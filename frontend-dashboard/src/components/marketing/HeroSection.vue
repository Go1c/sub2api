<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const router = useRouter()
const auth = useAuthStore()

function primary() {
  router.push(auth.isAuthed ? '/dashboard' : '/login')
}
function secondary() {
  const el = document.querySelector('#pricing')
  el?.scrollIntoView({ behavior: 'smooth' })
}
</script>

<template>
  <section class="relative overflow-hidden">
    <!-- soft background gradient -->
    <div class="absolute inset-0 bg-gradient-to-b from-brand-50/50 via-white to-white pointer-events-none"></div>
    <div class="absolute -top-40 -right-40 w-96 h-96 bg-brand-200/40 rounded-full blur-3xl pointer-events-none"></div>
    <div class="absolute -bottom-32 -left-32 w-80 h-80 bg-brand-100/50 rounded-full blur-3xl pointer-events-none"></div>

    <div class="relative max-w-6xl mx-auto px-6 pt-24 pb-28 text-center">
      <div class="inline-flex items-center gap-1.5 rounded-full bg-brand-50 border border-brand-100 px-3 py-1 text-xs font-medium text-brand-700">
        <span class="w-1.5 h-1.5 rounded-full bg-brand-500 animate-pulse"></span>
        {{ t('marketing.hero.eyebrow') }}
      </div>

      <h1 class="mt-6 text-5xl md:text-6xl font-semibold text-ink-900 leading-tight tracking-tight">
        {{ t('marketing.hero.title') }}
      </h1>

      <p class="mt-6 max-w-2xl mx-auto text-base md:text-lg text-ink-600 leading-relaxed">
        {{ t('marketing.hero.subtitle') }}
      </p>

      <div class="mt-10 flex items-center justify-center gap-3">
        <button class="btn-brand px-8 py-3 text-base" @click="primary">
          {{ t('marketing.hero.primary') }}
        </button>
        <button class="btn-ghost px-8 py-3 text-base" @click="secondary">
          {{ t('marketing.hero.secondary') }}
        </button>
      </div>
    </div>
  </section>
</template>
