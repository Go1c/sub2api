<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <div v-else-if="visibleSubscriptions.length === 0" class="card p-12 text-center">
        <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700">
          <Icon name="creditCard" size="xl" class="text-gray-400" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('userSubscriptions.noActiveSubscriptions') }}
        </h3>
        <p class="text-gray-500 dark:text-dark-400">
          {{ t('userSubscriptions.noActiveSubscriptionsDesc') }}
        </p>
      </div>

      <template v-else>
        <div v-if="orderedUsableSubscriptions.length > 0" class="space-y-4">
          <section
            v-for="subscription in orderedUsableSubscriptions"
            :key="subscription.id"
            class="overflow-hidden rounded-3xl border bg-white shadow-sm dark:bg-dark-800"
            :class="platformBorderClass(subscription.group?.platform || '')"
          >
            <div :class="['h-2', platformAccentBarClass(subscription.group?.platform || '')]" />
            <div class="space-y-6 p-5 sm:p-7">
              <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div class="min-w-0">
                  <div class="mb-2 flex flex-wrap items-center gap-2">
                    <span :class="['rounded-md border px-2 py-0.5 text-xs font-medium', platformBadgeClass(subscription.group?.platform || '')]">
                      {{ platformLabel(subscription.group?.platform || '') }}
                    </span>
                    <span class="rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-semibold text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">{{ t('userSubscriptions.currentlyUsable') }}</span>
                    <span :class="['rounded-full px-2 py-0.5 text-xs font-semibold', platformBadgeLightClass(subscription.group?.platform || '')]">{{ scopeSummary(subscription) }}</span>
                  </div>
                  <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
                    {{ subscriptionName(subscription) }}
                  </h2>
                  <p v-if="subscription.group?.description" class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                    {{ subscription.group.description }}
                  </p>
                </div>
                <div class="flex flex-shrink-0 flex-wrap items-center gap-2">
                  <button
                    v-if="canShowResetWeeklyLimit(subscription)"
                    type="button"
                    data-testid="reset-weekly-limit"
                    class="rounded-xl bg-red-600 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-red-600 dark:hover:bg-red-500"
                    :disabled="isResetWeeklyLimitDisabled(subscription)"
                    @click="handleResetWeeklyLimit(subscription)"
                  >
                    {{ t('userSubscriptions.resetWeeklyLimit') }}
                  </button>
                  <button
                    type="button"
                    :class="['rounded-xl px-4 py-2 text-sm font-semibold text-white transition-colors', platformButtonClass(subscription.group?.platform || '')]"
                    @click="router.push({ path: '/purchase', query: { tab: 'subscription' } })"
                  >
                    {{ t('payment.renewNow') }}
                  </button>
                </div>
              </div>

              <div class="grid gap-4 md:grid-cols-[1.3fr_0.7fr]">
                <div class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/50">
                  <div class="mb-3 flex items-end justify-between gap-3">
                    <div>
                      <p class="text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-dark-400">{{ t('userSubscriptions.totalCredit') }}</p>
                      <p class="mt-1 text-3xl font-black text-gray-900 dark:text-white">${{ formatUsd(quotaRemaining(subscription)) }}</p>
                    </div>
                    <p class="text-sm font-medium text-gray-500 dark:text-dark-400">
                      {{ t('userSubscriptions.usageOf', { used: `$${formatUsd(quotaUsed(subscription))}`, limit: `$${formatUsd(quotaLimit(subscription))}` }) }}
                    </p>
                  </div>
                  <ProgressBar :used="quotaUsed(subscription)" :limit="quotaLimit(subscription)" />
                  <p v-if="subscription.exhausted_at" class="mt-2 text-xs text-amber-600 dark:text-amber-300">{{ t('userSubscriptions.exhaustedAt', { date: formatDateTimeValue(subscription.exhausted_at) }) }}</p>
                </div>

                <div class="rounded-2xl bg-gray-50 p-4 text-sm dark:bg-dark-700/50">
                  <div class="flex items-center justify-between">
                    <span class="text-gray-500 dark:text-dark-400">{{ t('userSubscriptions.expires') }}</span>
                    <span :class="getExpirationClass(subscription.expires_at)">{{ expirationDisplay(subscription.expires_at) }}</span>
                  </div>
                  <div class="mt-3 flex items-center justify-between">
                    <span class="text-gray-500 dark:text-dark-400">{{ t('payment.planCard.scope') }}</span>
                    <span class="font-medium text-gray-800 dark:text-gray-200">{{ scopeSummary(subscription) }}</span>
                  </div>
                </div>
              </div>

              <div class="grid gap-4 sm:grid-cols-2">
                <LimitPanel :title="t('userSubscriptions.daily')" :used="subscription.daily_usage_usd" :limit="subscription.daily_limit_usd ?? undefined" :reset-at="subscription.daily_reset_at ?? undefined" />
                <LimitPanel :title="t('userSubscriptions.weekly')" :used="subscription.weekly_usage_usd" :limit="subscription.weekly_limit_usd ?? undefined" :reset-at="subscription.weekly_reset_at ?? undefined" />
              </div>
              <p v-if="subscription.weekly_limit_usd != null" class="text-xs text-gray-500 dark:text-dark-400">
                {{ weeklyResetHint(subscription) }}
              </p>
            </div>
          </section>
        </div>

        <details v-if="exhaustedSubscriptions.length > 0" class="rounded-2xl border border-amber-200 bg-amber-50/60 p-4 dark:border-amber-500/30 dark:bg-amber-900/10">
          <summary class="cursor-pointer text-sm font-semibold text-amber-800 dark:text-amber-200">
            {{ t('userSubscriptions.exhaustedAwaitingExpiry', { count: exhaustedSubscriptions.length }) }}
          </summary>
          <div class="mt-3 space-y-2">
            <div v-for="subscription in exhaustedSubscriptions" :key="subscription.id" class="flex flex-wrap items-center justify-between gap-2 rounded-xl bg-white px-3 py-2 text-sm dark:bg-dark-800">
              <span class="font-medium text-gray-900 dark:text-white">{{ subscriptionName(subscription) }}</span>
              <span class="text-gray-500 dark:text-dark-400">{{ t('userSubscriptions.exhaustedSummary', { used: `$${formatUsd(quotaUsed(subscription))}`, limit: `$${formatUsd(quotaLimit(subscription))}`, expires: expirationDisplay(subscription.expires_at) }) }}</span>
            </div>
          </div>
        </details>
      </template>
    </div>

    <ConfirmDialog
      :show="showResetWeeklyLimitConfirm"
      :title="t('userSubscriptions.resetWeeklyLimitTitle')"
      :message="resetWeeklyLimitConfirmMessage"
      :confirm-text="t('userSubscriptions.resetWeeklyLimit')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmResetWeeklyLimit"
      @cancel="closeResetWeeklyLimitDialog"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import subscriptionsAPI, { subscriptionCreditErrorMessages } from '@/api/subscriptions'
