import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { TurnStateProbeOverview } from '@/api/admin/turnStateProbe'
import ChannelTurnStateView from '@/views/admin/ChannelTurnStateView.vue'

const {
  getOverview,
  updatePolicy,
  setImportBatchRuntime,
  runOne,
  clearTicket,
  getAllProxies,
  showSuccess,
  showError
} = vi.hoisted(() => ({
  getOverview: vi.fn(),
  updatePolicy: vi.fn(),
  setImportBatchRuntime: vi.fn(),
  runOne: vi.fn(),
  clearTicket: vi.fn(),
  getAllProxies: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    turnStateProbe: {
      getOverview,
      updatePolicy,
      setImportBatchRuntime,
      runOne,
      clearTicket
    },
    proxies: {
      getAll: getAllProxies
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess,
    showError
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (params) return `${key}:${JSON.stringify(params)}`
        return key
      },
      te: (key: string) => key.startsWith('admin.channelTurnState.status.')
    })
  }
})

const AppLayoutStub = defineComponent({
  template: '<main><slot /></main>'
})

const TablePageLayoutStub = defineComponent({
  template: '<section><slot name="filters" /><slot name="table" /></section>'
})

const DataTableStub = defineComponent({
  props: {
    data: { type: Array, default: () => [] },
    columns: { type: Array, default: () => [] },
    loading: { type: Boolean, default: false }
  },
  template: `
    <div>
      <div v-if="!data.length" class="empty"><slot name="empty" /></div>
      <div v-for="row in data" :key="row.account_id" class="probe-row">
        <span class="name">{{ row.name }}</span>
        <slot name="cell-status" :row="row" />
        <slot name="cell-state_hash" :row="row" />
        <slot name="cell-state_length" :row="row" />
        <slot name="cell-actions" :row="row" />
      </div>
    </div>
  `
})

const BaseDialogStub = defineComponent({
  props: {
    show: { type: Boolean, default: false },
    title: { type: String, default: '' }
  },
  template: '<div v-if="show" class="dialog"><slot /><slot name="footer" /></div>'
})

const ConfirmDialogStub = defineComponent({
  props: {
    show: { type: Boolean, default: false },
    title: { type: String, default: '' },
    message: { type: String, default: '' }
  },
  emits: ['confirm', 'cancel'],
  template:
    '<div v-if="show" class="confirm"><p>{{ message }}</p><button data-testid="confirm-clear" type="button" @click="$emit(\'confirm\')">ok</button></div>'
})

function makeOverview(overrides: Partial<TurnStateProbeOverview> = {}): TurnStateProbeOverview {
  return {
    policy: {
      enabled: true,
      proxy_ids: [3],
      dynamic: {
        host: 'us.lajiaohttp.net:2000',
        username: 'probe-user',
        password_set: true,
        region: 'Random',
        session_minutes: 5
      },
      model: 'gpt-6-astra',
      length_filter_enabled: true,
      min_state_length: 160,
      question: 'candy',
      answer: '21',
      fuzzy_match: true,
      recheck_minutes: 10,
      rpm: 6
    },
    accounts: [
      {
        account_id: 11,
        name: 'astra',
        enabled: true,
        status: 'holding',
        state_hash: 'a1b2c3d4e5f6',
        state_length: 180,
        model: 'gpt-6-astra'
      }
    ],
    ...overrides
  }
}

function mountView() {
  return mount(ChannelTurnStateView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: ConfirmDialogStub,
        EmptyState: true,
        Toggle: defineComponent({
          props: { modelValue: { type: Boolean, default: false } },
          emits: ['update:modelValue'],
          inheritAttrs: false,
          setup(props, { attrs, emit }) {
            return () =>
              h('input', {
                ...attrs,
                type: 'checkbox',
                checked: props.modelValue,
                onChange: (event: Event) =>
                  emit('update:modelValue', (event.target as HTMLInputElement).checked)
              })
          }
        }),
        Icon: true
      }
    }
  })
}

describe('ChannelTurnStateView', () => {
  beforeEach(() => {
    for (const fn of [getOverview, updatePolicy, setImportBatchRuntime, runOne, clearTicket, getAllProxies, showSuccess, showError]) {
      fn.mockReset()
    }
    getOverview.mockResolvedValue(makeOverview())
    updatePolicy.mockResolvedValue(makeOverview().policy)
    runOne.mockResolvedValue({ started: true })
    clearTicket.mockResolvedValue({ cleared: true })
    getAllProxies.mockResolvedValue([
      { id: 3, name: 'probe-a', host: '1.1.1.1', port: 8080 },
      { id: 8, name: 'probe-b', host: '2.2.2.2', port: 8080 }
    ])
  })

  it('loads overview accounts with hash and length only', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(getOverview).toHaveBeenCalled()
    expect(wrapper.text()).toContain('astra')
    expect(wrapper.get('[data-testid="turn-state-hash-11"]').text()).toBe('a1b2c3d4e5f6')
    expect(wrapper.get('[data-testid="turn-state-length-11"]').text()).toBe('180')
    expect(wrapper.text()).not.toContain('harvested-blob')
  })

  it('saves policy including selected exits', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="turn-state-settings"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="turn-state-password-set"]').text()).toContain(
      'admin.channelTurnState.passwordSet'
    )
    await wrapper.get('[data-testid="turn-state-overload-threshold"]').setValue(5)
    const extraProxy = wrapper.get('[data-testid="turn-state-proxy-8"]')
    await extraProxy.setValue(true)
    await wrapper.get('[data-testid="turn-state-save-policy"]').trigger('click')
    await flushPromises()
    expect(updatePolicy).toHaveBeenCalledWith(
      expect.objectContaining({
        enabled: true,
        proxy_ids: [3, 8],
        model: 'gpt-6-astra',
        length_filter_enabled: true,
        min_state_length: 160,
        answer: '21',
        fuzzy_match: true,
        recheck_minutes: 20,
        overload_threshold: 5,
        rpm: 6,
        dynamic: expect.objectContaining({
          host: 'us.lajiaohttp.net:2000',
          username: 'probe-user',
          region: 'Random',
          session_minutes: 5
        })
      })
    )
    const payload = updatePolicy.mock.calls[0]?.[0]
    expect(payload.dynamic.password).toBeUndefined()
    expect(showSuccess).toHaveBeenCalledWith('admin.channelTurnState.saveSuccess')
    expect(setImportBatchRuntime).not.toHaveBeenCalled()
  })

  it('toggles import-batch routing without saving probe policy', async () => {
    getOverview.mockResolvedValue(makeOverview({ import_batch_runtime_suspended: false }))
    setImportBatchRuntime.mockResolvedValue({ import_batch_runtime_suspended: true })
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="turn-state-settings"]').trigger('click')
    const runtime = wrapper.get('[data-testid="turn-state-import-batch-runtime"]')
    expect((runtime.element as HTMLInputElement).checked).toBe(true)
    await runtime.setValue(false)
    await flushPromises()
    expect(setImportBatchRuntime).toHaveBeenCalledWith(true)
    expect(updatePolicy).not.toHaveBeenCalled()
  })

  it('starts a probe for one account', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="turn-state-run-one-11"]').trigger('click')
    await flushPromises()
    expect(runOne).toHaveBeenCalledWith(11)
    expect(showSuccess).toHaveBeenCalledWith('admin.channelTurnState.started')
  })
})
