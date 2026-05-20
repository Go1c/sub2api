<template>
  <Teleport to="body">
    <transition
      enter-active-class="transition-opacity duration-200"
      leave-active-class="transition-opacity duration-200"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <div
        v-if="open"
        class="fixed inset-0 z-[1000] flex items-center justify-center bg-black/70 p-4 backdrop-blur-md"
        @click.self="onBackdrop"
      >
        <transition
          enter-active-class="transition duration-300 ease-out"
          leave-active-class="transition duration-200 ease-in"
          enter-from-class="scale-95 opacity-0"
          leave-to-class="scale-95 opacity-0"
        >
          <div
            v-if="open"
            class="relative w-[min(92vw,460px)] overflow-hidden rounded-3xl bg-[#0a0a14] shadow-[0_30px_90px_-12px_rgba(139,92,246,0.5)] ring-1 ring-white/10"
          >
            <!-- Gradient halo top -->
            <div class="pointer-events-none absolute -top-32 left-1/2 -z-0 h-72 w-[120%] -translate-x-1/2 rounded-full bg-gradient-to-br from-blue-500/25 via-violet-500/25 to-fuchsia-500/20 blur-3xl"></div>

            <!-- Close -->
            <button
              v-if="!isSpinning && !result"
              type="button"
              class="absolute right-3 top-3 z-10 rounded-full p-1.5 text-slate-400 hover:bg-white/5 hover:text-white"
              @click="$emit('close')"
              aria-label="关闭"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M6 6 L18 18 M18 6 L6 18" stroke-linecap="round" />
              </svg>
            </button>

            <div class="relative px-6 pb-6 pt-8">
              <!-- Header -->
              <div class="text-center">
                <div class="inline-flex items-center gap-2 rounded-full bg-white/5 px-3 py-1 text-xs font-medium text-slate-200 ring-1 ring-white/10">
                  <span class="inline-block h-1.5 w-1.5 rounded-full bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]"></span>
                  <span>登录有礼 · 限时活动</span>
                </div>
                <h2 class="mt-4 text-2xl font-extrabold tracking-tight text-white">
                  {{ campaignTitle }}
                </h2>
                <p class="mt-1.5 text-sm text-slate-400">
                  {{ subtitle }}
                </p>
              </div>

              <!-- Stats -->
              <div class="mt-5 grid grid-cols-3 gap-2 rounded-2xl bg-white/[0.04] p-3 text-center text-xs ring-1 ring-white/10">
                <div>
                  <div class="bg-gradient-to-br from-blue-400 to-fuchsia-400 bg-clip-text font-mono text-lg font-bold text-transparent">
                    {{ prizeCount }}
                  </div>
                  <div class="mt-0.5 text-slate-400">奖品</div>
                </div>
                <div class="border-x border-white/5">
                  <div class="bg-gradient-to-br from-blue-400 to-fuchsia-400 bg-clip-text font-mono text-lg font-bold text-transparent">
                    {{ joined }}/{{ maxParticipants }}
                  </div>
                  <div class="mt-0.5 text-slate-400">已参与</div>
                </div>
                <div>
                  <div class="bg-gradient-to-br from-blue-400 to-fuchsia-400 bg-clip-text font-mono text-lg font-bold text-transparent">
                    {{ remaining }}
                  </div>
                  <div class="mt-0.5 text-slate-400">剩余</div>
                </div>
              </div>

              <!-- Wheel -->
              <div class="mt-6 flex justify-center">
                <LotteryWheel
                  ref="wheelRef"
                  :segments="segments"
                  :size="280"
                  :disabled="!canSpin"
                  @start="onStart"
                  @spin-end="onSpinEnd"
                />
              </div>

              <!-- Result panel -->
              <transition
                enter-active-class="transition duration-300 ease-out"
                enter-from-class="translate-y-2 opacity-0"
                leave-active-class="transition duration-200"
                leave-to-class="opacity-0"
              >
                <div v-if="result" class="mt-6">
                  <div
                    v-if="result.won"
                    class="relative overflow-hidden rounded-2xl bg-white/[0.03] p-4 ring-1 ring-white/10"
                  >
                    <div class="pointer-events-none absolute inset-0 -z-0 bg-gradient-to-br from-blue-500/15 via-violet-500/15 to-fuchsia-500/15"></div>
                    <div class="relative">
                      <div class="text-xs font-medium text-blue-300">🎉 恭喜你抽中</div>
                      <div class="mt-1 text-lg font-bold text-white">
                        {{ result.label }}
                      </div>
                      <div class="mt-3">
                        <div class="text-xs text-slate-400">兑换码</div>
                        <div class="mt-1 flex items-center gap-2">
                          <code class="flex-1 rounded-xl bg-black/40 px-3 py-2.5 font-mono text-sm font-semibold tracking-wider text-fuchsia-300 ring-1 ring-white/10">
                            {{ result.code }}
                          </code>
                          <button
                            type="button"
                            class="rounded-xl bg-white/5 px-3 py-2.5 text-xs font-medium text-white ring-1 ring-white/10 transition-colors hover:bg-white/10"
                            @click="copyCode"
                          >
                            {{ copied ? '已复制' : '复制' }}
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div
                    v-else
                    class="rounded-2xl bg-white/[0.03] p-4 text-center ring-1 ring-white/10"
                  >
                    <div class="text-sm text-slate-300">很遗憾，本次未中奖</div>
                    <div class="mt-1 text-base font-semibold text-slate-100">下次再来 🍀</div>
                  </div>

                  <button
                    type="button"
                    class="mt-3 w-full rounded-full bg-gradient-to-r from-blue-500 to-violet-600 px-4 py-2.5 text-sm font-semibold text-white shadow-[0_8px_24px_-4px_rgba(99,102,241,0.5)] transition-transform hover:-translate-y-0.5"
                    @click="$emit('close')"
                  >
                    完成
                  </button>
                </div>
              </transition>

              <div v-if="!result" class="mt-4 text-center">
                <p class="text-xs text-slate-500">每位用户每场活动仅限一次</p>
              </div>
            </div>
          </div>
        </transition>
      </div>
    </transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import LotteryWheel, { type WheelSegment } from './LotteryWheel.vue'

