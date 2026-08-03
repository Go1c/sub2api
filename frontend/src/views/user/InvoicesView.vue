<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="grid gap-4 md:grid-cols-3">
        <div class="card p-5">
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('invoice.totalRecharged') }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
            {{ formatMoney(overview.total_recharged) }}
          </p>
        </div>
        <div class="card p-5">
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('invoice.usedInvoiceAmount') }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
            {{ formatMoney(overview.used_invoice_amount) }}
          </p>
        </div>
        <div class="card p-5">
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('invoice.remainingAmount') }}</p>
          <p class="mt-2 text-2xl font-semibold text-primary-600 dark:text-primary-400">
            {{ formatMoney(overview.remaining_amount) }}
          </p>
        </div>
      </div>

      <div
        v-if="createdOrderNo"
        class="rounded-lg border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-800 dark:border-emerald-800/60 dark:bg-emerald-900/20 dark:text-emerald-200"
      >
        <span>{{ t('invoice.createdOrder') }}</span>
        <span class="ml-2 font-mono font-semibold">{{ createdOrderNo }}</span>
      </div>

      <div class="grid gap-4 lg:grid-cols-[minmax(0,420px)_1fr]">
        <div class="card p-5">
          <div class="mb-4 flex items-center justify-between gap-3">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('invoice.applyTitle') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('invoice.mailDeliveryHint') }}</p>
            </div>
            <Icon name="document" size="lg" class="text-primary-500" />
          </div>

          <form class="space-y-4" @submit.prevent="handleSubmit">
            <div>
              <label class="input-label">{{ t('invoice.form.title') }}</label>
              <input v-model.trim="form.title" type="text" class="input" :placeholder="t('invoice.form.titlePlaceholder')" />
            </div>
            <div>
              <label class="input-label">{{ t('invoice.form.taxNumber') }}</label>
              <input v-model.trim="form.tax_number" type="text" class="input" :placeholder="t('invoice.form.taxNumberPlaceholder')" />
            </div>
            <div>
              <label class="input-label">{{ t('invoice.form.amount') }}</label>
              <input v-model.number="form.amount" type="number" :min="MIN_INVOICE_AMOUNT" step="0.01" class="input" />
              <p class="input-hint">{{ t('invoice.form.amountHint', { amount: MIN_INVOICE_AMOUNT }) }}</p>
            </div>
            <div>
              <label class="input-label">{{ t('invoice.form.email') }}</label>
              <input v-model.trim="form.recipient_email" type="email" class="input" :placeholder="t('invoice.form.emailPlaceholder')" />
            </div>
            <div class="flex flex-col gap-2 sm:flex-row">
              <button
                type="button"
                class="btn btn-secondary w-full sm:w-auto sm:flex-1"
                :disabled="loading || !lastCompletedInvoice"
                @click="fillFromLastSuccess"
              >
                {{ t('invoice.fillLastSuccess') }}
              </button>
              <button type="submit" class="btn btn-primary w-full sm:flex-1" :disabled="submitting || loading">
                <Icon v-if="!submitting" name="check" size="sm" class="mr-2" />
                {{ submitting ? t('common.submitting') : t('invoice.submit') }}
              </button>
            </div>
          </form>
        </div>

        <div class="card p-5">
          <div class="flex items-start gap-3">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
              <Icon name="infoCircle" size="md" />
            </div>
            <div class="space-y-3 text-sm text-gray-600 dark:text-dark-300">
              <p>{{ t('invoice.minimumAmountRule', { amount: MIN_INVOICE_AMOUNT }) }}</p>
              <p>{{ t('invoice.quotaRule') }}</p>
              <p>{{ t('invoice.taxRule') }}</p>
              <p>{{ t('invoice.readyRule') }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 p-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('invoice.history') }}</h2>
          <button class="btn btn-secondary" :disabled="loading" @click="reload">
            <Icon name="refresh" size="sm" class="mr-2" :class="loading ? 'animate-spin' : ''" />
            {{ t('common.refresh') }}
          </button>
        </div>

        <DataTable :columns="columns" :data="items" :loading="loading" row-key="id" :actions-count="1">
          <template #cell-order_no="{ row }">
            <span class="font-mono text-sm font-medium">{{ row.order_no }}</span>
          </template>
          <template #cell-amount="{ value }">
            <span>{{ formatMoney(value) }}</span>
          </template>
          <template #cell-tax_amount="{ row }">
            <span v-if="row.status === 'completed'">{{ formatMoney(row.tax_amount) }}</span>
            <span v-else class="text-gray-400 dark:text-dark-500">-</span>
          </template>
          <template #cell-status="{ value }">
            <span :class="['inline-flex rounded-full px-2 py-1 text-xs font-medium', statusClass(value)]">
              {{ statusLabel(value) }}
            </span>
          </template>
          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-600 dark:text-dark-300">{{ formatDateTime(value) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <button
              v-if="row.status === 'completed' && row.has_file"
              class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20"
              :disabled="downloadingId === row.id"
              @click="downloadInvoice(row)"
            >
              <Icon name="download" size="sm" />
              {{ t('invoice.download') }}
            </button>
            <span v-else class="text-xs text-gray-400 dark:text-dark-500">-</span>
          </template>
        </DataTable>
      </div>

      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { invoicesAPI, type InvoiceOverview, type InvoiceRequest, type InvoiceStatus } from '@/api/invoices'
