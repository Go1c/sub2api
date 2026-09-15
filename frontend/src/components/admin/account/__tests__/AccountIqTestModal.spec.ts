import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AccountIqTestModal from '../AccountIqTestModal.vue'

const { getAvailableModels, copyToClipboard } = vi.hoisted(() => ({
  getAvailableModels: vi.fn(),
  copyToClipboard: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getAvailableModels
    }
  }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'admin.accounts.iqPromptDefault': 'Generate an SVG of a pelican riding a bicycle'
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'admin.accounts.startingIqTestForAccount' && params?.name) {
          return `start-${params.name}`
        }
        if (key === 'admin.accounts.usingModel' && params?.model) {
          return `model-${params.model}`
        }
        return messages[key] || key
      }
    })
  }
})

function createStreamResponse(lines: string[]) {
  const encoder = new TextEncoder()
  const chunks = lines.map((line) => encoder.encode(line))
  let index = 0

  return {
    ok: true,
    body: {
      getReader: () => ({
        read: vi.fn().mockImplementation(async () => {
          if (index < chunks.length) {
            return { done: false, value: chunks[index++] }
          }
          return { done: true, value: undefined }
        })
      })
    }
  } as Response
}

const pelicanSvg = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40"><circle cx="20" cy="20" r="8"/></svg>'

function mountModal() {
  return mount(AccountIqTestModal, {
    props: {
      show: false,
      account: {
        id: 117,
        name: 'CPA-B18-Pro20x',
        platform: 'openai',
        type: 'oauth',
        status: 'active'
      }
    } as any,
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        Select: { template: '<div class="select-stub"></div>' },
        TextArea: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
        },
        Icon: true
      }
    }
  })
}

describe('AccountIqTestModal', () => {
  beforeEach(() => {
    getAvailableModels.mockResolvedValue([
      { id: 'gpt-image-1.5', display_name: 'GPT Image 1.5' },
      { id: 'gpt-5.4', display_name: 'GPT-5.4' },
      { id: 'gpt-6-astra', display_name: 'GPT-6 Astra' },
      { id: 'gpt-5.3-codex', display_name: 'GPT-5.3 Codex' }
    ])
    copyToClipboard.mockReset()
    Object.defineProperty(globalThis, 'localStorage', {
      value: {
        getItem: vi.fn((key: string) => (key === 'auth_token' ? 'test-token' : null)),
        setItem: vi.fn(),
        removeItem: vi.fn(),
        clear: vi.fn()
      },
      configurable: true
    })
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gpt-5.4"}\n',
        `data: ${JSON.stringify({ type: 'content', text: `thinking...\n${pelicanSvg}` })}\n`,
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('sends the pelican prompt and renders the extracted svg as an image', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const promptInput = wrapper.find('textarea.textarea-stub')
    expect((promptInput.element as HTMLTextAreaElement).value).toBe(
      'Generate an SVG of a pelican riding a bicycle'
    )

    const startButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.startIqTest'))
    expect(startButton).toBeTruthy()
    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toEqual({
      model_id: 'gpt-6-astra',
      prompt: 'Generate an SVG of a pelican riding a bicycle',
      mode: 'iq'
    })

    const preview = wrapper.find('img[alt="iq-svg-preview"]')
    expect(preview.exists()).toBe(true)
    expect(preview.attributes('src')).toContain('data:image/svg+xml')
    expect(decodeURIComponent(preview.attributes('src') || '')).toContain('<svg')
  })

  it('falls back to gpt-5.4 when gpt-6-astra is unavailable', async () => {
    getAvailableModels.mockResolvedValue([
      { id: 'gpt-5.4', display_name: 'GPT-5.4' },
      { id: 'gpt-5.3-codex', display_name: 'GPT-5.3 Codex' }
    ])
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()
    const startButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.startIqTest'))
    await startButton!.trigger('click')
    await flushPromises()
    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body).model_id).toBe('gpt-5.4')
    expect(JSON.parse(request.body).mode).toBe('iq')
  })

  it('does not render a preview for incomplete svg and reports extract failure', async () => {
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gpt-5.4"}\n',
        'data: {"type":"content","text":"thinking about the pelican <svg viewBox=\\"0 0 10 10\\">"}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any

    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const startButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.startIqTest'))
    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    expect(wrapper.find('img[alt="iq-svg-preview"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('admin.accounts.iqExtractFailed')
  })
})
