<template>
  <LotteryDialog
    v-if="dialogCampaign"
    :open="open"
    :campaign-title="dialogCampaign.name"
    :subtitle="dialogCampaign.subtitle"
    :prize-count="dialogCampaign.prize_count"
    :max-participants="dialogCampaign.max_participants"
    :joined="dialogCampaign.joined_count"
    :segments="dialogSegments"
    :draw-fn="doDraw"
    :promo-text="dialogCampaign.promo_text"
    :promo-image-url="dialogCampaign.promo_image_url"
    @drawn="handleDrawn"
    @close="dismiss"
  />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useLotteryStore } from '@/stores/lottery'
import LotteryDialog, { type DrawResult } from './LotteryDialog.vue'
import type { LotteryActiveCampaign } from '@/types'

const authStore = useAuthStore()
const lotteryStore = useLotteryStore()

const ACTIVE_REFRESH_INTERVAL_MS = 60_000
let loginSessionUserId: number | null = null
let loginSessionToken: string | null = null
const dismissedCampaignsForLogin = new Set<number>()

const open = ref(false)
const dialogCampaign = ref<LotteryActiveCampaign | null>(null)
const drawCompleted = ref(false)
let refreshToken = 0
let refreshInterval: ReturnType<typeof setInterval> | null = null

const userId = computed<number | null>(() => authStore.user?.id ?? null)
const authToken = computed<string | null>(() => authStore.token ?? null)
const dialogSegments = computed(() =>
  (dialogCampaign.value?.segments ?? []).map((segment) => ({
    label: segment.label,
    isPrize: segment.is_prize,
  })),
)

function isDismissedThisSession(campaignId: number): boolean {
  return dismissedCampaignsForLogin.has(campaignId)
}

function markDismissed(campaignId: number) {
  dismissedCampaignsForLogin.add(campaignId)
}

function syncDialog() {
  const c = lotteryStore.activeCampaign
  if (!c || !userId.value) {
    if (!drawCompleted.value) {
      dialogCampaign.value = null
      open.value = false
    }
    return
  }
  if (isDismissedThisSession(c.id)) {
    dialogCampaign.value = null
    open.value = false
    return
  }
  dialogCampaign.value = c
  open.value = true
}

async function refreshActiveCampaign() {
  if (!authStore.isAuthenticated || !userId.value) return
  refreshToken += 1
  const currentRefresh = refreshToken
  try {
    await lotteryStore.fetchActive()
    if (currentRefresh !== refreshToken) return
    syncDialog()
  } catch {
    if (currentRefresh !== refreshToken) return
    open.value = false
    dialogCampaign.value = null
  }
}

function handleVisibilityChange() {
  if (document.visibilityState === 'visible') {
    refreshActiveCampaign()
  }
}

function startRefreshHooks() {
  if (refreshInterval === null) {
    refreshInterval = setInterval(refreshActiveCampaign, ACTIVE_REFRESH_INTERVAL_MS)
  }
  document.addEventListener('visibilitychange', handleVisibilityChange)
}

function stopRefreshHooks() {
  if (refreshInterval !== null) {
    clearInterval(refreshInterval)
    refreshInterval = null
  }
  document.removeEventListener('visibilitychange', handleVisibilityChange)
}

watch(
  () => [authStore.isAuthenticated, userId.value, authToken.value] as const,
  async ([isAuthenticated, currentUserId, currentToken]) => {
    if (!isAuthenticated || !currentUserId) {
      loginSessionUserId = null
      loginSessionToken = null
      dismissedCampaignsForLogin.clear()
      stopRefreshHooks()
      refreshToken += 1
      open.value = false
      dialogCampaign.value = null
      drawCompleted.value = false
      lotteryStore.clearActive()
      lotteryStore.clearLastResult()
      return
    }

    if (loginSessionUserId !== currentUserId || loginSessionToken !== currentToken) {
      loginSessionUserId = currentUserId
      loginSessionToken = currentToken
      dismissedCampaignsForLogin.clear()
    }
    startRefreshHooks()
    await refreshActiveCampaign()
  },
  { immediate: true },
)

watch(
  () => lotteryStore.activeCampaign?.id,
  () => syncDialog(),
)

async function doDraw(): Promise<DrawResult> {
  const c = dialogCampaign.value
  if (!c || !userId.value) throw new Error('no active campaign or user')
  return lotteryStore.draw(c.id)
}

function handleDrawn() {
  drawCompleted.value = true
}

function dismiss() {
  const c = dialogCampaign.value
  if (c && !drawCompleted.value) {
    markDismissed(c.id)
  }
  if (drawCompleted.value) {
    lotteryStore.clearActive()
    lotteryStore.clearLastResult()
  }
  open.value = false
  dialogCampaign.value = null
  drawCompleted.value = false
}

onBeforeUnmount(() => {
  stopRefreshHooks()
})
</script>
