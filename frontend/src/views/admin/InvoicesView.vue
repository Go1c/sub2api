<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-4">
        <div class="flex flex-wrap items-center gap-3">
          <div class="relative w-full md:w-72">
            <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              v-model="filters.search"
              type="text"
              class="input pl-10"
              :placeholder="t('admin.invoice.searchPlaceholder')"
              @input="debounceLoad"
            />
          </div>
          <Select v-model="filters.status" :options="statusOptions" class="w-full sm:w-40" @change="loadItems" />
          <input
            v-model.number="filters.user_id"
            type="number"
            min="1"
            class="input w-full sm:w-36"
            :placeholder="t('admin.invoice.userId')"
            @input="debounceLoad"
          />
          <div class="flex flex-1 justify-end">
            <button class="btn btn-secondary" :disabled="loading" @click="loadItems">
              <Icon name="refresh" size="sm" class="mr-2" :class="loading ? 'animate-spin' : ''" />
              {{ t('common.refresh') }}
            </button>
          </div>
        </div>
      </div>

      <div class="card">
        <DataTable :columns="columns" :data="items" :loading="loading" row-key="id" :actions-count="3">
          <template #cell-order_no="{ row }">
            <span class="font-mono text-sm font-medium">{{ row.order_no }}</span>
          </template>
          <template #cell-user="{ row }">
            <div class="min-w-0">
              <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ row.user_email }}</p>
              <p class="text-xs text-gray-500 dark:text-dark-400">#{{ row.user_id }}</p>
            </div>
          </template>
          <template #cell-amount="{ value }">
            <span>{{ formatMoney(value) }}</span>
          </template>
          <template #cell-user_completed_invoice_amount="{ value }">
            <span>{{ formatMoney(value) }}</span>
          </template>
          <template #cell-user_total_recharged="{ row }">
            <span>{{ formatMoney(row.user?.total_recharged ?? row.user_total_recharged ?? 0) }}</span>
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
            <div class="flex items-center gap-1">
              <button
                v-if="row.status === 'processing'"
                class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20"
                @click="openCompleteDialog(row)"
              >
                <Icon name="upload" size="sm" />
                {{ t('admin.invoice.complete') }}
              </button>
              <button
                v-if="row.status === 'processing'"
                class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
                @click="openFailDialog(row)"
              >
                <Icon name="xCircle" size="sm" />
                {{ t('admin.invoice.fail') }}
              </button>
              <button
                v-if="row.status === 'completed' && row.has_file"
                class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
                :disabled="downloadingId === row.id"
                @click="downloadInvoice(row)"
              >
                <Icon name="download" size="sm" />
                {{ t('invoice.download') }}
              </button>
            </div>
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

    <BaseDialog :show="!!completeTarget" :title="t('admin.invoice.completeTitle')" width="normal" @close="closeCompleteDialog">
      <div v-if="completeTarget" class="space-y-4">
        <div class="rounded-lg bg-gray-50 p-4 text-sm dark:bg-dark-800">
          <div class="flex justify-between gap-4">
            <span class="text-gray-500 dark:text-dark-400">{{ t('invoice.columns.orderNo') }}</span>
            <span class="font-mono font-medium text-gray-900 dark:text-white">{{ completeTarget.order_no }}</span>
          </div>
          <div class="mt-2 flex justify-between gap-4">
            <span class="text-gray-500 dark:text-dark-400">{{ t('invoice.columns.amount') }}</span>
            <span class="font-medium text-gray-900 dark:text-white">{{ formatMoney(completeTarget.amount) }}</span>
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('admin.invoice.file') }}</label>
          <input type="file" accept=".pdf,.png,.jpg,.jpeg" class="input" @change="handleFileChange" />
          <p v-if="completionForm.selectedFile" class="mt-2 text-xs text-gray-500 dark:text-dark-400">
            {{ completionForm.selectedFile.name }}
          </p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.invoice.taxRate') }}</label>
          <input v-model.number="completionForm.taxRate" type="number" min="0" step="0.001" class="input" />
          <p class="input-hint">{{ t('admin.invoice.taxRateHint') }}</p>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" type="button" @click="closeCompleteDialog">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" type="button" :disabled="actionLoading" @click="completeInvoice">
            {{ actionLoading ? t('common.processing') : t('admin.invoice.complete') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="!!failTarget" :title="t('admin.invoice.failTitle')" width="narrow" @close="closeFailDialog">
      <div class="space-y-4">
        <textarea v-model.trim="failReason" rows="3" class="input" :placeholder="t('admin.invoice.failReasonPlaceholder')" />
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" type="button" @click="closeFailDialog">{{ t('common.cancel') }}</button>
          <button class="btn btn-danger" type="button" :disabled="actionLoading || !failReason" @click="failInvoice">
            {{ actionLoading ? t('common.processing') : t('admin.invoice.fail') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminInvoicesAPI } from '@/api/admin/invoices'
import type { InvoiceRequest, InvoiceStatus } from '@/api/invoices'
import { useAppStore } from '@/stores/app'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatCurrency, formatDateTime } from '@/utils/format'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const items = ref<InvoiceRequest[]>([])
const loading = ref(false)
const actionLoading = ref(false)
const downloadingId = ref<number | null>(null)
const completeTarget = ref<InvoiceRequest | null>(null)
const failTarget = ref<InvoiceRequest | null>(null)
const failReason = ref('')
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const filters = reactive({
  search: '',
  status: '' as InvoiceStatus | '',
  user_id: undefined as number | undefined,
})
const completionForm = reactive({
  taxRate: 0.01,
  selectedFile: null as File | null,
})

let searchTimer: ReturnType<typeof setTimeout> | null = null

const columns = computed<Column[]>(() => [
  { key: 'order_no', label: t('invoice.columns.orderNo') },
  { key: 'user', label: t('admin.invoice.user') },
  { key: 'title', label: t('invoice.columns.title') },
  { key: 'amount', label: t('invoice.columns.amount') },
  { key: 'user_completed_invoice_amount', label: t('admin.invoice.userCompletedInvoiceAmount') },
  { key: 'user_total_recharged', label: t('admin.invoice.userTotalRecharged') },
  { key: 'tax_amount', label: t('admin.invoice.taxAmount') },
  { key: 'status', label: t('invoice.columns.status') },
  { key: 'created_at', label: t('invoice.columns.createdAt') },
  { key: 'actions', label: t('common.actions') },
])

const statusOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'processing', label: t('invoice.status.processing') },
  { value: 'completed', label: t('invoice.status.completed') },
  { value: 'failed', label: t('invoice.status.failed') },
])

