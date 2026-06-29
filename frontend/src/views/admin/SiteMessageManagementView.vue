<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="inline-flex rounded-lg border border-gray-200 bg-white p-1 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <button
            v-for="tab in viewTabs"
            :key="tab.value"
            type="button"
            class="rounded-md px-4 py-2 text-sm font-semibold transition-colors"
            :class="activeView === tab.value
              ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200'
              : 'text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-white'"
            @click="activeView = tab.value"
          >
            {{ tab.label }}
          </button>
        </div>
        <span class="badge badge-primary self-start sm:self-center">
          {{ t('admin.siteMessageManagement.previewBadge') }}
        </span>
      </div>

      <template v-if="activeView === 'history'">
        <section class="grid grid-cols-1 gap-4 md:grid-cols-3">
          <div
            v-for="item in historySummaryItems"
            :key="item.label"
            class="rounded-lg border border-gray-100 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800/60"
          >
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ item.label }}</p>
                <p class="mt-1 truncate text-lg font-semibold text-gray-900 dark:text-white">
                  {{ item.value }}
                </p>
              </div>
              <div
                class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg"
                :class="item.tint"
              >
                <Icon :name="item.icon" size="md" />
              </div>
            </div>
          </div>
        </section>

        <section class="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,1.05fr)_minmax(360px,0.95fr)]">
          <div class="rounded-lg border border-gray-100 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800/60">
            <div class="mb-5 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.siteMessageManagement.history.title') }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                  {{ t('admin.siteMessageManagement.history.subtitle') }}
                </p>
              </div>
              <button type="button" class="btn btn-primary shrink-0" @click="activeView = 'new'">
                <Icon name="plus" size="sm" />
                {{ t('admin.siteMessageManagement.history.addNew') }}
              </button>
            </div>

            <div class="mb-4 grid grid-cols-1 gap-3 md:grid-cols-[minmax(0,1fr)_180px]">
              <div class="relative">
                <Icon
                  name="search"
                  size="sm"
                  class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
                />
                <input
                  v-model="historySearch"
                  class="input pl-9"
                  :placeholder="t('admin.siteMessageManagement.history.searchPlaceholder')"
                />
              </div>
              <select v-model="historyStatusFilter" class="input">
                <option value="all">{{ t('admin.siteMessageManagement.history.statusAll') }}</option>
                <option
                  v-for="status in compensationStatuses"
                  :key="status"
                  :value="status"
                >
                  {{ statusLabel(status) }}
                </option>
              </select>
            </div>

            <div class="space-y-3">
              <button
                v-for="item in filteredHistoryItems"
                :key="item.id"
                type="button"
                class="w-full rounded-lg border p-4 text-left transition-colors"
                :class="selectedHistoryId === item.id
                  ? 'border-primary-500 bg-primary-50 dark:border-primary-500 dark:bg-primary-900/20'
                  : 'border-gray-100 bg-gray-50 hover:border-primary-200 hover:bg-white dark:border-dark-700 dark:bg-dark-900/40 dark:hover:border-primary-700 dark:hover:bg-dark-800'"
                @click="selectedHistoryId = item.id"
              >
                <div class="flex items-start justify-between gap-3">
                  <div class="min-w-0">
                    <p class="truncate text-sm font-semibold text-gray-900 dark:text-white">
                      {{ item.subject }}
                    </p>
                    <p class="mt-1 font-mono text-xs text-gray-500 dark:text-dark-400">
                      {{ item.id }}
                    </p>
                  </div>
                  <span class="badge shrink-0" :class="statusBadgeClass(item.status)">
                    {{ statusLabel(item.status) }}
                  </span>
                </div>
                <div class="mt-3 flex flex-wrap gap-2 text-xs font-medium text-gray-500 dark:text-dark-400">
                  <span>{{ item.sentAt }}</span>
                  <span>{{ item.audience }}</span>
                  <span>{{ formatMoney(item.amount) }} / {{ t('admin.siteMessageManagement.history.codesCount', { count: item.codeCount }) }}</span>
                </div>
              </button>

              <div
                v-if="filteredHistoryItems.length === 0"
                class="rounded-lg border border-dashed border-gray-200 p-6 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
              >
                {{ t('admin.siteMessageManagement.history.empty') }}
              </div>
            </div>
          </div>

          <aside
            v-if="selectedHistory"
            class="rounded-lg border border-gray-100 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800/60"
          >
            <div class="mb-5 flex items-start justify-between gap-3">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.siteMessageManagement.history.detailTitle') }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                  {{ t('admin.siteMessageManagement.history.detailSubtitle', {
                    operator: selectedHistory.operator,
                    amount: formatMoney(totalCompensation(selectedHistory)),
                  }) }}
                </p>
              </div>
              <span class="badge shrink-0" :class="statusBadgeClass(selectedHistory.status)">
                {{ statusLabel(selectedHistory.status) }}
              </span>
            </div>

            <div class="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div class="rounded-lg border border-gray-100 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900/40">
                <p class="text-xs font-medium text-gray-500 dark:text-dark-400">
                  {{ t('admin.siteMessageManagement.history.audience') }}
                </p>
                <p class="mt-1 truncate text-sm font-semibold text-gray-900 dark:text-white">
                  {{ selectedHistory.audience }}
                </p>
              </div>
              <div class="rounded-lg border border-gray-100 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900/40">
                <p class="text-xs font-medium text-gray-500 dark:text-dark-400">
                  {{ t('admin.siteMessageManagement.history.amount') }}
                </p>
                <p class="mt-1 truncate text-sm font-semibold text-gray-900 dark:text-white">
                  {{ formatMoney(selectedHistory.amount) }}
                </p>
              </div>
              <div class="rounded-lg border border-gray-100 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900/40">
                <p class="text-xs font-medium text-gray-500 dark:text-dark-400">
                  {{ t('admin.siteMessageManagement.history.codes') }}
                </p>
                <p class="mt-1 truncate text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.siteMessageManagement.history.codesCount', { count: selectedHistory.codeCount }) }}
                </p>
              </div>
            </div>

            <div class="rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-600 dark:bg-dark-900/40">
              <div class="border-b border-gray-200 p-4 dark:border-dark-600">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="badge badge-primary">{{ selectedHistory.id }}</span>
                  <span class="badge" :class="selectedHistory.mode === 'all' ? 'badge-warning' : 'badge-gray'">
                    {{ historyModeLabel(selectedHistory.mode) }}
                  </span>
                </div>
                <h3 class="mt-3 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ selectedHistory.subject }}
                </h3>
                <div class="mt-3 space-y-1 text-sm text-gray-500 dark:text-dark-400">
                  <p>{{ t('admin.siteMessageManagement.history.operator', { operator: selectedHistory.operator }) }}</p>
                  <p>{{ t('admin.siteMessageManagement.history.sentAt', { time: selectedHistory.sentAt }) }}</p>
                </div>
              </div>

              <div class="p-4">
                <div
                  class="min-h-[220px] whitespace-pre-wrap rounded-lg bg-white p-4 text-sm leading-6 text-gray-800 shadow-sm dark:bg-dark-800 dark:text-gray-200"
                >
                  {{ selectedHistoryContent }}
                </div>
              </div>
            </div>

            <div class="mt-5">
              <p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.siteMessageManagement.history.codeAssignments') }}
              </p>
              <div v-if="selectedHistory.codes.length > 0" class="max-h-56 space-y-2 overflow-auto pr-1">
                <div
                  v-for="code in selectedHistory.codes"
                  :key="`${selectedHistory.id}-${code.code}-${code.recipient}`"
                  class="grid grid-cols-1 gap-2 rounded-lg border border-gray-100 bg-gray-50 px-3 py-2 text-xs dark:border-dark-700 dark:bg-dark-900/40 sm:grid-cols-[minmax(0,1fr)_minmax(160px,auto)_auto] sm:items-center"
                >
                  <span class="truncate font-medium text-gray-700 dark:text-gray-300">{{ code.recipient }}</span>
                  <span class="truncate font-mono font-semibold text-primary-700 dark:text-primary-300">{{ code.code }}</span>
                  <span class="badge justify-self-start" :class="code.status === 'used' ? 'badge-gray' : 'badge-success'">
                    {{ codeStatusLabel(code.status) }}
                  </span>
                </div>
              </div>
              <div
                v-else
                class="rounded-lg border border-dashed border-gray-200 p-4 text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
              >
                {{ t('admin.siteMessageManagement.history.noCodes') }}
              </div>
            </div>

            <div class="mt-5 flex flex-wrap items-center justify-end gap-3 border-t border-gray-100 pt-5 dark:border-dark-700">
              <button type="button" class="btn btn-secondary" @click="copySelectedHistory">
                <Icon name="copy" size="sm" />
                {{ t('admin.siteMessageManagement.history.copyContent') }}
              </button>
              <button type="button" class="btn btn-secondary" @click="loadHistoryAsDraft">
                <Icon name="refresh" size="sm" />
                {{ t('admin.siteMessageManagement.history.resend') }}
              </button>
            </div>
          </aside>
        </section>
      </template>

      <template v-else>
        <section class="grid grid-cols-1 gap-4 md:grid-cols-3">
          <div
            v-for="item in composeSummaryItems"
            :key="item.label"
            class="rounded-lg border border-gray-100 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800/60"
          >
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ item.label }}</p>
                <p class="mt-1 truncate text-lg font-semibold text-gray-900 dark:text-white">
                  {{ item.value }}
                </p>
              </div>
              <div
                class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg"
                :class="item.tint"
              >
                <Icon :name="item.icon" size="md" />
              </div>
            </div>
          </div>
        </section>

        <section class="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,1.05fr)_minmax(360px,0.95fr)]">
          <form
            class="rounded-lg border border-gray-100 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800/60"
            @submit.prevent="handlePreviewSend"
          >
            <div class="mb-5 flex items-center justify-between gap-3">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.siteMessageManagement.composeTitle') }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                  {{ t('admin.siteMessageManagement.composeSubtitle') }}
                </p>
              </div>
              <span class="badge badge-primary">{{ t('admin.siteMessageManagement.previewBadge') }}</span>
            </div>

            <div class="space-y-5">
              <div>
                <label class="input-label">{{ t('admin.siteMessageManagement.recipientMode') }}</label>
                <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <button
                    v-for="mode in recipientModes"
                    :key="mode.value"
                    type="button"
                    class="min-h-[88px] rounded-lg border p-4 text-left transition-colors"
                    :class="recipientMode === mode.value
                      ? 'border-primary-500 bg-primary-50 text-primary-900 dark:border-primary-500 dark:bg-primary-900/20 dark:text-primary-100'
                      : 'border-gray-200 bg-white text-gray-700 hover:border-primary-200 hover:bg-primary-50/40 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:border-primary-700 dark:hover:bg-primary-900/10'"
                    :aria-pressed="recipientMode === mode.value"
                    @click="recipientMode = mode.value"
                  >
                    <span class="flex items-center gap-2 font-medium">
                      <Icon :name="mode.icon" size="sm" />
                      {{ mode.label }}
                    </span>
                    <span class="mt-2 block text-sm text-gray-500 dark:text-dark-400">
                      {{ mode.description }}
                    </span>
                  </button>
                </div>
              </div>

              <div v-if="recipientMode === 'selected'">
                <div class="mb-1.5 flex items-center justify-between gap-3">
                  <label class="input-label mb-0">{{ t('admin.siteMessageManagement.recipientList') }}</label>
                  <button
                    type="button"
                    class="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
                    @click="recipientInput = ''"
                  >
                    {{ t('admin.siteMessageManagement.clear') }}
                  </button>
                </div>
                <textarea
                  v-model="recipientInput"
                  rows="7"
                  class="input resize-none"
                  :placeholder="t('admin.siteMessageManagement.recipientPlaceholder')"
                ></textarea>
                <div class="mt-2 flex flex-wrap items-center gap-2">
                  <span class="badge badge-gray">
                    {{ t('admin.siteMessageManagement.parsedRecipients', { count: recipientEmails.length }) }}
                  </span>
                  <span v-if="invalidRecipientCount > 0" class="badge badge-warning">
                    {{ t('admin.siteMessageManagement.invalidRecipients', { count: invalidRecipientCount }) }}
                  </span>
                </div>
              </div>

              <div>
                <label class="input-label">{{ t('admin.siteMessageManagement.subject') }}</label>
                <input
                  v-model="messageSubject"
                  type="text"
                  class="input"
                  :placeholder="t('admin.siteMessageManagement.subjectPlaceholder')"
                />
              </div>

              <div>
                <label class="input-label">{{ t('admin.siteMessageManagement.content') }}</label>
                <textarea
                  v-model="messageContent"
                  rows="8"
                  class="input resize-none"
                  :placeholder="t('admin.siteMessageManagement.contentPlaceholder')"
                ></textarea>
              </div>

              <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
                <div class="flex items-center justify-between gap-4">
                  <div>
                    <label class="text-sm font-medium text-gray-800 dark:text-gray-200">
                      {{ t('admin.siteMessageManagement.compensation') }}
                    </label>
                    <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                      {{ t('admin.siteMessageManagement.compensationHint') }}
                    </p>
                  </div>
                  <Toggle v-model="compensationEnabled" />
                </div>

                <div v-if="compensationEnabled" class="mt-4 space-y-4">
                  <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
                    <div>
                      <label class="input-label">{{ t('admin.siteMessageManagement.compensationAmount') }}</label>
                      <input
                        v-model.number="compensationAmount"
                        type="number"
                        min="0.01"
                        step="0.01"
                        class="input"
                      />
                      <p class="input-hint">{{ t('admin.siteMessageManagement.compensationAmountHint') }}</p>
                    </div>
                    <div>
                      <label class="input-label">{{ t('admin.siteMessageManagement.compensationFormat') }}</label>
                      <div class="grid grid-cols-2 gap-2">
                        <button
                          v-for="format in compensationFormats"
                          :key="format.value"
                          type="button"
                          class="rounded-lg border px-3 py-2 text-sm font-medium transition-colors"
                          :class="compensationFormat === format.value
                            ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-900/20 dark:text-primary-300'
                            : 'border-gray-200 text-gray-600 hover:bg-gray-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700'"
                          @click="compensationFormat = format.value"
                        >
                          {{ format.label }}
                        </button>
                      </div>
                    </div>
                  </div>

                  <div>
                    <div class="mb-1.5 flex items-center justify-between gap-3">
                      <label class="input-label mb-0">{{ t('admin.siteMessageManagement.generatedCodes') }}</label>
                      <router-link
                        to="/admin/redeem"
                        class="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
                      >
                        {{ t('admin.siteMessageManagement.openRedeemManagement') }}
                      </router-link>
                    </div>
                    <textarea
                      v-model="compensationCodesInput"
                      rows="5"
                      class="input resize-none font-mono"
                      :placeholder="t('admin.siteMessageManagement.generatedCodesPlaceholder')"
                    ></textarea>
                    <div class="mt-2 flex flex-wrap items-center gap-2">
                      <span class="badge badge-gray">
                        {{ t('admin.siteMessageManagement.preparedCodes', { count: compensationCodes.length }) }}
                      </span>
                      <span v-if="needsMoreCompensationCodes" class="badge badge-warning">
                        {{ t('admin.siteMessageManagement.codesShortage', { count: compensationCodeShortage }) }}
                      </span>
                      <span v-else-if="compensationCodes.length > 0" class="badge badge-success">
                        {{ t('admin.siteMessageManagement.codesReady') }}
                      </span>
                    </div>
                    <p v-if="recipientMode === 'all'" class="input-hint">
                      {{ t('admin.siteMessageManagement.allUsersCodesHint') }}
                    </p>
                  </div>
                </div>
              </div>

              <div class="flex flex-wrap items-center justify-end gap-3 border-t border-gray-100 pt-5 dark:border-dark-700">
                <button type="button" class="btn btn-secondary" @click="resetDraft">
                  <Icon name="refresh" size="sm" />
                  {{ t('admin.siteMessageManagement.reset') }}
                </button>
                <button type="submit" class="btn btn-primary" :disabled="sendDisabled">
                  <Icon name="mail" size="sm" />
                  {{ t('admin.siteMessageManagement.previewSend') }}
                </button>
              </div>
            </div>
          </form>

          <aside
            class="rounded-lg border border-gray-100 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800/60"
          >
            <div class="mb-5 flex items-center justify-between gap-3">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.siteMessageManagement.previewTitle') }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                  {{ t('admin.siteMessageManagement.previewSubtitle') }}
                </p>
              </div>
              <span
                class="inline-flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-900/20 dark:text-primary-300"
              >
                <Icon name="inbox" size="md" />
              </span>
            </div>

            <div class="rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-600 dark:bg-dark-900/40">
              <div class="border-b border-gray-200 p-4 dark:border-dark-600">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="badge" :class="recipientMode === 'all' ? 'badge-warning' : 'badge-primary'">
                    {{ recipientModeLabel }}
                  </span>
                  <span v-if="compensationEnabled" class="badge badge-success">
                    {{ t('admin.siteMessageManagement.preparedRedeemCodes') }}
                  </span>
                </div>
                <h3 class="mt-3 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ previewSubject }}
                </h3>
                <div class="mt-3 space-y-1 text-sm text-gray-500 dark:text-dark-400">
                  <p>{{ t('admin.siteMessageManagement.fromOfficial') }}</p>
                  <p>{{ t('admin.siteMessageManagement.toPreview', { recipient: previewRecipientLabel }) }}</p>
                </div>
              </div>

              <div class="p-4">
                <div
                  class="min-h-[260px] whitespace-pre-wrap rounded-lg bg-white p-4 text-sm leading-6 text-gray-800 shadow-sm dark:bg-dark-800 dark:text-gray-200"
                >
                  {{ finalMessageContent }}
                </div>
              </div>
            </div>

            <div class="mt-5 space-y-3">
              <div class="flex items-center justify-between gap-3 rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-900/50">
                <span class="text-sm text-gray-500 dark:text-dark-400">
                  {{ t('admin.siteMessageManagement.previewRecipients') }}
                </span>
                <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ recipientCountLabel }}</span>
              </div>
              <div class="flex items-center justify-between gap-3 rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-900/50">
                <span class="text-sm text-gray-500 dark:text-dark-400">
                  {{ t('admin.siteMessageManagement.previewCompensation') }}
                </span>
                <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ compensationSummary }}</span>
              </div>
            </div>

            <div v-if="recipientMode === 'selected'" class="mt-5">
              <p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.siteMessageManagement.selectedRecipients') }}
              </p>
              <div class="flex flex-wrap gap-2">
                <span
                  v-for="email in visibleRecipientEmails"
                  :key="email"
                  class="rounded-full bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300"
                >
                  {{ email }}
                </span>
                <span
                  v-if="remainingRecipientCount > 0"
                  class="rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300"
                >
                  {{ t('admin.siteMessageManagement.moreRecipients', { count: remainingRecipientCount }) }}
                </span>
              </div>
            </div>
          </aside>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'