import { useAppStore } from '@/stores/app'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatCurrency, formatDateTime } from '@/utils/format'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()
const MIN_INVOICE_AMOUNT = 100

const overview = ref<InvoiceOverview>({
  total_recharged: 0,
  used_invoice_amount: 0,
  remaining_amount: 0,
  enabled: true,
})
const items = ref<InvoiceRequest[]>([])
const loading = ref(false)
const submitting = ref(false)
const downloadingId = ref<number | null>(null)
const createdOrderNo = ref('')
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const form = reactive({
  title: '',
  tax_number: '',
  amount: 0,
  recipient_email: '',
})

const columns = computed<Column[]>(() => [
  { key: 'order_no', label: t('invoice.columns.orderNo') },
  { key: 'title', label: t('invoice.columns.title') },
  { key: 'amount', label: t('invoice.columns.amount') },
  { key: 'tax_amount', label: t('invoice.columns.taxAmount') },
  { key: 'recipient_email', label: t('invoice.columns.email') },
  { key: 'status', label: t('invoice.columns.status') },
  { key: 'created_at', label: t('invoice.columns.createdAt') },
  { key: 'actions', label: t('common.actions') },
])

/** Latest completed invoice on the current page (list is created_at DESC). */
const lastCompletedInvoice = computed(() =>
  items.value.find((item) => item.status === 'completed') ?? null,
)

function formatMoney(value: number | null | undefined): string {
  return formatCurrency(value ?? 0, 'USD')
}

function fillFromLastSuccess() {
  const last = lastCompletedInvoice.value
  if (!last) {
    appStore.showWarning(t('invoice.noCompletedHistory'))
    return
  }
  form.title = last.title
  form.tax_number = last.tax_number
  form.recipient_email = last.recipient_email
  // Intentionally do not copy amount — each request needs a fresh amount.
  appStore.showSuccess(t('invoice.fillLastSuccessDone'))
}

function statusLabel(status: InvoiceStatus): string {
  return t(`invoice.status.${status}`)
}

function statusClass(status: InvoiceStatus): string {
  if (status === 'completed') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (status === 'failed') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
}

async function loadOverview() {
  overview.value = await invoicesAPI.getOverview()
}

async function loadItems() {
  const response = await invoicesAPI.list(pagination.page, pagination.page_size)
  items.value = response.items || []
  pagination.total = response.total || 0
}

async function reload() {
  loading.value = true
  try {
    await Promise.all([loadOverview(), loadItems()])
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('invoice.failedToLoad')))
  } finally {
    loading.value = false
  }
}

async function handleSubmit() {
  if (!form.title || !form.tax_number || !form.recipient_email || !form.amount) {
    appStore.showError(t('invoice.formRequired'))
    return
  }
  if (form.amount <= 0) {
    appStore.showError(t('invoice.amountInvalid'))
    return
  }
  if (form.amount < MIN_INVOICE_AMOUNT) {
    appStore.showError(t('invoice.amountTooLow', { amount: MIN_INVOICE_AMOUNT }))
    return
  }
  if (form.amount > overview.value.remaining_amount) {
    appStore.showError(t('invoice.amountExceeded'))
    return
  }

  submitting.value = true
  try {
    const created = await invoicesAPI.create({
      title: form.title,
      tax_number: form.tax_number,
      amount: form.amount,
      recipient_email: form.recipient_email,
    })
    createdOrderNo.value = created.order_no
    form.title = ''
    form.tax_number = ''
    form.amount = 0
    form.recipient_email = ''
    appStore.showSuccess(t('invoice.createSuccess'))
    await reload()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('invoice.createFailed')))
  } finally {
    submitting.value = false
  }
}

function saveBlob(blob: Blob, fileName: string) {
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
}

async function downloadInvoice(row: InvoiceRequest) {
  downloadingId.value = row.id
  try {
    const blob = await invoicesAPI.download(row.id)
    saveBlob(blob, row.file_name || `${row.order_no}.pdf`)
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('invoice.downloadFailed')))
  } finally {
    downloadingId.value = null
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadItems()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadItems()
}

onMounted(() => {
  void reload()
})
</script>
