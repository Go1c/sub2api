<template>
  <BaseDialog :show="show" :title="plan ? t('payment.admin.editPlan') : t('payment.admin.createPlan')" width="wide" @close="emit('close')">
    <form id="plan-form" @submit.prevent="handleSavePlan" class="space-y-4">
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="input-label">{{ t('payment.admin.planName') }} <span class="text-red-500">*</span></label>
          <input v-model="planForm.name" type="text" class="input" required />
        </div>
        <div>
          <label class="input-label">{{ t('payment.admin.group') }}</label>
          <Select v-model="planForm.group_id" :options="groupOptions" :placeholder="t('payment.admin.selectGroup')" class="w-full">
            <template #selected="{ option }">
              <span v-if="option?.platform" :class="platformTextClass(String(option.platform))">{{ option.label }}</span>
              <span v-else>{{ option?.label || t('payment.admin.selectGroup') }}</span>
            </template>
            <template #option="{ option, selected }">
              <span class="flex-1 truncate text-left" :class="option.platform ? platformTextClass(String(option.platform)) : ''">{{ option.label }}</span>
              <Icon v-if="selected" name="check" size="sm" class="text-primary-500" :stroke-width="2" />
            </template>
          </Select>
        </div>
      </div>

      <!-- Group Info Preview -->
      <div v-if="selectedGroupInfo" class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-800">
        <div class="mb-2 flex items-center gap-2">
          <GroupBadge :name="selectedGroupInfo.name" :platform="selectedGroupInfo.platform" :rate-multiplier="selectedGroupInfo.rate_multiplier" />
        </div>
        <div class="grid grid-cols-2 gap-2 text-xs">
          <div><span class="text-gray-500">{{ t('payment.admin.dailyLimit') }}:</span> <span class="ml-1 font-medium text-gray-700 dark:text-gray-300">{{ selectedGroupInfo.daily_limit_usd != null ? '$' + selectedGroupInfo.daily_limit_usd : t('payment.admin.unlimited') }}</span></div>
          <div><span class="text-gray-500">{{ t('payment.admin.weeklyLimit') }}:</span> <span class="ml-1 font-medium text-gray-700 dark:text-gray-300">{{ selectedGroupInfo.weekly_limit_usd != null ? '$' + selectedGroupInfo.weekly_limit_usd : t('payment.admin.unlimited') }}</span></div>
          <div><span class="text-gray-500">{{ t('payment.admin.monthlyLimit') }}:</span> <span class="ml-1 font-medium text-gray-700 dark:text-gray-300">{{ selectedGroupInfo.monthly_limit_usd != null ? '$' + selectedGroupInfo.monthly_limit_usd : t('payment.admin.unlimited') }}</span></div>
        </div>
      </div>

      <div class="rounded-xl border border-blue-100 bg-blue-50/60 p-4 dark:border-blue-900/40 dark:bg-blue-950/20">
        <div class="mb-3 flex items-center justify-between gap-3">
          <div>
            <h3 class="text-sm font-semibold text-blue-900 dark:text-blue-100">{{ t('payment.admin.creditPoolSettings') }}</h3>
            <p class="text-xs text-blue-700/80 dark:text-blue-300/80">{{ t('payment.admin.creditPoolHint') }}</p>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">{{ t('payment.admin.quotaUsd') }} <span class="text-red-500">*</span></label>
            <input v-model.number="planForm.quota_usd" type="number" step="0.01" min="0.01" class="input" required />
          </div>
          <div>
            <label class="input-label">{{ t('payment.admin.scopeType') }} <span class="text-red-500">*</span></label>
            <Select v-model="planForm.scope_type" :options="scopeTypeOptions" class="w-full" />
          </div>
          <div>
            <label class="input-label">{{ t('payment.admin.dailyLimit') }}</label>
            <input v-model.number="planForm.daily_limit_usd" type="number" step="0.01" min="0" class="input" :placeholder="t('payment.admin.unlimited')" />
          </div>
          <div>
            <label class="input-label">{{ t('payment.admin.weeklyLimit') }}</label>
            <input v-model.number="planForm.weekly_limit_usd" type="number" step="0.01" min="0" class="input" :placeholder="t('payment.admin.unlimited')" />
          </div>
        </div>

        <div v-if="planForm.scope_type === 'selected_groups'" class="mt-4">
          <label class="input-label">{{ t('payment.admin.selectedGroups') }}</label>
          <div class="grid max-h-40 grid-cols-1 gap-2 overflow-auto rounded-lg border border-blue-100 bg-white p-2 dark:border-blue-900/40 dark:bg-dark-800 sm:grid-cols-2">
            <label v-for="group in subscriptionGroups" :key="group.id" class="flex items-center gap-2 rounded-md px-2 py-1 text-sm text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-dark-700">
              <input v-model="planForm.scope_group_ids" type="checkbox" :value="group.id" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              <span class="truncate">{{ group.name }} — {{ group.platform }}</span>
            </label>
          </div>
        </div>

        <div v-if="planForm.scope_type === 'platforms'" class="mt-4">
          <label class="input-label">{{ t('payment.admin.selectedPlatforms') }}</label>
          <div class="flex flex-wrap gap-2">
            <label v-for="platform in platformOptions" :key="platform.value" class="inline-flex items-center gap-2 rounded-full border border-blue-100 bg-white px-3 py-1.5 text-sm text-gray-700 dark:border-blue-900/40 dark:bg-dark-800 dark:text-gray-200">
              <input v-model="planForm.scope_platforms" type="checkbox" :value="platform.value" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              <span>{{ platform.label }}</span>
            </label>
          </div>
        </div>
      </div>

      <div><label class="input-label">{{ t('payment.admin.planDescription') }} <span class="text-red-500">*</span></label><textarea v-model="planForm.description" rows="2" class="input" required></textarea></div>
      <div>
        <label class="input-label">{{ t('payment.admin.purchaseNotice') }}</label>
        <textarea v-model="planForm.purchase_notice" rows="3" class="input" :placeholder="t('payment.admin.purchaseNoticePlaceholder')"></textarea>
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div><label class="input-label">{{ t('payment.admin.price') }} <span class="text-red-500">*</span></label><input v-model.number="planForm.price" type="number" step="0.01" min="0.01" class="input" required /></div>
        <div><label class="input-label">{{ t('payment.admin.originalPrice') }}</label><input v-model.number="planForm.original_price" type="number" step="0.01" min="0" class="input" /></div>
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div><label class="input-label">{{ t('payment.admin.validityDays') }} <span class="text-red-500">*</span></label><input v-model.number="planForm.validity_days" type="number" min="1" max="30" class="input" required /></div>
        <div><label class="input-label">{{ t('payment.admin.validityUnit') }} <span class="text-red-500">*</span></label><Select v-model="planForm.validity_unit" :options="validityUnitOptions" /></div>
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div><label class="input-label">{{ t('payment.admin.sortOrder') }}</label><input v-model.number="planForm.sort_order" type="number" min="0" class="input" /></div>
        <div>
          <label class="input-label">{{ t('payment.admin.currency') }}</label>
          <input v-model="planForm.currency" type="text" maxlength="3" class="input uppercase" :placeholder="t('payment.admin.currencyPlaceholder')" />
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('payment.admin.currencyHint') }}</p>
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('payment.admin.features') }}</label>
        <textarea v-model="planFeaturesText" rows="3" class="input" :placeholder="t('payment.admin.featuresPlaceholder')"></textarea>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('payment.admin.featuresHint') }}</p>
      </div>
      <div class="flex items-center gap-3">
        <label class="text-sm text-gray-700 dark:text-gray-300">{{ t('payment.admin.forSale') }}</label>
        <button
          type="button"
          :class="[
            'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
            planForm.for_sale ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
          ]"
          @click="planForm.for_sale = !planForm.for_sale"
        >
          <span :class="[
            'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
            planForm.for_sale ? 'translate-x-5' : 'translate-x-0'
          ]" />
        </button>
      </div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" @click="emit('close')" class="btn btn-secondary">{{ t('common.cancel') }}</button>
        <button type="submit" form="plan-form" :disabled="saving" class="btn btn-primary">{{ saving ? t('common.saving') : t('common.save') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminPaymentAPI } from '@/api/admin/payment'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { SubscriptionPlan } from '@/types/payment'
import type { AdminGroup } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import { platformTextClass } from '@/utils/platformColors'

const props = defineProps<{
  show: boolean
  plan: SubscriptionPlan | null
  groups: AdminGroup[]
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const { t } = useI18n()
const appStore = useAppStore()

const saving = ref(false)
const planForm = reactive({
  name: '',
  group_id: null as number | null,
  description: '',
  price: 0,
  original_price: 0,
  currency: '',
  quota_usd: 0,
  daily_limit_usd: null as number | null | '',
  weekly_limit_usd: null as number | null | '',
  validity_days: 30,
  validity_unit: 'day',
  scope_type: 'all_available_groups',
  scope_group_ids: [] as number[],
  scope_platforms: [] as string[],
  purchase_notice: '',
  sort_order: 0,
  for_sale: true
})
const planFeaturesText = ref('')

const validityUnitOptions = computed(() => [
  { value: 'day', label: t('payment.admin.days') },
  { value: 'week', label: t('payment.admin.weeks') },
  { value: 'month', label: t('payment.admin.months') },
])

const groupOptions = computed(() =>
  [
    { value: null, label: t('groups.noGroup') },
    ...subscriptionGroups.value.map(g => ({
      value: g.id,
      label: `${g.name} — ${g.platform} (${g.rate_multiplier}x)`,
      platform: g.platform,
    })),
  ]
)

const subscriptionGroups = computed(() =>
  props.groups.filter(g => g.subscription_type === 'subscription')
)

const scopeTypeOptions = computed(() => [
  { value: 'all_available_groups', label: t('payment.admin.scopeAllAvailableGroups') },
  { value: 'selected_groups', label: t('payment.admin.scopeSelectedGroups') },
  { value: 'platforms', label: t('payment.admin.scopeSelectedPlatforms') },
])

const platformOptions = computed(() =>
  [...new Set(subscriptionGroups.value.map(g => String(g.platform)).filter(Boolean))]
    .map(platform => ({ value: platform, label: platform }))
)

const selectedGroupInfo = computed(() => {
  if (!planForm.group_id) return null
  return props.groups.find(g => g.id === planForm.group_id) || null
})

// Reset form when dialog opens
watch(() => props.show, (visible) => {
  if (!visible) return
  if (props.plan) {
    const scopeConfig = props.plan.scope_config || {}
    Object.assign(planForm, {
      name: props.plan.name,
      group_id: props.plan.group_id,
      description: props.plan.description,
      price: props.plan.price,
      original_price: props.plan.original_price || 0,
      currency: props.plan.currency || '',
      quota_usd: props.plan.quota_usd || 0,
      daily_limit_usd: props.plan.daily_limit_usd ?? null,
      weekly_limit_usd: props.plan.weekly_limit_usd ?? null,
      validity_days: props.plan.validity_days,
      validity_unit: props.plan.validity_unit || 'day',
      scope_type: props.plan.scope_type || 'all_available_groups',
      scope_group_ids: Array.isArray(scopeConfig.group_ids) ? scopeConfig.group_ids : [],
      scope_platforms: Array.isArray(scopeConfig.platforms) ? scopeConfig.platforms : [],
      purchase_notice: props.plan.purchase_notice || '',
      sort_order: props.plan.sort_order || 0,
      for_sale: props.plan.for_sale
    })
    planFeaturesText.value = (props.plan.features || []).join('\n')
  } else {
    Object.assign(planForm, {
      name: '',
      group_id: null,
      description: '',
      price: 0,
      original_price: 0,
      currency: '',
      quota_usd: 0,
      daily_limit_usd: null,
      weekly_limit_usd: null,
      validity_days: 30,
      validity_unit: 'day',
      scope_type: 'all_available_groups',
      scope_group_ids: [],
      scope_platforms: [],
      purchase_notice: '',
      sort_order: 0,
      for_sale: true
    })
    planFeaturesText.value = ''
  }
})

watch(() => planForm.scope_type, (scopeType) => {
  if (scopeType !== 'selected_groups') planForm.scope_group_ids = []
  if (scopeType !== 'platforms') planForm.scope_platforms = []
})

function optionalUsd(value: number | null | '') {
  return value == null || value === '' || Number(value) <= 0 ? null : Number(value)
}

function optionalUsdForSave(value: number | null | '') {
  const normalized = optionalUsd(value)
  return props.plan && normalized == null ? 0 : normalized
}

function buildScopeConfig() {
  if (planForm.scope_type === 'selected_groups') {
    return { group_ids: planForm.scope_group_ids }
  }
  if (planForm.scope_type === 'platforms') {
    return { platforms: planForm.scope_platforms }
  }
  return null
}

/** Build request payload with snake_case keys matching backend JSON tags */
function buildPlanPayload() {
  const features = planFeaturesText.value.split('\n').map(f => f.trim()).filter(Boolean).join('\n')
  return {
    name: planForm.name,
    group_id: props.plan && planForm.group_id == null ? 0 : planForm.group_id,
    description: planForm.description,
    price: planForm.price,
    original_price: planForm.original_price || 0,
    currency: planForm.currency.trim().toUpperCase(),
    quota_usd: planForm.quota_usd,
    daily_limit_usd: optionalUsdForSave(planForm.daily_limit_usd),
    weekly_limit_usd: optionalUsdForSave(planForm.weekly_limit_usd),
    validity_days: planForm.validity_days,
    validity_unit: planForm.validity_unit,
    scope_type: planForm.scope_type,
    scope_config: buildScopeConfig(),
    purchase_notice: planForm.purchase_notice.trim(),
    sort_order: planForm.sort_order,
    for_sale: planForm.for_sale,
    features,
  }
}

async function handleSavePlan() {
  if (!planForm.price || planForm.price <= 0) {
    appStore.showError(t('payment.admin.priceRequired'))
    return
  }
  if (!planForm.quota_usd || planForm.quota_usd <= 0) {
    appStore.showError(t('payment.admin.quotaRequired'))
    return
  }
  if (!planForm.validity_days || planForm.validity_days < 1 || planForm.validity_days > 30) {
    appStore.showError(t('payment.admin.validityDaysRequired'))
    return
  }
  if (planForm.scope_type === 'selected_groups' && planForm.scope_group_ids.length === 0) {
    appStore.showError(t('payment.admin.scopeSelectionRequired'))
    return
  }
  if (planForm.scope_type === 'platforms' && planForm.scope_platforms.length === 0) {
    appStore.showError(t('payment.admin.scopeSelectionRequired'))
    return
  }
  saving.value = true
  try {
    const data = buildPlanPayload()
    if (props.plan) { await adminPaymentAPI.updatePlan(props.plan.id, data) }
    else { await adminPaymentAPI.createPlan(data) }
    appStore.showSuccess(t('common.saved'))
    emit('close')
    emit('saved')
  } catch (err: unknown) { appStore.showError(extractApiErrorMessage(err, t('common.error'))) }
  finally { saving.value = false }
}
</script>
