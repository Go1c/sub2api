import { describe, expect, it, vi, beforeEach } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const { sendToUser } = vi.hoisted(() => ({
  sendToUser: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    siteMessages: {
      sendToUser
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

import UserSiteMessageModal from '../UserSiteMessageModal.vue'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: {
      type: Boolean,
      default: false
    }
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const user = {
  id: 7,
  email: 'recipient@example.com',
  username: 'recipient',
  role: 'user',
  balance: 0,
  concurrency: 1,
  status: 'active',
  allowed_groups: [],
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '2026-05-10T00:00:00Z',
  updated_at: '2026-05-10T00:00:00Z',
  notes: '',
  last_active_at: null,
  last_used_at: null,
  current_concurrency: 0
} as any

function mountModal() {
  return mount(UserSiteMessageModal, {
    props: {
      show: true,
      user
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub
      }
    }
  })
}

describe('UserSiteMessageModal', () => {
  beforeEach(() => {
    sendToUser.mockReset()
    sendToUser.mockResolvedValue({ id: 1 })
  })

  it('sends an email copy by default', async () => {
    const wrapper = mountModal()

    await wrapper.get('input[type="text"]').setValue('Notice')
    await wrapper.get('textarea').setValue('Message body')
    await wrapper.get('form#admin-user-site-message-form').trigger('submit.prevent')
    await flushPromises()

    expect(sendToUser).toHaveBeenCalledWith(7, {
      subject: 'Notice',
      content: 'Message body',
      send_email: true
    })
  })

  it('can send only the site message when email copy is unchecked', async () => {
    const wrapper = mountModal()

    await wrapper.get('input[type="text"]').setValue('Notice')
    await wrapper.get('textarea').setValue('Message body')
    await wrapper.get('input[type="checkbox"]').setValue(false)
    await wrapper.get('form#admin-user-site-message-form').trigger('submit.prevent')
    await flushPromises()

    expect(sendToUser).toHaveBeenCalledWith(7, {
      subject: 'Notice',
      content: 'Message body',
      send_email: false
    })
  })
})
