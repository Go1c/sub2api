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
        class="fixed inset-0 z-[1000] flex items-start justify-center overflow-y-auto bg-black/70 p-3 py-4 backdrop-blur-md sm:items-center sm:p-4"
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
            class="relative max-h-[calc(100dvh-2rem)] w-[min(92vw,460px)] overflow-y-auto overflow-x-hidden rounded-3xl bg-[#0a0a14] shadow-[0_30px_90px_-12px_rgba(139,92,246,0.5)] ring-1 ring-white/10 transition duration-300"
            :class="result ? 'scale-[0.98] opacity-45 blur-[1px]' : ''"
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

            <div class="relative px-4 pb-5 pt-7 sm:px-6 sm:pb-6 sm:pt-8">
              <!-- Header -->
              <div class="text-center">
                <div class="inline-flex items-center gap-2 rounded-full bg-white/5 px-3 py-1 text-xs font-medium text-slate-200 ring-1 ring-white/10">
                  <span class="inline-block h-1.5 w-1.5 rounded-full bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]"></span>
                  <span>登录有礼 · 限时活动</span>
                </div>
                <h2 class="mt-4 text-2xl font-extrabold tracking-tight text-white">
                  {{ stage === 'promo' ? '关注公众号再抽奖' : campaignTitle }}
                </h2>
                <p class="mt-1.5 text-sm text-slate-400">
                  {{ stage === 'promo' ? (promoText || '关注后可了解更多活动与兑换码') : subtitle }}
                </p>
              </div>

              <div v-if="stage === 'promo'" class="mt-5 text-center" data-test="lottery-promo-step">
                <img
                  :src="promoImageSrc"
                  alt="公众号关注海报"
                  class="mx-auto max-h-[360px] w-full max-w-[280px] rounded-3xl bg-white object-contain shadow-[0_18px_48px_-18px_rgba(79,140,255,0.55)] ring-1 ring-white/15"
                />
                <button
                  type="button"
                  data-test="lottery-promo-continue"
                  class="mt-5 w-full rounded-full bg-gradient-to-r from-[#4f8cff] via-[#2f6dff] to-[#1a2f5a] px-5 py-3 text-sm font-extrabold text-white shadow-[0_14px_42px_-12px_rgba(79,140,255,0.9)]"
                  @click="stage = 'wheel'"
                >
                  去抽奖
                </button>
                <p class="mt-3 text-xs text-slate-500">可跳过，不验证是否已经关注</p>
              </div>

              <template v-else>
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
                  :size="wheelSize"
                  :disabled="!canSpin"
                  @start="onStart"
                  @spin-end="onSpinEnd"
                />
              </div>

              <div
                v-if="errorMessage && !result"
                class="mt-4 rounded-2xl border border-red-500/20 bg-red-500/10 px-4 py-3 text-sm text-red-100"
              >
                {{ errorMessage }}
              </div>

              <div v-if="!result" class="mt-4 text-center">
                <p class="text-xs text-slate-500">每位用户每场活动仅限一次</p>
              </div>
              </template>
            </div>
          </div>
        </transition>

        <transition
          enter-active-class="transition duration-500 ease-out"
          enter-from-class="scale-75 opacity-0"
          leave-active-class="transition duration-200 ease-in"
          leave-to-class="scale-95 opacity-0"
        >
          <div
            v-if="result"
            data-test="lottery-result-modal"
            class="fixed inset-0 z-[1010] flex items-center justify-center p-4"
          >
            <div class="absolute inset-0 bg-black/35"></div>
            <div
              class="result-modal-card relative w-[min(92vw,420px)] overflow-hidden rounded-[2rem] p-6 text-center shadow-[0_30px_100px_-12px_rgba(0,0,0,0.65)] ring-1 ring-white/20 sm:p-7"
              :class="result.won ? 'result-modal-card--win' : 'result-modal-card--lose'"
            >
              <div v-if="result.won" class="result-confetti" aria-hidden="true">
                <span v-for="i in 16" :key="i" :style="{ '--i': i }"></span>
              </div>
              <div class="result-burst" aria-hidden="true"></div>

              <div class="relative">
                <div
                  class="mx-auto flex h-20 w-20 items-center justify-center rounded-full text-4xl shadow-[0_18px_48px_-12px_rgba(255,255,255,0.75)] ring-1 ring-white/30"
                  :class="result.won ? 'bg-gradient-to-br from-[#4f8cff] via-[#7db2ff] to-cyan-400 text-white' : 'bg-white/10'"
                >
                  {{ result.won ? '✦' : '🍀' }}
                </div>

                <div
                  class="mt-5 text-sm font-bold uppercase tracking-[0.26em]"
                  :class="result.won ? 'text-sky-100' : 'text-slate-300'"
                >
                  {{ result.won ? 'Lucky Hit' : 'Next Time' }}
                </div>
                <h3 class="mt-2 text-3xl font-black tracking-tight text-white sm:text-4xl">
                  {{ result.won ? '恭喜中奖' : '本次未中奖' }}
                </h3>
                <div
                  class="mx-auto mt-3 inline-flex rounded-full px-4 py-1.5 text-sm font-bold"
                  :class="result.won ? 'bg-[#4f8cff]/20 text-sky-100 ring-1 ring-[#4f8cff]/45' : 'bg-white/10 text-slate-100 ring-1 ring-white/15'"
                >
                  {{ result.label }}
                </div>
                <p class="mx-auto mt-4 max-w-[21rem] text-sm font-medium leading-7 text-slate-100">
                  {{ result.message }}
                </p>

                <div
                  v-if="result.won && (promoImageSrc || promoText)"
                  class="mt-5 rounded-3xl bg-white/8 p-3 ring-1 ring-white/15"
                  data-test="lottery-result-promo"
                >
                  <img
                    v-if="promoImageSrc"
                    :src="promoImageSrc"
                    alt="公众号关注海报"
                    class="mx-auto max-h-48 w-full max-w-[200px] rounded-2xl bg-white object-contain"
                  />
                  <p v-if="promoText" class="mt-3 text-xs leading-6 text-slate-200">
                    {{ promoText }}
                  </p>
                </div>

                <div class="mt-6 flex flex-col gap-3">
                  <a
                    v-if="result.won"
                    href="/site-messages"
                    class="rounded-full bg-gradient-to-r from-[#4f8cff] via-[#2f6dff] to-[#1a2f5a] px-5 py-3 text-sm font-extrabold text-white shadow-[0_14px_42px_-12px_rgba(79,140,255,0.9)] transition-transform hover:-translate-y-0.5"
                  >
                    前往站内信领取
                  </a>
                  <button
                    type="button"
                    class="rounded-full px-5 py-3 text-sm font-bold text-white ring-1 ring-white/15 transition-transform hover:-translate-y-0.5"
                    :class="result.won ? 'bg-white/10 hover:bg-white/15' : 'bg-gradient-to-r from-blue-500 to-violet-600 shadow-[0_14px_42px_-12px_rgba(99,102,241,0.8)]'"
                    @click="$emit('close')"
                  >
                    {{ result.won ? '完成' : '下次再来' }}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </transition>
      </div>
    </transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import LotteryWheel, { type WheelSegment } from './LotteryWheel.vue'