type ViewMode = 'history' | 'new'
type RecipientMode = 'selected' | 'all'
type CompensationFormat = 'block' | 'compact'
type CompensationStatus = 'sent' | 'sending' | 'cancelled'
type CodeStatus = 'unused' | 'used' | 'reserved' | 'recorded'
type HistoryStatusFilter = 'all' | CompensationStatus
type IconName = 'users' | 'mail' | 'gift' | 'globe' | 'inbox' | 'document' | 'dollar' | 'copy'

interface CompensationCode {
  recipient: string
  code: string
  status: CodeStatus
}

interface CompensationHistoryItem {
  id: string
  subject: string
  content: string
  status: CompensationStatus
  mode: RecipientMode
  audience: string
  recipientCount: number
  amount: number
  codeCount: number
  operator: string
  sentAt: string
  codes: CompensationCode[]
}

const { t } = useI18n()
const appStore = useAppStore()

const activeView = ref<ViewMode>('history')
const recipientMode = ref<RecipientMode>('selected')
const recipientInput = ref('alice@example.com\nbob@example.com\nvip-user@example.com')
const messageSubject = ref('')
const messageContent = ref('')
const compensationEnabled = ref(true)
const compensationAmount = ref(5)
const compensationFormat = ref<CompensationFormat>('block')
const compensationCodesInput = ref('9f3a8c1e4b6d0a72c5f9183e2d7b4a60\n62d4f0a9c1e83b57a6d2f49b0c7e5a31\nc8a137f06e4b92d5a0f6c31e7d9b4825')
const historySearch = ref('')
const historyStatusFilter = ref<HistoryStatusFilter>('all')
const selectedHistoryId = ref('CMP-20260629-001')

