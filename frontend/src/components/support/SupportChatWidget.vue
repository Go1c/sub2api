<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import {
  fetchSupportChatConfig,
  fetchSupportChatPublicSettings,
  isSupportChatEnabled,
  mergeSupportChatConfig,
  resolveSupportChatGatewayURL,
  streamSupportChat,
  type SupportChatConfig,
  type SupportChatPublicSettings,
  type SupportChatSource
} from '@/api/supportChat'
import { useAuthStore } from '@/stores/auth'

type ChatRole = 'assistant' | 'user'

interface ChatMessage {
  id: string
  role: ChatRole
  content: string
  sources?: SupportChatSource[]
}

const { t, locale } = useI18n()
const auth = useAuthStore()
const enabled = ref(isSupportChatEnabled())
const gatewayUrl = ref(resolveSupportChatGatewayURL())
const open = ref(false)
const draft = ref('')
const loading = ref(false)
const error = ref('')
const lastUserMessage = ref('')
const conversationId = ref<string>()
const inputRef = ref<HTMLTextAreaElement | null>(null)
const messages = ref<ChatMessage[]>([])
const config = ref<SupportChatConfig>({
  title: t('supportChat.title'),
  welcomeMessage: t('supportChat.welcome'),
  officialContactText: t('supportChat.contactSupport')
})

const gatewayLocale = computed(() => normalizeSupportChatLocale(locale.value))
const contactEmailHref = computed(() => {
  return config.value.supportEmail ? `mailto:${config.value.supportEmail}` : ''
})

watch(locale, loadConfig, { immediate: true })

async function loadConfig() {
  let publicSettings: SupportChatPublicSettings | null = null
  try {
    publicSettings = await fetchSupportChatPublicSettings()
    enabled.value = isSupportChatEnabled(publicSettings)
    gatewayUrl.value = resolveSupportChatGatewayURL(publicSettings)
  } catch {
    enabled.value = isSupportChatEnabled()
    gatewayUrl.value = resolveSupportChatGatewayURL()
  }

  if (!enabled.value) {
    open.value = false
    return
  }

  try {
    const gatewayConfig = await fetchSupportChatConfig({
      locale: gatewayLocale.value,
      gatewayUrl: gatewayUrl.value
    })
    config.value = mergeSupportChatConfig(gatewayConfig, publicSettings)
    error.value = ''
  } catch {
    error.value = t('supportChat.configError')
  }
}

function toggleOpen() {
  open.value = !open.value
  if (open.value) {
    nextTick(() => inputRef.value?.focus())
  }
}

function clearConversation() {
  messages.value = []
  conversationId.value = undefined
  error.value = ''
  draft.value = ''
}

async function retryLastMessage() {
  if (!lastUserMessage.value || loading.value) return
  draft.value = lastUserMessage.value
  await sendMessage()
}

async function sendMessage() {
  const message = draft.value.trim()
  if (!message || loading.value) return

  lastUserMessage.value = message
  draft.value = ''
  error.value = ''
  messages.value.push({ id: crypto.randomUUID(), role: 'user', content: message })

  const assistantMessage: ChatMessage = {
    id: crypto.randomUUID(),
    role: 'assistant',
    content: '',
    sources: []
  }
  messages.value.push(assistantMessage)
  loading.value = true

  try {
    await streamSupportChat(
      {
        message,
        locale: gatewayLocale.value,
        ...(conversationId.value ? { conversationId: conversationId.value } : {}),
        ...(auth.user
          ? {
              user: {
                id: auth.user.id,
                email: auth.user.email
              }
            }
          : {})
      },
      {
        onAnswer(chunk) {
          assistantMessage.content += chunk
        },
        onSources(sources) {
          assistantMessage.sources = [...(assistantMessage.sources ?? []), ...sources]
        },
        onConversationId(id) {
          conversationId.value = id
        },
        onError(messageText) {
          error.value = messageText
        },
        onEnd() {
          loading.value = false
        }
      },
      { gatewayUrl: gatewayUrl.value }
    )
  } catch {
    error.value = t('supportChat.error')
    if (!assistantMessage.content) {
      assistantMessage.content = t('supportChat.errorAssistant')
    }
  } finally {
    loading.value = false
  }
}

function sourceLabel(source: SupportChatSource, index: number) {
  return (
    source.title ||
    source.name ||
    source.source ||
    source.url ||
    t('supportChat.sourceFallback', { n: index + 1 })
  )
}

function normalizeSupportChatLocale(currentLocale: string) {
  if (currentLocale === 'zh-Hant') return 'zh-Hant'
  if (currentLocale === 'zh') return 'zh-CN'
  return 'en-US'
}
</script>