import { extractApiErrorCode, extractApiErrorMessage } from '@/utils/apiError'
import type { UserSubscription } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { formatDateTime, formatDateTimeToMinute } from '@/utils/format'
import { platformAccentBarClass, platformBadgeClass, platformBadgeLightClass, platformBorderClass, platformButtonClass, platformLabel } from '@/utils/platformColors'
import { subscriptionDisplayName } from '@/utils/subscriptionDisplay'

const ProgressBar = defineComponent({
  props: { used: { type: Number, default: 0 }, limit: { type: Number, default: 0 } },
  setup(props) {
    return () => h('div', { class: 'relative h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600' }, [
      h('div', {
        class: ['absolute inset-y-0 left-0 rounded-full transition-all duration-300', getProgressBarClass(props.used, props.limit)],
        style: { width: getProgressWidth(props.used, props.limit) },
      }),
    ])
  },
})

const LimitPanel = defineComponent({
  props: {
    title: { type: String, required: true },
    used: { type: Number, default: 0 },
    limit: { type: Number, default: null },
    resetAt: { type: String, default: '' },
  },
  setup(props) {
    return () => h('div', { class: 'rounded-2xl border border-gray-100 p-4 dark:border-dark-700' }, [
      h('div', { class: 'mb-2 flex items-center justify-between' }, [
        h('span', { class: 'text-sm font-semibold text-gray-700 dark:text-gray-300' }, props.title),
        h('span', { class: 'text-sm text-gray-500 dark:text-dark-400' }, props.limit == null
          ? t('userSubscriptions.unlimited')
          : t('userSubscriptions.usageOf', { used: `$${formatUsd(props.used)}`, limit: `$${formatUsd(props.limit)}` })),
      ]),
      props.limit == null ? null : h(ProgressBar, { used: props.used, limit: props.limit }),
      h('p', { class: 'mt-2 text-xs text-gray-500 dark:text-dark-400' }, formatResetLabel(props.resetAt || '')),
    ])
  },
})

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const subscriptions = ref<UserSubscription[]>([])
const loading = ref(true)
const showResetWeeklyLimitConfirm = ref(false)
const weeklyLimitSubscription = ref<UserSubscription | null>(null)
const resettingWeeklyLimit = ref(false)

