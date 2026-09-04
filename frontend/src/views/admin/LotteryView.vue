<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Page header -->
      <header class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ t('admin.lottery.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">
            {{ t('admin.lottery.description') }}
          </p>
        </div>
        <div class="inline-flex rounded-xl bg-white p-1 shadow-sm ring-1 ring-gray-200 dark:bg-dark-800 dark:ring-dark-700">
          <button
            v-for="tab in subTabs"
            :key="tab.key"
            class="rounded-lg px-4 py-1.5 text-sm font-medium transition-colors"
            :class="activeTab === tab.key
              ? 'bg-primary-500 text-white shadow-sm'
              : 'text-gray-600 hover:bg-gray-50 dark:text-dark-300 dark:hover:bg-dark-700'"
            @click="activeTab = tab.key"
          >
            {{ tab.label }}
          </button>
        </div>
      </header>

      <!-- Create campaign -->
      <section v-if="activeTab === 'create'" class="card card-body">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('admin.lottery.create.title') }}
        </h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">
          {{ t('admin.lottery.create.description') }}
        </p>
        <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">
          {{ t('admin.lottery.create.wheelNote') }}
        </p>

        <div
          v-if="showGuaranteedWin"
          data-test="lottery-guaranteed-win-hint"
          class="mt-4 rounded-2xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800 dark:border-emerald-900/60 dark:bg-emerald-950/40 dark:text-emerald-200"
        >
          {{ t('admin.lottery.create.guaranteedWinHint') }}
        </div>

        <div class="mt-5 grid gap-4 md:grid-cols-2">
          <label class="block">
            <span class="input-label">{{ t('admin.lottery.create.name') }}</span>
            <input
              v-model="form.name"
              data-test="lottery-name"
              type="text"
              :placeholder="t('admin.lottery.create.namePlaceholder')"
              class="input"
            />
          </label>
          <label class="block">
            <span class="input-label">{{ t('admin.lottery.create.subtitle') }}</span>
            <input
              v-model="form.subtitle"
              data-test="lottery-subtitle"
              type="text"
              :placeholder="t('admin.lottery.create.subtitlePlaceholder')"
              class="input"
            />
          </label>
          <label class="block">
            <span class="input-label">{{ t('admin.lottery.create.prizeCount') }}</span>
            <input v-model.number="form.prizeCount" data-test="lottery-prize-count" type="number" min="1" class="input font-mono" />
          </label>
          <label class="block">
            <span class="input-label">{{ t('admin.lottery.create.maxParticipants') }}</span>
            <input v-model.number="form.maxParticipants" data-test="lottery-max-participants" type="number" min="1" class="input font-mono" />
          </label>
          <label class="block">
            <span class="input-label">{{ t('admin.lottery.create.earlyBoostParticipantPercent') }}</span>
            <input
              v-model.number="form.earlyBoostParticipantPercent"
              data-test="lottery-early-boost-percent"
              type="number"
              min="0"
              max="100"
              class="input font-mono"
            />
            <span class="input-hint">{{ t('admin.lottery.create.earlyBoostParticipantPercentHint') }}</span>
          </label>
          <label class="block">
            <span class="input-label">{{ t('admin.lottery.create.rechargeBoostCapPercent') }}</span>
            <input
              v-model.number="form.rechargeBoostCapPercent"
              data-test="lottery-recharge-boost-cap"
              type="number"
              min="0"
              max="50"
              class="input font-mono"
            />
            <span class="input-hint">{{ t('admin.lottery.create.rechargeBoostCapPercentHint') }}</span>
          </label>
        </div>

        <section class="mt-6 rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/60 md:p-5">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('admin.lottery.create.promoSection') }}
          </h3>
          <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">
            {{ t('admin.lottery.create.promoSectionHint') }}
          </p>
          <div class="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_220px]">
            <div class="space-y-4">
              <label class="block">
                <span class="input-label">{{ t('admin.lottery.create.promoText') }}</span>
                <input
                  v-model="form.promoText"
                  data-test="lottery-promo-text"
                  type="text"
                  maxlength="240"
                  :placeholder="t('admin.lottery.create.promoTextPlaceholder')"
                  class="input"
                />
              </label>
              <label class="block">
                <span class="input-label">{{ t('admin.lottery.create.promoImageUrl') }}</span>
                <input
                  v-model="form.promoImageUrl"
                  data-test="lottery-promo-image-url"
                  type="url"
                  :placeholder="t('admin.lottery.create.promoImageUrlPlaceholder')"
                  class="input font-mono text-xs"
                />
                <span class="input-hint">{{ t('admin.lottery.create.promoImageUrlHint') }}</span>
              </label>
            </div>
            <div class="flex items-center justify-center rounded-2xl bg-white p-3 ring-1 ring-gray-200 dark:bg-dark-900 dark:ring-dark-700">
              <img
                v-if="livePromoImageUrl"
                data-test="lottery-promo-preview"
                :src="livePromoImageUrl"
                alt=""
                class="max-h-48 w-full max-w-[180px] rounded-xl object-contain"
              />
              <p v-else class="px-2 text-center text-xs text-gray-400 dark:text-dark-500">
                {{ t('admin.lottery.create.promoPreviewEmpty') }}
              </p>
            </div>
          </div>
        </section>

        <div class="mt-5">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <span class="input-label !mb-0">{{ t('admin.lottery.create.codes') }}</span>
            <span
              class="font-mono text-xs"
              :class="codeLines.length >= form.prizeCount ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-300'"
            >
              {{ t('admin.lottery.create.codesProgress', { filled: codeLines.length, needed: form.prizeCount }) }}
            </span>
          </div>
          <textarea
            v-model="form.codesRaw"
            data-test="lottery-codes"
            rows="6"
            :placeholder="t('admin.lottery.create.codesPlaceholder')"
            class="input mt-1 font-mono"
          ></textarea>
          <p class="input-hint">{{ t('admin.lottery.create.codesHint') }}</p>
        </div>

        <div class="mt-6 flex flex-wrap items-center gap-3">
          <button class="btn btn-primary" data-test="lottery-submit" :disabled="!canSubmit || submitting" @click="submitCampaign">
            {{ submitting ? t('common.saving') : t('admin.lottery.create.submit') }}
          </button>
          <button class="btn btn-secondary" @click="resetForm">
            {{ t('common.reset') }}
          </button>
          <button class="btn btn-ghost btn-sm" @click="openPreview = true" :disabled="submitting">
            {{ t('admin.lottery.create.preview') }}
          </button>
          <button class="btn btn-ghost btn-sm" data-test="lottery-ux-preview" :disabled="submitting" @click="openUxPreview = true">
            {{ t('admin.lottery.create.uxPreview') }}
          </button>
          <span v-if="formError" class="text-xs text-red-500">{{ formError }}</span>
          <span v-if="formSaved" class="text-xs text-emerald-500">
            {{ t('admin.lottery.create.saved') }}
          </span>
        </div>
      </section>

      <!-- History -->
      <section v-else-if="activeTab === 'history'" class="card">
        <div class="card-header">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('admin.lottery.history.title') }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">
            {{ t('admin.lottery.history.description') }}
          </p>
        </div>

        <div
          v-if="lotteryStore.loadingCampaigns && displayCampaigns.length === 0"
          class="px-6 py-12 text-center text-sm text-gray-500 dark:text-dark-400"
        >
          {{ t('common.loading') }}
        </div>

        <div v-else-if="displayCampaigns.length === 0" class="px-6 py-12 text-center">
          <div class="text-sm text-gray-500 dark:text-dark-400">
            {{ t('admin.lottery.history.empty') }}
          </div>
        </div>

        <ul v-else class="divide-y divide-gray-100 dark:divide-dark-700">
          <li v-for="c in displayCampaigns" :key="c.id" class="px-6 py-4">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div class="flex items-center gap-2">
                  <span class="text-base font-semibold text-gray-900 dark:text-white">{{ c.name }}</span>
                  <span
                    class="badge"
                    :class="c.status === 'active' ? 'badge-success' : 'badge-gray'"
                  >
                    {{ c.status === 'active' ? t('admin.lottery.status.active') : t('admin.lottery.status.finished') }}
                  </span>
                </div>
                <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ c.created_at }}</div>
              </div>
              <div class="flex flex-wrap items-center gap-3 text-right">
                <div class="font-mono text-xs text-gray-500 dark:text-dark-300">
                  <span class="font-bold text-gray-900 dark:text-white">{{ c.joined_count }}</span>/{{ c.max_participants }}
                  {{ t('admin.lottery.history.joined') }}
                </div>
                <div class="font-mono text-xs text-gray-500 dark:text-dark-300">
                  <span class="font-bold text-amber-600 dark:text-amber-300">{{ c.winner_count }}</span>/{{ c.prize_count }}
                  {{ t('admin.lottery.history.won') }}
                </div>
                <button
                  v-if="c.status === 'active'"
                  class="btn btn-secondary btn-sm"
                  @click="finishCampaign(c.id)"
                >
                  {{ t('admin.lottery.history.finish') }}
                </button>
                <button class="btn btn-secondary btn-sm" @click="toggleExpand(c.id)">
                  {{ expanded === c.id ? t('common.collapse') : t('admin.lottery.history.details') }}
                </button>
              </div>
            </div>

            <div
              v-if="expanded === c.id"
              class="mt-4 grid gap-4 rounded-xl bg-gray-50 p-4 md:grid-cols-2 dark:bg-dark-800"
            >
              <div>
                <div class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">
                  {{ t('admin.lottery.history.winnersList') }}
                </div>
                <ul class="mt-2 space-y-1.5 text-xs">
                  <li
                    v-for="winner in winnerRows(c)"
                    :key="winner.key"
                    class="flex items-center justify-between gap-3 rounded-lg bg-white px-3 py-2 ring-1 ring-gray-200 dark:bg-dark-900 dark:ring-dark-700"
                  >
                    <span class="text-gray-700 dark:text-dark-100">{{ winner.userLabel }}</span>
                    <span class="font-mono text-amber-600 dark:text-amber-300">{{ winner.code }}</span>
                  </li>
                  <li v-if="winnerRows(c).length === 0" class="text-gray-400 dark:text-dark-500">
                    {{ t('admin.lottery.history.noWinnersYet') }}
                  </li>
                </ul>
              </div>
              <div>
                <div class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400">
                  {{ t('admin.lottery.history.unclaimedCodes') }}
                </div>
                <ul class="mt-2 space-y-1.5 text-xs">
                  <li
                    v-for="(code, i) in lotteryStore.unclaimedCodes(c)"
                    :key="i"
                    class="rounded-lg bg-white px-3 py-2 font-mono text-gray-700 ring-1 ring-gray-200 dark:bg-dark-900 dark:text-dark-100 dark:ring-dark-700"
                  >
                    {{ code.code }}
                  </li>
                  <li
                    v-if="lotteryStore.unclaimedCodes(c).length === 0"
                    class="text-gray-400 dark:text-dark-500"
                  >
                    {{ t('admin.lottery.history.allClaimed') }}
                  </li>
                </ul>
              </div>
            </div>
          </li>
        </ul>
      </section>
    </div>

    <!-- Admin preview dialog (lets admin try the wheel without using a real user account) -->
    <LotteryDialog
      v-if="previewSegments.length > 0"
      :open="openPreview"
      :campaign-title="form.name || t('admin.lottery.create.previewTitle')"
      :subtitle="form.subtitle || t('admin.lottery.create.previewSubtitle')"
      :prize-count="form.prizeCount"
      :max-participants="form.maxParticipants"
      :joined="0"
      :segments="previewSegments"
      :promo-text="livePromoText"
      :promo-image-url="previewPromoImageSrc"
      :draw-fn="previewDrawFn"
      @close="openPreview = false"
    />

    <Teleport to="body">
      <div
        v-if="openUxPreview"
        class="fixed inset-0 z-[1100] flex items-center justify-center bg-black/70 p-3 backdrop-blur-sm"
        data-test="lottery-ux-preview-modal"
        @click.self="openUxPreview = false"
      >
        <div class="relative flex h-[min(92dvh,880px)] w-[min(96vw,1180px)] flex-col overflow-hidden rounded-3xl bg-[#07101f] shadow-2xl ring-1 ring-white/10">
          <div class="flex items-center justify-between gap-3 border-b border-white/10 px-4 py-3">
            <div class="text-sm font-semibold text-white">{{ t('admin.lottery.create.uxPreview') }}</div>
            <button type="button" class="rounded-full px-3 py-1 text-sm text-slate-300 hover:bg-white/10" @click="openUxPreview = false">
              {{ t('common.close') }}
            </button>
          </div>
          <iframe
            class="h-full w-full flex-1 border-0 bg-[#07101f]"
            title="抽奖体验预览"
            :srcdoc="uxPreviewHtml"
          />
        </div>
      </div>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import LotteryDialog, { type DrawResult } from '@/components/lottery/LotteryDialog.vue'
