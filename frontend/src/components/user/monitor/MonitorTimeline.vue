<template>
  <div :class="embedded ? 'mt-3' : 'mt-4 pt-3 border-t border-gray-100 dark:border-dark-700/60'">
    <div
      class="flex justify-between text-[10px] font-semibold uppercase tracking-widest text-gray-400 mb-2"
    >
      <span>{{ heading }}</span>
      <span v-if="showCountdown" class="tabular-nums">{{ t('monitorCommon.nextUpdateIn', { n: countdownSeconds }) }}</span>
      <span v-else></span>
    </div>

    <div
      v-if="maintenance"
      class="flex h-5 w-full items-center justify-center rounded border border-dashed border-gray-300 dark:border-dark-600 text-[10px] uppercase tracking-widest text-gray-400"
    >
      {{ t('monitorCommon.maintenancePaused') }}
    </div>
    <div v-else class="flex items-end gap-[2px] h-5 w-full">
      <div
        v-for="(bar, idx) in displayBars"
        :key="idx"
        class="flex-1 min-w-[3px] rounded-sm"
        :class="bar.colorClass"
        :style="{ height: bar.heightPct + '%' }"
        :title="bar.title"
      ></div>
    </div>

    <div
      class="mt-1 flex justify-between text-[9px] uppercase tracking-widest text-gray-400"
    >
      <span>{{ t('monitorCommon.past') }}</span>
      <span>{{ t('monitorCommon.now') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { MonitorTimelinePoint } from '@/api/channelMonitor'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'

const props = withDefaults(defineProps<{
  buckets?: MonitorTimelinePoint[]
  countdownSeconds: number
  length?: number
  maintenance?: boolean
  kind?: 'server' | 'iq'
  label?: string
  showCountdown?: boolean
  embedded?: boolean
}>(), {
  buckets: () => [],
  length: 60,
  maintenance: false,
  kind: 'server',
  showCountdown: true,
  embedded: false,
})

const { t } = useI18n()
const { iqStatusLabel, formatLatency, formatRelativeTime } = useChannelMonitorFormat()

const heading = computed(() => {
  if (props.label) return props.label
  return t('monitorCommon.history60pts', { n: props.length })
})

interface Bar {
  colorClass: string
  heightPct: number
  title: string
}

const STATUS_HEIGHT: Record<string, number> = {
  operational: 100,
  degraded: 65,
  failed: 35,
  error: 35,
  iq_ok: 100,
  iq_down: 65,
  test_timeout: 65,
  test_error: 35,
  monitor_network: 15,
  empty: 15,
}

const STATUS_COLOR: Record<string, string> = {
  operational: 'bg-emerald-500',
  degraded: 'bg-amber-500',
  failed: 'bg-red-500',
  error: 'bg-red-500',
  iq_ok: 'bg-emerald-500',
  iq_down: 'bg-amber-500',
  test_timeout: 'bg-amber-500',
  test_error: 'bg-red-500',
  monitor_network: 'bg-gray-300 dark:bg-dark-600',
  empty: 'bg-gray-300 dark:bg-dark-600',
}

const displayBars = computed<Bar[]>(() => {
  const real = [...(props.buckets ?? [])]
    .slice(0, props.length)
    .reverse()

  const padCount = Math.max(0, props.length - real.length)
  const bars: Bar[] = []

  for (let i = 0; i < padCount; i += 1) {
    bars.push({
      colorClass: STATUS_COLOR.empty,
      heightPct: STATUS_HEIGHT.empty,
      title: '',
    })
  }

  for (const point of real) {
    const status = String(point.status || 'empty')
    const colorClass = STATUS_COLOR[status] ?? STATUS_COLOR.empty
    const heightPct = STATUS_HEIGHT[status] ?? STATUS_HEIGHT.empty
    const latency = formatLatency(point.latency_ms)
    const relative = formatRelativeTime(point.checked_at)
    const label = iqStatusLabel(status)
    bars.push({
      colorClass,
      heightPct,
      title: `${relative} · ${label} · ${latency}ms`,
    })
  }

  return bars
})
</script>
