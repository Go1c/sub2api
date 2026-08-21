<template>
  <div class="min-h-screen bg-[#f7f8fb] text-gray-900 dark:bg-dark-950 dark:text-white">
    <header class="sticky top-0 z-30 border-b border-gray-200 bg-white/85 backdrop-blur-md dark:border-dark-800 dark:bg-dark-950/85">
      <div class="mx-auto flex h-16 max-w-[96rem] items-center justify-between px-4 sm:px-6">
        <button class="flex min-w-0 items-center gap-3 text-left" @click="goHome">
          <span class="relative inline-flex h-10 w-10 shrink-0 items-center justify-center">
            <span class="absolute inset-0 rounded-xl bg-gradient-to-br from-blue-600 via-indigo-500 to-purple-600 shadow-[0_6px_18px_rgba(99,102,241,0.3)]"></span>
            <img
              v-if="siteLogo"
              :src="siteLogo"
              :alt="siteName"
              class="relative h-7 w-7 rounded-lg object-contain"
            />
            <Icon v-else name="grid" size="lg" class="relative text-white" />
          </span>
          <span class="truncate text-lg font-semibold tracking-tight">{{ siteName }}</span>
        </button>

        <div class="flex items-center gap-2">
          <button
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="isDark ? labels.switchToLight : labels.switchToDark"
            @click="toggleTheme"
          >
            <Icon :name="isDark ? 'sun' : 'moon'" size="md" />
          </button>
          <button
            class="rounded-full border border-gray-200 bg-white px-4 py-2 text-sm font-semibold text-gray-700 shadow-sm transition-colors hover:border-gray-300 hover:text-gray-900 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-200 dark:hover:text-white"
            @click="goConsole"
          >
            {{ labels.console }}
          </button>
        </div>
      </div>
    </header>

    <main class="mx-auto grid max-w-[96rem] gap-5 px-4 py-5 sm:px-6 lg:grid-cols-[18rem_minmax(0,1fr)]">
      <aside class="lg:sticky lg:top-20 lg:self-start">
        <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-800 dark:bg-dark-900">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-lg font-semibold">{{ labels.filters }}</h2>
            <button
              class="rounded-md border border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-600 transition-colors hover:bg-gray-50 hover:text-gray-900 dark:border-dark-700 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
              @click="resetFilters"
            >
              {{ labels.reset }}
            </button>
          </div>

          <section class="border-t border-gray-100 py-4 first:border-t-0 first:pt-0 dark:border-dark-800">
            <div class="mb-3 text-sm font-semibold text-gray-700 dark:text-dark-200">{{ labels.providers }}</div>
            <div class="grid grid-cols-2 gap-2">
              <button
                v-for="option in providerOptions"
                :key="option.value"
                :class="[
                  'flex min-w-0 items-center justify-between gap-2 rounded-lg border px-3 py-2 text-left text-sm font-semibold transition-colors',
                  selectedProvider === option.value
                    ? 'border-blue-500 bg-blue-50 text-blue-700 dark:border-blue-400/60 dark:bg-blue-500/10 dark:text-blue-200'
                    : 'border-gray-200 bg-gray-50 text-gray-600 hover:bg-white hover:text-gray-900 dark:border-dark-700 dark:bg-dark-950/60 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white'
                ]"
                @click="selectedProvider = option.value"
              >
                <span class="truncate">{{ option.label }}</span>
                <span class="rounded-full bg-gray-200 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-dark-300">
                  {{ option.count }}
                </span>
              </button>
            </div>
          </section>

          <section class="border-t border-gray-100 py-4 dark:border-dark-800">
            <div class="mb-3 text-sm font-semibold text-gray-700 dark:text-dark-200">{{ labels.groups }}</div>
            <div class="flex max-h-80 flex-col gap-2 overflow-y-auto pr-1">
              <button
                v-for="option in groupOptions"
                :key="option.value"
                :class="[
                  'flex min-w-0 items-center justify-between gap-2 rounded-lg border px-3 py-2 text-left text-sm font-semibold transition-colors',
                  selectedGroup === option.value
                    ? 'border-teal-500 bg-teal-50 text-teal-700 dark:border-teal-400/60 dark:bg-teal-500/10 dark:text-teal-200'
                    : 'border-gray-200 bg-gray-50 text-gray-600 hover:bg-white hover:text-gray-900 dark:border-dark-700 dark:bg-dark-950/60 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white'
                ]"
                @click="selectGroup(option.value)"
              >
                <span class="truncate">{{ option.label }}</span>
                <span class="rounded-full bg-gray-200 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-dark-300">
                  {{ option.count }}
                </span>
              </button>
            </div>
          </section>

          <section class="border-t border-gray-100 pt-4 dark:border-dark-800">
            <div class="mb-3 text-sm font-semibold text-gray-700 dark:text-dark-200">{{ labels.billingType }}</div>
            <div class="grid grid-cols-2 gap-2">
              <button
                v-for="option in billingOptions"
                :key="option.value"
                :class="[
                  'rounded-lg border px-3 py-2 text-sm font-semibold transition-colors',
                  selectedBillingMode === option.value
                    ? 'border-amber-500 bg-amber-50 text-amber-700 dark:border-amber-400/60 dark:bg-amber-500/10 dark:text-amber-200'
                    : 'border-gray-200 bg-gray-50 text-gray-600 hover:bg-white hover:text-gray-900 dark:border-dark-700 dark:bg-dark-950/60 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white'
                ]"
                @click="selectedBillingMode = option.value"
              >
                {{ option.label }}
                <span class="ml-1 text-xs opacity-70">{{ option.count }}</span>
              </button>
            </div>
          </section>
        </div>
      </aside>

      <section class="min-w-0">
        <div class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-800 dark:bg-dark-900">
          <div class="relative overflow-hidden bg-gradient-to-r from-blue-600 via-indigo-600 to-teal-500 px-6 py-7 text-white sm:px-8">
            <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(255,255,255,0.26),transparent_30%),radial-gradient(circle_at_80%_10%,rgba(255,255,255,0.18),transparent_28%)]"></div>
            <div class="relative flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
              <div class="min-w-0">
                <div class="mb-3 inline-flex items-center gap-2 rounded-full bg-white/15 px-3 py-1 text-sm font-semibold backdrop-blur">
                  <Icon name="sparkles" size="sm" />
                  {{ labels.kicker }}
                </div>
                <h1 class="text-3xl font-semibold tracking-tight sm:text-4xl">{{ marketTitle || labels.title }}</h1>
                <p class="mt-3 max-w-3xl text-sm leading-6 text-blue-50 sm:text-base">
                  {{ marketDescription || labels.description }}
                </p>
              </div>
              <div class="grid grid-cols-3 gap-2 text-center sm:min-w-[24rem]">
                <div class="rounded-lg bg-white/14 px-3 py-3 backdrop-blur">
                  <div class="text-2xl font-bold">{{ models.length }}</div>
                  <div class="mt-1 text-xs text-blue-50">{{ labels.models }}</div>
                </div>
                <div class="rounded-lg bg-white/14 px-3 py-3 backdrop-blur">
                  <div class="text-2xl font-bold">{{ providerOptions.length - 1 }}</div>
                  <div class="mt-1 text-xs text-blue-50">{{ labels.providers }}</div>
                </div>
                <div class="rounded-lg bg-white/14 px-3 py-3 backdrop-blur">
                  <div class="text-2xl font-bold">{{ groupOptions.length - 1 }}</div>
                  <div class="mt-1 text-xs text-blue-50">{{ labels.groups }}</div>
                </div>
              </div>
            </div>
          </div>

          <div class="border-b border-gray-200 p-4 dark:border-dark-800">
            <div class="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
              <div class="relative min-w-0 flex-1">
                <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                <input
                  v-model="searchQuery"
                  class="w-full rounded-lg border border-gray-200 bg-gray-50 py-2.5 pl-10 pr-4 text-sm outline-none transition-colors focus:border-blue-500 focus:bg-white dark:border-dark-700 dark:bg-dark-950 dark:text-white dark:focus:border-blue-400"
                  :placeholder="labels.searchPlaceholder"
                  type="text"
                />
              </div>

              <div class="flex flex-wrap items-center gap-2">
                <span v-if="!modelMarketEnabled && !loading" class="rounded-full border border-amber-200 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-700 dark:border-amber-400/30 dark:bg-amber-500/10 dark:text-amber-200">
                  {{ labels.disabled }}
                </span>
                <button
                  class="inline-flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-sm font-semibold text-gray-700 transition-colors hover:bg-gray-50 dark:border-dark-700 dark:text-dark-200 dark:hover:bg-dark-800"
                  @click="copyVisibleModels"
                >
                  <Icon name="copy" size="sm" />
                  {{ copied ? labels.copied : labels.copy }}
                </button>
                <button
                  class="inline-flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-sm font-semibold text-gray-700 transition-colors hover:bg-gray-50 dark:border-dark-700 dark:text-dark-200 dark:hover:bg-dark-800"
                  :disabled="loading"
                  @click="loadModelMarket"
                >
                  <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
                  {{ labels.refresh }}
                </button>
                <div class="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-1 dark:border-dark-700 dark:bg-dark-950">
                  <button
                    :class="viewMode === 'card' ? activeViewButtonClass : inactiveViewButtonClass"
                    @click="viewMode = 'card'"
                  >
                    <Icon name="grid" size="sm" />
                    {{ labels.cardView }}
                  </button>
                  <button
                    :class="viewMode === 'table' ? activeViewButtonClass : inactiveViewButtonClass"
                    @click="viewMode = 'table'"
                  >
                    <Icon name="menu" size="sm" />
                    {{ labels.tableView }}
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div class="p-4 sm:p-5">
            <div v-if="loading && models.length === 0" class="py-20 text-center text-gray-500 dark:text-dark-400">
              <Icon name="refresh" size="xl" class="mx-auto mb-3 animate-spin" />
              {{ labels.loading }}
            </div>

            <div v-else-if="filteredModels.length === 0" class="py-20 text-center">
              <Icon name="inbox" size="xl" class="mx-auto mb-3 text-gray-400" />
              <div class="text-sm font-semibold text-gray-700 dark:text-dark-200">{{ labels.empty }}</div>
              <button class="mt-4 rounded-lg bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700" @click="resetFilters">
                {{ labels.resetFilters }}
              </button>
            </div>

            <div v-else-if="viewMode === 'card'" class="grid gap-4 xl:grid-cols-2">
              <article
                v-for="model in filteredModels"
                :key="model.key"
                class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm transition-all hover:-translate-y-0.5 hover:shadow-md dark:border-dark-700 dark:bg-dark-950"
              >
                <div class="flex items-start justify-between gap-4">
                  <div class="flex min-w-0 items-start gap-3">
                    <span
                      :class="[
                        'inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border',
                        platformBadgeClass(model.platform)
                      ]"
                    >
                      <PlatformIcon :platform="model.platform as GroupPlatform" size="lg" />
                    </span>
                    <div class="min-w-0">
                      <h3 class="truncate text-xl font-semibold tracking-tight">{{ model.name }}</h3>
                      <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                        <span
                          :class="[
                            'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 font-semibold',
                            platformBadgeClass(model.platform)
                          ]"
                        >
                          {{ platformLabel(model.platform) }}
                        </span>
                        <span>{{ billingModeLabel(model.billingMode) }}</span>
                      </div>
                    </div>
                  </div>
                  <button
                    class="rounded-md p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-800 dark:hover:text-white"
                    :title="labels.copyModel"
                    @click="copyText(model.name)"
                  >
                    <Icon name="copy" size="sm" />
                  </button>
                </div>

                <div class="mt-5 grid gap-2 sm:grid-cols-2">
                  <div
                    v-for="line in priceLines(model)"
                    :key="line.label"
                    class="rounded-lg border border-gray-100 bg-gray-50 px-3 py-2 dark:border-dark-800 dark:bg-dark-900"
                  >
                    <div class="text-xs text-gray-500 dark:text-dark-400">{{ line.label }}</div>
                    <div class="mt-1 font-mono text-sm font-semibold text-gray-900 dark:text-white">{{ line.value }}</div>
                  </div>
                </div>

                <div class="mt-4 flex items-start gap-3 text-xs">
                  <div class="min-w-0 flex-1 space-y-2">
                    <div class="flex flex-wrap items-center gap-2">
                      <span
                        v-if="tokenDiscountGroups(model).length > 0"
                        class="shrink-0 font-semibold text-gray-500 dark:text-dark-300"
                      >
                        {{ labels.tokenDiscountTitle }}
                      </span>
                      <span
                        v-for="group in model.groups"
                        :key="group.id"
                        class="inline-flex max-w-full flex-wrap items-center gap-1.5 rounded-full border border-gray-200 bg-gray-50 px-2.5 py-1 font-semibold text-gray-600 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-300"
                      >
                        <span class="min-w-0 max-w-full truncate">{{ group.name }}</span>
                        <span v-if="isTokenBilling(model) && group.rateLabel" class="rounded-full bg-blue-600 px-2 py-0.5 font-mono text-[11px] font-bold text-white dark:bg-blue-500">
                          {{ group.rateLabel }}x
                        </span>
                      </span>
                    </div>
                    <div v-if="tokenDiscountGroups(model).length > 0" class="flex flex-wrap items-center gap-2">
                      <span class="shrink-0 font-semibold text-gray-500 dark:text-dark-300">
                        {{ labels.rechargeRateTitle }}
                      </span>
                      <span class="rounded-full border border-gray-200 bg-gray-50 px-2.5 py-1 font-semibold text-gray-600 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-300">
                        {{ labels.rechargeRateValue }}
                      </span>
                    </div>
                  </div>
                  <span
                    v-if="tokenDiscountGroups(model).length > 0"
                    class="mt-1 shrink-0 text-2xl font-semibold leading-none text-gray-400 dark:text-dark-300"
                    aria-hidden="true"
                  >
                    ×
                  </span>
                  <div
                    v-if="tokenDiscountGroups(model).length > 0"
                    class="flex shrink-0 flex-col items-end gap-1.5"
                  >
                    <div class="flex flex-wrap items-center justify-end gap-2">
                      <span class="font-semibold text-gray-500 dark:text-dark-300">
                        {{ labels.subscriptionRateTitle }}
                      </span>
                      <span class="inline-flex items-center gap-1.5 rounded-full border border-violet-200 bg-violet-50 px-2.5 py-1 font-semibold text-violet-700 dark:border-violet-400/30 dark:bg-violet-500/10 dark:text-violet-200">
                        {{ labels.subscriptionRatePrefix }}
                        <span class="rounded-full bg-violet-600 px-2 py-0.5 font-mono text-[11px] font-bold text-white dark:bg-violet-500">
                          {{ labels.subscriptionRateValue }}
                        </span>
                      </span>
                    </div>
                    <button
                      type="button"
                      class="rounded-md bg-violet-600 px-3 py-1 text-xs font-semibold text-white transition-colors hover:bg-violet-500 dark:bg-violet-500 dark:hover:bg-violet-400"
                      @click="goSubscribe"
                    >
                      {{ labels.subscriptionCta }}
                    </button>
                  </div>
                </div>
              </article>
            </div>

            <div v-else class="overflow-x-auto rounded-lg border border-gray-200 dark:border-dark-800">
              <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-800">
                <thead class="bg-gray-50 text-left text-xs font-semibold uppercase text-gray-500 dark:bg-dark-900 dark:text-dark-400">
                  <tr>
                    <th class="px-4 py-3">{{ labels.model }}</th>
                    <th class="px-4 py-3">{{ labels.provider }}</th>
                    <th class="px-4 py-3">{{ labels.billingType }}</th>
                    <th class="px-4 py-3">{{ labels.price }}</th>
                    <th class="px-4 py-3">{{ labels.groups }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-800 dark:bg-dark-950">
                  <tr v-for="model in filteredModels" :key="model.key" class="hover:bg-gray-50 dark:hover:bg-dark-900">
                    <td class="px-4 py-3 font-semibold">{{ model.name }}</td>
                    <td class="px-4 py-3">
                      <span
                        :class="[
                          'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-xs font-semibold',
                          platformBadgeClass(model.platform)
                        ]"
                      >
                        <PlatformIcon :platform="model.platform as GroupPlatform" size="xs" />
                        {{ platformLabel(model.platform) }}
                      </span>
                    </td>
                    <td class="px-4 py-3 text-gray-600 dark:text-dark-300">{{ billingModeLabel(model.billingMode) }}</td>
                    <td class="px-4 py-3 font-mono text-xs">
                      <div v-for="line in priceLines(model)" :key="line.label" class="whitespace-nowrap">
                        {{ line.label }}: {{ line.value }}
                      </div>
                    </td>
                    <td class="px-4 py-3">
                      <div class="flex flex-wrap gap-1">
                        <span
                          v-for="group in model.groups"
                          :key="group.id"
                          class="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300"
                        >
                          <span>{{ group.name }}</span>
                          <span v-if="model.billingMode === BILLING_MODE_TOKEN && group.rateLabel" class="font-mono font-semibold text-blue-600 dark:text-blue-300">
                            {{ group.rateLabel }}x
                          </span>
                        </span>
                        <button
                          v-if="tokenDiscountGroups(model).length > 0"
                          type="button"
                          class="rounded-md bg-violet-600 px-2.5 py-1 text-xs font-bold text-white hover:bg-violet-500 dark:bg-violet-500 dark:hover:bg-violet-400"
                          @click="goSubscribe"
                        >
                          {{ labels.subscriptionCta }}
                        </button>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import modelMarketAPI, {
  type ModelMarketGroup as ApiModelMarketGroup,
  type ModelMarketModel as ApiModelMarketModel,
  type ModelMarketPricing
} from '@/api/modelMarket'
import { useAppStore, useAuthStore } from '@/stores'
import type { BillingMode } from '@/constants/channel'
import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_TOKEN
} from '@/constants/channel'
import type { GroupPlatform } from '@/types'
import { formatScaled } from '@/utils/pricing'
import { platformBadgeClass, platformLabel } from '@/utils/platformColors'