import type { WheelSegment } from '@/components/lottery/LotteryWheel.vue'
import { useAppStore } from '@/stores/app'
import { useLotteryStore } from '@/stores/lottery'
import { extractApiErrorMessage } from '@/utils/apiError'
import { publicHttpsImageUrl } from '@/utils/siteMessageContent'
import type { CreateLotteryCampaignRequest, LotteryCampaign, LotteryDraw } from '@/types'
import uxPreviewHtml from '@/assets/lottery-ux-preview.html?raw'

const { t } = useI18n()
const appStore = useAppStore()
const lotteryStore = useLotteryStore()

const subTabs = computed(() => [
  { key: 'create' as const, label: t('admin.lottery.tabs.create') },
  { key: 'history' as const, label: t('admin.lottery.tabs.history') }
])
const activeTab = ref<'create' | 'history'>('create')

const form = ref({
  name: '',
  subtitle: '',
  prizeCount: 5,
  maxParticipants: 20,
  earlyBoostParticipantPercent: 25,
  rechargeBoostCapPercent: 0,
  promoText: '',
  promoImageUrl: '',
  codesRaw: ''
})
const formError = ref('')
const formSaved = ref(false)
const submitting = ref(false)

const displayCampaigns = computed(() =>
  lotteryStore.campaigns.map(
    (campaign) => lotteryStore.getCampaignDetail(campaign.id) ?? campaign,
  ),
)

