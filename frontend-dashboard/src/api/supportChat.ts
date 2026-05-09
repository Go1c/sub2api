export interface SupportChatConfig {
  title: string
  welcomeMessage: string
  supportEmail?: string
  supportUrl?: string
  officialContactText: string
}

export interface SupportChatUser {
  id?: string
  email?: string
}

export interface SupportChatRequest {
  message: string
  conversationId?: string
  locale?: string
  user?: SupportChatUser
}

export interface SupportChatSource {
  title?: string
  name?: string
  url?: string
  source?: string
  page?: number
  content?: string
  [key: string]: unknown
}

export interface SupportChatStreamHandlers {
  onAnswer?: (chunk: string) => void
  onSources?: (sources: SupportChatSource[]) => void
  onConversationId?: (conversationId: string) => void
  onError?: (message: string) => void
  onEnd?: () => void
}

interface SupportChatClientOptions {
  gatewayUrl?: string
  fetcher?: typeof fetch
  locale?: string
  signal?: AbortSignal
}

type SupportChatEvent = {
  type?: string
  answer?: string
  content?: string
  text?: string
  error?: string
  conversation_id?: string
  conversationId?: string
  id?: string
  source?: SupportChatSource | SupportChatSource[]
  sources?: SupportChatSource[]
}

const defaultConfig: SupportChatConfig = {
  title: 'LumioAPI Support',
  welcomeMessage: 'Ask a question and the AI support assistant will answer from the LumioAPI docs.',
  officialContactText: 'Contact official support'
}

export function isSupportChatEnabled() {
  return (
    import.meta.env.VITE_SUPPORT_CHAT_ENABLED === 'true' &&
    Boolean(import.meta.env.VITE_SUPPORT_CHAT_GATEWAY_URL)
  )
}

export async function fetchSupportChatConfig(
  options: SupportChatClientOptions = {}
): Promise<SupportChatConfig> {
  const fetcher = options.fetcher ?? fetch
  const path = options.locale
    ? `/widget-config?locale=${encodeURIComponent(options.locale)}`
    : '/widget-config'
  const response = await fetcher(buildGatewayURL(path, options.gatewayUrl), {
    headers: { Accept: 'application/json' }
  })

  if (!response.ok) {
    throw new Error(`Support chat config request failed (${response.status})`)
  }

  return { ...defaultConfig, ...(await response.json()) }
}

export async function streamSupportChat(
  request: SupportChatRequest,
  handlers: SupportChatStreamHandlers,
  options: SupportChatClientOptions = {}
) {
  const fetcher = options.fetcher ?? fetch
  const response = await fetcher(buildGatewayURL('/chat/stream', options.gatewayUrl), {
    method: 'POST',
    headers: {
      Accept: 'text/event-stream',
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(request),
    signal: options.signal
  })

  if (!response.ok) {
    throw new Error(`Support chat request failed (${response.status})`)
  }
  if (!response.body) {
    throw new Error('Support chat response did not include a stream')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    buffer = consumeSSEBuffer(buffer, (event) => dispatchEvent(event, handlers))
  }

  buffer += decoder.decode()
  consumeSSEBuffer(`${buffer}\n\n`, (event) => dispatchEvent(event, handlers))
}

function buildGatewayURL(path: string, gatewayUrl = import.meta.env.VITE_SUPPORT_CHAT_GATEWAY_URL) {
  if (!gatewayUrl) {
    throw new Error('VITE_SUPPORT_CHAT_GATEWAY_URL is not configured')
  }
  return `${gatewayUrl.replace(/\/+$/, '')}${path}`
}

function consumeSSEBuffer(buffer: string, onEvent: (event: SupportChatEvent) => void) {
  let remaining = buffer

  while (true) {
    const match = remaining.match(/\r?\n\r?\n/)
    if (!match || match.index === undefined) break

    const frame = remaining.slice(0, match.index)
    remaining = remaining.slice(match.index + match[0].length)
    const event = parseSSEFrame(frame)
    if (event) onEvent(event)
  }

  return remaining
}

function parseSSEFrame(frame: string): SupportChatEvent | null {
  const data = frame
    .split(/\r?\n/)
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice(5).trimStart())
    .join('\n')
    .trim()

  if (!data) return null
  if (data === '[DONE]') return { type: 'end' }

  try {
    return JSON.parse(data) as SupportChatEvent
  } catch {
    return { type: 'answer', answer: data }
  }
}

function dispatchEvent(event: SupportChatEvent, handlers: SupportChatStreamHandlers) {
  const type = event.type ?? inferEventType(event)
  if (type === 'answer') {
    const chunk = event.answer ?? event.content ?? event.text ?? ''
    if (chunk) handlers.onAnswer?.(chunk)
    return
  }
  if (type === 'source') {
    const sources = normalizeSources(event.sources ?? event.source)
    if (sources.length > 0) handlers.onSources?.(sources)
    return
  }
  if (type === 'id' || type === 'conversation_id') {
    const conversationId = event.conversation_id ?? event.conversationId ?? event.id
    if (conversationId) handlers.onConversationId?.(conversationId)
    return
  }
  if (type === 'error') {
    handlers.onError?.(event.error ?? 'Support chat failed')
    return
  }
  if (type === 'end') {
    handlers.onEnd?.()
  }
}

function inferEventType(event: SupportChatEvent) {
  if (event.error) return 'error'
  if (event.sources || event.source) return 'source'
  if (event.conversation_id || event.conversationId) return 'id'
  if (event.answer || event.content || event.text) return 'answer'
  return 'end'
}

function normalizeSources(value: SupportChatSource | SupportChatSource[] | undefined) {
  if (!value) return []
  return Array.isArray(value) ? value : [value]
}