const historyItems = ref<CompensationHistoryItem[]>([
  {
    id: 'CMP-20260629-001',
    subject: '服务补偿通知',
    content: '您好，\n\n感谢您近期的理解与支持。我们已为本次服务波动准备了一份余额补偿，请在站内信末尾查看兑换码。',
    status: 'sent',
    mode: 'selected',
    audience: '指定 3 个用户',
    recipientCount: 3,
    amount: 5,
    codeCount: 3,
    operator: 'admin@lumioapi.com',
    sentAt: '2026-06-29 09:20',
    codes: [
      { recipient: 'alice@example.com', code: '9f3a8c1e4b6d0a72c5f9183e2d7b4a60', status: 'unused' },
      { recipient: 'bob@example.com', code: '62d4f0a9c1e83b57a6d2f49b0c7e5a31', status: 'used' },
      { recipient: 'vip-user@example.com', code: 'c8a137f06e4b92d5a0f6c31e7d9b4825', status: 'unused' },
    ],
  },
  {
    id: 'CMP-20260625-004',
    subject: '模型延迟补偿说明',
    content: '您好，\n\n6 月 25 日部分模型请求出现延迟。我们已为受影响时段准备余额补偿，兑换码随站内信发放。',
    status: 'sent',
    mode: 'all',
    audience: '全部用户',
    recipientCount: 26,
    amount: 3,
    codeCount: 26,
    operator: 'admin@lumioapi.com',
    sentAt: '2026-06-25 18:40',
    codes: [
      { recipient: '全员批次 1', code: 'b7613a04c9d84f2180e7a6c5f3b2d910', status: 'unused' },
      { recipient: '全员批次 2', code: '0fd91e2a7c6b4385a4e019f8d2c5736b', status: 'reserved' },
      { recipient: '全员批次 3', code: 'd53b0c1a6e8f49a2b7c3081f95e64d20', status: 'used' },
      { recipient: '还有 23 个兑换码', code: '后端按用户逐个分配', status: 'recorded' },
    ],
  },
  {
    id: 'CMP-20260618-002',
    subject: '维护窗口补偿',
    content: '您好，\n\n本次维护窗口比预计时间更长。我们为受影响用户发放 8 元余额补偿，请在站内信末尾复制兑换码使用。',
    status: 'sent',
    mode: 'selected',
    audience: '指定 6 个用户',
    recipientCount: 6,
    amount: 8,
    codeCount: 6,
    operator: 'ops@lumioapi.com',
    sentAt: '2026-06-18 23:10',
    codes: [
      { recipient: 'team-a@example.com', code: '4d7a91c063f54a2eb8d0c1359f6e72ab', status: 'used' },
      { recipient: 'team-b@example.com', code: '18a03f9c52e64d7b8f1a6c0d93e245ab', status: 'used' },
      { recipient: 'team-c@example.com', code: '8e6f21b0a5c94d37b2f8016a3c9d74e5', status: 'unused' },
    ],
  },
  {
    id: 'CMP-20260612-001',
    subject: '小额体验补偿',
    content: '该批次在发送前取消，未占用兑换码。',
    status: 'cancelled',
    mode: 'selected',
    audience: '指定 3 个用户',
    recipientCount: 3,
    amount: 2,
    codeCount: 0,
    operator: 'admin@lumioapi.com',
    sentAt: '2026-06-12 11:06',
    codes: [],
  },
])