interface MarketGroup {
  id: string
  name: string
  platform: string
  rateMultiplier: number | null
  rateLabel: string
  exclusive: boolean
}

interface MarketModel {
  key: string
  name: string
  platform: string
  billingMode: BillingMode | 'unknown'
  pricing: ModelMarketPricing | null
  groups: MarketGroup[]
  channels: string[]
  searchable: string
}

interface FilterOption {
  value: string
  label: string
  count: number
}

type BillingFilterValue = 'all' | BillingMode | 'unknown'

interface BillingFilterOption {
  value: BillingFilterValue
  label: string
  count: number
}

const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const { locale } = useI18n()

const isZh = computed(() => locale.value.startsWith('zh'))
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')

const isDark = ref(document.documentElement.classList.contains('dark'))
const liveModels = ref<ApiModelMarketModel[]>([])
const modelMarketEnabled = ref(true)
const marketTitle = ref('')
const marketDescription = ref('')
const loading = ref(false)
const selectedProvider = ref('all')
const selectedGroup = ref('all')
const selectedBillingMode = ref<BillingFilterValue>('all')
const searchQuery = ref('')
const viewMode = ref<'card' | 'table'>('card')
const copied = ref(false)

const activeViewButtonClass =
  'inline-flex items-center gap-1.5 rounded-md bg-white px-3 py-1.5 text-sm font-semibold text-blue-700 shadow-sm dark:bg-dark-800 dark:text-blue-200'
