import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import AdminPaymentPlansView from '../orders/AdminPaymentPlansView.vue'

const mocks = vi.hoisted(() => ({
  getPlans: vi.fn(),
  previewPlanLimitSync: vi.fn(),
  syncPlanLimits: vi.fn(),
  updatePlan: vi.fn(),
  deletePlan: vi.fn(),
  getAllGroups: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin/payment', () => ({
  adminPaymentAPI: {
    getPlans: mocks.getPlans,
    previewPlanLimitSync: mocks.previewPlanLimitSync,
    syncPlanLimits: mocks.syncPlanLimits,
    updatePlan: mocks.updatePlan,
    deletePlan: mocks.deletePlan,
  },
}))

vi.mock('@/api/admin', () => ({
  default: {
    groups: {
      getAll: mocks.getAllGroups,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: mocks.showSuccess,
    showError: mocks.showError,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'payment.admin.syncPlanLimitsPreviewMessage') {
          return `matched ${params?.matched}, changed ${params?.changed}, daily ${params?.dailyLimit}, weekly ${params?.weeklyLimit}`
        }
        if (key === 'payment.admin.syncPlanLimitsSuccess') {
          return `updated ${params?.updated}`
        }
        if (key === 'payment.admin.noLimit') {
          return 'No limit'
        }
        return key
      },
    }),
  }
})

function mountView() {
  return mount(AdminPaymentPlansView, {
    global: {
      plugins: [createPinia()],
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        DataTable: {
          props: ['data'],
          template: `
            <div>
              <div v-for="row in data" :key="row.id" data-test="plan-row">
                <slot name="cell-actions" :row="row" />
              </div>
            </div>
          `,
        },
        ConfirmDialog: {
          props: ['show', 'title', 'message', 'confirmText'],
          emits: ['confirm', 'cancel'],
          template: `
            <div v-if="show" data-test="confirm-dialog">
              <h2>{{ title }}</h2>
              <p>{{ message }}</p>
              <button data-test="confirm-dialog-confirm" @click="$emit('confirm')">{{ confirmText }}</button>
            </div>
          `,
        },
        Icon: true,
        GroupBadge: true,
        PlanEditDialog: true,
      },
    },
  })
}

describe('AdminPaymentPlansView plan limit sync', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mocks.getPlans.mockReset()
    mocks.previewPlanLimitSync.mockReset()
    mocks.syncPlanLimits.mockReset()
    mocks.updatePlan.mockReset()
    mocks.deletePlan.mockReset()
    mocks.getAllGroups.mockReset()
    mocks.showSuccess.mockReset()
    mocks.showError.mockReset()
    mocks.getAllGroups.mockResolvedValue([])
    mocks.getPlans.mockResolvedValue({
      data: [
        {
          id: 42,
          group_id: null,
          name: 'Plan A',
          description: '',
          price: 9.9,
          validity_days: 30,
          validity_unit: 'day',
          quota_usd: 100,
          daily_limit_usd: 20,
          weekly_limit_usd: 80,
          features: '',
          for_sale: true,
          sort_order: 1,
        },
      ],
    })
    mocks.previewPlanLimitSync.mockResolvedValue({
      data: {
        plan_id: 42,
        daily_limit_usd: 20,
        weekly_limit_usd: 80,
        matched_count: 3,
        changed_count: 2,
      },
    })
    mocks.syncPlanLimits.mockResolvedValue({
      data: {
        plan_id: 42,
        daily_limit_usd: 20,
        weekly_limit_usd: 80,
        matched_count: 3,
        changed_count: 2,
        updated_count: 2,
      },
    })
  })

  it('previews and confirms syncing plan daily and weekly limits', async () => {
    const wrapper = mountView()
    await flushPromises()

    const syncButton = wrapper.get('[data-test="sync-plan-limits"]')
    await syncButton.trigger('click')
    await flushPromises()

    expect(mocks.previewPlanLimitSync).toHaveBeenCalledWith(42)
    expect(wrapper.text()).toContain('matched 3, changed 2, daily 20, weekly 80')

    await wrapper.get('[data-test="confirm-dialog-confirm"]').trigger('click')
    await flushPromises()

    expect(mocks.syncPlanLimits).toHaveBeenCalledWith(42)
    expect(mocks.showSuccess).toHaveBeenCalledWith('updated 2')
  })
})
