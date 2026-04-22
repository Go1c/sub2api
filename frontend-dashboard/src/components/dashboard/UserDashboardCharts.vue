<script setup lang="ts">
import { computed } from 'vue'
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

ChartJS.register(Title, Tooltip, Legend, PointElement, LineElement, CategoryScale, LinearScale, Filler)

const { t } = useI18n()

const props = defineProps<{
  labels: string[]
  requests: number[]
  tokens: number[]
}>()

const data = computed(() => ({
  labels: props.labels,
  datasets: [
    {
      label: t('dashboard.charts.requests'),
      data: props.requests,
      borderColor: '#4f8cff',
      backgroundColor: 'rgba(79, 140, 255, 0.12)',
      fill: true,
      tension: 0.35,
      yAxisID: 'y1'
    },
    {
      label: t('dashboard.charts.tokens'),
      data: props.tokens,
      borderColor: '#1e3a8a',
      backgroundColor: 'rgba(30, 58, 138, 0.08)',
      fill: false,
      tension: 0.35,
      yAxisID: 'y2'
    }
  ]
}))

const options = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { position: 'bottom' as const, labels: { usePointStyle: true, boxWidth: 8 } },
    tooltip: { mode: 'index' as const, intersect: false }
  },
  scales: {
    x: { grid: { display: false } },
    y1: {
      type: 'linear' as const,
      position: 'left' as const,
      grid: { color: 'rgba(15, 23, 42, 0.05)' }
    },
    y2: {
      type: 'linear' as const,
      position: 'right' as const,
      grid: { drawOnChartArea: false }
    }
  }
}))
</script>

<template>
  <div class="card">
    <div class="flex items-baseline justify-between mb-4">
      <h3 class="brand-serif text-base font-semibold text-ink-900">{{ t('dashboard.charts.title') }}</h3>
    </div>
    <div class="h-64">
      <Line :data="data" :options="options" />
    </div>
  </div>
</template>
