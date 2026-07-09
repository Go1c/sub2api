<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
        ></div>
      </div>

      <template v-else-if="detail">
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div class="card p-5">
            <p class="flex items-center gap-1.5 text-sm text-gray-500 dark:text-dark-400">
              <Icon name="dollar" size="sm" class="text-primary-500" />
              {{ t('affiliate.stats.rebateRate') }}
            </p>
            <p class="mt-2 text-2xl font-semibold text-primary-600 dark:text-primary-400">
              {{ hasEffectiveRebateRate ? formattedRebateRate : t('affiliate.tiers.unconfigured') }}<span v-if="hasEffectiveRebateRate" class="ml-0.5 text-base font-medium">%</span>
            </p>
            <p class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              {{ t('affiliate.stats.rebateRateHint') }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.invitedUsers') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCount(detail.aff_count) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.availableQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-emerald-600 dark:text-emerald-400">
              {{ formatCurrency(detail.aff_quota) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.totalQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCurrency(detail.aff_history_quota) }}
            </p>
            <p v-if="detail.aff_frozen_quota > 0" class="mt-1 text-xs text-amber-600 dark:text-amber-400">
              {{ t('affiliate.stats.frozenQuota') }}: {{ formatCurrency(detail.aff_frozen_quota) }}
            </p>
          </div>
        </div>

        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.title') }}</h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.description') }}</p>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.yourCode') }}</p>
              <div class="flex items-center gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900">
                <code class="flex-1 truncate text-sm font-semibold text-gray-900 dark:text-white">{{ detail.aff_code }}</code>
                <button class="btn btn-secondary btn-sm" @click="copyCode">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyCode') }}</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.inviteLink') }}</p>
              <div class="flex items-center gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900">
                <code class="flex-1 truncate text-sm text-gray-700 dark:text-gray-300">{{ inviteLink }}</code>
                <button class="btn btn-secondary btn-sm" @click="copyInviteLink">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyLink') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="mt-5 rounded-xl border border-primary-200 bg-primary-50 p-4 dark:border-primary-900/40 dark:bg-primary-900/20">
            <p class="text-sm font-medium text-primary-800 dark:text-primary-200">{{ t('affiliate.tips.title') }}</p>
            <ul class="mt-2 space-y-1 text-sm text-primary-700 dark:text-primary-300">
              <li>1. {{ t('affiliate.tips.line1') }}</li>
              <li>2. {{ t('affiliate.tips.line2', { rate: effectiveRebateRateLabel }) }}</li>
              <li>3. {{ t('affiliate.tips.line3') }}</li>
              <li v-if="detail.aff_frozen_quota > 0">4. {{ t('affiliate.tips.line4') }}</li>
            </ul>
          </div>
        </div>

        <div class="card overflow-hidden">
          <div class="border-b border-gray-100 px-6 py-5 dark:border-dark-700">
            <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <p class="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-primary-600 dark:text-primary-300">
                  <Icon name="trendingUp" size="sm" />
                  {{ t('affiliate.tiers.eyebrow') }}
                </p>
                <h3 class="mt-2 text-base font-semibold text-gray-900 dark:text-white">
                  {{ t('affiliate.tiers.title') }}
                </h3>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                  {{ t('affiliate.tiers.description') }}
                </p>
                <p class="mt-2 text-xs text-gray-400 dark:text-dark-500">
                  {{ t('affiliate.tiers.configNote') }}
                </p>
              </div>
              <div class="rounded-xl border border-primary-200 bg-primary-50 px-4 py-3 dark:border-primary-900/50 dark:bg-primary-900/20">
                <p class="text-xs font-medium text-primary-700 dark:text-primary-300">
                  {{ t('affiliate.tiers.currentLevel') }}
                </p>
                <p class="mt-1 flex items-baseline gap-2">
                  <span class="text-2xl font-semibold text-primary-700 dark:text-primary-200">
                    {{ currentAffiliateTier?.level ?? t('affiliate.tiers.noLevel') }}
                  </span>
                  <span class="text-sm font-medium text-primary-600 dark:text-primary-300">
                    {{ effectiveRebateRateLabel }}
                  </span>
                </p>
              </div>
            </div>
          </div>

          <div class="grid gap-5 p-6 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
            <div class="rounded-xl border border-gray-200 bg-gray-50 p-5 dark:border-dark-700 dark:bg-dark-900/50">
              <div class="flex items-start justify-between gap-4">
                <div>
                  <p class="text-sm text-gray-500 dark:text-dark-400">
                    {{ t('affiliate.tiers.previewData') }}
                  </p>
                  <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">
                    {{ currentAffiliateTier?.level ?? t('affiliate.tiers.noLevel') }} · {{ effectiveRebateRateLabel }}
                  </p>
                </div>
              </div>

              <dl class="mt-5 grid gap-3 sm:grid-cols-2">
                <div class="rounded-lg bg-white p-3 dark:bg-dark-800/80">
                  <dt class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400">
                    <Icon name="users" size="xs" />
                    {{ t('affiliate.tiers.inviteCount') }}
                  </dt>
                  <dd class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                    {{ formatCount(detail.aff_count) }}
                  </dd>
                </div>
                <div class="rounded-lg bg-white p-3 dark:bg-dark-800/80">
                  <dt class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400">
                    <Icon name="creditCard" size="xs" />
                    {{ t('affiliate.tiers.rechargeTotal') }}
                  </dt>
                  <dd class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                    {{ formatCnyAmount(inviteeRechargeTotal) }}
                  </dd>
                </div>
              </dl>

              <div class="mt-5 space-y-4">
                <div>
                  <div class="mb-1.5 flex items-center justify-between gap-3 text-xs">
                    <span class="font-medium text-gray-600 dark:text-gray-300">
                      {{ t('affiliate.tiers.nextInviteProgress') }}
                    </span>
                    <span class="text-gray-500 dark:text-dark-400">
                      {{ nextAffiliateTier ? `${formatCount(detail.aff_count)} / ${formatCount(nextAffiliateTier.minInvitees)}` : t('affiliate.tiers.maxLevel') }}
                    </span>
                  </div>
                  <div class="h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
                    <div
                      class="h-full rounded-full bg-primary-500 transition-all"
                      :style="{ width: `${nextInviteeProgressPercent}%` }"
                    ></div>
                  </div>
                </div>
                <div>
                  <div class="mb-1.5 flex items-center justify-between gap-3 text-xs">
                    <span class="font-medium text-gray-600 dark:text-gray-300">
                      {{ t('affiliate.tiers.nextRechargeProgress') }}
                    </span>
                    <span class="text-gray-500 dark:text-dark-400">
                      {{ nextAffiliateTier ? `${formatCnyAmount(inviteeRechargeTotal)} / ${formatCnyAmount(nextAffiliateTier.minRecharge)}` : t('affiliate.tiers.maxLevel') }}
                    </span>
                  </div>
                  <div class="h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
                    <div
                      class="h-full rounded-full bg-emerald-500 transition-all"
                      :style="{ width: `${nextRechargeProgressPercent}%` }"
                    ></div>
                  </div>
                </div>
              </div>

              <p class="mt-4 rounded-lg bg-white px-3 py-2 text-sm text-gray-600 dark:bg-dark-800/80 dark:text-gray-300">
                {{ nextTierHint }}
              </p>
            </div>

            <div class="grid gap-3 sm:grid-cols-2 2xl:grid-cols-4">
              <div
                v-for="tier in affiliateTiers"
                :key="tier.level"
                :class="[
                  'rounded-xl border p-4',
                  currentAffiliateTier && tier.level === currentAffiliateTier.level
                    ? 'border-primary-300 bg-primary-50 shadow-sm dark:border-primary-700/70 dark:bg-primary-900/20'
                    : affiliateTierReached(tier)
                      ? 'border-emerald-200 bg-emerald-50/60 dark:border-emerald-900/40 dark:bg-emerald-900/10'
                      : 'border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900/40'
                ]"
              >
                <div class="flex items-center justify-between gap-3">
                  <div class="flex items-center gap-2">
                    <span
                      :class="[
                        'flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold',
                        currentAffiliateTier && tier.level === currentAffiliateTier.level
                          ? 'bg-primary-600 text-white'
                          : affiliateTierReached(tier)
                            ? 'bg-emerald-600 text-white'
                            : 'bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-dark-300'
                      ]"
                    >
                      {{ tier.level }}
                    </span>
                    <span v-if="currentAffiliateTier && tier.level === currentAffiliateTier.level" class="text-xs font-medium text-primary-700 dark:text-primary-300">
                      {{ t('affiliate.tiers.active') }}
                    </span>
                  </div>
                  <Icon
                    v-if="affiliateTierReached(tier)"
                    name="checkCircle"
                    size="sm"
                    class="text-emerald-500"
                  />
                </div>
                <dl class="mt-4 space-y-2 text-sm">
                  <div class="flex items-center justify-between gap-3">
                    <dt class="text-gray-500 dark:text-dark-400">{{ t('affiliate.tiers.inviteThreshold') }}</dt>
                    <dd class="font-medium text-gray-900 dark:text-white">
                      {{ tier.minInvitees === 0 ? t('common.none') : formatCount(tier.minInvitees) }}
                    </dd>
                  </div>
                  <div class="flex items-center justify-between gap-3">
                    <dt class="text-gray-500 dark:text-dark-400">{{ t('affiliate.tiers.rechargeThreshold') }}</dt>
                    <dd class="font-medium text-gray-900 dark:text-white">
                      {{ tier.minRecharge === 0 ? t('common.none') : formatCnyAmount(tier.minRecharge) }}
                    </dd>
                  </div>
                  <div class="flex items-center justify-between gap-3 border-t border-gray-100 pt-2 dark:border-dark-700">
                    <dt class="text-gray-500 dark:text-dark-400">{{ t('affiliate.tiers.shareRate') }}</dt>
                    <dd class="text-lg font-semibold text-primary-600 dark:text-primary-300">
                      {{ formatPercent(tier.ratePercent) }}
                    </dd>
                  </div>
                </dl>
              </div>
            </div>
          </div>
        </div>

        <div class="card p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.transfer.title') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.transfer.description') }}</p>
            </div>
            <button
              class="btn btn-primary"
              :disabled="transferring || detail.aff_quota <= 0"
              @click="transferQuota"
            >
              <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="dollar" size="sm" />
              <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
            </button>
          </div>
          <p v-if="detail.aff_quota <= 0" class="mt-3 text-sm text-amber-600 dark:text-amber-400">
            {{ t('affiliate.transfer.empty') }}
          </p>
        </div>

        <div class="card p-6">
          <div class="flex items-center justify-between gap-3">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.logs.title') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.logs.description') }}</p>
            </div>
            <button class="btn btn-secondary btn-sm" :disabled="logsLoading" @click="loadAffiliateInviteLogs">
              <Icon v-if="logsLoading" name="refresh" size="sm" class="animate-spin" />
              <span>{{ t('common.refresh') }}</span>
            </button>
          </div>
          <div v-if="logs.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('affiliate.logs.empty') }}
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="w-full min-w-[640px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.logs.columns.time') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.logs.columns.account') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.logs.columns.code') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.logs.columns.result') }}</th>
                  <th class="px-3 py-2 text-right font-medium">{{ t('affiliate.logs.columns.bonus') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="entry in logs" :key="entry.id" class="border-b border-gray-100 last:border-b-0 dark:border-dark-800">
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ formatDateTime(entry.created_at) || '-' }}</td>
                  <td class="px-3 py-3 text-gray-900 dark:text-white">
                    {{ entry.invitee_email || entry.inviter_email || `#${entry.invitee_id || entry.inviter_id || '-'}` }}
                  </td>
                  <td class="px-3 py-3 font-mono text-gray-700 dark:text-gray-300">{{ entry.affiliate_code || '-' }}</td>
                  <td class="px-3 py-3">
                    <span :class="entry.success && !entry.failure_reason ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">
                      {{ entry.failure_message || (entry.success ? t('common.success') : entry.failure_reason) || '-' }}
                    </span>
                  </td>
                  <td class="px-3 py-3 text-right font-medium text-emerald-600 dark:text-emerald-400">
                    {{ entry.bonus_amount > 0 ? formatCurrency(entry.bonus_amount) : '-' }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-if="logsTotal > logsPageSize" class="mt-4 flex items-center justify-between text-sm text-gray-500">
            <span>{{ t('affiliate.logs.total', { total: logsTotal }) }}</span>
            <div class="flex items-center gap-2">
              <button class="btn btn-secondary btn-sm" :disabled="logsPage <= 1" @click="changeLogPage(logsPage - 1)">
                {{ t('pagination.previous') }}
              </button>
              <span>{{ logsPage }} / {{ Math.max(1, Math.ceil(logsTotal / logsPageSize)) }}</span>
              <button class="btn btn-secondary btn-sm" :disabled="logsPage >= Math.ceil(logsTotal / logsPageSize)" @click="changeLogPage(logsPage + 1)">
                {{ t('pagination.next') }}
              </button>
            </div>
          </div>
        </div>

        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.invitees.title') }}</h3>
          <div v-if="detail.invitees.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('affiliate.invitees.empty') }}
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="w-full min-w-[560px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.email') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="px-3 py-2 font-medium text-right">{{ t('affiliate.invitees.columns.rebate') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.joinedAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in detail.invitees"
                  :key="item.user_id"
                  class="border-b border-gray-100 last:border-b-0 dark:border-dark-800"
                >
                  <td class="px-3 py-3 text-gray-900 dark:text-white">{{ item.email || '-' }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ item.username || '-' }}</td>
                  <td class="px-3 py-3 text-right font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import userAPI from '@/api/user'