const inactiveViewButtonClass =
  'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-semibold text-gray-500 transition-colors hover:text-gray-900 dark:text-dark-400 dark:hover:text-white'

const labels = computed(() => {
  if (isZh.value) {
    return {
      title: '模型广场',
      kicker: 'Model Market',
      description: '按供应商、分组和计费类型浏览可用模型。登录后会优先展示你当前可访问的真实分组和定价。',
      filters: '筛选',
      reset: '重置',
      providers: '供应商',
      groups: '可用分组',
      billingType: '计费类型',
      allProviders: '全部供应商',
      allGroups: '全部分组',
      allTypes: '全部类型',
      tokenBilling: '按量计费',
      requestBilling: '按次计费',
      imageBilling: '图片计费',
      unknownBilling: '未配置',
      models: '模型',
      model: '模型',
      provider: '供应商',
      price: '价格',
      searchPlaceholder: '搜索模型、供应商、分组或渠道',
      disabled: '未启用',
      copy: '复制',
      copied: '已复制',
      copyModel: '复制模型名称',
      refresh: '刷新',
      cardView: '卡片视图',
      tableView: '表格视图',
      loading: '正在加载模型数据...',
      empty: '没有匹配的模型',
      resetFilters: '清空筛选',
      console: '控制台',
      switchToLight: '切换到浅色模式',
      switchToDark: '切换到深色模式',
      inputPrice: '输入价格（官方价）',
      outputPrice: '输出价格（官方价）',
      cacheWritePrice: '缓存创建（官方价）',
      cacheReadPrice: '缓存读取（官方价）',
      imageOutputPrice: '图片输出（官方价）',
      perRequestPrice: '请求价格（官方价）',
      noPricing: '暂无定价',
      unitPerMillion: '/ 1M Tokens',
      unitPerRequest: '/ 次',
      unitPerImage: '/ 张',
      tokenDiscountTitle: '折扣倍率',
      rechargeRateTitle: '充值倍率',
      rechargeRateValue: '1积分 = 1美元',
      subscriptionRateTitle: '订阅倍率',
      subscriptionRatePrefix: '最低',
      subscriptionRateValue: '0.8x',
      subscriptionCta: '去订阅',
    }
  }

  return {
    title: 'Model Market',
    kicker: 'Model Market',
    description: 'Browse available models by provider, group, and billing type. Signed-in users see their live accessible groups and pricing first.',
    filters: 'Filters',
    reset: 'Reset',
    providers: 'Providers',
    groups: 'Groups',
    billingType: 'Billing',
    allProviders: 'All providers',
    allGroups: 'All groups',
    allTypes: 'All types',
    tokenBilling: 'Token',
    requestBilling: 'Per request',
    imageBilling: 'Image',
    unknownBilling: 'Unset',
    models: 'Models',
    model: 'Model',
    provider: 'Provider',
    price: 'Price',
    searchPlaceholder: 'Search models, providers, groups, or channels',
    disabled: 'Disabled',
    copy: 'Copy',
    copied: 'Copied',
    copyModel: 'Copy model name',
    refresh: 'Refresh',
    cardView: 'Cards',
    tableView: 'Table',
    loading: 'Loading model data...',
    empty: 'No matching models',
    resetFilters: 'Clear filters',
    console: 'Console',
    switchToLight: 'Switch to light mode',
    switchToDark: 'Switch to dark mode',
    inputPrice: 'Input (official)',
    outputPrice: 'Output (official)',
    cacheWritePrice: 'Cache write (official)',
    cacheReadPrice: 'Cache read (official)',
    imageOutputPrice: 'Image output (official)',
    perRequestPrice: 'Request (official)',
    noPricing: 'No pricing',
    unitPerMillion: '/ 1M Tokens',
    unitPerRequest: '/ request',
    unitPerImage: '/ image',
    tokenDiscountTitle: 'Discount',
    rechargeRateTitle: 'Recharge rate',
    rechargeRateValue: '1 credit = $1',
    subscriptionRateTitle: 'Subscription rate',
    subscriptionRatePrefix: 'from',
    subscriptionRateValue: '0.8x',
    subscriptionCta: 'Subscribe',
  }
})

