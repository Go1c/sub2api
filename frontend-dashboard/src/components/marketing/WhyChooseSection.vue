<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'

const { tm } = useI18n()

const items = computed(() => (tm('marketing.why.items') as any[]) || [])
</script>

<template>
  <section class="bg-white py-24">
    <div class="max-w-6xl mx-auto px-6">
      <h2 class="text-3xl md:text-4xl font-semibold text-ink-900 leading-tight">
        {{ $t('marketing.why.title') }}
        <span class="text-brand-500">{{ $t('marketing.why.titleBrand') }}</span>
        <span class="text-brand-500 ml-1">✦</span>
      </h2>
      <p class="mt-4 text-sm md:text-base text-ink-500 max-w-2xl">
        {{ $t('marketing.why.subtitle') }}
      </p>

      <div class="mt-12 grid grid-cols-1 md:grid-cols-2 gap-5">
        <div
          v-for="(item, i) in items"
          :key="i"
          class="rounded-2xl bg-ink-50 border border-ink-100 p-8 hover:bg-white hover:shadow-card-hover transition-all"
        >
          <div class="text-5xl md:text-6xl font-semibold text-brand-500 leading-none tabular-nums">
            {{ String(i + 1).padStart(2, '0') }}
          </div>
          <h3 class="mt-8 text-lg font-semibold text-ink-900">{{ item.title }}</h3>
          <p class="mt-3 text-sm text-ink-500 leading-relaxed">{{ item.desc }}</p>
        </div>
      </div>
    </div>
  </section>
</template>
