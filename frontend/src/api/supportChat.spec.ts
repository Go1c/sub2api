import { describe, expect, it, vi } from 'vitest'
import {
  streamSupportChat,
  fetchSupportChatConfig,
  fetchSupportChatPublicSettings,
  isSupportChatEnabled,
  mergeSupportChatConfig,
  resolveSupportChatGatewayURL
} from './supportChat'

function streamResponse(chunks: string[]) {
  const encoder = new TextEncoder()
  return new Response(
    new ReadableStream({
      start(controller) {
        for (const chunk of chunks) controller.enqueue(encoder.encode(chunk))
        controller.close()
      }
    }),
    {
      status: 200,
      headers: { 'Content-Type': 'text/event-stream' }
    }
  )
}

describe('support chat API', () => {
  it('loads support chat enablement from public backend settings', async () => {
    const fetcher = vi.fn<typeof fetch>(async () =>
      new Response(
        JSON.stringify({
          code: 0,
          message: 'success',
          data: {
            support_chat_enabled: true,
            support_chat_gateway_url: 'https://gateway.example.com/',
            support_chat_title: 'LumioAPI Helper',
            support_chat_welcome_message: 'Ask from the LumioAPI docs.',
            support_chat_official_contact_text: 'Contact human support'
          }
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } }
      )
    )

    const settings = await fetchSupportChatPublicSettings({
      apiBaseUrl: 'https://api.example.com/api/v1/',
      fetcher
    })

    expect(fetcher).toHaveBeenCalledWith('https://api.example.com/api/v1/settings/public', {
      headers: { Accept: 'application/json' }
    })
    expect(isSupportChatEnabled(settings)).toBe(true)
    expect(resolveSupportChatGatewayURL(settings)).toBe('https://gateway.example.com')
    expect(
      mergeSupportChatConfig(
        {
          title: 'Gateway title',
          welcomeMessage: 'Gateway welcome',
          officialContactText: 'Gateway contact'
        },
        settings
      )
    ).toEqual({
      title: 'LumioAPI Helper',
      welcomeMessage: 'Ask from the LumioAPI docs.',
      officialContactText: 'Contact human support'
    })
  })

  it('loads public widget configuration from the gateway', async () => {
    const fetcher = vi.fn<typeof fetch>(async () =>
      new Response(
        JSON.stringify({
          title: 'LumioAPI Support',
          welcomeMessage: 'Ask us anything',
          supportEmail: 'support@example.com',
          supportUrl: 'https://support.example.com',
          officialContactText: 'Contact support'
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } }
      )
    )

    const config = await fetchSupportChatConfig({
      locale: 'zh-Hant',
      gatewayUrl: 'https://gateway.example.com/',
      fetcher
    })

    expect(fetcher).toHaveBeenCalledWith(
      'https://gateway.example.com/widget-config?locale=zh-Hant',
      {
        headers: { Accept: 'application/json' }
      }
    )
    expect(config.supportEmail).toBe('support@example.com')
    expect(JSON.stringify(config)).not.toContain('agent-secret')
  })

  it('streams answer, source, conversation id, and end events from fragmented SSE chunks', async () => {
    const fetcher = vi.fn<typeof fetch>(async () =>
      streamResponse([
        'data: {"type":"answer","answer":"Hel',
        'lo"}\n\n',
        'data: {"type":"source","sources":[{"title":"Billing FAQ","url":"https://docs.example.com/billing"}]}\n\n',
        'data: {"type":"id","conversation_id":"conv-2"}\n\n',
        'data: {"type":"end"}\n\n'
      ])
    )
    const onAnswer = vi.fn()
    const onSources = vi.fn()
    const onConversationId = vi.fn()
    const onEnd = vi.fn()

    await streamSupportChat(
      {
        message: 'How do I recharge?',
        conversationId: 'conv-1',
        locale: 'en-US',
        user: { id: 'u-1', email: 'u@example.com' }
      },
      { onAnswer, onSources, onConversationId, onEnd },
      { gatewayUrl: 'https://gateway.example.com', fetcher }
    )

    const [, init] = fetcher.mock.calls[0]
    expect(fetcher.mock.calls[0][0]).toBe('https://gateway.example.com/chat/stream')
    expect(JSON.parse(String(init?.body))).toEqual({
      message: 'How do I recharge?',
      conversationId: 'conv-1',
      locale: 'en-US',
      user: { id: 'u-1', email: 'u@example.com' }
    })
    expect(onAnswer).toHaveBeenCalledWith('Hello')
    expect(onSources).toHaveBeenCalledWith([
      { title: 'Billing FAQ', url: 'https://docs.example.com/billing' }
    ])
    expect(onConversationId).toHaveBeenCalledWith('conv-2')
    expect(onEnd).toHaveBeenCalledTimes(1)
  })

  it('throws a readable error when the gateway rejects the request', async () => {
    const fetcher = vi.fn<typeof fetch>(async () => new Response('bad gateway', { status: 502 }))

    await expect(
      streamSupportChat({ message: 'hello' }, {}, { gatewayUrl: 'https://gateway.example.com', fetcher })
    ).rejects.toThrow('Support chat request failed (502)')
  })
})
