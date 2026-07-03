import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const { listCompensationBatches, sendCompensationBatch, showError, showSuccess, showWarning, showInfo } = vi.hoisted(() => ({
  listCompensationBatches: vi.fn(),
  sendCompensationBatch: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
  showInfo: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    siteMessages: {
      listCompensationBatches,
      sendCompensationBatch,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showWarning,
    showInfo,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (!params) return key
        return `${key}:${JSON.stringify(params)}`
      },
    }),
  }
})

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (_error: unknown, fallback: string) => fallback,
}))

import SiteMessageManagementView from '../SiteMessageManagementView.vue'

const AppLayoutStub = { template: '<div><slot /></div>' }
const IconStub = { template: '<span />' }
const RouterLinkStub = { template: '<a><slot /></a>' }
const ToggleStub = defineComponent({
  props: {
    modelValue: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['update:modelValue'],
  template: `
    <input
      v-bind="$attrs"
      type="checkbox"
      :checked="modelValue"
      @change="$emit('update:modelValue', $event.target.checked)"
    />
  `,
})

function mountView() {
  return mount(SiteMessageManagementView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        Icon: IconStub,
        RouterLink: RouterLinkStub,
        Toggle: ToggleStub,
      },
    },
  })
}

async function openCompose(wrapper: ReturnType<typeof mountView>) {
  await wrapper
    .findAll('button')
    .find((button) => button.text().includes('admin.siteMessageManagement.tabs.new'))
    ?.trigger('click')
  await flushPromises()
}

describe('SiteMessageManagementView', () => {
  beforeEach(() => {
    listCompensationBatches.mockReset()
    sendCompensationBatch.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    showWarning.mockReset()
    showInfo.mockReset()
    listCompensationBatches.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 100,
      pages: 0,
    })
    sendCompensationBatch.mockResolvedValue({
      id: 'CMP-20260703-120000',
      subject: '回归邀请',
      content: '欢迎回来',
      mode: 'selected',
      audience: '指定 1 个用户',
      recipient_count: 1,
      success_count: 1,
      failed_count: 0,
      amount: 0,
      code_count: 0,
      operator: 'admin@example.com',
      sent_at: '2026-07-03T12:00:00Z',
      codes: [],
      results: [],
      message_ids: [1],
    })
  })

  it('sends an email copy when the admin enables the batch email switch', async () => {
    const wrapper = mountView()
    await flushPromises()
    await openCompose(wrapper)

    await wrapper.find('[data-testid="site-message-recipient-input"]').setValue('alice@example.com')
    await wrapper.find('[data-testid="site-message-subject-input"]').setValue('回归邀请')
    await wrapper.find('[data-testid="site-message-content-input"]').setValue('欢迎回来')
    await wrapper.find('[data-testid="site-message-send-email-toggle"]').setValue(true)
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(sendCompensationBatch).toHaveBeenCalledWith(expect.objectContaining({
      recipient_mode: 'selected',
      recipient_emails: ['alice@example.com'],
      subject: '回归邀请',
      content: '欢迎回来',
      send_email: true,
    }))
  })

  it('passes inactive days for all-user comeback campaigns', async () => {
    const wrapper = mountView()
    await flushPromises()
    await openCompose(wrapper)

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.siteMessageManagement.mode.all'))
      ?.trigger('click')
    await wrapper.find('[data-testid="site-message-recipient-filter-inactive"]').trigger('click')
    await wrapper.find('[data-testid="site-message-inactive-days-input"]').setValue('3')
    await wrapper.find('[data-testid="site-message-subject-input"]').setValue('回归邀请')
    await wrapper.find('[data-testid="site-message-content-input"]').setValue('欢迎回来')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(sendCompensationBatch).toHaveBeenCalledWith(expect.objectContaining({
      recipient_mode: 'all',
      recipient_emails: [],
      inactive_days: 3,
      send_email: false,
    }))
  })
})
