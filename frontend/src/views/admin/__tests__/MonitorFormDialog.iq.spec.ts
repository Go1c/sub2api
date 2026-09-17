import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import MonitorFormDialog from '@/components/admin/monitor/MonitorFormDialog.vue'
import { DEFAULT_IQ_ANSWER, DEFAULT_IQ_QUESTION } from '@/constants/channelMonitor'

const { listTemplates, accountsList, monitorCreate } = vi.hoisted(() => ({
  listTemplates: vi.fn(),
  accountsList: vi.fn(),
  monitorCreate: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    channelMonitor: {
      create: monitorCreate,
      update: vi.fn(),
    },
    channelMonitorTemplate: {
      list: listTemplates,
    },
    accounts: {
      list: (...args: unknown[]) => accountsList(...args),
      getById: vi.fn(),
    },
  },
}))

vi.mock('@/api/keys', () => ({
  keysAPI: { list: vi.fn() },
}))

vi.mock('@/api/groups', () => ({
  userGroupsAPI: { getUserGroupRates: vi.fn() },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    cachedPublicSettings: null,
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

const BaseDialogStub = defineComponent({
  props: { show: { type: Boolean, default: false } },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
})

let unmountWrapper: (() => void) | undefined

function mountDialog() {
  const wrapper = mount(MonitorFormDialog, {
    props: { show: true, monitor: null },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        Toggle: true,
        ModelTagInput: true,
        MonitorKeyPickerDialog: true,
        MonitorAdvancedRequestConfig: true,
      },
    },
  })
  unmountWrapper = () => wrapper.unmount()
  return wrapper
}

describe('MonitorFormDialog IQ mode', () => {
  beforeEach(() => {
    listTemplates.mockResolvedValue({ items: [] })
    accountsList.mockResolvedValue({ items: [] })
    monitorCreate.mockResolvedValue({ id: 1 })
  })

  afterEach(() => {
    unmountWrapper?.()
    unmountWrapper = undefined
    vi.clearAllMocks()
  })

  it('keeps interval settings and does not show the account selector', async () => {
    const wrapper = mountDialog()
    await flushPromises()

    expect(wrapper.get('[data-testid="monitor-check-mode-iq"]').exists()).toBe(true)
    await wrapper.get('[data-testid="monitor-check-mode-iq"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="monitor-linked-account"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="monitor-interval-seconds"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="monitor-interval-seconds"]').attributes('min')).toBe('15')
    expect(wrapper.get('[data-testid="monitor-interval-seconds"]').attributes('max')).toBe('3600')
    expect(wrapper.get('[data-testid="monitor-iq-question"]').element).toBeTruthy()
    expect((wrapper.get('[data-testid="monitor-iq-answer"]').element as HTMLInputElement).value).toBe(DEFAULT_IQ_ANSWER)
    expect((wrapper.get('[data-testid="monitor-iq-question"]').element as HTMLTextAreaElement).value).toContain(DEFAULT_IQ_QUESTION.slice(0, 12))
  })

  it('submits check_mode=iq with original interval and candy quiz', async () => {
    const wrapper = mountDialog()
    await flushPromises()
    await wrapper.get('[data-testid="monitor-check-mode-iq"]').trigger('click')
    await flushPromises()

    const textInputs = wrapper.findAll('input[type="text"]')
    await textInputs[0].setValue('iq monitor')
    await wrapper.get('[data-testid="monitor-endpoint"]').setValue('https://api.example.com')
    await wrapper.get('[data-testid="monitor-primary-model"]').setValue('gpt-4o-mini')
    await wrapper.find('input[type="password"]').setValue('sk-test')
    await wrapper.get('[data-testid="monitor-interval-seconds"]').setValue('90')

    await wrapper.get('#channel-monitor-form').trigger('submit')
    await flushPromises()

    expect(monitorCreate).toHaveBeenCalledWith(expect.objectContaining({
      name: 'iq monitor',
      check_mode: 'iq',
      interval_seconds: 90,
      iq_answer: DEFAULT_IQ_ANSWER,
      iq_fuzzy_match: true,
      account_id: null,
    }))
    expect(accountsList).not.toHaveBeenCalled()
  })
})