const viewTabs = computed<Array<{ value: ViewMode; label: string }>>(() => [
  { value: 'history', label: t('admin.siteMessageManagement.tabs.history') },
  { value: 'new', label: t('admin.siteMessageManagement.tabs.new') },
])

const compensationStatuses: CompensationStatus[] = ['sent', 'sending', 'cancelled']

const recipientModes = computed<Array<{ value: RecipientMode; label: string; description: string; icon: IconName }>>(() => [
  {
    value: 'selected',
    label: t('admin.siteMessageManagement.mode.selected'),
    description: t('admin.siteMessageManagement.mode.selectedDescription'),
    icon: 'users',
  },
  {
    value: 'all',
    label: t('admin.siteMessageManagement.mode.all'),
    description: t('admin.siteMessageManagement.mode.allDescription'),
    icon: 'globe',
  },
])

const compensationFormats = computed<Array<{ value: CompensationFormat; label: string }>>(() => [
  { value: 'block', label: t('admin.siteMessageManagement.format.block') },
  { value: 'compact', label: t('admin.siteMessageManagement.format.compact') },
])

const recipientTokens = computed(() =>
  recipientInput.value
    .split(/[\s,;]+/)
    .map((item) => item.trim())
    .filter(Boolean),
)

const recipientEmails = computed(() => {
  const seen = new Set<string>()
  const emails: string[] = []
  for (const token of recipientTokens.value) {
    const email = token.toLowerCase()
    if (isEmail(email) && !seen.has(email)) {
      seen.add(email)
      emails.push(email)
    }
  }
  return emails
})

