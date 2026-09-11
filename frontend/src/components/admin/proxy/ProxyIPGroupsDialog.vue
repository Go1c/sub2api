<template>
  <BaseDialog
    :show="show"
    :title="t('admin.proxies.ipGroups')"
    width="wide"
    @close="emit('close')"
  >
    <div class="space-y-4">
      <div class="flex justify-end">
        <button type="button" class="btn btn-primary" @click="openCreate">
          <Icon name="plus" size="md" class="mr-2" />
          {{ t('admin.proxies.ipGroupCreate') }}
        </button>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-8 text-sm text-gray-500">
        <Icon name="refresh" size="md" class="mr-2 animate-spin" />
        {{ t('common.loading') }}
      </div>
      <div v-else-if="groups.length === 0" class="py-6 text-center text-sm text-gray-500">
        {{ t('admin.proxies.ipGroupEmpty') }}
      </div>
      <div v-else class="max-h-80 overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
        <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
          <thead class="bg-gray-50 text-xs uppercase text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="px-3 py-2 text-left">{{ t('admin.proxies.ipGroupName') }}</th>
              <th class="px-3 py-2 text-left">{{ t('admin.proxies.ipGroupConcurrency') }}</th>
              <th class="px-3 py-2 text-left">{{ t('admin.proxies.ipGroupMembers') }}</th>
              <th class="px-3 py-2 text-right">{{ t('admin.proxies.columns.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
            <tr v-for="group in groups" :key="group.id">
              <td class="px-3 py-2 font-medium text-gray-900 dark:text-white">{{ group.name }}</td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-300">{{ group.per_ip_concurrency }}</td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-300">
                {{ memberLabel(group) }}
              </td>
              <td class="px-3 py-2 text-right">
                <button type="button" class="btn btn-secondary mr-2" @click="openEdit(group)">
                  {{ t('common.edit') }}
                </button>
                <button type="button" class="btn btn-danger" @click="askDelete(group)">
                  {{ t('common.delete') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </BaseDialog>

  <BaseDialog
    :show="showForm"
    :title="editing ? t('admin.proxies.ipGroupEdit') : t('admin.proxies.ipGroupCreate')"
    width="normal"
    @close="closeForm"
  >
    <form id="proxy-ip-group-form" class="space-y-4" @submit.prevent="submitForm">
      <div>
        <label class="input-label">{{ t('admin.proxies.ipGroupName') }}</label>
        <input
          v-model="form.name"
          type="text"
          required
          maxlength="100"
          class="input"
          :placeholder="t('admin.proxies.ipGroupNamePlaceholder')"
        />
      </div>
      <div>
        <label class="input-label">{{ t('admin.proxies.ipGroupConcurrency') }}</label>
        <input
          v-model.number="form.per_ip_concurrency"
          type="number"
          min="1"
          max="1000"
          required
          class="input"
        />
      </div>
      <div>
        <label class="input-label">{{ t('admin.proxies.ipGroupMembers') }}</label>
        <div class="max-h-56 overflow-auto rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <label
            v-for="proxy in allProxies"
            :key="proxy.id"
            class="mb-2 flex items-center gap-2 text-sm last:mb-0"
          >
            <input
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="form.proxy_ids.includes(proxy.id)"
              @change="toggleMember(proxy.id, $event)"
            />
            <span class="text-gray-800 dark:text-gray-200">
              {{ proxy.name }}
              <span class="text-xs text-gray-500">({{ proxy.host }}:{{ proxy.port }})</span>
            </span>
          </label>
          <p v-if="allProxies.length === 0" class="text-sm text-gray-500">
            {{ t('common.noData') }}
          </p>
        </div>
      </div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="closeForm">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="proxy-ip-group-form"
          class="btn btn-primary"
          :disabled="submitting"
        >
          {{ submitting ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <ConfirmDialog
    :show="showDelete"
    :title="t('admin.proxies.ipGroupDelete')"
    :message="t('admin.proxies.ipGroupDeleteConfirm', { name: deleting?.name || '' })"
    :confirm-text="t('common.delete')"
    :cancel-text="t('common.cancel')"
    :danger="true"
    @confirm="confirmDelete"
    @cancel="showDelete = false"
  />
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { Proxy, ProxyIPGroup } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const submitting = ref(false)
const groups = ref<ProxyIPGroup[]>([])
const allProxies = ref<Proxy[]>([])
const showForm = ref(false)
const showDelete = ref(false)
const editing = ref<ProxyIPGroup | null>(null)
const deleting = ref<ProxyIPGroup | null>(null)
const form = reactive({
  name: '',
  per_ip_concurrency: 10,
  proxy_ids: [] as number[]
})

const memberLabel = (group: ProxyIPGroup) => {
  const ids = group.proxy_ids || []
  if (ids.length === 0) return '0'
  const names = ids
    .map((id) => allProxies.value.find((proxy) => proxy.id === id)?.name || `#${id}`)
    .slice(0, 3)
  const extra = ids.length > 3 ? ` +${ids.length - 3}` : ''
  return `${ids.length}: ${names.join(', ')}${extra}`
}

const loadGroups = async () => {
  loading.value = true
  try {
    const [nextGroups, nextProxies] = await Promise.all([
      adminAPI.proxyIpGroups.list(),
      adminAPI.proxies.getAll()
    ])
    groups.value = nextGroups
    allProxies.value = nextProxies
  } catch (error) {
    console.error('Error loading IP groups:', error)
    appStore.showError(t('admin.proxies.failedToLoad'))
  } finally {
    loading.value = false
  }
}

watch(
  () => props.show,
  (show) => {
    if (show) {
      void loadGroups()
    }
  }
)

const resetForm = () => {
  form.name = ''
  form.per_ip_concurrency = 10
  form.proxy_ids = []
  editing.value = null
}

const openCreate = () => {
  resetForm()
  showForm.value = true
}

const openEdit = (group: ProxyIPGroup) => {
  editing.value = group
  form.name = group.name
  form.per_ip_concurrency = group.per_ip_concurrency
  form.proxy_ids = [...(group.proxy_ids || [])]
  showForm.value = true
}

const closeForm = () => {
  showForm.value = false
  resetForm()
}

const toggleMember = (id: number, event: Event) => {
  const checked = (event.target as HTMLInputElement).checked
  if (checked) {
    if (!form.proxy_ids.includes(id)) {
      form.proxy_ids.push(id)
    }
    return
  }
  form.proxy_ids = form.proxy_ids.filter((item) => item !== id)
}

const submitForm = async () => {
  const name = form.name.trim()
  if (!name) return
  submitting.value = true
  try {
    if (editing.value) {
      await adminAPI.proxyIpGroups.update(editing.value.id, {
        name,
        per_ip_concurrency: form.per_ip_concurrency
      })
      await adminAPI.proxyIpGroups.setMembers(editing.value.id, form.proxy_ids)
    } else {
      await adminAPI.proxyIpGroups.create({
        name,
        per_ip_concurrency: form.per_ip_concurrency,
        proxy_ids: form.proxy_ids
      })
    }
    appStore.showSuccess(t('admin.proxies.ipGroupSaved'))
    closeForm()
    await loadGroups()
  } catch (error: any) {
    const detail = error.reason === 'PROXY_IP_GROUP_NAME_TAKEN'
      ? error.message
      : error.message || error.response?.data?.message || t('admin.proxies.failedToLoad')
    appStore.showError(detail)
  } finally {
    submitting.value = false
  }
}

const askDelete = (group: ProxyIPGroup) => {
  deleting.value = group
  showDelete.value = true
}

const confirmDelete = async () => {
  if (!deleting.value) return
  try {
    await adminAPI.proxyIpGroups.delete(deleting.value.id)
    appStore.showSuccess(t('admin.proxies.ipGroupDeleted'))
    showDelete.value = false
    deleting.value = null
    await loadGroups()
  } catch (error: any) {
    const reason = error.reason || error.response?.data?.reason
    const message = error.message || error.response?.data?.message
    if (reason === 'PROXY_IP_GROUP_IN_USE' || String(message || '').includes('still assigned')) {
      appStore.showError(t('admin.proxies.ipGroupInUse'))
      return
    }
    appStore.showError(message || t('admin.proxies.failedToLoad'))
  }
}
</script>