const models = computed(() => buildMarketModels(liveModels.value))

const providerOptions = computed<FilterOption[]>(() => {
  const counts = new Map<string, number>()
  for (const model of models.value) {
    counts.set(model.platform, (counts.get(model.platform) || 0) + 1)
  }
  return [
    { value: 'all', label: labels.value.allProviders, count: models.value.length },
    ...Array.from(counts.entries())
      .sort((a, b) => providerSortKey(a[0]) - providerSortKey(b[0]) || platformLabel(a[0]).localeCompare(platformLabel(b[0])))
      .map(([platform, count]) => ({ value: platform, label: platformLabel(platform), count })),
  ]
})

const groupOptions = computed<FilterOption[]>(() => {
  const groups = new Map<string, { label: string; count: number }>()
  for (const model of models.value) {
    for (const group of model.groups) {
      const current = groups.get(group.id) || { label: group.name, count: 0 }
      current.count += 1
      groups.set(group.id, current)
    }
  }
  return [
    { value: 'all', label: labels.value.allGroups, count: models.value.length },
    ...Array.from(groups.entries())
      .sort((a, b) => b[1].count - a[1].count || a[1].label.localeCompare(b[1].label))
      .map(([value, item]) => ({ value, label: item.label, count: item.count })),
  ]
})

