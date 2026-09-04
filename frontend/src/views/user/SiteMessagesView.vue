<template>
  <AppLayout>
    <div class="min-w-0 max-w-full overflow-x-hidden space-y-5">
      <div class="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="min-w-0">
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('siteMessages.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('siteMessages.description') }}</p>
        </div>
        <div class="flex shrink-0 items-center gap-2">
          <button class="btn btn-secondary" :disabled="loading" @click="loadMessages">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
          <button class="btn btn-primary" @click="openCompose">
            <Icon name="plus" size="md" class="mr-1.5" />
            {{ t('siteMessages.compose') }}
          </button>
        </div>
      </div>

      <div
        data-testid="site-messages-mailbox"
        class="grid min-w-0 grid-cols-1 gap-4 lg:grid-cols-[minmax(0,390px)_minmax(0,1fr)]"
      >
        <section
          data-testid="site-messages-list"
          class="min-w-0 overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800"
          :class="selectedMessage ? 'hidden lg:block' : ''"
        >
          <div class="border-b border-gray-100 p-3 dark:border-dark-700">
            <div class="grid grid-cols-2 gap-1 rounded-lg bg-gray-100 p-1 dark:bg-dark-700">
              <button
                type="button"
                class="rounded-md px-3 py-2 text-sm font-medium transition-colors"
                :class="activeTab === 'inbox' ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-900 dark:text-white' : 'text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'"
                @click="switchTab('inbox')"
              >
                {{ t('siteMessages.inbox') }}
              </button>
              <button
                type="button"
                class="rounded-md px-3 py-2 text-sm font-medium transition-colors"
                :class="activeTab === 'sent' ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-900 dark:text-white' : 'text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'"
                @click="switchTab('sent')"
              >
                {{ t('siteMessages.sent') }}
              </button>
            </div>
          </div>

          <div v-if="loading" class="flex min-h-[360px] items-center justify-center text-sm text-gray-500 dark:text-gray-400">
            {{ t('common.loading') }}
          </div>
          <div v-else-if="messages.length === 0" class="flex min-h-[360px] items-center justify-center p-8 text-center text-sm text-gray-500 dark:text-gray-400">
            {{ activeTab === 'inbox' ? t('siteMessages.emptyInbox') : t('siteMessages.emptySent') }}
          </div>
          <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
            <button
              v-for="message in messages"
              :key="message.id"
              type="button"
              class="block w-full min-w-0 px-4 py-3 text-left transition-colors hover:bg-gray-50 dark:hover:bg-dark-700/60"
              :class="[
                selectedMessage?.id === message.id ? 'bg-primary-50/70 dark:bg-primary-900/20' : '',
                isUnread(message) ? 'font-semibold' : ''
              ]"
              @click="selectMessage(message)"
            >
              <div class="flex min-w-0 items-start gap-3">
                <span
                  v-if="isUnread(message)"
                  class="mt-1.5 h-2 w-2 flex-shrink-0 rounded-full bg-red-500"
                  aria-hidden="true"
                ></span>
                <span v-else class="mt-1.5 h-2 w-2 flex-shrink-0"></span>
                <div class="min-w-0 flex-1">
                  <div class="flex min-w-0 items-center gap-2">
                    <span class="min-w-0 flex-1 truncate text-sm text-gray-900 dark:text-white">{{ message.subject }}</span>
                    <AdminSenderBadge
                      v-if="activeTab === 'inbox'"
                      class="shrink-0"
                      :is-admin="Boolean(message.sender?.is_admin)"
                      :label="t('siteMessages.adminSender')"
                    />
                  </div>
                  <div
                    data-testid="site-message-row-meta"
                    class="mt-1 flex min-w-0 flex-col gap-0.5 text-xs text-gray-500 dark:text-gray-400 sm:flex-row sm:items-center sm:gap-2"
                  >
                    <span class="min-w-0 truncate">{{ activeTab === 'inbox' ? displayUser(message.sender, message.sender_id) : displayUser(message.recipient, message.recipient_id) }}</span>
                    <span class="shrink-0 tabular-nums">{{ formatDateTime(message.created_at) }}</span>
                  </div>
                  <p class="mt-1 truncate text-sm text-gray-500 dark:text-gray-400">{{ message.content }}</p>
                </div>
              </div>
            </button>
          </div>

          <div v-if="pagination.total > pagination.page_size" class="border-t border-gray-100 p-3 dark:border-dark-700">
            <Pagination
              :page="pagination.page"
              :total="pagination.total"
              :page-size="pagination.page_size"
              @update:page="handlePageChange"
              @update:pageSize="handlePageSizeChange"
            />
          </div>
        </section>

        <section
          data-testid="site-messages-detail"
          class="min-w-0 overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800 lg:min-h-[520px]"
          :class="selectedMessage ? '' : 'hidden lg:block'"
        >
          <div v-if="detailLoading" class="flex h-full min-h-[240px] items-center justify-center text-sm text-gray-500 dark:text-gray-400 lg:min-h-[520px]">
            {{ t('common.loading') }}
          </div>
          <div v-else-if="!selectedMessage" class="flex h-full min-h-[240px] items-center justify-center p-8 text-center text-sm text-gray-500 dark:text-gray-400 lg:min-h-[520px]">
            {{ t('siteMessages.emptyDetail') }}
          </div>
          <div v-else class="flex h-full min-h-[240px] flex-col lg:min-h-[520px]">
            <div class="border-b border-gray-100 p-4 sm:p-5 dark:border-dark-700">
              <button
                type="button"
                data-testid="site-messages-back"
                class="mb-3 inline-flex items-center text-sm text-gray-500 lg:hidden dark:text-gray-400"
                @click="clearSelection"
              >
                {{ t('common.back') }}
              </button>
              <div class="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div class="min-w-0">
                  <h2 class="break-words text-xl font-semibold text-gray-900 dark:text-white">{{ selectedMessage.subject }}</h2>
                  <div class="mt-2 flex min-w-0 flex-wrap items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
                    <span>{{ t(activeTab === 'inbox' ? 'siteMessages.from' : 'siteMessages.to') }}</span>
                    <span class="min-w-0 break-all font-medium text-gray-700 dark:text-gray-200">
                      {{ activeTab === 'inbox' ? displayUser(selectedMessage.sender, selectedMessage.sender_id) : displayUser(selectedMessage.recipient, selectedMessage.recipient_id) }}
                    </span>
                    <AdminSenderBadge
                      v-if="activeTab === 'inbox'"
                      class="shrink-0"
                      :is-admin="Boolean(selectedMessage.sender?.is_admin)"
                      :label="t('siteMessages.adminSender')"
                    />
                  </div>
                </div>
                <span class="shrink-0 text-sm tabular-nums text-gray-500 dark:text-gray-400">{{ formatDateTime(selectedMessage.created_at) }}</span>
              </div>
            </div>

            <div class="min-h-0 flex-1 overflow-y-auto p-4 sm:p-5">
              <div class="space-y-4">
                <div
                  v-for="message in conversationMessages"
                  :key="message.id"
                  class="flex min-w-0"
                  :class="isOwnConversationMessage(message) ? 'justify-end' : 'justify-start'"
                >
                  <div class="min-w-0 w-[min(78%,42rem)] max-w-full">
                    <div
                      class="mb-1 flex min-w-0 items-center gap-2 text-xs text-gray-500 dark:text-gray-400"
                      :class="isOwnConversationMessage(message) ? 'justify-end text-right' : 'justify-start'"
                    >
                      <span class="min-w-0 flex-1 truncate">{{ displayUser(message.sender, message.sender_id) }}</span>
                      <span class="shrink-0 tabular-nums">{{ formatDateTime(message.created_at) }}</span>
                    </div>
                    <div
                      class="whitespace-pre-wrap break-words rounded-2xl px-4 py-3 text-sm leading-6 shadow-sm [overflow-wrap:anywhere]"
                      :class="isOwnConversationMessage(message)
                        ? 'rounded-br-md bg-primary-600 text-white shadow-primary-600/20'
                        : 'rounded-bl-md border border-gray-200 bg-gray-50 text-gray-800 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-100'"
                    >
                      {{ message.content }}
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <form class="border-t border-gray-100 p-4 dark:border-dark-700 sm:p-5" @submit.prevent="sendReply">
              <label class="input-label">{{ t('siteMessages.reply') }}</label>
              <textarea
                v-model="replyContent"
                rows="4"
                class="input"
                :placeholder="t('siteMessages.replyPlaceholder')"
              ></textarea>
              <div class="mt-3 flex justify-end">
                <button type="submit" class="btn btn-primary" :disabled="replying || replyContent.trim().length === 0">
                  {{ replying ? t('siteMessages.sending') : t('siteMessages.sendReply') }}
                </button>
              </div>
            </form>
          </div>
        </section>
      </div>
    </div>

    <BaseDialog :show="composeOpen" :title="t('siteMessages.compose')" width="wide" @close="closeCompose">
      <form id="site-message-compose-form" class="space-y-4" @submit.prevent="sendMessage">
        <div>
          <label class="input-label">{{ t('siteMessages.recipient') }}</label>
          <div class="flex flex-col gap-2 sm:flex-row">
            <input
              v-model="composeForm.recipient"
              type="text"
              class="input"
              :placeholder="t('siteMessages.recipientPlaceholder')"
              @input="clearResolvedRecipient"
            />
            <button type="button" class="btn btn-secondary sm:w-28" :disabled="resolvingRecipient || composeForm.recipient.trim().length === 0" @click="resolveComposeRecipient">
              {{ resolvingRecipient ? t('common.verifying') : t('common.search') }}
            </button>
          </div>
          <div v-if="resolvedRecipient" class="mt-2 rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-300">
            {{ displayUser(resolvedRecipient, resolvedRecipient.id) }}
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('siteMessages.subject') }}</label>
          <input v-model="composeForm.subject" type="text" maxlength="200" class="input" required />
        </div>
        <div>
          <label class="input-label">{{ t('siteMessages.content') }}</label>
          <textarea v-model="composeForm.content" rows="8" class="input" required></textarea>
        </div>
      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeCompose">{{ t('common.cancel') }}</button>
          <button type="submit" form="site-message-compose-form" class="btn btn-primary" :disabled="sending || !canSendCompose">
            {{ sending ? t('siteMessages.sending') : t('siteMessages.send') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import AdminSenderBadge from '@/components/site-message/AdminSenderBadge.vue'
import { siteMessagesAPI } from '@/api/siteMessages'
import { useAppStore, useAuthStore, useSiteMessageStore } from '@/stores'
import type { SiteMessage, SiteMessageRecipient } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

type MailboxTab = 'inbox' | 'sent'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const siteMessageStore = useSiteMessageStore()

const activeTab = ref<MailboxTab>('inbox')
const messages = ref<SiteMessage[]>([])
const selectedMessage = ref<SiteMessage | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const replying = ref(false)
const replyContent = ref('')
const composeOpen = ref(false)
const sending = ref(false)
const resolvingRecipient = ref(false)
const resolvedRecipient = ref<SiteMessageRecipient | null>(null)
const resolvedRecipientQuery = ref('')

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0,
  pages: 0,
})

