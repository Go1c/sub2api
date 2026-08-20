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

import type { SiteMessageCompensationBatch } from '@/types'
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

function expectedMoney(value: number): string {
  const amount = value.toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
  return `admin.siteMessageManagement.money:${JSON.stringify({ amount })}`
}

function compensationBatch(
  overrides: Partial<SiteMessageCompensationBatch> = {},
): SiteMessageCompensationBatch {
  return {
    id: 'CMP-TEST',
    subject: '补偿',
    content: '基础内容',
    mode: 'selected',
    audience: '指定用户',
    recipient_count: 1,
    success_count: 1,
    failed_count: 0,
    amount: 0,
    code_count: 0,
    operator: 'admin@example.com',
    sent_at: '2026-07-16T12:00:00Z',
    codes: [],
    results: [],
    message_ids: [],
    ...overrides,
  }
}

function mockHistory(items: SiteMessageCompensationBatch[]) {
  listCompensationBatches.mockResolvedValue({
    items,
    total: items.length,
    page: 1,
    page_size: 100,
    pages: 1,
  })
}

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

  it('sums history compensation as batch totals instead of amount times code count', async () => {
    mockHistory([
      compensationBatch({
        id: 'CMP-OPUS5-20260728',
        subject: 'Opus 补偿',
        amount: 487.4,
        code_count: 24,
        recipient_count: 24,
        success_count: 24,
      }),
      compensationBatch({
        id: 'CMP-GPT-PRO-30PCT-20260813',
        subject: 'GPT Pro 补偿',
        amount: 443,
        code_count: 63,
        recipient_count: 63,
        success_count: 63,
      }),
      compensationBatch({
        id: 'CMP-ALL-NOTICE',
        subject: '全员通知',
        mode: 'all',
        audience: '全员',
        amount: 0,
        code_count: 0,
        recipient_count: 1286,
        success_count: 1286,
      }),
    ])

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain(expectedMoney(930.4))
    expect(wrapper.text()).toContain(expectedMoney(487.4))
    expect(wrapper.text()).toContain(expectedMoney(443))
    expect(wrapper.text()).not.toContain(expectedMoney(11697.6))
    expect(wrapper.text()).not.toContain(expectedMoney(27909))
    expect(wrapper.text()).not.toContain('81402.73')
  })

  it('loads a unified-face-value batch as a per-user compensation draft', async () => {
    mockHistory([
      compensationBatch({
        id: 'CMP-ADMIN-2',
        subject: '管理后台补偿',
        amount: 10,
        code_count: 2,
        recipient_count: 2,
        success_count: 2,
        results: [
          { recipient: 'alice@example.com', status: 'sent', message_id: 1 },
          { recipient: 'bob@example.com', status: 'sent', message_id: 2 },
        ],
      }),
    ])

    const wrapper = mountView()
    await flushPromises()

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.siteMessageManagement.history.resend'))
      ?.trigger('click')
    await flushPromises()

    const amountInput = wrapper.find('input[inputmode="decimal"]')
    expect(amountInput.exists()).toBe(true)
    expect((amountInput.element as HTMLInputElement).value).toBe('5.00')
  })
})