const billingOptions = computed<BillingFilterOption[]>(() => {
  const count = (mode: BillingMode | 'unknown') => models.value.filter((model) => model.billingMode === mode).length
  const options: BillingFilterOption[] = [
    { value: 'all', label: labels.value.allTypes, count: models.value.length },
    { value: BILLING_MODE_TOKEN, label: labels.value.tokenBilling, count: count(BILLING_MODE_TOKEN) },
    { value: BILLING_MODE_PER_REQUEST, label: labels.value.requestBilling, count: count(BILLING_MODE_PER_REQUEST) },
    { value: BILLING_MODE_IMAGE, label: labels.value.imageBilling, count: count(BILLING_MODE_IMAGE) },
    { value: 'unknown', label: labels.value.unknownBilling, count: count('unknown') },
  ]
  return options.filter((option) => option.value === 'all' || option.count > 0)
})

const filteredModels = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  return models.value.filter((model) => {
    if (selectedProvider.value !== 'all' && model.platform !== selectedProvider.value) return false
    if (selectedGroup.value !== 'all' && !model.groups.some((group) => group.id === selectedGroup.value)) return false
    if (selectedBillingMode.value !== 'all' && model.billingMode !== selectedBillingMode.value) return false
    if (query && !model.searchable.includes(query)) return false
    return true
  })
})

