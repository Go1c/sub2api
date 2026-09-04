import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

import SiteMessagesView from '../SiteMessagesView.vue'
import { siteMessagesAPI } from '@/api/siteMessages'
import type { SiteMessage } from '@/types'

const { showError, showSuccess, refreshUnreadCount } = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
  refreshUnreadCount: vi.fn(),
}))

const cachedPublicSettings = vi.hoisted(() => ({
  value: {
    site_messages_default_recipient_email: '',
  } as Record<string, unknown>,
}))

vi.mock('@/api/siteMessages', () => ({
  siteMessagesAPI: {
    listInbox: vi.fn(),
    listSent: vi.fn(),
    getById: vi.fn(),
    send: vi.fn(),
    reply: vi.fn(),
    resolveRecipient: vi.fn(),
    getUnreadCount: vi.fn(),
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    cachedPublicSettings: cachedPublicSettings.value,
    showError,
    showSuccess,
  }),
  useSiteMessageStore: () => ({
    refreshUnreadCount,
  }),
  useAuthStore: () => ({
    user: { id: 3, email: 'user@example.com', role: 'user' },
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (_error: unknown, fallback: string) => fallback,
}))

const AppLayoutStub = { template: '<div><slot /></div>' }
const BaseDialogStub = defineComponent({
  props: {
    show: {
      type: Boolean,
      default: false,
    },
    title: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () =>
      props.show
        ? h('div', { class: 'dialog-stub', 'data-title': props.title }, [
            slots.default?.(),
            slots.footer?.(),
          ])
        : null
  },
})

const IconStub = { template: '<span />' }
const PaginationStub = { template: '<div />' }
const AdminSenderBadgeStub = {
  props: ['isAdmin', 'label'],
  template: '<span v-if="isAdmin">{{ label }}</span>',
}

const now = '2026-05-10T03:15:01Z'

function message(overrides: Partial<SiteMessage> = {}): SiteMessage {
  return {
    id: 1,
    sender_id: 2,
    recipient_id: 3,
    subject: '测试一下',
    content: '你收到消息吗',
    created_at: now,
    updated_at: now,
    sender: {
      id: 2,
      email: 'admin@lumio.games',
      username: 'admin',
      is_admin: true,
    },
    recipient: {
      id: 3,
      email: 'user@example.com',
      username: 'user',
    },
    ...overrides,
  }
}

function mountView() {
  return mount(SiteMessagesView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        BaseDialog: BaseDialogStub,
        Icon: IconStub,
        Pagination: PaginationStub,
        AdminSenderBadge: AdminSenderBadgeStub,
      },
    },
  })
}

describe('SiteMessagesView', () => {
  beforeEach(() => {
    vi.mocked(siteMessagesAPI.listInbox).mockReset()
    vi.mocked(siteMessagesAPI.listSent).mockReset()
    vi.mocked(siteMessagesAPI.getById).mockReset()
    vi.mocked(siteMessagesAPI.send).mockReset()
    vi.mocked(siteMessagesAPI.reply).mockReset()
    vi.mocked(siteMessagesAPI.resolveRecipient).mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    refreshUnreadCount.mockReset()
    refreshUnreadCount.mockResolvedValue(0)
    cachedPublicSettings.value = {
      site_messages_default_recipient_email: '',
    }
    vi.mocked(siteMessagesAPI.listInbox).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0,
    })
  })

  it('renders existing replies and appends the reply just sent by the user', async () => {
    const parent = message()
    const existingReply = message({
      id: 2,
      sender_id: 3,
      recipient_id: 2,
      parent_id: 1,
      subject: 'Re: 测试一下',
      content: '之前回复',
      sender: {
        id: 3,
        email: 'user@example.com',
        username: 'user',
      },
      recipient: {
        id: 2,
        email: 'admin@lumio.games',
        username: 'admin',
        is_admin: true,
      },
    })
    const newReply = message({
      id: 3,
      sender_id: 3,
      recipient_id: 2,
      parent_id: 1,
      subject: 'Re: 测试一下',
      content: '新的回复',
      sender: existingReply.sender,
      recipient: existingReply.recipient,
    })

    vi.mocked(siteMessagesAPI.listInbox).mockResolvedValue({
      items: [parent],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    vi.mocked(siteMessagesAPI.getById).mockResolvedValue({
      ...parent,
      replies: [existingReply],
    })
    vi.mocked(siteMessagesAPI.reply).mockResolvedValue(newReply)

    const wrapper = mountView()
    await flushPromises()

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('测试一下'))
      ?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('你收到消息吗')
    expect(wrapper.text()).toContain('之前回复')

    await wrapper.get('textarea').setValue('新的回复')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(siteMessagesAPI.reply).toHaveBeenCalledWith(1, { content: '新的回复' })
    expect(wrapper.text()).toContain('新的回复')
  })

  it('keeps mailbox rows shrinkable so long sender addresses cannot stretch the page', async () => {
    vi.mocked(siteMessagesAPI.listInbox).mockResolvedValue({
      items: [
        message({
          subject: '恭喜中奖：关注才能抽奖哦这是一段很长的标题',
          content: '兑换码：ABCDEFGH-1234-5678-VERY-LONG',
          sender: {
            id: 2,
            email: 'admin@lumio.games',
            username: 'admin',
            is_admin: true,
          },
        }),
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    const wrapper = mountView()
    await flushPromises()

    const mailbox = wrapper.get('[data-testid="site-messages-mailbox"]')
    expect(mailbox.classes()).toContain('min-w-0')
    expect(mailbox.classes().some((cls) => cls.includes('minmax(0'))).toBe(true)

    const meta = wrapper.get('[data-testid="site-message-row-meta"]')
    expect(meta.classes()).toContain('min-w-0')
    expect(meta.get('span').classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
  })

  it('shows a back control after opening a message so the list can be restored', async () => {
    const parent = message()
    vi.mocked(siteMessagesAPI.listInbox).mockResolvedValue({
      items: [parent],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    vi.mocked(siteMessagesAPI.getById).mockResolvedValue(parent)

    const wrapper = mountView()
    await flushPromises()

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('测试一下'))
      ?.trigger('click')
    await flushPromises()

    const list = wrapper.get('[data-testid="site-messages-list"]')
    expect(list.classes()).toContain('hidden')
    expect(wrapper.get('[data-testid="site-messages-back"]').text()).toBe('common.back')

    await wrapper.get('[data-testid="site-messages-back"]').trigger('click')
    expect(wrapper.get('[data-testid="site-messages-list"]').classes()).not.toContain('hidden')
  })

  it('prefills compose recipient from the public site-message setting', async () => {
    cachedPublicSettings.value = {
      site_messages_default_recipient_email: 'support@lumio.games',
    }

    const wrapper = mountView()
    await flushPromises()

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('siteMessages.compose'))
      ?.trigger('click')
    await flushPromises()

    const recipientInput = wrapper.get('input[type="text"]').element as HTMLInputElement
    expect(recipientInput.value).toBe('support@lumio.games')
  })
})
