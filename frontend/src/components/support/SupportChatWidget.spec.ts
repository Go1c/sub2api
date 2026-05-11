// @vitest-environment jsdom

import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'
import zhHant from '@/i18n/locales/zh-Hant'
import { useAuthStore } from '@/stores/auth'
import SupportChatWidget from './SupportChatWidget.vue'
import { fetchSupportChatConfig, fetchSupportChatPublicSettings, streamSupportChat } from '@/api/supportChat'

vi.mock('@/api/supportChat', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/supportChat')>()
  return {
    ...actual,
    fetchSupportChatConfig: vi.fn(),
    fetchSupportChatPublicSettings: vi.fn(),
    streamSupportChat: vi.fn()
  }
})

describe('SupportChatWidget', () => {
  beforeEach(() => {
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    vi.unstubAllGlobals()
    vi.stubGlobal('localStorage', createMemoryStorage())
    localStorage.clear()
    vi.clearAllMocks()
    vi.unstubAllEnvs()
    vi.stubEnv('VITE_SUPPORT_CHAT_ENABLED', 'false')
    vi.stubEnv('VITE_SUPPORT_CHAT_GATEWAY_URL', '')
    vi.mocked(fetchSupportChatPublicSettings).mockResolvedValue({
      support_chat_enabled: true,
      support_chat_gateway_url: 'https://gateway.example.com',
      support_chat_title: '',
      support_chat_welcome_message: '',
      support_chat_official_contact_text: ''
    })
    vi.mocked(fetchSupportChatConfig).mockResolvedValue({
      title: 'LumioAPI Support',
      welcomeMessage: 'Ask us anything',
      supportEmail: 'support@example.com',
      supportUrl: 'https://support.example.com',
      officialContactText: 'Contact support'
    })
  })

  it('stays hidden when public settings disable support chat', async () => {
    vi.mocked(fetchSupportChatPublicSettings).mockResolvedValue({
      support_chat_enabled: false,
      support_chat_gateway_url: 'https://gateway.example.com',
      support_chat_title: '',
      support_chat_welcome_message: '',
      support_chat_official_contact_text: ''
    })

    const wrapper = mountWidget()
    await flushPromises()

    expect(wrapper.find('[data-testid="support-chat-toggle"]').exists()).toBe(false)
  })

  it('opens the chat panel and loads public gateway config', async () => {
    const wrapper = mountWidget()
    await flushPromises()

    expect(fetchSupportChatConfig).toHaveBeenCalledWith({
      locale: 'en-US',
      gatewayUrl: 'https://gateway.example.com'
    })

    await wrapper.find('[data-testid="support-chat-toggle"]').trigger('click')

    const panel = wrapper.find('[data-testid="support-chat-panel"]')
    expect(panel.exists()).toBe(true)
    expect(panel.classes()).toContain('h-[min(620px,calc(100vh-7rem))]')
    expect(panel.classes()).toContain('sm:w-[420px]')
    expect(wrapper.text()).toContain('LumioAPI Support')
    expect(wrapper.text()).toContain('Ask us anything')
  })

  it('sends logged-in user context and hides streamed answer sources', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const auth = useAuthStore()
    auth.token = 'token'
    auth.user = {
      id: 1,
      username: 'user',
      email: 'u@example.com',
      role: 'user',
      balance: 10,
      concurrency: 1,
      status: 'active',
      allowed_groups: null,
      balance_notify_enabled: false,
      balance_notify_threshold: null,
      balance_notify_extra_emails: [],
      created_at: '2026-05-11T00:00:00Z',
      updated_at: '2026-05-11T00:00:00Z'
    }

    vi.mocked(streamSupportChat).mockImplementation(async (_request, handlers) => {
      handlers.onAnswer?.('Recharge from Billing.')
      handlers.onSources?.([{ title: 'Billing FAQ', url: 'https://docs.example.com/billing' }])
      handlers.onConversationId?.('conv-2')
      handlers.onEnd?.()
    })

    const wrapper = mountWidget(pinia)
    await flushPromises()
    await wrapper.find('[data-testid="support-chat-toggle"]').trigger('click')
    await wrapper.find('[data-testid="support-chat-input"]').setValue('How do I recharge?')
    await wrapper.find('[data-testid="support-chat-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(streamSupportChat).toHaveBeenCalledWith(
      {
        message: 'How do I recharge?',
        locale: 'en-US',
        user: { id: '1', email: 'u@example.com' }
      },
      expect.any(Object),
      { gatewayUrl: 'https://gateway.example.com' }
    )
    expect(wrapper.text()).toContain('Recharge from Billing.')
    expect(wrapper.text()).not.toContain('Billing FAQ')
    expect(wrapper.find('a[href="https://docs.example.com/billing"]').exists()).toBe(false)
  })

  it('renders assistant answers as safe LLM markdown with highlighted code blocks', async () => {
    vi.mocked(streamSupportChat).mockImplementation(async (_request, handlers) => {
      handlers.onAnswer?.(
        [
          '**Codex CLI** 安装步骤：',
          '',
          '- 需要 `Node.js 18+`',
          '- 执行安装命令',
          '',
          '```bash',
          'npm install -g @openai/codex',
          '```',
          '',
          '> 如果失败，请检查网络代理。'
        ].join('\n')
      )
      handlers.onEnd?.()
    })

    const wrapper = mountWidget()
    await flushPromises()
    await wrapper.find('[data-testid="support-chat-toggle"]').trigger('click')
    await wrapper.find('[data-testid="support-chat-input"]').setValue('如何安装 Codex？')
    await wrapper.find('[data-testid="support-chat-form"]').trigger('submit.prevent')
    await flushPromises()

    const markdown = wrapper.find('[data-testid="support-chat-markdown"]')
    expect(markdown.exists()).toBe(true)
    expect(markdown.find('strong').text()).toBe('Codex CLI')
    expect(markdown.find('ul li code').text()).toBe('Node.js 18+')
    expect(markdown.find('.support-chat-code-block pre code').text()).toContain('npm install -g @openai/codex')
    expect(markdown.find('.support-chat-code-language').text()).toBe('bash')
    expect(markdown.find('blockquote').text()).toContain('检查网络代理')
  })

  it('sanitizes assistant markdown and copies code block contents', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(window, 'isSecureContext', { value: true, configurable: true })
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true
    })

    vi.mocked(streamSupportChat).mockImplementation(async (_request, handlers) => {
      handlers.onAnswer?.(
        [
          '<img src=x onerror="alert(1)">',
          '[bad](javascript:alert(1))',
          '',
          '```ts',
          'console.log("safe")',
          '```'
        ].join('\n')
      )
      handlers.onEnd?.()
    })

    const wrapper = mountWidget()
    await flushPromises()
    await wrapper.find('[data-testid="support-chat-toggle"]').trigger('click')
    await wrapper.find('[data-testid="support-chat-input"]').setValue('测试格式')
    await wrapper.find('[data-testid="support-chat-form"]').trigger('submit.prevent')
    await flushPromises()

    const html = wrapper.find('[data-testid="support-chat-markdown"]').html()
    expect(html).not.toContain('onerror')
    expect(html).not.toContain('javascript:alert')

    await wrapper.find('.support-chat-code-copy').trigger('click')

    expect(writeText).toHaveBeenCalledWith('console.log("safe")')
  })

  it('uses homepage-aligned gradient accents for the main actions', async () => {
    const wrapper = mountWidget()
    await flushPromises()

    const toggle = wrapper.find('[data-testid="support-chat-toggle"]')
    expect(toggle.classes()).toContain('bg-gradient-to-br')
    expect(toggle.classes()).toContain('from-blue-600')

    await toggle.trigger('click')

    const sendButton = wrapper.find('[data-testid="support-chat-form"] button[type="submit"]')
    expect(sendButton.classes()).toContain('bg-gradient-to-br')
    expect(sendButton.classes()).toContain('from-blue-600')
  })

  it('sends the current message when Enter is pressed and preserves Shift+Enter for new lines', async () => {
    vi.mocked(streamSupportChat).mockImplementation(async (_request, handlers) => {
      handlers.onAnswer?.('Sure.')
      handlers.onEnd?.()
    })

    const wrapper = mountWidget()
    await flushPromises()
    await wrapper.find('[data-testid="support-chat-toggle"]').trigger('click')

    const input = wrapper.find('[data-testid="support-chat-input"]')
    await input.setValue('first line')
    await input.trigger('keydown.enter', { shiftKey: true })

    expect(streamSupportChat).not.toHaveBeenCalled()

    await input.trigger('keydown.enter')
    await flushPromises()

    expect(streamSupportChat).toHaveBeenCalledWith(
      {
        message: 'first line',
        locale: 'en-US'
      },
      expect.any(Object),
      { gatewayUrl: 'https://gateway.example.com' }
    )
  })

  it('lets the floating chat launcher move to a dragged viewport position', async () => {
    const wrapper = mountWidget()
    await flushPromises()

    const launcher = wrapper.find('[data-testid="support-chat-toggle"]')
    await launcher.trigger('pointerdown', {
      pointerId: 1,
      clientX: 760,
      clientY: 560
    })
    window.dispatchEvent(createPointerLikeEvent('pointermove', { pointerId: 1, clientX: 620, clientY: 420 }))
    window.dispatchEvent(createPointerLikeEvent('pointerup', { pointerId: 1, clientX: 620, clientY: 420 }))
    await flushPromises()

    const root = wrapper.find('[data-testid="support-chat-root"]')
    expect(root.attributes('style')).toContain('left:')
    expect(root.attributes('style')).toContain('top:')
    expect(root.classes()).not.toContain('bottom-4')
    expect(root.classes()).not.toContain('right-4')
  })

  it('uses a dark-mode chat surface instead of the light message gradient', async () => {
    const wrapper = mountWidget()
    await flushPromises()
    await wrapper.find('[data-testid="support-chat-toggle"]').trigger('click')

    const surface = wrapper.find('[data-testid="support-chat-surface"]')
    expect(surface.classes().join(' ')).toContain('dark:bg-[linear-gradient')
  })

  it('uses the active Traditional Chinese locale for the next support request', async () => {
    const i18n = createTestI18n('zh-Hant')
    const wrapper = mountWidget(createPinia(), i18n)
    await flushPromises()
    await wrapper.find('[data-testid="support-chat-toggle"]').trigger('click')
    await wrapper.find('[data-testid="support-chat-input"]').setValue('如何充值？')
    await wrapper.find('[data-testid="support-chat-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(fetchSupportChatConfig).toHaveBeenCalledWith({
      locale: 'zh-Hant',
      gatewayUrl: 'https://gateway.example.com'
    })
    expect(streamSupportChat).toHaveBeenCalledWith(
      {
        message: '如何充值？',
        locale: 'zh-Hant'
      },
      expect.any(Object),
      { gatewayUrl: 'https://gateway.example.com' }
    )
  })
})

function createMemoryStorage(): Storage {
  const values = new Map<string, string>()
  return {
    get length() {
      return values.size
    },
    clear() {
      values.clear()
    },
    getItem(key: string) {
      return values.get(key) ?? null
    },
    key(index: number) {
      return Array.from(values.keys())[index] ?? null
    },
    removeItem(key: string) {
      values.delete(key)
    },
    setItem(key: string, value: string) {
      values.set(key, value)
    }
  }
}

function createPointerLikeEvent(type: string, init: MouseEventInit & { pointerId: number }) {
  const event = new MouseEvent(type, init)
  Object.defineProperty(event, 'pointerId', { value: init.pointerId })
  return event
}

function mountWidget(pinia = createPinia(), i18n = createTestI18n()) {
  setActivePinia(pinia)
  return mount(SupportChatWidget, {
    global: {
      plugins: [pinia, i18n]
    }
  })
}

function createTestI18n(locale = 'en') {
  return createI18n({
    legacy: false,
    locale,
    fallbackLocale: 'en',
    messages: {
      en,
      zh,
      'zh-Hant': zhHant
    }
  })
}