function buildMarketModels(items: ApiModelMarketModel[]): MarketModel[] {
  return items.map((model) => {
    const entry: MarketModel = {
      key: model.key,
      name: model.name,
      platform: model.platform || 'api',
      billingMode: model.billing_mode || model.pricing?.billing_mode || 'unknown',
      pricing: model.pricing,
      groups: (model.groups || []).map(toMarketGroup),
      channels: model.channels || [],
      searchable: '',
    }
    entry.searchable = buildSearchable(entry)
    return entry
  }).sort((a, b) => {
    const platformOrder = providerSortKey(a.platform) - providerSortKey(b.platform)
    if (platformOrder !== 0) return platformOrder
    return a.name.localeCompare(b.name)
  })
}

function toMarketGroup(group: ApiModelMarketGroup): MarketGroup {
  const rate = Number.isFinite(group.rate_multiplier) ? Number(group.rate_multiplier) : null
  return {
    id: String(group.id),
    name: group.name,
    platform: group.platform,
    rateMultiplier: rate,
    rateLabel: rate == null ? '' : formatCompactNumber(rate),
    exclusive: Boolean(group.is_exclusive),
  }
}

function buildSearchable(model: MarketModel) {
  return [
    model.name,
    model.platform,
    platformLabel(model.platform),
    model.billingMode,
    ...model.channels,
    ...model.groups.map((group) => group.name),
  ].join(' ').toLowerCase()
}