const visibleSubscriptions = computed(() => subscriptions.value.filter(isVisibleSubscription))
const usableSubscriptions = computed(() => visibleSubscriptions.value.filter(subscription => subscription.is_usable === true))
const orderedUsableSubscriptions = computed(() => [...usableSubscriptions.value].sort(compareSubscriptionRecency))
const exhaustedSubscriptions = computed(() => visibleSubscriptions.value.filter(subscription => subscription.exhausted_at && subscription.is_usable !== true))

const resetWeeklyLimitConfirmMessage = computed(() => {
  const remaining = weeklyLimitSubscription.value?.weekly_limit_reset_remaining ?? 0
  return t('userSubscriptions.resetWeeklyLimitConfirm', { remaining })
})

async function loadSubscriptions() {
  try {
    loading.value = true
    subscriptions.value = await subscriptionsAPI.getMySubscriptions()
  } catch (error) {
    console.error('Failed to load subscriptions:', error)
    const code = extractApiErrorCode(error)
    appStore.showError(code && subscriptionCreditErrorMessages[code]
      ? subscriptionCreditErrorMessages[code]
      : extractApiErrorMessage(error, t('userSubscriptions.failedToLoad')))
  } finally {
    loading.value = false
  }
}

function canShowResetWeeklyLimit(subscription: UserSubscription): boolean {
  return subscription.is_usable === true && subscription.weekly_limit_usd != null
}

function isResetWeeklyLimitDisabled(subscription: UserSubscription): boolean {
  return (subscription.weekly_limit_reset_remaining ?? 0) <= 0
    || (resettingWeeklyLimit.value && weeklyLimitSubscription.value?.id === subscription.id)
}

function handleResetWeeklyLimit(subscription: UserSubscription) {
  if (isResetWeeklyLimitDisabled(subscription)) return
  weeklyLimitSubscription.value = subscription
  showResetWeeklyLimitConfirm.value = true
}

function closeResetWeeklyLimitDialog() {
  if (resettingWeeklyLimit.value) return
  showResetWeeklyLimitConfirm.value = false
  weeklyLimitSubscription.value = null
}

async function confirmResetWeeklyLimit() {
  if (!weeklyLimitSubscription.value || resettingWeeklyLimit.value) return

  resettingWeeklyLimit.value = true
  try {
    await subscriptionsAPI.resetWeeklyLimit(weeklyLimitSubscription.value.id)
    appStore.showSuccess(t('userSubscriptions.weeklyLimitResetSuccess'))
    showResetWeeklyLimitConfirm.value = false
    weeklyLimitSubscription.value = null
    await loadSubscriptions()
  } catch (error) {
    console.error('Failed to reset weekly limit:', error)
    const code = extractApiErrorCode(error)
    appStore.showError(code && subscriptionCreditErrorMessages[code]
      ? subscriptionCreditErrorMessages[code]
      : extractApiErrorMessage(error, t('userSubscriptions.failedToResetWeeklyLimit')))
  } finally {
    resettingWeeklyLimit.value = false
  }
}

function isVisibleSubscription(subscription: UserSubscription): boolean {
  if (subscription.status !== 'active') return false
  if (!subscription.expires_at) return true
  return new Date(subscription.expires_at).getTime() > Date.now()
}

function subscriptionName(subscription: UserSubscription): string {
  return subscriptionDisplayName(subscription, t)
}

function compareSubscriptionRecency(a: UserSubscription, b: UserSubscription): number {
  const diff = subscriptionCreatedAt(a) - subscriptionCreatedAt(b)
  return diff !== 0 ? diff : a.id - b.id
}

