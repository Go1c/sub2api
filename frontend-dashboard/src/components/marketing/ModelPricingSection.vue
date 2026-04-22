<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { fetchPublicPricing, type PublicPricing } from '@/api/pricing'

const { t } = useI18n()
const router = useRouter()
const auth = useAuthStore()

const data = ref<PublicPricing | null>(null)
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    data.value = await fetchPublicPricing()
  } catch (e: any) {
    error.value = e?.message || t('marketing.pricing.error')
  } finally {
    loading.value = false
  }
}

onMounted(load)

function ctaClick() {
  router.push(auth.isAuthed ? '/dashboard' : '/register')
}

function fmtYuan(v: number) {
  return `¥${v.toFixed(2).replace(/\.00$/, '.00')}`
}

function fmtOfficial(input: number, output: number) {
  return `¥${input}/¥${output}`
}
</script>

<template>
  <section id="pricing" class="bg-white py-24">
    <div class="max-w-6xl mx-auto px-6">
      <div class="text-center">
        <h2 class="text-4xl md:text-5xl font-semibold text-ink-900 inline-flex items-center gap-3">
          {{ t('marketing.pricing.title') }}
          <span class="text-brand-500 text-3xl md:text-4xl">✦</span>
        </h2>
        <p class="mt-4 text-sm md:text-base text-ink-500 max-w-3xl mx-auto">
          {{ data?.rateNote || t('marketing.pricing.note') }}
        </p>
      </div>

      <div class="mt-12 rounded-2xl bg-white border border-ink-100 shadow-card overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full text-sm min-w-[720px]">
            <thead class="bg-ink-50 text-xs text-ink-500 font-medium">
              <tr>
                <th class="text-left px-5 py-4">{{ t('marketing.pricing.cols.model') }}</th>
                <th class="text-left px-5 py-4">{{ t('marketing.pricing.cols.group') }}</th>
                <th class="text-left px-5 py-4">{{ t('marketing.pricing.cols.multiplier') }}</th>
                <th class="text-left px-5 py-4">{{ t('marketing.pricing.cols.input') }}</th>
                <th class="text-left px-5 py-4">{{ t('marketing.pricing.cols.output') }}</th>
                <th class="text-left px-5 py-4">{{ t('marketing.pricing.cols.official') }}</th>
                <th class="text-left px-5 py-4">{{ t('marketing.pricing.cols.discount') }}</th>
                <th class="text-left px-5 py-4">{{ t('marketing.pricing.cols.openClaw') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loading">
                <td colspan="8" class="px-5 py-12 text-center text-ink-400">
                  {{ t('marketing.pricing.loading') }}
                </td>
              </tr>
              <tr v-else-if="error">
                <td colspan="8" class="px-5 py-12 text-center text-rose-500">
                  {{ error }}
                </td>
              </tr>
              <tr
                v-for="(r, i) in data?.rows || []"
                :key="`${r.model}-${r.group}`"
                class="border-t border-ink-100 transition-colors"
                :class="i % 2 === 0 ? 'bg-white' : 'bg-ink-50/40'"
              >
                <td class="px-5 py-4 font-medium text-ink-900">{{ r.model }}</td>
                <td class="px-5 py-4 text-ink-600">{{ r.group }}</td>
                <td class="px-5 py-4 ui-mono text-ink-700">{{ r.multiplier }}</td>
                <td class="px-5 py-4 ui-mono text-ink-900">{{ fmtYuan(r.inputPrice) }}</td>
                <td class="px-5 py-4 ui-mono text-ink-900">{{ fmtYuan(r.outputPrice) }}</td>
                <td class="px-5 py-4 ui-mono text-ink-500">{{ fmtOfficial(r.officialInput, r.officialOutput) }}</td>
                <td class="px-5 py-4">
                  <span class="chip bg-brand-100 text-brand-700 ui-mono">{{ r.discount }}</span>
                </td>
                <td class="px-5 py-4 text-ink-600">
                  {{ r.openClaw ? t('marketing.pricing.yes') : t('marketing.pricing.no') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="mt-10 text-center">
        <button
          class="inline-flex items-center gap-2 rounded-full bg-gradient-to-r from-brand-500 to-brand-900 px-8 py-3 text-base font-medium text-white shadow-[0_8px_30px_rgba(79,140,255,0.35)] hover:shadow-[0_12px_40px_rgba(79,140,255,0.45)] transition-shadow"
          @click="ctaClick"
        >
          {{ t('marketing.pricing.ctaRegister') }} {{ t('marketing.pricing.ctaExperience') }}
        </button>
      </div>
    </div>
  </section>
</template>
