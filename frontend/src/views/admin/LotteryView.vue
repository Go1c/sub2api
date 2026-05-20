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

        <div class="mt-5 grid gap-4 md:grid-cols-2">
          <label class="block">
            <span class="input-label">{{ t('admin.lottery.create.name') }}</span>
            <input
              v-model="form.name"
              type="text"
              :placeholder="t('admin.lottery.create.namePlaceholder')"
              class="input"
            />
          </label>
          <label class="block">
            <span class="input-label">{{ t('admin.lottery.create.subtitle') }}</span>
            <input
              v-model="form.subtitle"
              type="text"
              :placeholder="t('admin.lottery.create.subtitlePlaceholder')"
              class="input"
            />
          </label>
          <label class="block">
            <span class="input-label">{{ t('admin.lottery.create.prizeCount') }}</span>
            <input v-model.number="form.prizeCount" type="number" min="1" class="input font-mono" />
          </label>
          <label class="block">
            <span class="input-label">{{ t('admin.lottery.create.maxParticipants') }}</span>
            <input v-model.number="form.maxParticipants" type="number" min="1" class="input font-mono" />
          </label>
        </div>

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
            rows="6"
            :placeholder="t('admin.lottery.create.codesPlaceholder')"
            class="input mt-1 font-mono"
          ></textarea>
          <p class="input-hint">{{ t('admin.lottery.create.codesHint') }}</p>
        </div>

        <div class="mt-6 flex flex-wrap items-center gap-3">
          <button class="btn btn-primary" :disabled="!canSubmit" @click="submitCampaign">
            {{ t('admin.lottery.create.submit') }}
          </button>
          <button class="btn btn-secondary" @click="resetForm">
            {{ t('common.reset') }}
          </button>
          <button class="btn btn-ghost btn-sm" @click="openPreview = true" :disabled="!canSubmit">
            {{ t('admin.lottery.create.preview') }}
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

        <div v-if="lotteryStore.campaigns.length === 0" class="px-6 py-12 text-center">
          <div class="text-sm text-gray-500 dark:text-dark-400">
            {{ t('admin.lottery.history.empty') }}
          </div>
        </div>

        <ul v-else class="divide-y divide-gray-100 dark:divide-dark-700">
          <li v-for="c in lotteryStore.campaigns" :key="c.id" class="px-6 py-4">
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
                <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ c.createdAt }}</div>
              </div>
              <div class="flex flex-wrap items-center gap-3 text-right">
                <div class="font-mono text-xs text-gray-500 dark:text-dark-300">
                  <span class="font-bold text-gray-900 dark:text-white">{{ c.joined }}</span>/{{ c.maxParticipants }}
                  {{ t('admin.lottery.history.joined') }}
                </div>
                <div class="font-mono text-xs text-gray-500 dark:text-dark-300">
                  <span class="font-bold text-amber-600 dark:text-amber-300">{{ c.winners.length }}</span>/{{ c.prizeCount }}
                  {{ t('admin.lottery.history.won') }}
                </div>
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
                    v-for="(w, i) in c.winners"
                    :key="i"
                    class="flex items-center justify-between gap-3 rounded-lg bg-white px-3 py-2 ring-1 ring-gray-200 dark:bg-dark-900 dark:ring-dark-700"
                  >
                    <span class="text-gray-700 dark:text-dark-100">{{ w.userName }}</span>
                    <span class="font-mono text-amber-600 dark:text-amber-300">{{ w.code }}</span>
                  </li>
                  <li v-if="c.winners.length === 0" class="text-gray-400 dark:text-dark-500">
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
                    {{ code }}
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
      v-if="previewSegments && previewSegments.length > 0"
      :open="openPreview"
      :campaign-title="form.name || t('admin.lottery.create.previewTitle')"
      :subtitle="form.subtitle || t('admin.lottery.create.previewSubtitle')"
      :prize-count="form.prizeCount"
      :max-participants="form.maxParticipants"
      :joined="0"
      :segments="previewSegments"
      :draw-fn="previewDrawFn"
      @close="openPreview = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import LotteryDialog, { type DrawResult } from '@/components/lottery/LotteryDialog.vue'
import type { WheelSegment } from '@/components/lottery/LotteryWheel.vue'
import { useLotteryStore } from '@/stores/lottery'

const { t } = useI18n()
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
  codesRaw: ''
})
const formError = ref('')
const formSaved = ref(false)

const codeLines = computed(() =>
  form.value.codesRaw.split(/\r?\n/).map(s => s.trim()).filter(Boolean)
)
const canSubmit = computed(
  () =>
    form.value.name.trim().length > 0 &&
    form.value.prizeCount > 0 &&
    form.value.maxParticipants >= form.value.prizeCount &&
    codeLines.value.length >= form.value.prizeCount
)

function submitCampaign() {
  formError.value = ''
  formSaved.value = false
  if (!canSubmit.value) {
    formError.value = t('admin.lottery.create.invalid')
    return
  }
  lotteryStore.createCampaign({
    name: form.value.name,
    subtitle: form.value.subtitle,
    prizeCount: form.value.prizeCount,
    maxParticipants: form.value.maxParticipants,
    codes: codeLines.value
  })
  formSaved.value = true
}

function resetForm() {
  form.value = { name: '', subtitle: '', prizeCount: 5, maxParticipants: 20, codesRaw: '' }
  formError.value = ''
  formSaved.value = false
}

const expanded = ref<string | null>(null)
function toggleExpand(id: string) {
  expanded.value = expanded.value === id ? null : id
}

// Preview wheel (no persistence, no user impact)
const openPreview = ref(false)
const previewSegments = computed<WheelSegment[]>(() => {
  const total = Math.max(form.value.prizeCount + 2, 8)
  const segs: WheelSegment[] = []
  for (let i = 0; i < total; i++) {
    segs.push(
      i < form.value.prizeCount
        ? { label: `奖品 ${i + 1}`, isPrize: true }
        : { label: '谢谢参与', isPrize: false }
    )
  }
  return segs
})

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
    code: won ? 'PREVIEW-XXXX-XXXX' : undefined
  }
}
</script>
