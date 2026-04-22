<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { http } from '@/api/http'

const { t } = useI18n()

interface KeyRow {
  id: string
  name: string
  prefix: string
  created: string
  status: string
}

const rows = ref<KeyRow[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  const res = await http.get('/user/keys')
  rows.value = res.data.data
  loading.value = false
}
onMounted(load)
</script>

<template>
  <div class="space-y-5 animate-fade-in">
    <div class="flex items-center justify-between">
      <h1 class="brand-serif text-2xl font-semibold text-ink-900">{{ t('keys.title') }}</h1>
      <button class="btn-brand">{{ t('keys.create') }}</button>
    </div>

    <div class="card !p-0 overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-ink-50 text-xs text-ink-500 ui-sans">
          <tr>
            <th class="text-left px-4 py-3 font-normal">{{ t('keys.name') }}</th>
            <th class="text-left px-4 py-3 font-normal">{{ t('keys.prefix') }}</th>
            <th class="text-left px-4 py-3 font-normal">{{ t('keys.created') }}</th>
            <th class="text-left px-4 py-3 font-normal">{{ t('keys.status') }}</th>
            <th class="text-right px-4 py-3 font-normal">{{ t('keys.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading" class="border-t border-ink-100">
            <td colspan="5" class="px-4 py-6 text-center text-ink-400 ui-sans">
              {{ t('common.loading') }}
            </td>
          </tr>
          <tr v-else-if="rows.length === 0" class="border-t border-ink-100">
            <td colspan="5" class="px-4 py-6 text-center text-ink-400 ui-sans">
              {{ t('common.empty') }}
            </td>
          </tr>
          <tr v-for="r in rows" :key="r.id" class="border-t border-ink-100 hover:bg-ink-50">
            <td class="px-4 py-3 brand-serif text-ink-900">{{ r.name }}</td>
            <td class="px-4 py-3 ui-mono text-xs text-ink-600">{{ r.prefix }}…</td>
            <td class="px-4 py-3 ui-sans text-ink-600">{{ r.created }}</td>
            <td class="px-4 py-3">
              <span
                class="inline-block px-2 py-0.5 rounded-full text-[11px] ui-sans"
                :class="
                  r.status === 'active'
                    ? 'bg-emerald-50 text-emerald-600'
                    : 'bg-ink-100 text-ink-500'
                "
              >
                {{ r.status }}
              </span>
            </td>
            <td class="px-4 py-3 text-right">
              <button class="text-xs text-brand-600 hover:underline ui-sans">view</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