const codeLines = computed(() =>
  form.value.codesRaw.split(/\r?\n/).map(s => s.trim()).filter(Boolean)
)
const promoImageUrlValid = computed(() => {
  const url = form.value.promoImageUrl.trim()
  return url === '' || Boolean(publicHttpsImageUrl(url))
})
const showGuaranteedWin = computed(
  () => form.value.prizeCount > 0 && form.value.maxParticipants > 0 && form.value.prizeCount >= form.value.maxParticipants,
)
const livePromoImageUrl = computed(() => publicHttpsImageUrl(form.value.promoImageUrl))
const livePromoText = computed(
  () => form.value.promoText.trim() || t('admin.lottery.create.previewPromoText'),
)
const canSubmit = computed(
  () =>
    form.value.name.trim().length > 0 &&
    form.value.prizeCount > 0 &&
    form.value.maxParticipants >= form.value.prizeCount &&
    form.value.earlyBoostParticipantPercent >= 0 &&
    form.value.earlyBoostParticipantPercent <= 100 &&
    form.value.rechargeBoostCapPercent >= 0 &&
    form.value.rechargeBoostCapPercent <= 50 &&
    promoImageUrlValid.value &&
    codeLines.value.length >= form.value.prizeCount
)

onMounted(async () => {
  try {
    await lotteryStore.loadCampaigns()
  } catch (error) {
    appStore.showError(
      extractApiErrorMessage(error, t('admin.lottery.history.failedToLoad')),
    )
  }
})