<template>
  <div v-if="enabled" class="fixed bottom-4 right-4 z-50 sm:bottom-6 sm:right-6">
    <section
      v-if="open"
      data-testid="support-chat-panel"
      class="fixed bottom-20 left-3 right-3 flex max-h-[min(680px,calc(100vh-7rem))] flex-col overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-2xl dark:border-dark-700 dark:bg-dark-900 sm:left-auto sm:right-6 sm:w-[380px]"
      aria-live="polite"
    >
      <header class="flex items-center justify-between gap-3 border-b border-gray-200 px-4 py-3 dark:border-dark-700">
        <div class="min-w-0">
          <h2 class="truncate text-base font-semibold text-gray-900 dark:text-white">
            {{ config.title }}
          </h2>
          <p class="truncate text-xs text-gray-500 dark:text-dark-400">
            {{ t('supportChat.subtitle') }}
          </p>
        </div>
        <div class="flex items-center gap-1">
          <button
            type="button"
            class="grid h-8 w-8 place-items-center rounded-md text-gray-500 transition hover:bg-gray-100 hover:text-gray-900 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :aria-label="t('supportChat.clear')"
            :title="t('supportChat.clear')"
            @click="clearConversation"
          >
            <Icon name="trash" size="sm" />
          </button>
          <button
            type="button"
            class="grid h-8 w-8 place-items-center rounded-md text-gray-500 transition hover:bg-gray-100 hover:text-gray-900 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :aria-label="t('supportChat.close')"
            :title="t('supportChat.close')"
            @click="toggleOpen"
          >
            <Icon name="x" size="sm" />
          </button>
        </div>
      </header>

      <div class="flex-1 space-y-3 overflow-y-auto bg-gray-50/70 px-4 py-4 dark:bg-dark-950/50">
        <div class="rounded-xl border border-primary-100 bg-white px-3 py-3 text-sm text-gray-700 dark:border-primary-900/40 dark:bg-dark-800 dark:text-dark-100">
          {{ config.welcomeMessage }}
        </div>

        <div
          v-for="message in messages"
          :key="message.id"
          class="flex"
          :class="message.role === 'user' ? 'justify-end' : 'justify-start'"
        >
          <div
            class="max-w-[86%] rounded-xl px-3 py-2 text-sm leading-relaxed"
            :class="
              message.role === 'user'
                ? 'bg-primary-600 text-white'
                : 'border border-gray-200 bg-white text-gray-800 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-100'
            "
          >
            <p class="whitespace-pre-wrap break-words">
              {{ message.content || (loading && message.role === 'assistant' ? t('supportChat.thinking') : '') }}
            </p>
            <div
              v-if="message.sources?.length"
              class="mt-2 flex flex-wrap gap-1.5 border-t border-gray-200 pt-2 dark:border-dark-700"
            >
              <a
                v-for="(source, index) in message.sources"
                :key="`${message.id}-${index}`"
                :href="source.url"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex max-w-full items-center gap-1 rounded-md bg-primary-50 px-2 py-1 text-[11px] text-primary-700 transition hover:bg-primary-100 dark:bg-primary-900/30 dark:text-primary-200 dark:hover:bg-primary-900/50"
              >
                <span class="truncate">{{ sourceLabel(source, index) }}</span>
                <Icon v-if="source.url" name="externalLink" class="h-3 w-3 shrink-0" />
              </a>
            </div>
          </div>
        </div>
      </div>

      <div
        v-if="error"
        class="border-t border-amber-200 bg-amber-50 px-4 py-2 text-xs text-amber-800 dark:border-amber-800/50 dark:bg-amber-950/40 dark:text-amber-200"
      >
        <div class="flex items-center justify-between gap-3">
          <span>{{ error }}</span>
          <button
            type="button"
            class="inline-flex shrink-0 items-center gap-1 rounded-md px-2 py-1 font-medium transition hover:bg-amber-100 dark:hover:bg-amber-900/30"
            @click="retryLastMessage"
          >
            <Icon name="refresh" class="h-3.5 w-3.5" />
            {{ t('supportChat.retry') }}
          </button>
        </div>
      </div>

      <div class="border-t border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-900">
        <form
          data-testid="support-chat-form"
          class="flex items-end gap-2"
          @submit.prevent="sendMessage"
        >
          <textarea
            ref="inputRef"
            v-model="draft"
            data-testid="support-chat-input"
            rows="2"
            class="min-h-[44px] flex-1 resize-none rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none transition focus:border-primary-400 focus:ring-2 focus:ring-primary-100 dark:border-dark-700 dark:bg-dark-800 dark:text-white dark:focus:border-primary-500 dark:focus:ring-primary-900/40"
            :placeholder="t('supportChat.placeholder')"
            :disabled="loading"
          />
          <button
            type="submit"
            class="grid h-11 w-11 shrink-0 place-items-center rounded-xl bg-primary-600 text-white shadow-sm transition hover:bg-primary-700 disabled:cursor-not-allowed disabled:bg-gray-300 dark:disabled:bg-dark-700"
            :disabled="loading || !draft.trim()"
            :aria-label="t('supportChat.send')"
            :title="t('supportChat.send')"
          >
            <Icon name="arrowUp" size="md" />
          </button>
        </form>

        <div class="mt-3 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
          <a
            v-if="contactEmailHref"
            :href="contactEmailHref"
            class="inline-flex items-center gap-1 rounded-md border border-gray-200 px-2 py-1 transition hover:border-primary-200 hover:text-primary-700 dark:border-dark-700 dark:hover:border-primary-700 dark:hover:text-primary-300"
          >
            <Icon name="mail" class="h-3.5 w-3.5" />
            {{ config.supportEmail }}
          </a>
          <a
            v-if="config.supportUrl"
            :href="config.supportUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex items-center gap-1 rounded-md border border-gray-200 px-2 py-1 transition hover:border-primary-200 hover:text-primary-700 dark:border-dark-700 dark:hover:border-primary-700 dark:hover:text-primary-300"
          >
            <Icon name="externalLink" class="h-3.5 w-3.5" />
            {{ config.officialContactText }}
          </a>
        </div>
      </div>
    </section>

    <button
      type="button"
      data-testid="support-chat-toggle"
      class="grid h-14 w-14 place-items-center rounded-full bg-primary-600 text-white shadow-xl transition hover:-translate-y-0.5 hover:bg-primary-700 focus:outline-none focus:ring-4 focus:ring-primary-200 dark:focus:ring-primary-900/40"
      :aria-label="open ? t('supportChat.close') : t('supportChat.open')"
      :title="open ? t('supportChat.close') : t('supportChat.open')"
      @click="toggleOpen"
    >
      <Icon :name="open ? 'x' : 'chatBubble'" size="lg" />
    </button>
  </div>
</template>
