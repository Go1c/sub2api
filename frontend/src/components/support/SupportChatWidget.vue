<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import Icon from '@/components/icons/Icon.vue'
import {
  fetchSupportChatConfig,
  fetchSupportChatPublicSettings,
  isSupportChatEnabled,
  mergeSupportChatConfig,
  resolveSupportChatGatewayURL,
  streamSupportChat,
  type SupportChatConfig,
  type SupportChatPublicSettings
} from '@/api/supportChat'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'

type ChatRole = 'assistant' | 'user'

interface ChatMessage {
  id: string
  role: ChatRole
  content: string
}

const { t, locale } = useI18n()
const auth = useAuthStore()
const { copyToClipboard } = useClipboard()
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
    content: ''
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
                id: String(auth.user.id),
                email: auth.user.email
              }
            }
          : {})
      },
      {
        onAnswer(chunk) {
          assistantMessage.content += chunk
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

function normalizeSupportChatLocale(currentLocale: string) {
  if (currentLocale === 'zh-Hant') return 'zh-Hant'
  if (currentLocale === 'zh') return 'zh-CN'
  return 'en-US'
}

function renderAssistantContent(content: string) {
  const rendered = marked.parse(stripUnsafeMarkdownLinks(content), { breaks: true, gfm: true }) as string
  const sanitized = DOMPurify.sanitize(rendered)
  return enhanceAssistantMarkdown(sanitized)
}

function stripUnsafeMarkdownLinks(content: string) {
  return content.replace(/\[([^\]]+)\]\(\s*(?:javascript|data|vbscript):[^\n]*\)/gi, '$1')
}

function enhanceAssistantMarkdown(html: string) {
  if (!html || typeof document === 'undefined') return html

  const template = document.createElement('template')
  template.innerHTML = html

  template.content.querySelectorAll('a').forEach((link) => {
    link.setAttribute('target', '_blank')
    link.setAttribute('rel', 'noopener noreferrer')
  })

  template.content.querySelectorAll('pre').forEach((pre) => {
    const code = pre.querySelector('code')
    if (!code) return

    const wrapper = document.createElement('div')
    wrapper.className = 'support-chat-code-block'

    const toolbar = document.createElement('div')
    toolbar.className = 'support-chat-code-toolbar'

    const language = document.createElement('span')
    language.className = 'support-chat-code-language'
    language.textContent = codeLanguage(code)

    const copyButton = document.createElement('button')
    copyButton.type = 'button'
    copyButton.className = 'support-chat-code-copy'
    copyButton.dataset.code = (code.textContent ?? '').replace(/\n$/, '')
    copyButton.textContent = copyCodeLabel()

    toolbar.append(language, copyButton)
    wrapper.append(toolbar)
    pre.parentNode?.insertBefore(wrapper, pre)
    wrapper.appendChild(pre)
  })

  return template.innerHTML
}

function copyCodeLabel() {
  const label = t('common.copy')
  return label === 'common.copy' ? (locale.value.startsWith('zh') ? '复制' : 'Copy') : label
}

function codeLanguage(code: Element) {
  const languageClass = Array.from(code.classList).find((className) =>
    className.startsWith('language-')
  )
  return languageClass?.replace(/^language-/, '') || 'code'
}

async function handleMarkdownClick(event: MouseEvent) {
  const target = event.target
  if (!(target instanceof Element)) return

  const button = target.closest<HTMLButtonElement>('.support-chat-code-copy')
  if (!button) return

  await copyToClipboard(button.dataset.code ?? '')
}
</script>

<template>
  <div v-if="enabled" class="fixed bottom-4 right-4 z-50 sm:bottom-6 sm:right-6">
    <section
      v-if="open"
      data-testid="support-chat-panel"
      class="fixed bottom-20 left-3 right-3 flex h-[min(620px,calc(100vh-7rem))] max-h-[min(680px,calc(100vh-7rem))] flex-col overflow-hidden rounded-[28px] border border-slate-200/80 bg-white/95 shadow-[0_28px_80px_rgba(15,23,42,0.18)] backdrop-blur dark:border-dark-700 dark:bg-dark-900/95 sm:left-auto sm:right-6 sm:w-[420px]"
      aria-live="polite"
    >
      <header class="flex items-center justify-between gap-3 border-b border-white/10 bg-gradient-to-r from-blue-600 via-indigo-500 to-purple-600 px-4 py-3 text-white">
        <div class="min-w-0">
          <h2 class="truncate text-base font-semibold text-white">
            {{ config.title }}
          </h2>
          <p class="truncate text-xs text-white/75">
            {{ t('supportChat.subtitle') }}
          </p>
        </div>
        <div class="flex items-center gap-1">
          <button
            type="button"
            class="grid h-8 w-8 place-items-center rounded-md text-white/80 transition hover:bg-white/10 hover:text-white"
            :aria-label="t('supportChat.clear')"
            :title="t('supportChat.clear')"
            @click="clearConversation"
          >
            <Icon name="trash" size="sm" />
          </button>
          <button
            type="button"
            class="grid h-8 w-8 place-items-center rounded-md text-white/80 transition hover:bg-white/10 hover:text-white"
            :aria-label="t('supportChat.close')"
            :title="t('supportChat.close')"
            @click="toggleOpen"
          >
            <Icon name="x" size="sm" />
          </button>
        </div>
      </header>

      <div class="flex-1 space-y-3 overflow-y-auto bg-[linear-gradient(180deg,rgba(248,250,252,0.92),rgba(238,242,255,0.88))] px-4 py-4 dark:bg-dark-950/50">
        <div class="rounded-2xl border border-indigo-100 bg-white/90 px-3 py-3 text-sm text-gray-700 shadow-[0_10px_24px_rgba(99,102,241,0.06)] dark:border-indigo-900/40 dark:bg-dark-800 dark:text-dark-100">
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
                ? 'bg-gradient-to-br from-blue-600 via-indigo-500 to-purple-600 text-white shadow-[0_12px_28px_rgba(99,102,241,0.28)]'
                : 'border border-gray-200 bg-white text-gray-800 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-100'
            "
          >
            <p
              v-if="message.role === 'user' || !message.content"
              class="whitespace-pre-wrap break-words"
            >
              {{ message.content || (loading && message.role === 'assistant' ? t('supportChat.thinking') : '') }}
            </p>
            <div
              v-else
              data-testid="support-chat-markdown"
              class="support-chat-markdown"
              v-html="renderAssistantContent(message.content)"
              @click="handleMarkdownClick"
            ></div>
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
            class="min-h-[44px] flex-1 resize-none rounded-2xl border border-slate-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none transition focus:border-indigo-400 focus:ring-2 focus:ring-indigo-100 dark:border-dark-700 dark:bg-dark-800 dark:text-white dark:focus:border-indigo-500 dark:focus:ring-indigo-900/40"
            :placeholder="t('supportChat.placeholder')"
            :disabled="loading"
          />
          <button
            type="submit"
            class="grid h-11 w-11 shrink-0 place-items-center rounded-2xl bg-gradient-to-br from-blue-600 via-indigo-500 to-purple-600 text-white shadow-[0_12px_24px_rgba(99,102,241,0.24)] transition hover:-translate-y-0.5 hover:from-blue-700 hover:via-indigo-600 hover:to-purple-700 disabled:cursor-not-allowed disabled:opacity-50 disabled:shadow-none"
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
            class="inline-flex items-center gap-1 rounded-full border border-slate-200 px-2.5 py-1 transition hover:border-indigo-200 hover:bg-indigo-50/70 hover:text-indigo-700 dark:border-dark-700 dark:hover:border-indigo-700 dark:hover:text-indigo-300"
          >
            <Icon name="mail" class="h-3.5 w-3.5" />
            {{ config.supportEmail }}
          </a>
          <a
            v-if="config.supportUrl"
            :href="config.supportUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex items-center gap-1 rounded-full border border-slate-200 px-2.5 py-1 transition hover:border-indigo-200 hover:bg-indigo-50/70 hover:text-indigo-700 dark:border-dark-700 dark:hover:border-indigo-700 dark:hover:text-indigo-300"
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
      class="grid h-14 w-14 place-items-center rounded-full bg-gradient-to-br from-blue-600 via-indigo-500 to-purple-600 text-white shadow-[0_16px_40px_rgba(99,102,241,0.32)] transition hover:-translate-y-0.5 hover:from-blue-700 hover:via-indigo-600 hover:to-purple-700 focus:outline-none focus:ring-4 focus:ring-indigo-200 dark:focus:ring-indigo-900/40"
      :aria-label="open ? t('supportChat.close') : t('supportChat.open')"
      :title="open ? t('supportChat.close') : t('supportChat.open')"
      @click="toggleOpen"
    >
      <Icon :name="open ? 'x' : 'chatBubble'" size="lg" />
    </button>
  </div>
</template>

<style scoped>
.support-chat-markdown {
  color: rgb(31 41 55);
  font-size: 0.875rem;
  line-height: 1.7;
}

.dark .support-chat-markdown {
  color: rgb(229 231 235);
}

.support-chat-markdown :deep(:first-child) {
  margin-top: 0;
}

.support-chat-markdown :deep(:last-child) {
  margin-bottom: 0;
}

.support-chat-markdown :deep(p),
.support-chat-markdown :deep(ul),
.support-chat-markdown :deep(ol),
.support-chat-markdown :deep(blockquote),
.support-chat-markdown :deep(.support-chat-code-block),
.support-chat-markdown :deep(table) {
  margin: 0.75rem 0;
}

.support-chat-markdown :deep(strong) {
  color: rgb(15 23 42);
  font-weight: 700;
}

.dark .support-chat-markdown :deep(strong) {
  color: white;
}

.support-chat-markdown :deep(ul),
.support-chat-markdown :deep(ol) {
  padding-left: 1.25rem;
}

.support-chat-markdown :deep(ul) {
  list-style: disc;
}

.support-chat-markdown :deep(ol) {
  list-style: decimal;
}

.support-chat-markdown :deep(li + li) {
  margin-top: 0.25rem;
}

.support-chat-markdown :deep(a) {
  color: rgb(79 70 229);
  font-weight: 600;
  text-decoration: none;
}

.support-chat-markdown :deep(a:hover) {
  text-decoration: underline;
}

.support-chat-markdown :deep(code) {
  border-radius: 0.375rem;
  background: rgb(241 245 249);
  color: rgb(67 56 202);
  padding: 0.125rem 0.375rem;
  font-size: 0.86em;
  font-weight: 600;
}

.dark .support-chat-markdown :deep(code) {
  background: rgb(30 41 59);
  color: rgb(199 210 254);
}

.support-chat-markdown :deep(blockquote) {
  border-left: 3px solid rgb(99 102 241);
  border-radius: 0 0.75rem 0.75rem 0;
  background: rgb(238 242 255);
  color: rgb(55 65 81);
  padding: 0.625rem 0.75rem;
}

.dark .support-chat-markdown :deep(blockquote) {
  background: rgba(49, 46, 129, 0.28);
  color: rgb(209 213 219);
}

.support-chat-markdown :deep(.support-chat-code-block) {
  overflow: hidden;
  border: 1px solid rgb(226 232 240);
  border-radius: 0.875rem;
  background: rgb(15 23 42);
}

.dark .support-chat-markdown :deep(.support-chat-code-block) {
  border-color: rgb(51 65 85);
}

.support-chat-markdown :deep(.support-chat-code-toolbar) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border-bottom: 1px solid rgba(148, 163, 184, 0.22);
  background: rgb(30 41 59);
  padding: 0.45rem 0.625rem;
}