const composeForm = reactive({
  recipient: '',
  subject: '',
  content: '',
})

const defaultRecipientEmail = computed(() => {
  const value = appStore.cachedPublicSettings?.site_messages_default_recipient_email
  return typeof value === 'string' ? value.trim() : ''
})

const conversationMessages = computed(() => {
  if (!selectedMessage.value) return []
  return [selectedMessage.value, ...(selectedMessage.value.replies ?? [])].sort((a, b) => {
    const createdDiff = new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
    if (Number.isFinite(createdDiff) && createdDiff !== 0) return createdDiff
    return a.id - b.id
  })
})

const canSendCompose = computed(() =>
  composeForm.recipient.trim().length > 0 &&
  composeForm.subject.trim().length > 0 &&
  composeForm.content.trim().length > 0
)

function displayUser(user: SiteMessageRecipient | undefined | null, fallbackId: number): string {
  if (!user) return `#${fallbackId}`
  const name = user.username?.trim()
  if (name) return `${name} <${user.email}>`
  return user.email || `#${user.id}`
}

function isUnread(message: SiteMessage): boolean {
  return activeTab.value === 'inbox' && !message.read_at
}

async function loadMessages(): Promise<void> {
  loading.value = true
  try {
    const request = {
      page: pagination.page,
      page_size: pagination.page_size,
      sort_by: 'created_at',
      sort_order: 'desc' as const,
    }
    const response = activeTab.value === 'inbox'
      ? await siteMessagesAPI.listInbox(request)
      : await siteMessagesAPI.listSent(request)

    messages.value = response.items ?? []
    pagination.total = response.total ?? 0
    pagination.pages = response.pages ?? 0

    if (!selectedMessage.value || !messages.value.some((message) => message.id === selectedMessage.value?.id)) {
      selectedMessage.value = null
      replyContent.value = ''
    }
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('siteMessages.failedToLoad')))
  } finally {
    loading.value = false
  }
}

