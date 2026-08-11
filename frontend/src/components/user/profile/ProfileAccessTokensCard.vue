<template>
  <div class="card" data-testid="profile-access-tokens-card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t('profile.accessTokens.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('profile.accessTokens.description') }}
      </p>
    </div>

    <div class="space-y-6 px-6 py-6">
      <!-- Create form -->
      <div class="space-y-3 rounded-lg border border-gray-100 p-4 dark:border-dark-700">
        <div class="grid gap-3 sm:grid-cols-2">
          <div>
            <label class="input-label" for="uat-name">{{ t('profile.accessTokens.name') }}</label>
            <input
              id="uat-name"
              v-model="formName"
              type="text"
              class="input w-full"
              maxlength="100"
              :placeholder="t('profile.accessTokens.namePlaceholder')"
            />
          </div>
          <div>
            <label class="input-label" for="uat-days">{{
              t('profile.accessTokens.expiresInDays')
            }}</label>
            <input
              id="uat-days"
              v-model.number="formDays"
              type="number"
              min="1"
              max="30"
              step="1"
              class="input w-full ui-mono"
            />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('profile.accessTokens.expiresHint') }}
            </p>
          </div>
        </div>
        <div class="flex justify-end">
          <button
            type="button"
            class="btn btn-primary btn-sm"
            :disabled="creating"
            @click="handleCreate"
          >
            {{ creating ? t('common.saving') : t('profile.accessTokens.create') }}
          </button>
        </div>
      </div>

      <!-- One-time plaintext banner -->
      <div
        v-if="createdPlaintext"
        class="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-900/20"
        data-testid="access-token-once-banner"
      >
        <p class="text-sm font-medium text-amber-900 dark:text-amber-100">
          {{ t('profile.accessTokens.onceWarning') }}
        </p>
        <div class="mt-3 flex flex-col gap-2 sm:flex-row sm:items-center">
          <code
            class="ui-mono flex-1 break-all rounded bg-white/80 px-3 py-2 text-sm text-gray-900 dark:bg-dark-800 dark:text-gray-100"
            >{{ createdPlaintext }}</code
          >
          <button type="button" class="btn btn-secondary btn-sm whitespace-nowrap" @click="copyToken">
            {{ copied ? t('profile.accessTokens.copied') : t('profile.accessTokens.copy') }}
          </button>
        </div>
        <button
          type="button"
          class="mt-3 text-xs text-amber-800 underline dark:text-amber-200"
          @click="createdPlaintext = ''"
        >
          {{ t('profile.accessTokens.dismissOnce') }}
        </button>
      </div>

      <!-- List -->
      <div v-if="loading" class="flex justify-center py-6">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-500" />
      </div>
      <div v-else-if="tokens.length === 0" class="py-4 text-center text-sm text-gray-500">
        {{ t('profile.accessTokens.empty') }}
      </div>
      <div v-else class="overflow-x-auto">
        <table class="min-w-full text-left text-sm">
          <thead>
            <tr class="border-b border-gray-100 text-xs text-gray-500 dark:border-dark-700">
              <th class="px-2 py-2 font-medium">{{ t('profile.accessTokens.name') }}</th>
              <th class="px-2 py-2 font-medium">{{ t('profile.accessTokens.prefix') }}</th>
              <th class="px-2 py-2 font-medium">{{ t('profile.accessTokens.status') }}</th>
              <th class="px-2 py-2 font-medium">{{ t('profile.accessTokens.expiresAt') }}</th>
              <th class="px-2 py-2 font-medium">{{ t('profile.accessTokens.createdAt') }}</th>
              <th class="px-2 py-2 font-medium" />
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in tokens"
              :key="item.id"
              class="border-b border-gray-50 dark:border-dark-800"
            >
              <td class="px-2 py-3 text-gray-900 dark:text-white">{{ item.name }}</td>
              <td class="ui-mono px-2 py-3 text-gray-600 dark:text-gray-300">
                {{ item.token_prefix }}…
              </td>
              <td class="px-2 py-3">
                <span
                  class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium"
                  :class="statusClass(item.status)"
                >
                  {{ t(`profile.accessTokens.statusLabels.${item.status}`) }}
                </span>
              </td>
              <td class="ui-mono px-2 py-3 text-gray-600 dark:text-gray-300">
                {{ formatDate(item.expires_at) }}
              </td>
              <td class="ui-mono px-2 py-3 text-gray-600 dark:text-gray-300">
                {{ formatDate(item.created_at) }}
              </td>
              <td class="px-2 py-3 text-right">
                <button
                  v-if="item.status === 'active'"
                  type="button"
                  class="btn btn-outline-danger btn-sm"
                  :disabled="revokingId === item.id"
                  @click="confirmRevoke(item)"
                >
                  {{ t('profile.accessTokens.revoke') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  createAccessToken,
  listAccessTokens,
  revokeAccessToken,
  type UserAccessToken
} from '@/api/accessTokens'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const creating = ref(false)
const revokingId = ref<number | null>(null)
const tokens = ref<UserAccessToken[]>([])
const formName = ref('')
const formDays = ref(7)
const createdPlaintext = ref('')
const copied = ref(false)

function statusClass(status: string): string {
  if (status === 'active') {
    return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300'
  }
  if (status === 'revoked') {
    return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
  return 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-200'
}

function formatDate(iso: string): string {
  if (!iso) return '—'
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}

async function load() {
  loading.value = true
  try {
    tokens.value = await listAccessTokens()
  } catch (e: unknown) {
    const msg = (e as { message?: string })?.message || t('profile.accessTokens.loadFailed')
    appStore.showError(msg)
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  const name = formName.value.trim()
  if (!name) {
    appStore.showError(t('profile.accessTokens.nameRequired'))
    return
  }
  const days = Number(formDays.value)
  if (!Number.isInteger(days) || days < 1 || days > 30) {
    appStore.showError(t('profile.accessTokens.invalidDays'))
    return
  }
  creating.value = true
  try {
    const created = await createAccessToken({ name, expires_in_days: days })
    createdPlaintext.value = created.token || ''
    formName.value = ''
    formDays.value = 7
    copied.value = false
    appStore.showSuccess(t('profile.accessTokens.createSuccess'))
    await load()
  } catch (e: unknown) {
    const msg = (e as { message?: string })?.message || t('profile.accessTokens.createFailed')
    appStore.showError(msg)
  } finally {
    creating.value = false
  }
}

async function copyToken() {
  if (!createdPlaintext.value) return
  try {
    await navigator.clipboard.writeText(createdPlaintext.value)
    copied.value = true
    appStore.showSuccess(t('profile.accessTokens.copied'))
  } catch {
    appStore.showError(t('profile.accessTokens.copyFailed'))
  }
}

async function confirmRevoke(item: UserAccessToken) {
  if (!window.confirm(t('profile.accessTokens.revokeConfirm', { name: item.name }))) {
    return
  }
  revokingId.value = item.id
  try {
    await revokeAccessToken(item.id)
    appStore.showSuccess(t('profile.accessTokens.revokeSuccess'))
    await load()
  } catch (e: unknown) {
    const msg = (e as { message?: string })?.message || t('profile.accessTokens.revokeFailed')
    appStore.showError(msg)
  } finally {
    revokingId.value = null
  }
}

onMounted(() => {
  void load()
})
</script>
