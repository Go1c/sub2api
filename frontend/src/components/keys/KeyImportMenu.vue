<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { keysAPI } from '@/api/keys'
import { useAppStore } from '@/stores/app'
import { CODEX_DOWNLOAD_URL } from '@/constants/codexDownload'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  attemptImportNavigation,
  providerImportUrl,
  resolveProviderImport,
  type ImportClient,
  type ProviderImportInput,
} from '@/utils/providerImport'

const props = defineProps<{
  keyId: number
  platform?: string | null
  keyName?: string
  groupName?: string
  apiBaseUrl?: string
  siteName?: string
}>()

const { t } = useI18n()
const appStore = useAppStore()

const busy = ref(false)
const attemptedClient = ref<ImportClient>('bestcodex')
const followUpOpen = ref(false)
const appPickerOpen = ref(false)
const pendingClient = ref<ImportClient | null>(null)
const prepared = ref<ProviderImportInput | null>(null)

let requestSerial = 0

const followUpTitle = computed(() =>
  attemptedClient.value === 'ccswitch' ? t('keys.importToCcSwitch') : t('keys.importToBestCodex')
)
const openLabel = computed(() =>
  attemptedClient.value === 'ccswitch' ? t('keys.openCcSwitch') : t('keys.openBestCodex')
)

function hrefFor(client: ImportClient): string {
  if (!followUpOpen.value || !prepared.value) return ''
  try {
    return providerImportUrl(client, prepared.value)
  } catch {
    return ''
  }
}

const retryHref = computed(() => hrefFor(attemptedClient.value))
const ccswitchHref = computed(() => (attemptedClient.value === 'bestcodex' ? hrefFor('ccswitch') : ''))

function importErrorMessage(error: unknown): string {
  const message = error instanceof Error ? error.message : ''
  if (message === '服务地址无效') return t('keys.importEndpointInvalid')
  if (message === 'Key 无效') return t('keys.importKeyInvalid')
  if (message === '配置过长，请使用客户端手动导入') return t('keys.importTooLong')
  if (message === '客户端无效') return t('keys.importClientInvalid')
  if (message === '应用类型无效') return t('keys.importAppInvalid')
  return t('keys.importPrepareFailed')
}

function clearSecret() {
  const serial = requestSerial
  prepared.value = null
  void nextTick(() => {
    if (serial !== requestSerial) return
    followUpOpen.value = false
  })
}

function closeAppPicker() {
  appPickerOpen.value = false
  pendingClient.value = null
}

async function begin(client: ImportClient) {
  if (busy.value) return
  if (followUpOpen.value) return
  if (props.platform === 'antigravity') {
    pendingClient.value = client
    appPickerOpen.value = true
    return
  }
  await launch(client, null)
}

async function chooseAntigravityApp(app: 'claude' | 'gemini') {
  const client = pendingClient.value || 'bestcodex'
  closeAppPicker()
  await launch(client, app)
}

async function launch(client: ImportClient, antigravityApp: 'claude' | 'gemini' | null) {
  if (busy.value) return
  const serial = ++requestSerial
  busy.value = true
  try {
    const detail = await keysAPI.getById(props.keyId)
    if (serial !== requestSerial) return
    const platform = detail.group?.platform || props.platform || null
    const resolved = resolveProviderImport({
      platform,
      antigravityApp,
      apiBaseUrl: props.apiBaseUrl,
      pageOrigin: window.location.origin,
      keyName: detail.name || props.keyName,
      groupName: detail.group?.name || props.groupName,
      siteName: props.siteName,
    })
    if (resolved.needsAppChoice) {
      pendingClient.value = client
      appPickerOpen.value = true
      return
    }
    const input: ProviderImportInput = {
      app: resolved.app,
      name: resolved.name,
      endpoint: resolved.endpoint,
      apiKey: detail.key,
    }
    const url = providerImportUrl(client, input)
    if (serial !== requestSerial) return
    prepared.value = input
    attemptedClient.value = client
    followUpOpen.value = true
    try {
      attemptImportNavigation(url)
    } catch {
      // A blocked navigation leaves the native link for the next click.
    }
  } catch (error: unknown) {
    if (serial !== requestSerial) return
    clearSecret()
    appStore.showError(importErrorMessage(error))
  } finally {
    if (serial === requestSerial) busy.value = false
  }
}

