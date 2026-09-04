<template>
  <div class="relative inline-block select-none">
    <!-- Pointer with brand gradient -->
    <div class="pointer-events-none absolute left-1/2 top-0 z-20 -translate-x-1/2 -translate-y-1">
      <svg width="40" height="52" viewBox="0 0 40 52">
        <defs>
          <linearGradient :id="`ptr-${uid}`" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stop-color="#4f8cff" />
            <stop offset="100%" stop-color="#1a2f5a" />
          </linearGradient>
        </defs>
        <path d="M20 52 L4 14 A16 16 0 0 1 36 14 Z" :fill="`url(#ptr-${uid})`" />
        <circle cx="20" cy="16" r="5" fill="#ffffff" />
      </svg>
    </div>

    <!-- Disk -->
    <div
      class="relative overflow-hidden rounded-full bg-[#0b0b14] shadow-[0_20px_60px_-12px_rgba(79,140,255,0.45)] ring-1 ring-white/10"
      :style="{
        width: `${size}px`,
        height: `${size}px`,
        padding: '6px',
        backgroundImage:
          'linear-gradient(#0b0b14, #0b0b14), linear-gradient(135deg, #4f8cff 0%, #2f6dff 50%, #1a2f5a 100%)',
        backgroundOrigin: 'border-box',
        backgroundClip: 'padding-box, border-box',
        border: '2px solid transparent'
      }"
    >
      <svg
        :viewBox="`0 0 ${size} ${size}`"
        :width="size - 16"
        :height="size - 16"
        class="block rounded-full"
        :style="{
          transform: `rotate(${rotation}deg)`,
          transition: spinning
            ? `transform ${durationMs}ms cubic-bezier(0.17, 0.85, 0.32, 1.04)`
            : 'none'
        }"
        @transitionend="onTransitionEnd"
      >
        <g v-for="(seg, idx) in segments" :key="idx">
          <path :d="seg.path" :fill="seg.color" :stroke="seg.stroke" stroke-width="1.5" />
          <g :transform="`translate(${seg.labelX}, ${seg.labelY}) rotate(${seg.labelRotate})`">
            <text
              :fill="seg.textColor"
              text-anchor="middle"
              dominant-baseline="middle"
              font-weight="700"
              :font-size="seg.fontSize"
            >
              {{ truncate(seg.label, 9) }}
            </text>
          </g>
        </g>
        <circle :cx="cxInner" :cy="cyInner" :r="hubR" fill="#0b0b14" stroke="#ffffff" stroke-width="2" stroke-opacity="0.2" />
      </svg>

      <!-- Center "Go" button: brand gradient capsule -->
      <button
        v-if="showSpinButton"
        type="button"
        :disabled="spinning || disabled"
        class="absolute left-1/2 top-1/2 z-10 flex h-[88px] w-[88px] -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full bg-gradient-to-br from-[#4f8cff] via-[#2f6dff] to-[#1a2f5a] text-sm font-bold text-white shadow-[0_8px_28px_-4px_rgba(79,140,255,0.65)] ring-2 ring-white/20 transition-transform hover:scale-105 disabled:cursor-not-allowed disabled:opacity-60 disabled:hover:scale-100"
        @click="$emit('start')"
      >
        <span v-if="spinning">抽奖中</span>
        <span v-else>开始抽奖</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

export interface WheelSegment {
  label: string
  isPrize: boolean
}

const props = withDefaults(
  defineProps<{
    segments: WheelSegment[]
    size?: number
    showSpinButton?: boolean
    disabled?: boolean
  }>(),
  { size: 320, showSpinButton: true, disabled: false }
)

const emit = defineEmits<{
  (e: 'start'): void
  (e: 'spin-end', payload: { index: number }): void
}>()

const uid = Math.random().toString(36).slice(2, 8)
const rotation = ref(0)
const spinning = ref(false)
const durationMs = ref(4200)
const pendingIndex = ref<number | null>(null)