import { safeLotteryPromoImageUrl } from '@/utils/siteMessageContent'

export interface DrawResult {
  won: boolean
  index: number
  label: string
  message: string
  site_message_id?: number | null
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
    promoText?: string
    promoImageUrl?: string
  }>(),
  {
    campaignTitle: '幸运转盘',
    subtitle: '登录就有机会，转一转赢取兑换码',
    promoText: '',
    promoImageUrl: ''
  }
)

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'drawn', result: DrawResult): void
}>()

const wheelRef = ref<InstanceType<typeof LotteryWheel> | null>(null)
const isSpinning = ref(false)
const result = ref<DrawResult | null>(null)
const pendingResult = ref<DrawResult | null>(null)
const errorMessage = ref('')
const viewportWidth = ref(getViewportWidth())
const promoImageSrc = computed(() => safeLotteryPromoImageUrl(props.promoImageUrl))
const hasPromo = computed(() => Boolean(promoImageSrc.value))
const stage = ref<'promo' | 'wheel'>(hasPromo.value ? 'promo' : 'wheel')

const remaining = computed(() => Math.max(0, props.maxParticipants - props.joined))
const canSpin = computed(() => props.joined < props.maxParticipants && !result.value)
const wheelSize = computed(() => Math.min(280, Math.max(180, viewportWidth.value - 96)))

