<template>
  <BaseDialog :show="show" :title="t('codexLogin.title')" width="normal" @close="close">
    <form id="codex-2fa-form" class="space-y-4" @submit.prevent="submit">
      <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('codexLogin.hint') }}</p>
      <p v-if="!available" class="text-sm text-amber-600">{{ t('codexLogin.unavailable') }}</p>
      <div>
        <label class="input-label" for="codex-2fa-files">{{ t('codexLogin.files') }}</label>
        <input id="codex-2fa-files" ref="fileInput" class="input w-full" type="file" multiple accept=".json,.txt" @change="chooseFiles" />
      </div>
      <div>
        <label class="input-label" for="codex-2fa-text">{{ t('codexLogin.content') }}</label>
        <textarea id="codex-2fa-text" v-model="content" class="input w-full font-mono text-xs" rows="4" autocomplete="off" spellcheck="false"
          placeholder='[{"email":"you@example.com","password":"...","totp_secret":"..."}]' />
      </div>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div>
          <label class="input-label" for="codex-2fa-groups">{{ t('codexLogin.groups') }}</label>
          <select id="codex-2fa-groups" v-model="selectedGroups" class="input w-full" multiple>
            <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
          </select>
        </div>
        <div>
          <label class="input-label" for="codex-2fa-proxy">{{ t('codexLogin.proxy') }}</label>
          <select id="codex-2fa-proxy" v-model="selectedProxy" class="input w-full">
            <option value="">{{ t('codexLogin.direct') }}</option>
            <optgroup :label="t('codexLogin.ipGroups')">
              <option v-for="group in ipGroups" :key="group.id" :value="`group:${group.id}`">{{ group.name }}</option>
            </optgroup>
            <optgroup :label="t('codexLogin.proxies')">
              <option v-for="proxy in proxies" :key="proxy.id" :value="`proxy:${proxy.id}`">{{ proxy.name }}</option>
            </optgroup>
          </select>
        </div>
      </div>
      <p v-if="message" class="text-sm" role="status">{{ message }}</p>
    </form>
    <section class="mt-5 space-y-2" aria-live="polite">
      <button class="btn btn-secondary" type="button" @click="refresh">{{ t('codexLogin.refresh') }}</button>
      <div v-for="job in jobs" :key="job.id" class="rounded-lg border border-gray-200 p-3 text-sm dark:border-dark-600">
        <div class="break-all font-medium">{{ job.email }}</div>
        <div class="mt-1 text-gray-600 dark:text-dark-300">{{ t(`codexLogin.${job.status}`) }}<span v-if="job.account_id"> · #{{ job.account_id }}</span></div>
        <p v-if="job.error_message" class="mt-1 text-red-600">{{ job.error_message }}</p>
        <button v-if="job.status === 'failed'" class="btn btn-secondary mt-2" type="button" :disabled="retrying === job.id" @click="retry(job.id)">{{ t('codexLogin.retry') }}</button>
      </div>
    </section>
    <template #footer>
      <button class="btn btn-secondary" type="button" @click="close">{{ t('common.close') }}</button>
      <button class="btn btn-primary" type="submit" form="codex-2fa-form" :disabled="submitting || !available">{{ t('codexLogin.submit') }}</button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import * as api from '@/api/admin/codexLogin'
import * as groupsAPI from '@/api/admin/groups'
import * as proxiesAPI from '@/api/admin/proxies'
import * as ipGroupsAPI from '@/api/admin/proxyIpGroups'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: []; imported: [] }>()
const { t } = useI18n()
const available = ref(false)
const content = ref('')
const files = ref<File[]>([])
const fileInput = ref<HTMLInputElement | null>(null)
const groups = ref<{ id: number; name: string }[]>([])
const proxies = ref<{ id: number; name: string }[]>([])
const ipGroups = ref<{ id: number; name: string }[]>([])
const selectedGroups = ref<number[]>([])
const selectedProxy = ref('')
const jobs = ref<api.CodexLoginJob[]>([])
const message = ref('')
const submitting = ref(false)
const retrying = ref<number | null>(null)
let timer: ReturnType<typeof setTimeout> | undefined
let generation = 0
const completed = new Set<number>()

function clearMaterial() {
  content.value = ''
  files.value = []
  if (fileInput.value) fileInput.value.value = ''
}
function close() { clearMaterial(); clearTimeout(timer); generation++; emit('close') }
function chooseFiles(event: Event) { files.value = Array.from((event.target as HTMLInputElement).files || []) }
async function refresh() {
  clearTimeout(timer)
  const current = generation
  try {
    const result = await api.jobs()
    if (!props.show || current !== generation) return
    available.value = result.available
    jobs.value = result.jobs
    for (const job of result.jobs) {
      if (job.status === 'succeeded' && !completed.has(job.id)) { completed.add(job.id); emit('imported') }
    }
    if (result.jobs.some(job => job.status === 'queued' || job.status === 'running')) timer = setTimeout(refresh, 3000)
  } catch { if (props.show) message.value = t('codexLogin.loadFailed') }
}
async function submit() {
  if (submitting.value) return
  if (files.value.reduce((n, file) => n + file.size, 0) + new Blob([content.value]).size > 1024 * 1024) { message.value = t('codexLogin.tooLarge'); return }
  submitting.value = true
  try {
    const documents = await Promise.all(files.value.map(file => file.text()))
    if (content.value.trim()) documents.push(content.value)
    if (!documents.length || documents.length > 20) { message.value = t('codexLogin.chooseFile'); return }
    const [mode, value] = selectedProxy.value.split(':')
    const result = await api.importAccounts({ documents, group_ids: selectedGroups.value,
      ...(mode === 'group' ? { proxy_ip_group_id: Number(value) } : mode === 'proxy' ? { proxy_id: Number(value) } : {}) })
    clearMaterial()
    message.value = t('codexLogin.accepted', { count: result.job_ids.length }) +
      (result.errors || []).map(error => ` [${error.document}:${error.index}] ${error.message}`).join('; ')
    await refresh()
  } catch { message.value = t('codexLogin.submitFailed') }
  finally { submitting.value = false }
}
async function retry(id: number) {
  retrying.value = id
  try { await api.retry(id); completed.delete(id); await refresh() }
  catch { message.value = t('codexLogin.retryFailed') }
  finally { retrying.value = null }
}
watch(() => props.show, async open => {
  generation++
  clearTimeout(timer)
  if (!open) { clearMaterial(); return }
  message.value = ''
  const current = generation
  await refresh()
  try {
    const [groupRows, proxyRows, ipRows] = await Promise.all([groupsAPI.getAll('openai'), proxiesAPI.getAll(), ipGroupsAPI.list()])
    if (!props.show || current !== generation) return
    groups.value = groupRows
    proxies.value = proxyRows
    ipGroups.value = ipRows
    selectedGroups.value = groupRows[0] ? [groupRows[0].id] : []
    selectedProxy.value = ipRows[0] ? `group:${ipRows[0].id}` : ''
  } catch { message.value = t('codexLogin.loadFailed') }
}, { immediate: true })
onBeforeUnmount(() => { generation++; clearTimeout(timer); clearMaterial() })
</script>