// SVG is inside a padded disk: effective drawing area is size - 16 (8px padding both sides)
const innerSize = computed(() => props.size - 16)
const cx = computed(() => innerSize.value / 2)
const cy = computed(() => innerSize.value / 2)
const cxInner = computed(() => innerSize.value / 2)
const cyInner = computed(() => innerSize.value / 2)
const radius = computed(() => innerSize.value / 2 - 2)
const hubR = computed(() => Math.max(14, innerSize.value * 0.07))

// Prize palette: stay within blue → purple → fuchsia → cyan spectrum
const PRIZE_PALETTE = ['#4f8cff', '#2f6dff', '#1a2f5a', '#7db2ff', '#3b82f6']
// Blank palette: dark translucent so the wheel feels "carbon" with bright prize accents
const BLANK_A = '#1f2030'
const BLANK_B = '#2a2b40'

const segments = computed(() => {
  const total = props.segments.length || 1
  const arc = 360 / total
  const prizeIndices = props.segments
    .map((s, idx) => (s.isPrize ? idx : -1))
    .filter(idx => idx !== -1)
  const blankIndices = props.segments
    .map((s, idx) => (!s.isPrize ? idx : -1))
    .filter(idx => idx !== -1)

  return props.segments.map((seg, i) => {
    const start = i * arc - 90
    const end = start + arc
    const sRad = (start * Math.PI) / 180
    const eRad = (end * Math.PI) / 180
    const r = radius.value
    const x1 = cx.value + r * Math.cos(sRad)
    const y1 = cy.value + r * Math.sin(sRad)
    const x2 = cx.value + r * Math.cos(eRad)
    const y2 = cy.value + r * Math.sin(eRad)
    const largeArc = arc > 180 ? 1 : 0
    const path = `M ${cx.value} ${cy.value} L ${x1} ${y1} A ${r} ${r} 0 ${largeArc} 1 ${x2} ${y2} Z`

    const mid = (start + end) / 2
    const midRad = (mid * Math.PI) / 180
    const labelDist = r * 0.66
    const labelX = cx.value + labelDist * Math.cos(midRad)
    const labelY = cy.value + labelDist * Math.sin(midRad)

    let color: string
    let textColor: string
    if (seg.isPrize) {
      const pIdx = prizeIndices.indexOf(i)
      color = PRIZE_PALETTE[pIdx % PRIZE_PALETTE.length]
      textColor = '#ffffff'
    } else {
      const bIdx = blankIndices.indexOf(i)
      color = bIdx % 2 === 0 ? BLANK_A : BLANK_B
      textColor = '#94a3b8'
    }

    return {
      path,
      color,
      stroke: '#0b0b14',
      labelX,
      labelY,
      labelRotate: mid + 90,
      label: seg.label,
      textColor,
      fontSize: arc < 30 ? 11 : arc < 45 ? 13 : 14
    }
  })
})

function truncate(s: string, n: number): string {
  return s.length > n ? s.slice(0, n - 1) + '…' : s
}

function spinTo(index: number): Promise<void> {
  if (spinning.value) return Promise.resolve()
  if (index < 0 || index >= props.segments.length) return Promise.resolve()
  const total = props.segments.length
  const arc = 360 / total
  const targetMod = -(index * arc + arc / 2)
  const currentMod = ((rotation.value % 360) + 360) % 360
  const desiredMod = ((targetMod % 360) + 360) % 360
  const delta = (desiredMod - currentMod + 360) % 360
  const turns = 6
  rotation.value = rotation.value + turns * 360 + delta
  spinning.value = true
  pendingIndex.value = index
  return Promise.resolve()
}

function onTransitionEnd() {
  if (!spinning.value) return
  spinning.value = false
  const idx = pendingIndex.value
  pendingIndex.value = null
  if (idx !== null) emit('spin-end', { index: idx })
}

watch(
  () => props.segments.length,
  () => {
    rotation.value = 0
  }
)

defineExpose({ spinTo, isSpinning: () => spinning.value })
</script>