function subscriptionCreatedAt(subscription: UserSubscription): number {
  const timestamp = Date.parse(subscription.created_at || '')
  return Number.isFinite(timestamp) ? timestamp : 0
}

function quotaLimit(subscription: UserSubscription): number {
  return subscription.quota_limit_usd ?? subscription.group?.monthly_limit_usd ?? 0
}

function quotaUsed(subscription: UserSubscription): number {
  return subscription.quota_used_usd ?? subscription.monthly_usage_usd ?? 0
}

function quotaRemaining(subscription: UserSubscription): number {
  const remaining = subscription.quota_remaining_usd ?? (quotaLimit(subscription) - quotaUsed(subscription))
  return Math.max(remaining, 0)
}

function scopeSummary(subscription: UserSubscription): string {
  const scopeType = subscription.scope_type || 'group'
  if (scopeType === 'group') {
    return subscription.group?.name || (subscription.group_id != null
      ? t('payment.groupFallback', { id: subscription.group_id })
      : t('userSubscriptions.groupScopeFallback'))
  }
  if (scopeType === 'all_available_groups') return t('userSubscriptions.allAvailableGroups')
  if (scopeType === 'selected_groups') {
    const groups = Array.isArray(subscription.scope_config?.group_ids) ? subscription.scope_config.group_ids : []
    return t('payment.planCard.scopeSelectedGroups', { count: groups.length })
  }
  if (scopeType === 'platforms') {
    const platforms = Array.isArray(subscription.scope_config?.platforms) ? subscription.scope_config.platforms : []
    return platforms.length > 0
      ? platforms.map(platform => String(platform)).join(' / ')
      : t('payment.planCard.scopePlatforms')
  }
  return scopeType.split('_').join(' ')
}

function getProgressWidth(used: number | undefined, limit: number | null | undefined): string {
  if (!limit || limit <= 0) return '0%'
  const percentage = Math.min(((used || 0) / limit) * 100, 100)
  return `${percentage}%`
}

function getProgressBarClass(used: number | undefined, limit: number | null | undefined): string {
  if (!limit || limit <= 0) return 'bg-gray-400'
  const percentage = ((used || 0) / limit) * 100
  if (percentage >= 90) return 'bg-red-500'
  if (percentage >= 70) return 'bg-orange-500'
  return 'bg-green-500'
}

function expirationDisplay(expiresAt: string | null | undefined): string {
  if (!expiresAt) return t('userSubscriptions.noExpiration')
  const now = new Date()
  const expires = new Date(expiresAt)
  const diff = expires.getTime() - now.getTime()
  const days = Math.ceil(diff / (1000 * 60 * 60 * 24))
  if (days < 0) return t('userSubscriptions.status.expired')
  const dateStr = formatDateTimeToMinute(expires)
  if (days === 0) return `${dateStr} (${t('common.today')})`
  if (days === 1) return `${dateStr} (${t('common.tomorrow')})`
  return t('userSubscriptions.daysRemainingWithDate', { days, date: dateStr })
}

function getExpirationClass(expiresAt: string | null | undefined): string {
  if (!expiresAt) return 'text-gray-700 dark:text-gray-300'
  const days = Math.ceil((new Date(expiresAt).getTime() - Date.now()) / (1000 * 60 * 60 * 24))
  if (days <= 0) return 'text-red-600 dark:text-red-400 font-medium'
  if (days <= 3) return 'text-red-600 dark:text-red-400'
  if (days <= 7) return 'text-orange-600 dark:text-orange-400'
  return 'text-gray-700 dark:text-gray-300'
}

function formatUsd(value: number | null | undefined): string {
  return Number((value || 0).toFixed(4)).toString()
}

function formatDateTimeValue(value: string): string {
  return formatDateTime(new Date(value))
}

function formatResetLabel(resetAt: string): string {
  if (!resetAt) return t('userSubscriptions.windowNotActive')
  return t('userSubscriptions.resetAt', { time: formatDateTimeValue(resetAt) })
}

function weeklyResetHint(subscription: UserSubscription): string {
  if (!subscription.weekly_reset_at) {
    return t('userSubscriptions.weeklyResetHint', { time: t('userSubscriptions.windowNotActive') })
  }
  return t('userSubscriptions.weeklyResetHint', { time: formatDateTimeValue(subscription.weekly_reset_at) })
}

onMounted(() => {
  loadSubscriptions()
})
</script>