onUnmounted(() => {
  requestSerial += 1
  clearSecret()
})
</script>

<template>
  <div data-testid="key-import">
    <button
      type="button"
      data-testid="import-primary"
      class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 disabled:opacity-50 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
      :disabled="busy"
      @click="begin('bestcodex')"
    >
      <Icon name="upload" size="sm" />
      <span class="text-xs">{{ t('keys.importToCcSwitch') }}</span>
    </button>

    <BaseDialog :show="appPickerOpen" :title="t('keys.ccsClientSelect.title')" width="narrow" @close="closeAppPicker">
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-400">{{ t('keys.ccsClientSelect.description') }}</p>
        <div class="grid grid-cols-2 gap-3">
          <button
            type="button"
            data-testid="import-app-claude"
            class="flex flex-col items-center gap-2 rounded-xl border-2 border-gray-200 p-4 transition-all hover:border-primary-500 hover:bg-primary-50 dark:border-dark-600 dark:hover:border-primary-500 dark:hover:bg-primary-900/20"
            @click="chooseAntigravityApp('claude')"
          >
            <Icon name="terminal" size="xl" class="text-gray-600 dark:text-gray-400" />
            <span class="font-medium text-gray-900 dark:text-white">{{ t('keys.ccsClientSelect.claudeCode') }}</span>
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('keys.ccsClientSelect.claudeCodeDesc') }}</span>
          </button>
          <button
            type="button"
            data-testid="import-app-gemini"
            class="flex flex-col items-center gap-2 rounded-xl border-2 border-gray-200 p-4 transition-all hover:border-primary-500 hover:bg-primary-50 dark:border-dark-600 dark:hover:border-primary-500 dark:hover:bg-primary-900/20"
            @click="chooseAntigravityApp('gemini')"
          >
            <Icon name="sparkles" size="xl" class="text-gray-600 dark:text-gray-400" />
            <span class="font-medium text-gray-900 dark:text-white">{{ t('keys.ccsClientSelect.geminiCli') }}</span>
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('keys.ccsClientSelect.geminiCliDesc') }}</span>
          </button>
        </div>
      </div>
    </BaseDialog>

    <BaseDialog :show="followUpOpen" :title="followUpTitle" width="narrow" @close="clearSecret">
      <div class="space-y-3" data-testid="import-follow-up">
        <p class="text-sm text-gray-600 dark:text-gray-400">{{ t('keys.importFollowUpHint') }}</p>
        <div class="flex flex-col gap-2">
          <a
            v-if="retryHref"
            data-testid="import-open-link"
            class="btn btn-primary justify-center"
            :href="retryHref"
          >
            {{ openLabel }}
          </a>
          <a
            v-if="ccswitchHref"
            data-testid="import-ccswitch-fallback"
            class="btn btn-secondary justify-center"
            :href="ccswitchHref"
          >
            {{ t('keys.importNotOpenedCcSwitch') }}
          </a>
          <div class="rounded-lg bg-gray-50 px-3 py-2.5 dark:bg-dark-700">
            <p class="text-sm leading-5 text-gray-600 dark:text-gray-300">{{ t('keys.downloadBestCodexPitch') }}</p>
            <a
              data-testid="import-follow-up-download"
              class="mt-2 inline-flex text-sm font-medium text-primary-600 hover:underline dark:text-primary-400"
              :href="CODEX_DOWNLOAD_URL"
              target="_blank"
              rel="noopener noreferrer"
            >
              {{ t('keys.downloadBestCodex') }}
            </a>
          </div>
        </div>
      </div>
    </BaseDialog>
  </div>
</template>
