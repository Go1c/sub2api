<template>
  <div
    :class="[
      'group relative flex flex-col overflow-hidden rounded-2xl border transition-all',
      'hover:shadow-xl hover:-translate-y-0.5',
      borderClass,
      'bg-white dark:bg-dark-800',
    ]"
  >
    <!-- Colored top accent bar -->
    <div :class="['h-1.5', accentClass]" />

    <div class="flex flex-1 flex-col p-4">
      <!-- Header: name + badge + price -->
      <div class="mb-3 flex items-start justify-between gap-2">
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <h3 class="truncate text-base font-bold text-gray-900 dark:text-white">{{ plan.name }}</h3>
            <span :class="['shrink-0 rounded-full px-2 py-0.5 text-[11px] font-medium', badgeLightClass]">
              {{ pLabel }}
            </span>
          </div>
          <p v-if="plan.description" class="mt-0.5 text-xs leading-relaxed text-gray-500 dark:text-dark-400 line-clamp-2">
            {{ plan.description }}
          </p>
        </div>
        <div class="shrink-0 text-right">
          <div class="flex items-baseline gap-1">
            <span class="text-xs text-gray-400 dark:text-dark-500">$</span>
            <span :class="['text-2xl font-extrabold tracking-tight', textClass]">{{ plan.price }}</span>
            <span v-if="plan.currency" class="text-xs font-medium text-gray-400 dark:text-dark-500">{{ plan.currency }}</span>
          </div>
          <span class="text-[11px] text-gray-400 dark:text-dark-500">/ {{ validitySuffix }}</span>
          <div v-if="plan.original_price" class="mt-0.5 flex items-center justify-end gap-1.5">
            <span class="text-xs text-gray-400 line-through dark:text-dark-500">${{ plan.original_price }}<template v-if="plan.currency"> {{ plan.currency }}</template></span>
            <span :class="['rounded px-1 py-0.5 text-[10px] font-semibold', discountClass]">{{ discountText }}</span>
          </div>
        </div>
      </div>

      <!-- Credit pool info (compact) -->
      <div class="mb-3 space-y-2 rounded-lg bg-gray-50 px-3 py-2 text-xs dark:bg-dark-700/50">
        <div class="flex items-center justify-between">
          <span class="text-gray-400 dark:text-dark-500">{{ t('payment.planCard.quota') }}</span>
          <span :class="['text-base font-extrabold', textClass]">{{ creditDisplay }}</span>
        </div>
        <div class="grid grid-cols-2 gap-x-3 gap-y-1">
          <div class="flex items-center justify-between">
            <span class="text-gray-400 dark:text-dark-500">{{ t('payment.planCard.validity') }}</span>
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ validitySuffix }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-gray-400 dark:text-dark-500">{{ t('payment.planCard.scope') }}</span>
            <span :class="['rounded-full px-1.5 py-0.5 text-[10px] font-semibold', badgeLightClass]">{{ scopeLabel }}</span>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-x-3 gap-y-1">
        <div class="flex items-center justify-between">
          <span class="text-gray-400 dark:text-dark-500">{{ t('payment.planCard.rate') }}</span>
          <span class="font-medium text-gray-700 dark:text-gray-300">{{ rateDisplay }}</span>
        </div>
        <div v-if="plan.daily_limit_usd != null" class="flex items-center justify-between">
          <span class="text-gray-400 dark:text-dark-500">{{ t('payment.planCard.dailyLimit') }}</span>
          <span class="font-medium text-gray-700 dark:text-gray-300">${{ plan.daily_limit_usd }}</span>
        </div>
        <div v-if="plan.weekly_limit_usd != null" class="flex items-center justify-between">
          <span class="text-gray-400 dark:text-dark-500">{{ t('payment.planCard.weeklyLimit') }}</span>
          <span class="font-medium text-gray-700 dark:text-gray-300">${{ plan.weekly_limit_usd }}</span>
        </div>
        <div v-if="plan.daily_limit_usd == null && plan.weekly_limit_usd == null" class="flex items-center justify-between">
          <span class="text-gray-400 dark:text-dark-500">{{ t('payment.planCard.dailyLimit') }}</span>
          <span class="font-medium text-gray-700 dark:text-gray-300">{{ t('payment.planCard.unlimited') }}</span>
        </div>
        </div>
        <div v-if="modelScopeLabels.length > 0" class="col-span-2 flex items-center justify-between">
          <span class="text-gray-400 dark:text-dark-500">{{ t('payment.planCard.models') }}</span>
          <div class="flex flex-wrap justify-end gap-1">
            <span v-for="scope in modelScopeLabels" :key="scope"
              class="rounded bg-gray-200/80 px-1.5 py-0.5 text-[10px] font-medium text-gray-600 dark:bg-dark-600 dark:text-gray-300">
              {{ scope }}
            </span>
          </div>
        </div>
      </div>

      <!-- Features list (compact) -->
      <div v-if="plan.features.length > 0" class="mb-3 space-y-1">
        <div v-for="feature in plan.features" :key="feature" class="flex items-start gap-1.5">
          <svg :class="['mt-0.5 h-3.5 w-3.5 flex-shrink-0', iconClass]" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
          </svg>
          <span class="text-xs text-gray-600 dark:text-gray-300">{{ feature }}</span>
        </div>
      </div>

      <div class="flex-1" />

      <!-- Subscribe Button -->
      <button
        type="button"
        :class="['w-full rounded-xl py-2.5 text-sm font-semibold transition-all active:scale-[0.98]', btnClass]"
        @click="emit('select', plan)"
      >
        {{ isRenewal ? t('payment.renewNow') : t('payment.subscribeNow') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubscriptionPlan } from '@/types/payment'
import type { UserSubscription } from '@/types'
import {
  platformAccentBarClass,
  platformBadgeLightClass,
  platformBorderClass,
  platformTextClass,
  platformIconClass,
  platformButtonClass,
  platformDiscountClass,
  platformLabel,
} from '@/utils/platformColors'

const props = defineProps<{ plan: SubscriptionPlan; activeSubscriptions?: UserSubscription[] }>()
const emit = defineEmits<{ select: [plan: SubscriptionPlan] }>()
const { t } = useI18n()

const platform = computed(() => props.plan.group_platform || '')
const isRenewal = computed(() =>
  props.activeSubscriptions?.some(s => {
    if (s.is_usable !== true) return false
    if (s.plan_id != null) return s.plan_id === props.plan.id
    return normalizeScopeType(s.scope_type) === normalizeScopeType(props.plan.scope_type)
  }) ?? false
)

// Derived color classes from central config
const accentClass = computed(() => platformAccentBarClass(platform.value))
const borderClass = computed(() => platformBorderClass(platform.value))
const badgeLightClass = computed(() => platformBadgeLightClass(platform.value))
const textClass = computed(() => platformTextClass(platform.value))
const iconClass = computed(() => platformIconClass(platform.value))
const btnClass = computed(() => platformButtonClass(platform.value))
const discountClass = computed(() => platformDiscountClass(platform.value))
const pLabel = computed(() => platformLabel(platform.value))
const creditDisplay = computed(() => {
  const quota = props.plan.quota_usd
  return quota != null && quota > 0 ? `$${formatUsd(quota)}` : t('payment.planCard.unlimited')
})

const scopeLabel = computed(() => {
  const type = props.plan.scope_type || 'all_available_groups'
  if (type === 'group') return props.plan.group_name || t('payment.groupFallback', { id: props.plan.group_id ?? '-' })
  if (type === 'all_available_groups') return t('payment.planCard.scopeAllAvailable')
  if (type === 'selected_groups') {
    const groups = Array.isArray(props.plan.scope_config?.group_ids) ? props.plan.scope_config.group_ids : []
    return t('payment.planCard.scopeSelectedGroups', { count: groups.length })
  }
  if (type === 'platforms') {
    const platforms = Array.isArray(props.plan.scope_config?.platforms) ? props.plan.scope_config.platforms : []
    return platforms.length > 0
      ? platforms.map(platform => String(platform)).join(' / ')
      : t('payment.planCard.scopePlatforms')
  }
  return type.split('_').join(' ')
})

const discountText = computed(() => {
  if (!props.plan.original_price || props.plan.original_price <= 0) return ''
  const pct = Math.round((1 - props.plan.price / props.plan.original_price) * 100)
  return pct > 0 ? `-${pct}%` : ''
})

const rateDisplay = computed(() => {
  const rate = props.plan.rate_multiplier ?? 1
  return `×${Number(rate.toPrecision(10))}`
})

const MODEL_SCOPE_LABELS: Record<string, string> = {
  claude: 'Claude',
  gemini_text: 'Gemini',
  gemini_image: 'Imagen',
}

const modelScopeLabels = computed(() => {
  const scopes = props.plan.supported_model_scopes
  if (!scopes || scopes.length === 0) return []
  return scopes.map(s => MODEL_SCOPE_LABELS[s] || s)
})

const validitySuffix = computed(() => {
  const unit = (props.plan.validity_unit || 'day').toLowerCase()
  if (unit === 'month' || unit === 'months') return t('payment.perMonth')
  if (unit === 'year' || unit === 'years') return t('payment.perYear')
  if (unit === 'week' || unit === 'weeks') return `${props.plan.validity_days}${t('payment.weeks')}`
  return `${props.plan.validity_days}${t('payment.days')}`
})

function formatUsd(value: number): string {
  return Number(value.toFixed(4)).toString()
}

function normalizeScopeType(scopeType?: string | null): string {
  return scopeType || 'all_available_groups'
}
</script>
