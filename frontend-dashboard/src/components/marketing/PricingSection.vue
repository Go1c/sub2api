<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

const { t, tm } = useI18n()
const router = useRouter()
const auth = useAuthStore()

interface Plan {
  name: string
  price: string
  priceNote: string
  features: string[]
  cta: string
  highlight?: boolean
}

const plans = computed<Plan[]>(() => (tm('marketing.pricing.plans') as any[]) || [])

function start() {
  router.push(auth.isAuthed ? '/dashboard' : '/login')
}
</script>

<template>
  <section id="pricing" class="bg-ink-50 py-24">
    <div class="max-w-6xl mx-auto px-6">
      <div class="text-center max-w-2xl mx-auto">
        <h2 class="text-3xl md:text-4xl font-semibold text-ink-900">
          {{ t('marketing.pricing.title') }}
        </h2>
        <p class="mt-4 text-sm md:text-base text-ink-500">
          {{ t('marketing.pricing.subtitle') }}
        </p>
      </div>

      <div class="mt-12 grid grid-cols-1 md:grid-cols-3 gap-5">
        <div
          v-for="(p, i) in plans"
          :key="i"
          class="rounded-2xl border p-8 flex flex-col"
          :class="
            p.highlight
              ? 'bg-gradient-to-br from-brand-500 to-brand-900 border-brand-500 text-white shadow-[0_20px_60px_rgba(79,140,255,0.3)]'
              : 'bg-white border-ink-100'
          "
        >
          <h3
            class="text-base font-semibold"
            :class="p.highlight ? 'text-white' : 'text-ink-900'"
          >
            {{ p.name }}
          </h3>
          <div class="mt-4">
            <div class="text-3xl font-semibold tracking-tight" :class="p.highlight ? 'text-white' : 'text-ink-900'">
              {{ p.price }}
            </div>
            <div class="text-xs mt-1" :class="p.highlight ? 'text-white/70' : 'text-ink-500'">
              {{ p.priceNote }}
            </div>
          </div>

          <ul class="mt-6 space-y-2.5 flex-1">
            <li
              v-for="(f, fi) in p.features"
              :key="fi"
              class="flex items-start gap-2 text-sm"
              :class="p.highlight ? 'text-white/90' : 'text-ink-700'"
            >
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="w-4 h-4 mt-0.5 shrink-0" :class="p.highlight ? 'text-brand-200' : 'text-brand-500'">
                <path d="M5 12l5 5L20 7" />
              </svg>
              <span>{{ f }}</span>
            </li>
          </ul>

          <button
            class="mt-8 rounded-full px-5 py-2.5 text-sm font-medium transition-colors"
            :class="
              p.highlight
                ? 'bg-white text-brand-700 hover:bg-ink-50'
                : 'border border-ink-200 text-ink-800 hover:border-brand-500 hover:text-brand-600'
            "
            @click="start"
          >
            {{ p.cta }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