const invalidRecipientCount = computed(() =>
  recipientTokens.value.filter((token) => !isEmail(token.toLowerCase())).length,
)

const visibleRecipientEmails = computed(() => recipientEmails.value.slice(0, 8))
const remainingRecipientCount = computed(() => Math.max(recipientEmails.value.length - visibleRecipientEmails.value.length, 0))

const normalizedCompensationAmount = computed(() => {
  const amount = Number(compensationAmount.value)
  return Number.isFinite(amount) && amount > 0 ? amount : 0
})

const formattedCompensationAmount = computed(() => currency(normalizedCompensationAmount.value))

const compensationCodes = computed(() => {
  const seen = new Set<string>()
  const codes: string[] = []
  for (const token of compensationCodesInput.value.split(/[\s,;]+/)) {
    const code = token.trim()
    if (code && !seen.has(code)) {
      seen.add(code)
      codes.push(code)
    }
  }
  return codes
})

const requiredCompensationCodeCount = computed(() =>
  recipientMode.value === 'selected' ? recipientEmails.value.length : 0,
)

const compensationCodeShortage = computed(() =>
  compensationEnabled.value
    ? Math.max(requiredCompensationCodeCount.value - compensationCodes.value.length, 0)
    : 0,
)

const needsMoreCompensationCodes = computed(() => compensationCodeShortage.value > 0)