function updateViewportWidth() {
  viewportWidth.value = getViewportWidth()
}

function getViewportWidth(): number {
  if (typeof window === 'undefined') return 460
  return window.innerWidth
}

onMounted(() => {
  updateViewportWidth()
  window.addEventListener('resize', updateViewportWidth)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateViewportWidth)
})

async function onStart() {
  if (!canSpin.value || isSpinning.value) return
  isSpinning.value = true
  errorMessage.value = ''
  try {
    const r = await props.drawFn()
    pendingResult.value = r
    await wheelRef.value?.spinTo(r.index)
  } catch (e) {
    isSpinning.value = false
    errorMessage.value = extractErrorMessage(e)
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
  if (isSpinning.value || result.value) return
  emit('close')
}

function extractErrorMessage(error: unknown): string {
  if (typeof error === 'object' && error !== null) {
    const message = (error as { message?: string; error?: string }).message
      ?? (error as { message?: string; error?: string }).error
    if (message) return message
  }
  if (error instanceof Error && error.message) return error.message
  return '抽奖失败，请稍后再试。'
}

watch(
  () => props.open,
  v => {
    if (v) {
      result.value = null
      pendingResult.value = null
      errorMessage.value = ''
      stage.value = hasPromo.value ? 'promo' : 'wheel'
    }
  }
)
</script>

<style scoped>
.result-modal-card {
  animation: result-pop 620ms cubic-bezier(0.16, 1, 0.3, 1);
}

.result-modal-card--win {
  background:
    radial-gradient(circle at 50% 0%, rgba(79, 140, 255, 0.62), transparent 34%),
    radial-gradient(circle at 12% 16%, rgba(34, 211, 238, 0.28), transparent 28%),
    radial-gradient(circle at 88% 24%, rgba(52, 211, 153, 0.18), transparent 24%),
    linear-gradient(145deg, #081631 0%, #1a2f5a 52%, #070b17 100%);
}

.result-modal-card--lose {
  background:
    radial-gradient(circle at 50% 0%, rgba(96, 165, 250, 0.28), transparent 36%),
    linear-gradient(145deg, #111827 0%, #1e1b4b 55%, #0b1020 100%);
}

.result-burst {
  position: absolute;
  inset: -35%;
  background:
    conic-gradient(
      from 0deg,
      transparent 0 8deg,
      rgba(255, 255, 255, 0.2) 8deg 10deg,
      transparent 10deg 28deg
    );
  animation: result-spin 6s linear infinite;
  opacity: 0.55;
}

.result-confetti {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.result-confetti span {
  position: absolute;
  left: calc(6% + (var(--i) * 5.5%));
  top: -18px;
  width: 8px;
  height: 14px;
  border-radius: 3px;
  background: linear-gradient(135deg, #4f8cff, #22d3ee);
  transform: rotate(calc(var(--i) * 23deg));
  animation: confetti-fall 1300ms ease-out both;
  animation-delay: calc(var(--i) * 35ms);
}

.result-confetti span:nth-child(3n) {
  background: linear-gradient(135deg, #7db2ff, #34d399);
}

.result-confetti span:nth-child(3n + 1) {
  background: linear-gradient(135deg, #1a2f5a, #4f8cff);
}

@keyframes result-pop {
  0% {
    opacity: 0;
    transform: translateY(22px) scale(0.82);
  }
  60% {
    opacity: 1;
    transform: translateY(-4px) scale(1.04);
  }
  100% {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@keyframes result-spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes confetti-fall {
  0% {
    opacity: 0;
    transform: translateY(-20px) rotate(0deg) scale(0.8);
  }
  15% {
    opacity: 1;
  }
  100% {
    opacity: 0;
    transform: translateY(230px) rotate(360deg) scale(1);
  }
}
</style>