async function selectMessage(message: SiteMessage): Promise<void> {
  detailLoading.value = true
  try {
    const detail = await siteMessagesAPI.getById(message.id)
    selectedMessage.value = detail
    replyContent.value = ''
    const index = messages.value.findIndex((item) => item.id === message.id)
    if (index >= 0) {
      messages.value[index] = detail
    }
    if (activeTab.value === 'inbox') {
      await siteMessageStore.refreshUnreadCount().catch(() => undefined)
    }
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('siteMessages.failedToLoad')))
  } finally {
    detailLoading.value = false
  }
}

function clearSelection(): void {
  selectedMessage.value = null
  replyContent.value = ''
}

function switchTab(tab: MailboxTab): void {
  if (activeTab.value === tab) return
  activeTab.value = tab
  pagination.page = 1
  clearSelection()
  void loadMessages()
}

function handlePageChange(page: number): void {
  pagination.page = Math.max(1, page)
  clearSelection()
  void loadMessages()
}

function handlePageSizeChange(pageSize: number): void {
  pagination.page_size = pageSize
  pagination.page = 1
  clearSelection()
  void loadMessages()
}

function openCompose(): void {
  composeOpen.value = true
}

function closeCompose(): void {
  composeOpen.value = false
}

function resetCompose(recipient = ''): void {
  composeForm.recipient = recipient
  composeForm.subject = ''
  composeForm.content = ''
  resolvedRecipient.value = null
  resolvedRecipientQuery.value = ''
}