const previewSubject = computed(() =>
  messageSubject.value.trim() || t('admin.siteMessageManagement.defaultSubject'),
)

const baseContent = computed(() =>
  messageContent.value.trim() || t('admin.siteMessageManagement.defaultContent'),
)

const compensationBlock = computed(() => {
  if (!compensationEnabled.value || normalizedCompensationAmount.value <= 0) {
    return ''
  }

  const params = {
    amount: formattedCompensationAmount.value,
    code: compensationCodes.value[0] || t('admin.siteMessageManagement.redeemCodePlaceholder'),
  }
  return compensationFormat.value === 'compact'
    ? t('admin.siteMessageManagement.compensationCompact', params)
    : t('admin.siteMessageManagement.compensationBlock', params)
})

const finalMessageContent = computed(() =>
  [baseContent.value, compensationBlock.value].filter(Boolean).join('\n\n'),
)

const recipientModeLabel = computed(() =>
  recipientMode.value === 'all'
    ? t('admin.siteMessageManagement.mode.all')
    : t('admin.siteMessageManagement.mode.selected'),
)

const previewRecipientLabel = computed(() => {
  if (recipientMode.value === 'all') {
    return t('admin.siteMessageManagement.allUsers')
  }
  if (recipientEmails.value.length === 1) {
    return recipientEmails.value[0]
  }
  return t('admin.siteMessageManagement.selectedUsersCount', { count: recipientEmails.value.length })
})

