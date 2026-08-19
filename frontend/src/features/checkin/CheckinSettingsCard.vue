<template>
  <section class="card" aria-labelledby="checkin-settings-title">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 id="checkin-settings-title" class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('checkin.settings.title') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('checkin.settings.description') }}</p>
    </div>
    <div v-if="loading" class="space-y-4 p-6" aria-busy="true">
      <div v-for="index in 4" :key="index" class="h-10 animate-pulse rounded bg-gray-100 dark:bg-dark-700"></div>
    </div>
    <div v-else-if="loadError" class="p-6">
      <p class="text-sm text-red-600 dark:text-red-400">{{ t('checkin.settings.loadFailed') }}</p>
      <button type="button" class="btn btn-secondary mt-3" @click="loadSettings">{{ t('checkin.retry') }}</button>
    </div>
    <div v-else class="space-y-6 p-6">
      <div class="flex items-start justify-between gap-6">
        <div>
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('checkin.settings.enabled') }}</label>
          <p class="mt-1 max-w-2xl text-xs text-gray-500 dark:text-gray-400">{{ t('checkin.settings.enabledHint') }}</p>
        </div>
        <Toggle v-model="draft.enabled" data-test="enabled-toggle" />
      </div>
      <div class="grid gap-4 md:grid-cols-2">
        <label v-for="field in moneyFields" :key="field.key" class="block">
          <span class="input-label">{{ t(field.label) }}</span>
          <div class="relative">
            <span class="pointer-events-none absolute inset-y-0 left-3 flex items-center text-sm text-gray-400">$</span>
            <input v-model.trim="draft[field.key]" class="input pl-7 ui-mono" inputmode="decimal" autocomplete="off" />
          </div>
          <span v-if="field.key === 'daily_cap'" class="mt-1 block text-xs text-gray-400">{{ t('checkin.settings.dailyCapHint') }}</span>
        </label>
        <label class="block">
          <span class="input-label">{{ t('checkin.settings.timezone') }}</span>
          <input v-model.trim="draft.timezone" class="input ui-mono" list="checkin-timezones" autocomplete="off" />
          <datalist id="checkin-timezones">
            <option value="Asia/Shanghai" /><option value="Asia/Singapore" /><option value="UTC" /><option value="America/Los_Angeles" /><option value="Europe/London" />
          </datalist>
        </label>
      </div>
      <div class="border-t border-gray-100 pt-5 dark:border-dark-700">
        <div class="mb-3 flex items-center justify-between gap-4">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('checkin.settings.milestones') }}</h3>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="draft.milestones.length >= 10" @click="addMilestone">
            <Icon name="plus" size="sm" /><span>{{ t('checkin.settings.addMilestone') }}</span>
          </button>
        </div>
        <div v-if="draft.milestones.length" class="space-y-3">
          <div v-for="(milestone, index) in draft.milestones" :key="index" class="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_2.5rem] items-end gap-3">
            <label class="block min-w-0"><span class="input-label">{{ t('checkin.settings.milestoneDay') }}</span><input v-model.number="milestone.day" type="number" min="1" step="1" class="input ui-mono" /></label>
            <label class="block min-w-0"><span class="input-label">{{ t('checkin.settings.milestoneBonus') }}</span><input v-model.trim="milestone.bonus" inputmode="decimal" class="input ui-mono" /></label>
            <button type="button" class="btn btn-secondary h-10 w-10 p-0" :title="t('checkin.settings.removeMilestone')" @click="removeMilestone(index)"><Icon name="trash" size="sm" /></button>
          </div>
        </div>
      </div>
      <div class="rounded-md border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('checkin.settings.maximumReward') }}</span>
          <strong data-test="maximum-reward" class="ui-mono text-base text-gray-900 dark:text-white">${{ liveMaximum }}</strong>
        </div>
        <p v-if="budgetRisk" data-test="daily-cap-warning" class="mt-2 flex items-start gap-2 text-sm text-amber-700 dark:text-amber-300"><Icon name="exclamationTriangle" size="sm" class="mt-0.5 flex-shrink-0" /><span>{{ t('checkin.settings.budgetWarning') }}</span></p>
      </div>
      <p v-if="saveError" class="text-sm text-red-600 dark:text-red-400">{{ t('checkin.settings.saveFailed') }}</p>
      <p v-else-if="saved" class="text-sm text-emerald-600 dark:text-emerald-400">{{ t('checkin.settings.saved') }}</p>
      <div class="flex justify-end">
        <button type="button" data-test="save-settings" class="btn btn-primary" :disabled="saving" @click="saveSettings">
          <Icon :name="saving ? 'refresh' : 'check'" size="sm" :class="saving ? 'animate-spin' : ''" /><span>{{ saving ? t('checkin.settings.saving') : t('checkin.settings.save') }}</span>
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { getActivePinia } from 'pinia'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import { checkinAPI } from './api'
import { hasDailyCapRisk, maximumSingleReward } from './settings'
import { useCheckinStore } from './store'
import type { CheckinSettings, CheckinSettingsRequest } from './types'

const { t } = useI18n()
const loading = ref(true), saving = ref(false), loadError = ref(false), saveError = ref(false), saved = ref(false)
const draft = reactive<CheckinSettingsRequest>({ enabled: false, min_reward: '0.1000', max_reward: '0.5000', timezone: 'Asia/Shanghai', daily_cap: '0.0000', milestones: [] })
const moneyFields: { key: 'min_reward' | 'max_reward' | 'daily_cap'; label: string }[] = [
  { key: 'min_reward', label: 'checkin.settings.minReward' }, { key: 'max_reward', label: 'checkin.settings.maxReward' }, { key: 'daily_cap', label: 'checkin.settings.dailyCap' }
]
const liveMaximum = computed(() => maximumSingleReward(draft))
const budgetRisk = computed(() => hasDailyCapRisk(draft))

function applySettings(settings: CheckinSettings) {
  draft.enabled = settings.enabled; draft.min_reward = settings.min_reward; draft.max_reward = settings.max_reward
  draft.timezone = settings.timezone; draft.daily_cap = settings.daily_cap; draft.milestones = settings.milestones.map(item => ({ ...item }))
}
async function loadSettings() {
  loading.value = true; loadError.value = false
  try { applySettings(await checkinAPI.getSettings()) } catch { loadError.value = true } finally { loading.value = false }
}
function addMilestone() {
  if (draft.milestones.length >= 10) return
  draft.milestones.push({ day: draft.milestones.reduce((max, item) => Math.max(max, item.day || 0), 0) + 1, bonus: '0.0000' })
}
function removeMilestone(index: number) { draft.milestones.splice(index, 1) }
async function saveSettings() {
  saving.value = true; saved.value = false; saveError.value = false
  try {
    const payload: CheckinSettingsRequest = { enabled: draft.enabled, min_reward: draft.min_reward, max_reward: draft.max_reward, timezone: draft.timezone, daily_cap: draft.daily_cap, milestones: draft.milestones.map(item => ({ ...item })) }
    applySettings(await checkinAPI.updateSettings(payload)); saved.value = true
    const pinia = getActivePinia(); if (pinia) void useCheckinStore(pinia).fetchStatus()
  } catch { saveError.value = true } finally { saving.value = false }
}
onMounted(() => void loadSettings())
</script>
