<template>
  <LotteryDialog
    v-if="campaign"
    :open="open"
    :campaign-title="campaign.name"
    :subtitle="campaign.subtitle"
    :prize-count="campaign.prizeCount"
    :max-participants="campaign.maxParticipants"
    :joined="campaign.joined"
    :segments="campaign.segments"
    :draw-fn="doDraw"
    @close="dismiss"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useLotteryStore } from '@/stores/lottery'
import LotteryDialog, { type DrawResult } from './LotteryDialog.vue'

const authStore = useAuthStore()
const lotteryStore = useLotteryStore()

const SESSION_DISMISS_KEY = 'lottery_dismissed_v1'
const open = ref(false)

const userId = computed<number | null>(() => authStore.user?.id ?? null)
const userName = computed<string>(() => authStore.user?.username || authStore.user?.email || `user-${userId.value ?? 'anon'}`)

const campaign = computed(() => lotteryStore.getActiveForUser(userId.value))

function isDismissedThisSession(campaignId: string): boolean {
  try {
    const raw = sessionStorage.getItem(SESSION_DISMISS_KEY)
    if (!raw) return false
    const list = JSON.parse(raw)
    return Array.isArray(list) && list.includes(campaignId)
  } catch {
    return false
  }
}

function markDismissed(campaignId: string) {
  try {
    const raw = sessionStorage.getItem(SESSION_DISMISS_KEY)
    const list = raw ? JSON.parse(raw) : []
    if (!list.includes(campaignId)) list.push(campaignId)
    sessionStorage.setItem(SESSION_DISMISS_KEY, JSON.stringify(list))
  } catch {
    // ignore storage failures
  }
}

function maybeOpen() {
  const c = campaign.value
  if (!c || !userId.value) {
    open.value = false
    return
  }
  if (isDismissedThisSession(c.id)) {
    open.value = false
    return
  }
  open.value = true
}

watch(
  () => [authStore.isAuthenticated, userId.value, campaign.value?.id],
  () => maybeOpen(),
  { immediate: true }
)

async function doDraw(): Promise<DrawResult> {
  const c = campaign.value
  if (!c || !userId.value) throw new Error('no active campaign or user')
  return lotteryStore.draw(userId.value, userName.value, c.id)
}

function dismiss() {
  const c = campaign.value
  if (c) markDismissed(c.id)
  open.value = false
}
</script>
