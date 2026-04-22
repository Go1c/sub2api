<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  Title,
  Tooltip,
  Legend,
  PointElement,
  LineElement,
  CategoryScale,
  LinearScale,
  Filler
} from 'chart.js'
import { http } from '@/api/http'

ChartJS.register(Title, Tooltip, Legend, PointElement, LineElement, CategoryScale, LinearScale, Filler)

const { t } = useI18n()

interface UsageData {
  labels: string[]
  requests: number[]
  tokens: number[]
  totals: { requests: number; tokens: number; cost: number }
}

const data = ref<UsageData | null>(null)

async function load() {
  const res = await http.get('/user/usage')
  data.value = res.data.data
}
onMounted(load)

const chartData = computed(() => {
  if (!data.value) return { labels: [], datasets: [] }
  return {
    labels: data.value.labels,
    datasets: [
      {
        label: t('dashboard.charts.requests'),
        data: data.value.requests,
        borderColor: '#4f8cff',
        backgroundColor: 'rgba(79, 140, 255, 0.15)',
        fill: true,
        tension: 0.35
      }
    ]
  }
})

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: { legend: { display: false } },
  scales: {
    x: { grid: { display: false } },
    y: { grid: { color: 'rgba(15, 23, 42, 0.05)' } }
  }
}
</script>

<template>
  <div class="space-y-5 animate-fade-in">
    <div>
      <h1 class="brand-serif text-2xl font-semibold text-ink-900">{{ t('usage.title') }}</h1>
      <p class="text-sm text-ink-500 ui-sans">{{ t('usage.subtitle') }}</p>
    </div>

    <div v-if="data" class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div class="card">
        <div class="text-xs text-ink-500 ui-sans">{{ t('usage.totalRequests') }}</div>
        <div class="mt-2 brand-serif text-2xl font-semibold">{{ data.totals.requests.toLocaleString() }}</div>
      </div>
      <div class="card">
        <div class="text-xs text-ink-500 ui-sans">{{ t('usage.totalTokens') }}</div>
        <div class="mt-2 brand-serif text-2xl font-semibold">{{ data.totals.tokens.toLocaleString() }}</div>
      </div>
      <div class="card">
        <div class="text-xs text-ink-500 ui-sans">{{ t('usage.cost') }}</div>
        <div class="mt-2 brand-serif text-2xl font-semibold">¥ {{ data.totals.cost.toFixed(2) }}</div>
      </div>
    </div>

    <div class="card">
      <div class="h-72">
        <Line v-if="data" :data="chartData" :options="chartOptions" />
        <div v-else class="h-full flex items-center justify-center text-ink-400 ui-sans">
          {{ t('common.loading') }}
        </div>
      </div>
    </div>
  </div>
</template>