.support-chat-markdown :deep(.support-chat-code-language) {
  color: rgb(203 213 225);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.support-chat-markdown :deep(.support-chat-code-copy) {
  border-radius: 0.5rem;
  background: rgba(255, 255, 255, 0.08);
  color: rgb(226 232 240);
  font-size: 0.7rem;
  font-weight: 700;
  padding: 0.2rem 0.5rem;
  transition: background-color 0.15s ease, color 0.15s ease;
}

.support-chat-markdown :deep(.support-chat-code-copy:hover) {
  background: rgba(255, 255, 255, 0.16);
  color: white;
}

.support-chat-markdown :deep(pre) {
  overflow-x: auto;
  margin: 0;
  padding: 0.8rem 0.9rem;
  color: rgb(226 232 240);
}

.support-chat-markdown :deep(pre code) {
  display: block;
  background: transparent;
  color: inherit;
  padding: 0;
  font-size: 0.78rem;
  font-weight: 500;
  line-height: 1.65;
  white-space: pre;
}

.support-chat-markdown :deep(table) {
  width: 100%;
  border-collapse: collapse;
  overflow: hidden;
  border-radius: 0.75rem;
  font-size: 0.8rem;
}

.support-chat-markdown :deep(th),
.support-chat-markdown :deep(td) {
  border: 1px solid rgb(226 232 240);
  padding: 0.45rem 0.55rem;
}

.support-chat-markdown :deep(th) {
  background: rgb(248 250 252);
  color: rgb(15 23 42);
  font-weight: 700;
}

.dark .support-chat-markdown :deep(th),
.dark .support-chat-markdown :deep(td) {
  border-color: rgb(51 65 85);
}

.dark .support-chat-markdown :deep(th) {
  background: rgb(30 41 59);
  color: white;
}
</style>