import type { AffiliateInviteLog, AffiliateRebateTier, UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t, locale } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const transferring = ref(false)
const detail = ref<UserAffiliateDetail | null>(null)
const logsLoading = ref(false)
const logs = ref<AffiliateInviteLog[]>([])
const logsTotal = ref(0)
const logsPage = ref(1)
const logsPageSize = 10

interface AffiliateTierRow {
  level: string
  minInvitees: number
  minRecharge: number
  ratePercent: number | null
}

const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})

// Rebate rate is a percentage in the range [0, 100]; backend already clamps it.
// We trim trailing zeros (e.g. 20.00 → "20", 12.50 → "12.5") for a cleaner UI.
const hasEffectiveRebateRate = computed(() => typeof detail.value?.effective_rebate_rate_percent === 'number')

const formattedRebateRate = computed(() => {
  const v = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(v * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

const effectiveRebateRateLabel = computed(() => {
  return hasEffectiveRebateRate.value ? `${formattedRebateRate.value}%` : t('affiliate.tiers.unconfigured')
})

const inviteeRechargeTotal = computed(() => detail.value?.invitee_recharge_total ?? 0)

const affiliateTiers = computed<AffiliateTierRow[]>(() => {
  return (detail.value?.affiliate_tiers ?? []).map(toAffiliateTierRow)
})

const currentAffiliateTier = computed<AffiliateTierRow | null>(() => {
  return detail.value?.current_affiliate_tier ? toAffiliateTierRow(detail.value.current_affiliate_tier) : null
})

const nextAffiliateTier = computed<AffiliateTierRow | null>(() => {
  return detail.value?.next_affiliate_tier ? toAffiliateTierRow(detail.value.next_affiliate_tier) : null
})

const nextInviteeProgressPercent = computed(() => {
  if (!nextAffiliateTier.value) return 100
  return progressPercent(detail.value?.aff_count ?? 0, nextAffiliateTier.value.minInvitees)
})

const nextRechargeProgressPercent = computed(() => {
  if (!nextAffiliateTier.value) return 100
  return progressPercent(inviteeRechargeTotal.value, nextAffiliateTier.value.minRecharge)
})

const nextTierHint = computed(() => {
  const next = nextAffiliateTier.value
  if (!next) return t('affiliate.tiers.maxLevelHint')

  const missingInvitees = Math.max(0, next.minInvitees - (detail.value?.aff_count ?? 0))
  const missingRecharge = Math.max(0, next.minRecharge - inviteeRechargeTotal.value)
  if (missingInvitees === 0 && missingRecharge === 0) {
    return t('affiliate.tiers.readyForNext', { level: next.level })
  }
  if (missingInvitees === 0) {
    return t('affiliate.tiers.nextHintRechargeOnly', {
      level: next.level,
      amount: formatCnyAmount(missingRecharge),
    })
  }
  if (missingRecharge === 0) {
    return t('affiliate.tiers.nextHintInviteOnly', {
      level: next.level,
      count: formatCount(missingInvitees),
    })
  }
  return t('affiliate.tiers.nextHintBoth', {
    level: next.level,
    count: formatCount(missingInvitees),
    amount: formatCnyAmount(missingRecharge),
  })
})

function formatCount(value: number): string {
  return value.toLocaleString()
}

function toAffiliateTierRow(tier: AffiliateRebateTier): AffiliateTierRow {
  return {
    level: tier.level,
    minInvitees: Math.max(0, Number(tier.min_invitees) || 0),
    minRecharge: Math.max(0, Number(tier.min_recharge) || 0),
    ratePercent: typeof tier.rebate_rate_percent === 'number' ? tier.rebate_rate_percent : null,
  }
}

function formatPercent(value: number | null | undefined): string {
  if (typeof value !== 'number') return t('affiliate.tiers.unconfigured')
  const rounded = Math.round(value * 100) / 100
  return `${Number.isInteger(rounded) ? rounded.toString() : rounded.toString()}%`
}

function formatCnyAmount(value: number): string {
  return new Intl.NumberFormat(locale.value, {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: value % 1 === 0 ? 0 : 2,
    maximumFractionDigits: 2,
  }).format(value)
}

function progressPercent(current: number, target: number): number {
  if (target <= 0) return 100
  return Math.min(100, Math.max(0, Math.round((current / target) * 100)))
}

function affiliateTierReached(tier: AffiliateTierRow): boolean {
  if (typeof tier.ratePercent !== 'number') return false
  const invitees = detail.value?.aff_count ?? 0
  return invitees >= tier.minInvitees && inviteeRechargeTotal.value >= tier.minRecharge
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (!silent) {
    loading.value = true
  }
  try {
    detail.value = await userAPI.getAffiliateDetail()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.loadFailed')))
  } finally {
    if (!silent) {
      loading.value = false
    }
  }
}

async function loadAffiliateInviteLogs(): Promise<void> {
  logsLoading.value = true
  try {
    const resp = await userAPI.getAffiliateInviteLogs({
      page: logsPage.value,
      page_size: logsPageSize,
    })
    logs.value = resp.items ?? []
    logsTotal.value = resp.total ?? 0
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.logs.loadFailed')))
  } finally {
    logsLoading.value = false
  }
}

function changeLogPage(page: number): void {
  if (page < 1) return
  logsPage.value = page
  void loadAffiliateInviteLogs()
}

async function copyCode(): Promise<void> {
  if (!detail.value?.aff_code) return
  await copyToClipboard(detail.value.aff_code, t('affiliate.codeCopied'))
}

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, t('affiliate.linkCopied'))
}

async function transferQuota(): Promise<void> {
  if (!detail.value || detail.value.aff_quota <= 0 || transferring.value) return
  transferring.value = true
  try {
    const resp = await userAPI.transferAffiliateQuota()
    appStore.showSuccess(t('affiliate.transfer.success', { amount: formatCurrency(resp.transferred_quota) }))
    await Promise.all([
      loadAffiliateDetail(true),
      authStore.refreshUser().catch(() => undefined),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.transferFailed')))
  } finally {
    transferring.value = false
  }
}

onMounted(() => {
  void loadAffiliateDetail()
  void loadAffiliateInviteLogs()
})
</script>
