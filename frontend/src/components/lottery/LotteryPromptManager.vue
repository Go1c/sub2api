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
    @drawn="handleDrawn"
    @close="dismiss"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useLotteryStore } from '@/stores/lottery'
import LotteryDialog, { type DrawResult } from './LotteryDialog.vue'
import type { LotteryActiveCampaign } from '@/types'

const authStore = useAuthStore()
const lotteryStore = useLotteryStore()

const SESSION_DISMISS_KEY = 'lottery_dismissed_v2'
const open = ref(false)
const dialogCampaign = ref<LotteryActiveCampaign | null>(null)
const drawCompleted = ref(false)
let refreshToken = 0

const userId = computed<number | null>(() => authStore.user?.id ?? null)
const dialogSegments = computed(() =>
  (dialogCampaign.value?.segments ?? []).map((segment) => ({
    label: segment.label,
    isPrize: segment.is_prize,
  })),
)

function loadDismissedCampaigns(): number[] {
  try {
    const raw = sessionStorage.getItem(SESSION_DISMISS_KEY)
    if (!raw) return []
    const list = JSON.parse(raw)
    if (!Array.isArray(list)) return []
    return list
      .map((value) => Number(value))
      .filter((value) => Number.isInteger(value) && value > 0)
  } catch {
    return []
  }
}

function isDismissedThisSession(campaignId: number): boolean {
  return loadDismissedCampaigns().includes(campaignId)
}

function markDismissed(campaignId: number) {
  try {
    const list = loadDismissedCampaigns()
    if (!list.includes(campaignId)) list.push(campaignId)
    sessionStorage.setItem(SESSION_DISMISS_KEY, JSON.stringify(list))
  } catch {
    // ignore storage failures
  }
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

watch(
  () => [authStore.isAuthenticated, userId.value],
  async ([isAuthenticated, currentUserId]) => {
    refreshToken += 1
    const currentRefresh = refreshToken

    if (!isAuthenticated || !currentUserId) {
      open.value = false
      dialogCampaign.value = null
      drawCompleted.value = false
      lotteryStore.clearActive()
      lotteryStore.clearLastResult()
      return
    }

    try {
      await lotteryStore.fetchActive()
      if (currentRefresh !== refreshToken) return
      syncDialog()
    } catch {
      if (currentRefresh !== refreshToken) return
      open.value = false
      dialogCampaign.value = null
    }
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
</script>
