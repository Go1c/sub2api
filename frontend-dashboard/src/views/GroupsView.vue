<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { http } from '@/api/http'

const { t } = useI18n()

interface Group {
  id: string
  name: string
  capacity: number
  members: number
  usage: string
}

const rows = ref<Group[]>([])

async function load() {
  const res = await http.get('/user/groups')
  rows.value = res.data.data
}
onMounted(load)
</script>

<template>
  <div class="space-y-5 animate-fade-in">
    <h1 class="brand-serif text-2xl font-semibold text-ink-900">{{ t('groups.title') }}</h1>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div v-for="g in rows" :key="g.id" class="card">
        <div class="flex items-center justify-between">
          <h3 class="brand-serif text-lg font-semibold text-ink-900">{{ g.name }}</h3>
          <span class="text-xs text-brand-600 ui-sans">{{ g.usage }}</span>
        </div>
        <div class="mt-4 grid grid-cols-2 gap-4 text-sm">
          <div>
            <div class="text-[11px] text-ink-500 ui-sans">{{ t('groups.capacity') }}</div>
            <div class="brand-serif text-ink-900">{{ g.capacity }}</div>
          </div>
          <div>
            <div class="text-[11px] text-ink-500 ui-sans">{{ t('groups.members') }}</div>
            <div class="brand-serif text-ink-900">{{ g.members }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