function providerSortKey(platform: string) {
  switch (platform) {
    case 'openai': return 1
    case 'anthropic': return 2
    case 'gemini': return 3
    case 'antigravity': return 4
    default: return 99
  }
}

function billingModeLabel(mode: MarketModel['billingMode']) {
  switch (mode) {
    case BILLING_MODE_TOKEN:
      return labels.value.tokenBilling
    case BILLING_MODE_PER_REQUEST:
      return labels.value.requestBilling
    case BILLING_MODE_IMAGE:
      return labels.value.imageBilling
    default:
      return labels.value.unknownBilling
  }
}

function isTokenBilling(model: MarketModel) {
  return model.billingMode === BILLING_MODE_TOKEN
}

function tokenDiscountGroups(model: MarketModel) {
  if (!isTokenBilling(model)) return []
  return model.groups.filter((group) => group.rateLabel)
}

function priceLines(model: MarketModel) {
  const pricing = model.pricing
  if (!pricing) return [{ label: labels.value.price, value: labels.value.noPricing }]
  if (pricing.billing_mode === BILLING_MODE_PER_REQUEST) {
    return [{ label: labels.value.perRequestPrice, value: `${formatScaled(pricing.per_request_price, 1)} ${labels.value.unitPerRequest}` }]
  }
  if (pricing.billing_mode === BILLING_MODE_IMAGE) {
    const tierLines = imageTierPriceLines(pricing)
    if (tierLines.length > 0) return tierLines
    if (pricing.image_output_price != null) {
      return [{ label: labels.value.imageOutputPrice, value: `${formatScaled(pricing.image_output_price, 1)} ${labels.value.unitPerImage}` }]
    }
    return [{ label: labels.value.price, value: labels.value.noPricing }]
  }

  const lines = [
    { label: labels.value.inputPrice, value: `${formatScaled(pricing.input_price, 1_000_000)} ${labels.value.unitPerMillion}` },
    { label: labels.value.outputPrice, value: `${formatScaled(pricing.output_price, 1_000_000)} ${labels.value.unitPerMillion}` },
  ]
  if (pricing.cache_read_price != null) {
    lines.push({ label: labels.value.cacheReadPrice, value: `${formatScaled(pricing.cache_read_price, 1_000_000)} ${labels.value.unitPerMillion}` })
  }
  if (pricing.cache_write_price != null) {
    lines.push({ label: labels.value.cacheWritePrice, value: `${formatScaled(pricing.cache_write_price, 1_000_000)} ${labels.value.unitPerMillion}` })
  }
  if (pricing.image_output_price != null && pricing.image_output_price > 0) {
    lines.push({ label: labels.value.imageOutputPrice, value: `${formatScaled(pricing.image_output_price, 1_000_000)} ${labels.value.unitPerMillion}` })
  }
  return lines
}

function imageTierPriceLines(pricing: ModelMarketPricing) {
  const tierOrder = new Map([
    ['1K', 1],
    ['2K', 2],
    ['4K', 3],
  ])
  return (pricing.intervals || [])
    .filter((interval) => interval.per_request_price != null && String(interval.tier_label || '').trim())
    .map((interval) => {
      const label = String(interval.tier_label || '').trim().toUpperCase()
      return {
        label,
        value: `${formatScaled(interval.per_request_price, 1)} ${labels.value.unitPerImage}`,
      }
    })
    .sort((left, right) => {
      const leftOrder = tierOrder.get(left.label) || 99
      const rightOrder = tierOrder.get(right.label) || 99
      if (leftOrder !== rightOrder) return leftOrder - rightOrder
      return left.label.localeCompare(right.label)
    })
}

async function loadModelMarket() {
  if (shouldUseMockMarket()) {
    modelMarketEnabled.value = true
    marketTitle.value = '模型广场'
    marketDescription.value = '本地预览数据：价格展示官方价，按量计费模型展示分组折扣倍率。'
    liveModels.value = mockModelMarketModels()
    loading.value = false
    return
  }

  loading.value = true
  try {
    const payload = await modelMarketAPI.getPublicModelMarket()
    modelMarketEnabled.value = payload.enabled
    marketTitle.value = payload.title || ''
    marketDescription.value = payload.description || ''
    liveModels.value = payload.enabled ? payload.models || [] : []
  } catch (error) {
    console.warn('Failed to load model market', error)
    liveModels.value = []
  } finally {
    loading.value = false
  }
}