const recipientCountLabel = computed(() =>
  recipientMode.value === 'all'
    ? t('admin.siteMessageManagement.allUsers')
    : t('admin.siteMessageManagement.countUsers', { count: recipientEmails.value.length }),
)

const compensationSummary = computed(() =>
  compensationEnabled.value && normalizedCompensationAmount.value > 0
    ? t('admin.siteMessageManagement.compensationPreparedSummary', {
        amount: formattedCompensationAmount.value,
        count: compensationCodes.value.length,
      })
    : t('common.disabled'),
)

const composeSummaryItems = computed<Array<{ label: string; value: string; icon: IconName; tint: string }>>(() => [
  {
    label: t('admin.siteMessageManagement.summary.target'),
    value: recipientCountLabel.value,
    icon: recipientMode.value === 'all' ? 'globe' : 'users',
    tint: 'bg-blue-50 text-blue-600 dark:bg-blue-900/20 dark:text-blue-300',
  },
  {
    label: t('admin.siteMessageManagement.summary.subject'),
    value: previewSubject.value,
    icon: 'mail',
    tint: 'bg-primary-50 text-primary-600 dark:bg-primary-900/20 dark:text-primary-300',
  },
  {
    label: t('admin.siteMessageManagement.summary.compensation'),
    value: compensationSummary.value,
    icon: 'gift',
    tint: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-300',
  },
])

const historySummaryItems = computed<Array<{ label: string; value: string; icon: IconName; tint: string }>>(() => [
  {
    label: t('admin.siteMessageManagement.history.summary.batches'),
    value: t('admin.siteMessageManagement.history.batchesCount', { count: historyItems.value.length }),
    icon: 'document',
    tint: 'bg-blue-50 text-blue-600 dark:bg-blue-900/20 dark:text-blue-300',
  },
  {
    label: t('admin.siteMessageManagement.history.summary.amount'),
    value: formatMoney(historyItems.value.reduce((sum, item) => sum + totalCompensation(item), 0)),
    icon: 'dollar',
    tint: 'bg-primary-50 text-primary-600 dark:bg-primary-900/20 dark:text-primary-300',
  },
  {
    label: t('admin.siteMessageManagement.history.summary.codes'),
    value: t('admin.siteMessageManagement.history.codesCount', {
      count: historyItems.value.reduce((sum, item) => sum + item.codeCount, 0),
    }),
    icon: 'gift',
    tint: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-300',
  },
])

const filteredHistoryItems = computed(() => {
  const keyword = historySearch.value.trim().toLowerCase()
  return historyItems.value.filter((item) => {
    const statusMatched = historyStatusFilter.value === 'all' || item.status === historyStatusFilter.value
    const searchable = [
      item.id,
      item.subject,
      item.content,
      item.operator,
      item.audience,
      ...item.codes.map((code) => `${code.recipient} ${code.code}`),
    ].join(' ').toLowerCase()
    return statusMatched && (!keyword || searchable.includes(keyword))
  })
})

const selectedHistory = computed(() => {
  const filteredSelected = filteredHistoryItems.value.find((item) => item.id === selectedHistoryId.value)
  const globalSelected = historyItems.value.find((item) => item.id === selectedHistoryId.value)
  return filteredSelected || filteredHistoryItems.value[0] || globalSelected || historyItems.value[0]
})

const selectedHistoryContent = computed(() => {
  const item = selectedHistory.value
  if (!item) return ''
  const code = item.codes[0]?.code || t('admin.siteMessageManagement.redeemCodePlaceholder')
  const compensation = item.codeCount > 0
    ? t('admin.siteMessageManagement.compensationBlock', {
        amount: currency(item.amount),
        code,
      })
    : ''
  return [item.content, compensation].filter(Boolean).join('\n\n')
})

const sendDisabled = computed(() => {
  if (recipientMode.value === 'selected' && recipientEmails.value.length === 0) {
    return true
  }
  if (compensationEnabled.value) {
    return compensationCodes.value.length === 0 || needsMoreCompensationCodes.value
  }
  return false
})

function isEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)
}

function currency(value: number): string {
  const amount = Number.isFinite(value) ? value : 0
  return amount.toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
}

function formatMoney(value: number): string {
  return t('admin.siteMessageManagement.money', { amount: currency(value) })
}