async function submitCampaign() {
  formError.value = ''
  formSaved.value = false
  if (!promoImageUrlValid.value) {
    formError.value = t('admin.lottery.create.invalidPromoImage')
    return
  }
  if (!canSubmit.value) {
    formError.value = t('admin.lottery.create.invalid')
    return
  }
  submitting.value = true
  try {
    const input: CreateLotteryCampaignRequest = {
      name: form.value.name.trim(),
      subtitle: form.value.subtitle.trim(),
      prize_count: form.value.prizeCount,
      max_participants: form.value.maxParticipants,
      early_boost_participant_percent: form.value.earlyBoostParticipantPercent,
      recharge_boost_cap_percent: form.value.rechargeBoostCapPercent,
      promo_text: form.value.promoText.trim(),
      promo_image_url: form.value.promoImageUrl.trim(),
      codes: codeLines.value,
    }
    await lotteryStore.createCampaign(input)
    formSaved.value = true
    activeTab.value = 'history'
  } catch (error) {
    formError.value = extractApiErrorMessage(
      error,
      t('admin.lottery.create.failed'),
    )
  } finally {
    submitting.value = false
  }
}

function resetForm() {
  form.value = {
    name: '',
    subtitle: '',
    prizeCount: 5,
    maxParticipants: 20,
    earlyBoostParticipantPercent: 25,
    rechargeBoostCapPercent: 0,
    promoText: '',
    promoImageUrl: '',
    codesRaw: '',
  }
  formError.value = ''
  formSaved.value = false
}

