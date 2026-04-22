<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'

const { t, tm } = useI18n()

interface Section {
  title: string
  links: string[]
}

const sections = computed<Section[]>(() => (tm('marketing.footer.sections') as any[]) || [])
</script>

<template>
  <footer class="bg-ink-900 text-ink-300 pt-16 pb-10">
    <div class="max-w-6xl mx-auto px-6">
      <div class="grid grid-cols-2 md:grid-cols-6 gap-8">
        <div class="col-span-2 md:col-span-1">
          <div class="flex items-center gap-2">
            <div class="w-8 h-8 rounded-md bg-gradient-to-br from-brand-500 to-brand-900 flex items-center justify-center text-white text-sm font-semibold">
              D
            </div>
            <span class="text-base font-semibold text-white">DragonCode</span>
          </div>
          <p class="mt-4 text-xs text-ink-400 leading-relaxed">{{ t('site.tagline') }}</p>
        </div>

        <div v-for="(s, i) in sections" :key="i">
          <h4 class="text-xs font-semibold text-white mb-4 tracking-wide">{{ s.title }}</h4>
          <ul class="space-y-2.5">
            <li v-for="(l, li) in s.links" :key="li">
              <a href="#" class="text-sm text-ink-400 hover:text-white transition-colors">{{ l }}</a>
            </li>
          </ul>
        </div>
      </div>

      <div class="mt-12 pt-6 border-t border-ink-800 text-xs text-ink-500 text-center">
        {{ t('marketing.footer.copyright', { year: new Date().getFullYear() }) }}
      </div>
    </div>
  </footer>
</template>