function totalCompensation(item: CompensationHistoryItem): number {
  if (item.status === 'cancelled') {
    return 0
  }
  return item.amount * item.codeCount
}

function statusLabel(status: CompensationStatus): string {
  return t(`admin.siteMessageManagement.history.status.${status}`)
}

function statusBadgeClass(status: CompensationStatus): string {
  if (status === 'sent') return 'badge-success'
  if (status === 'sending') return 'badge-warning'
  return 'badge-danger'
}

function codeStatusLabel(status: CodeStatus): string {
  return t(`admin.siteMessageManagement.history.codeStatus.${status}`)
}

function historyModeLabel(mode: RecipientMode): string {
  return mode === 'all'
    ? t('admin.siteMessageManagement.mode.all')
    : t('admin.siteMessageManagement.mode.selected')
}

function resetDraft() {
  recipientMode.value = 'selected'
  recipientInput.value = 'alice@example.com\nbob@example.com\nvip-user@example.com'
  messageSubject.value = ''
  messageContent.value = ''
  compensationEnabled.value = true
  compensationAmount.value = 5
  compensationFormat.value = 'block'
  compensationCodesInput.value = '9f3a8c1e4b6d0a72c5f9183e2d7b4a60\n62d4f0a9c1e83b57a6d2f49b0c7e5a31\nc8a137f06e4b92d5a0f6c31e7d9b4825'
}

function buildHistoryItemFromDraft(): CompensationHistoryItem {
  const now = new Date()
  const dateStamp = [
    now.getFullYear(),
    String(now.getMonth() + 1).padStart(2, '0'),
    String(now.getDate()).padStart(2, '0'),
  ].join('')
  const time = `${dateStamp.slice(0, 4)}-${dateStamp.slice(4, 6)}-${dateStamp.slice(6, 8)} ${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`
  const id = `CMP-${dateStamp}-${String(historyItems.value.length + 1).padStart(3, '0')}`
  const draftCodes = compensationEnabled.value ? compensationCodes.value : []
  const codes = recipientMode.value === 'all'
    ? draftCodes.slice(0, 12).map((code, index) => ({
        recipient: t('admin.siteMessageManagement.history.allUsersBatch', { index: index + 1 }),
        code,
        status: 'unused' as CodeStatus,
      }))
    : recipientEmails.value.map((recipient, index) => ({
        recipient,
        code: draftCodes[index] || t('admin.siteMessageManagement.history.missingCode'),
        status: 'unused' as CodeStatus,
      }))

  return {
    id,
    subject: previewSubject.value,
    content: baseContent.value,
    status: 'sent',
    mode: recipientMode.value,
    audience: recipientMode.value === 'all'
      ? t('admin.siteMessageManagement.allUsers')
      : t('admin.siteMessageManagement.selectedUsersCount', { count: recipientEmails.value.length }),
    recipientCount: recipientMode.value === 'all' ? draftCodes.length : recipientEmails.value.length,
    amount: normalizedCompensationAmount.value,
    codeCount: draftCodes.length,
    operator: 'admin@lumioapi.com',
    sentAt: time,
    codes,
  }
}

function handlePreviewSend() {
  if (sendDisabled.value) {
    appStore.showError(t('admin.siteMessageManagement.previewBlocked'))
    return
  }
  const item = buildHistoryItemFromDraft()
  historyItems.value.unshift(item)
  selectedHistoryId.value = item.id
  activeView.value = 'history'
  appStore.showSuccess(t('admin.siteMessageManagement.history.addedToast'))
}

function loadHistoryAsDraft() {
  const item = selectedHistory.value
  if (!item) return
  recipientMode.value = item.mode
  recipientInput.value = item.mode === 'all'
    ? ''
    : item.codes.map((code) => code.recipient).filter(isEmail).join('\n')
  messageSubject.value = item.subject
  messageContent.value = item.content
  compensationEnabled.value = item.codeCount > 0
  compensationAmount.value = item.amount || 5
  compensationFormat.value = 'block'
  compensationCodesInput.value = item.codes
    .map((code) => code.code)
    .filter((code) => /^[a-f0-9]{32}$/i.test(code))
    .join('\n')
  activeView.value = 'new'
  appStore.showInfo(t('admin.siteMessageManagement.history.loadedAsDraftToast'))
}

async function copySelectedHistory() {
  const item = selectedHistory.value
  if (!item) return
  const text = [item.subject, selectedHistoryContent.value].join('\n\n')
  try {
    await navigator.clipboard?.writeText(text)
    appStore.showSuccess(t('admin.siteMessageManagement.history.copiedToast'))
  } catch {
    appStore.showInfo(t('admin.siteMessageManagement.history.copyFallbackToast'))
  }
}
</script>