export interface DrawResult {
  won: boolean
  index: number
  label: string
  code?: string
}

const props = withDefaults(
  defineProps<{
    open: boolean
    campaignTitle?: string
    subtitle?: string
    prizeCount: number
    maxParticipants: number
    joined: number
    segments: WheelSegment[]
    drawFn: () => Promise<DrawResult>
  }>(),
  {
    campaignTitle: '幸运转盘',
    subtitle: '登录就有机会，转一转赢取兑换码'
  }
)

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'drawn', result: DrawResult): void
}>()

const wheelRef = ref<InstanceType<typeof LotteryWheel> | null>(null)
const isSpinning = ref(false)
const result = ref<DrawResult | null>(null)
const copied = ref(false)
const pendingResult = ref<DrawResult | null>(null)

const remaining = computed(() => Math.max(0, props.prizeCount))
const canSpin = computed(() => props.joined < props.maxParticipants && !result.value)

async function onStart() {
  if (!canSpin.value || isSpinning.value) return
  isSpinning.value = true
  try {
    const r = await props.drawFn()
    await wheelRef.value?.spinTo(r.index)
    pendingResult.value = r
  } catch (e) {
    isSpinning.value = false
    console.error('draw failed', e)
  }
}

function onSpinEnd() {
  isSpinning.value = false
  if (pendingResult.value) {
    result.value = pendingResult.value
    emit('drawn', pendingResult.value)
    pendingResult.value = null
  }
}

function onBackdrop() {
  if (isSpinning.value) return
  emit('close')
}

async function copyCode() {
  if (!result.value?.code) return
  try {
    await navigator.clipboard.writeText(result.value.code)
    copied.value = true
    setTimeout(() => (copied.value = false), 1600)
  } catch (e) {
    console.warn('clipboard failed', e)
  }
}

watch(
  () => props.open,
  v => {
    if (v) {
      result.value = null
      pendingResult.value = null
      copied.value = false
    }
  }
)
</script>