function formatMoney(value: number | null | undefined): string {
  return formatCurrency(value ?? 0, 'USD')
}

function statusLabel(status: InvoiceStatus): string {
  return t(`invoice.status.${status}`)
}

function statusClass(status: InvoiceStatus): string {
  if (status === 'completed') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (status === 'failed') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
}

async function loadItems() {
  loading.value = true
  try {
    const response = await adminInvoicesAPI.list({
      page: pagination.page,
      page_size: pagination.page_size,
      status: filters.status,
      search: filters.search || undefined,
      user_id: filters.user_id || undefined,
    })
    items.value = response.items || []
    pagination.total = response.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('admin.invoice.failedToLoad')))
  } finally {
    loading.value = false
  }
}

function debounceLoad() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    pagination.page = 1
    void loadItems()
  }, 300)
}

function openCompleteDialog(row: InvoiceRequest) {
  completeTarget.value = row
  completionForm.taxRate = 0.01
  completionForm.selectedFile = null
}

function closeCompleteDialog() {
  completeTarget.value = null
  completionForm.selectedFile = null
}

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  completionForm.selectedFile = input.files?.[0] ?? null
}

async function completeInvoice() {
  if (!completeTarget.value) return
  const selectedFile = completionForm.selectedFile
  if (!selectedFile) {
    appStore.showError(t('admin.invoice.fileRequired'))
    return
  }
  actionLoading.value = true
  try {
    await adminInvoicesAPI.complete(completeTarget.value.id, {
      file: selectedFile,
      tax_rate: completionForm.taxRate,
    })
    appStore.showSuccess(t('admin.invoice.completeSuccess'))
    closeCompleteDialog()
    await loadItems()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('admin.invoice.completeFailed')))
  } finally {
    actionLoading.value = false
  }
}

function openFailDialog(row: InvoiceRequest) {
  failTarget.value = row
  failReason.value = ''
}

function closeFailDialog() {
  failTarget.value = null
  failReason.value = ''
}

async function failInvoice() {
  if (!failTarget.value || !failReason.value) return
  actionLoading.value = true
  try {
    await adminInvoicesAPI.fail(failTarget.value.id, failReason.value)
    appStore.showSuccess(t('admin.invoice.failSuccess'))
    closeFailDialog()
    await loadItems()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'invoice.errors', t('admin.invoice.failFailed')))
  } finally {
    actionLoading.value = false
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
    const blob = await adminInvoicesAPI.download(row.id)
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
  void loadItems()
})
</script>