function shouldUseMockMarket() {
  return import.meta.env.DEV && new URLSearchParams(window.location.search).get('mock') === '1'
}

function mockModelMarketModels(): ApiModelMarketModel[] {
  const openaiDemoGroup: ApiModelMarketGroup = {
    id: 1,
    name: '【自建】Gpt-Pro20x支持Image2',
    platform: 'openai',
    subscription_type: 'standard',
    rate_multiplier: 0.35,
    is_exclusive: false,
  }

  return [
    {
      key: 'openai:gpt-5.3-codex',
      name: 'gpt-5.3-codex',
      platform: 'openai',
      billing_mode: BILLING_MODE_TOKEN,
      pricing: {
        billing_mode: BILLING_MODE_TOKEN,
        input_price: 0.00000175,
        output_price: 0.000014,
        cache_write_price: null,
        cache_read_price: 0.000000175,
        image_output_price: null,
        per_request_price: null,
        intervals: [],
      },
      groups: [openaiDemoGroup],
      channels: ['OpenAI'],
      sort_order: 0,
    },
    {
      key: 'openai:gpt-5.4',
      name: 'gpt-5.4',
      platform: 'openai',
      billing_mode: BILLING_MODE_TOKEN,
      pricing: {
        billing_mode: BILLING_MODE_TOKEN,
        input_price: 0.0000025,
        output_price: 0.000015,
        cache_write_price: null,
        cache_read_price: 0.00000025,
        image_output_price: null,
        per_request_price: null,
        intervals: [],
      },
      groups: [openaiDemoGroup],
      channels: ['OpenAI'],
      sort_order: 1,
    },
    {
      key: 'openai:gpt-image-2',
      name: 'gpt-image-2',
      platform: 'openai',
      billing_mode: BILLING_MODE_IMAGE,
      pricing: {
        billing_mode: BILLING_MODE_IMAGE,
        input_price: null,
        output_price: null,
        cache_write_price: null,
        cache_read_price: null,
        image_output_price: null,
        per_request_price: null,
        intervals: [
          {
            min_tokens: 0,
            max_tokens: null,
            tier_label: '1K',
            input_price: null,
            output_price: null,
            cache_write_price: null,
            cache_read_price: null,
            per_request_price: 0.05,
          },
          {
            min_tokens: 0,
            max_tokens: null,
            tier_label: '4K',
            input_price: null,
            output_price: null,
            cache_write_price: null,
            cache_read_price: null,
            per_request_price: 0.15,
          },
        ],
      },
      groups: [openaiDemoGroup],
      channels: ['OpenAI'],
      sort_order: 2,
    },
  ]
}

function resetFilters() {
  selectedProvider.value = 'all'
  selectedGroup.value = 'all'
  selectedBillingMode.value = 'all'
  searchQuery.value = ''
}

function selectGroup(value: string) {
  selectedProvider.value = 'all'
  selectedGroup.value = value
}

async function copyVisibleModels() {
  await copyText(filteredModels.value.map((model) => model.name).join('\n'))
}

async function copyText(text: string) {
  if (!text) return
  try {
    await navigator.clipboard?.writeText(text)
    copied.value = true
    window.setTimeout(() => {
      copied.value = false
    }, 1500)
  } catch (error) {
    console.warn('Failed to copy model market text', error)
  }
}

function formatCompactNumber(value: number) {
  return value
    .toFixed(value < 1 ? 2 : 1)
    .replace(/(\.\d*?[1-9])0+$/, '$1')
    .replace(/\.0$/, '')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  const prefersDark = typeof window.matchMedia === 'function' && window.matchMedia('(prefers-color-scheme: dark)').matches
  if (
    savedTheme === 'dark' ||
    (!savedTheme && prefersDark)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function goHome() {
  router.push('/home')
}

function goConsole() {
  router.push(authStore.isAuthenticated ? (authStore.isAdmin ? '/admin/dashboard' : '/dashboard') : '/login')
}

const subscribePurchasePath = '/purchase?tab=subscription'

function goSubscribe() {
  if (authStore.isAuthenticated) {
    router.push({ path: '/purchase', query: { tab: 'subscription' } })
    return
  }
  router.push({ path: '/login', query: { redirect: subscribePurchasePath } })
}

onMounted(() => {
  initTheme()
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    void appStore.fetchPublicSettings()
  }
  void loadModelMarket()
})
</script>
