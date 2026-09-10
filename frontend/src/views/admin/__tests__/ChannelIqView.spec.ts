import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { ChannelIQOverview } from '@/api/admin/channelIq'
import ChannelIqView from '@/views/admin/ChannelIqView.vue'

const { getOverview, updateSettings, runAll, runOne, getAllGroups, showSuccess, showError } = vi.hoisted(() => ({
  getOverview: vi.fn(),
  updateSettings: vi.fn(),
  runAll: vi.fn(),
  runOne: vi.fn(),
  getAllGroups: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    channelIq: {
      getOverview,
      updateSettings,
      runAll,
      runOne
    },
    groups: {
      getAll: getAllGroups
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
    useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => {
      if (params) return `${key}:${JSON.stringify(params)}`
      return key
    } })
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
      <div v-for="row in data" :key="row.account_id" class="iq-row">
        <span class="name">{{ row.name }}</span>
        <slot name="cell-status" :row="row" />
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

function makeOverview(overrides: Partial<ChannelIQOverview> = {}): ChannelIQOverview {
  return {
    settings: {
      group_ids: [7],
      auto_enabled: false,
      interval_seconds: 1800,
      prompt: 'Generate an SVG of a pelican riding a bicycle',
      model: 'gpt-6-astra'
    },
    items: [
      {
        account_id: 11,
        name: 'astra',
        status: 'idle',
        test_count: 1,
        model: 'gpt-6-astra',
        reasoning_effort: 'low',
        duration_ms: 149700,
        total_tokens: 4971,
        svg: '<svg xmlns="http://www.w3.org/2000/svg"></svg>'
      }
    ],
    ...overrides
  }
}

function mountView() {
  return mount(ChannelIqView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        BaseDialog: BaseDialogStub,
        EmptyState: true,
        Toggle: true,
        Icon: true,
        Teleport: true,
        Transition: false
      }
    }
  })
}

describe('ChannelIqView', () => {
  beforeEach(() => {
    for (const fn of [getOverview, updateSettings, runAll, runOne, getAllGroups, showSuccess, showError]) {
      fn.mockReset()
    }
    getOverview.mockResolvedValue(makeOverview())
    updateSettings.mockResolvedValue(makeOverview().settings)
    runAll.mockResolvedValue({ started: true })
    runOne.mockResolvedValue({ started: true })
    getAllGroups.mockResolvedValue([
      { id: 7, name: 'codex', platform: 'openai' },
      { id: 8, name: 'claude', platform: 'anthropic' }
    ])
  })

  it('lists accounts from selected groups', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(getOverview).toHaveBeenCalled()
    expect(wrapper.text()).toContain('astra')
  })

  it('starts a concurrent run for every listed account', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="channel-iq-run-all"]').trigger('click')
    await flushPromises()
    expect(runAll).toHaveBeenCalled()
    expect(showSuccess).toHaveBeenCalledWith('admin.channelIq.started')
  })

  it('saves selected groups from settings', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="channel-iq-settings"]').trigger('click')
    await flushPromises()
    const claude = wrapper.get('[data-test="channel-iq-group-8"]')
    await claude.setValue(true)
    await wrapper.get('[data-test="channel-iq-save-settings"]').trigger('click')
    await flushPromises()
    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        group_ids: [7, 8],
        auto_enabled: false,
        interval_seconds: 1800,
        model: 'gpt-6-astra'
      })
    )
  })
})