function clearResolvedRecipient(): void {
  if (composeForm.recipient.trim() !== resolvedRecipientQuery.value) {
    resolvedRecipient.value = null
    resolvedRecipientQuery.value = ''
  }
}

async function resolveComposeRecipient(): Promise<SiteMessageRecipient | null> {
  const query = composeForm.recipient.trim()
  if (!query) return null
  resolvingRecipient.value = true
  try {
    const recipient = await siteMessagesAPI.resolveRecipient(query)
    resolvedRecipient.value = recipient
    resolvedRecipientQuery.value = query
    return recipient
  } catch (error: unknown) {
    resolvedRecipient.value = null
    resolvedRecipientQuery.value = ''
    appStore.showError(extractApiErrorMessage(error, t('siteMessages.recipientNotFound')))
    return null
  } finally {
    resolvingRecipient.value = false
  }
}

async function sendMessage(): Promise<void> {
  if (!canSendCompose.value) return
  sending.value = true
  try {
    const query = composeForm.recipient.trim()
    if (!resolvedRecipient.value || resolvedRecipientQuery.value !== query) {
      const recipient = await resolveComposeRecipient()
      if (!recipient) return
    }
    await siteMessagesAPI.send({
      recipient: query,
      subject: composeForm.subject.trim(),
      content: composeForm.content.trim(),
    })
    appStore.showSuccess(t('siteMessages.sentSuccess'))
    closeCompose()
    resetCompose()
    activeTab.value = 'sent'
    pagination.page = 1
    await loadMessages()
    await siteMessageStore.refreshUnreadCount().catch(() => undefined)
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('siteMessages.failedToSend')))
  } finally {
    sending.value = false
  }
}

async function sendReply(): Promise<void> {
  if (!selectedMessage.value || replyContent.value.trim().length === 0) return
  replying.value = true
  try {
    const reply = await siteMessagesAPI.reply(selectedMessage.value.id, {
      content: replyContent.value.trim(),
    })
    selectedMessage.value = {
      ...selectedMessage.value,
      replies: [...(selectedMessage.value.replies ?? []), reply],
    }
    replyContent.value = ''
    appStore.showSuccess(t('siteMessages.replySent'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('siteMessages.failedToSend')))
  } finally {
    replying.value = false
  }
}

watch(composeOpen, (open) => {
  if (open) {
    resetCompose(defaultRecipientEmail.value)
  }
})

function isOwnConversationMessage(message: SiteMessage): boolean {
  const currentUserID = authStore.user?.id
  if (typeof currentUserID === 'number') return message.sender_id === currentUserID
  if (!selectedMessage.value) return false
  if (message.id === selectedMessage.value.id) return activeTab.value === 'sent'
  return message.sender_id === selectedMessage.value.recipient_id
}

onMounted(() => {
  void loadMessages()
  void siteMessageStore.refreshUnreadCount().catch(() => undefined)
})
</script>