const expanded = ref<number | null>(null)
async function toggleExpand(id: number) {
  if (expanded.value === id) {
    expanded.value = null
    return
  }

  expanded.value = id
  try {
    await lotteryStore.loadCampaign(id)
  } catch (error) {
    appStore.showError(
      extractApiErrorMessage(error, t('admin.lottery.history.failedToLoadDetail')),
    )
  }
}

async function finishCampaign(id: number) {
  try {
    await lotteryStore.finishCampaign(id)
  } catch (error) {
    appStore.showError(
      extractApiErrorMessage(error, t('admin.lottery.history.failedToFinish')),
    )
  }
}

function winnerRows(campaign: LotteryCampaign) {
  const codeById = new Map(
    (campaign.codes ?? []).map((code) => [code.id, code.code]),
  )
  return (campaign.draws ?? [])
    .filter((draw) => draw.won)
    .map((draw) => ({
      key: String(draw.id),
      userLabel: winnerUserLabel(draw),
      code: draw.lottery_code_id ? codeById.get(draw.lottery_code_id) ?? '-' : '-',
    }))
}

function winnerUserLabel(draw: LotteryDraw) {
  const email = draw.user_email?.trim()
  if (email) {
    return t('admin.lottery.history.userWithEmail', { id: draw.user_id, email })
  }
  return t('admin.lottery.history.userPrefix', { id: draw.user_id })
}

// Preview wheel (no persistence, no user impact)
const openPreview = ref(false)
const openUxPreview = ref(false)
const previewPromoImageUrl = '/lottery-preview-qr.svg'
const previewPromoImageSrc = computed(
  () => livePromoImageUrl.value || previewPromoImageUrl,
)
const previewSegments = computed<WheelSegment[]>(() => [
  { label: '奖品', isPrize: true },
  { label: '谢谢参与', isPrize: false },
  { label: '奖品', isPrize: true },
  { label: '奖品', isPrize: true },
  { label: '谢谢参与', isPrize: false },
  { label: '奖品', isPrize: true },
  { label: '谢谢参与', isPrize: false },
  { label: '奖品', isPrize: true },
])

async function previewDrawFn(): Promise<DrawResult> {
  await new Promise(r => setTimeout(r, 250))
  const prizeIdx = previewSegments.value
    .map((s, i) => ({ s, i }))
    .filter(({ s }) => s.isPrize)
  const blanks = previewSegments.value
    .map((s, i) => ({ s, i }))
    .filter(({ s }) => !s.isPrize)
  const won = Math.random() < 0.5 && prizeIdx.length > 0
  const pick = won
    ? prizeIdx[Math.floor(Math.random() * prizeIdx.length)]
    : blanks[Math.floor(Math.random() * blanks.length)]
  return {
    won,
    index: pick.i,
    label: pick.s.label,
    message: won
      ? '恭喜你中奖！兑换码已通过站内信发放，请前往站内信领取。'
      : '很遗憾，这次没有中奖。'
  }
}
</script>
