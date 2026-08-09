import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import type { AdminUser } from '@/types'
import UserEditModal from '../UserEditModal.vue'

const { updateUser, updateUserAttributeValues, showError, showSuccess } = vi.hoisted(() => ({
  updateUser: vi.fn(),
  updateUserAttributeValues: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      update: updateUser,
    },
    userAttributes: {
      updateUserAttributeValues,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn(),
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

const BaseDialogStub = {
  props: ['show'],
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
}

const createUser = (overrides: Partial<AdminUser> = {}): AdminUser => ({
  id: 42,
  username: 'invoice-user',
  email: 'invoice@example.com',
  role: 'user',
  balance: 10,
  total_recharged: 100,
  invoice_enabled: true,
  subscription_purchase_disabled: false,
  concurrency: 1,
  rpm_limit: 0,
  status: 'active',
  allowed_groups: [],
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '2026-05-01T00:00:00Z',
  updated_at: '2026-05-01T00:00:00Z',
  notes: '',
  last_active_at: null,
  last_used_at: null,
  current_concurrency: 0,
  ...overrides,
})

describe('UserEditModal invoice access', () => {
  beforeEach(() => {
    updateUser.mockReset()
    updateUserAttributeValues.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    updateUser.mockResolvedValue(createUser({ invoice_enabled: false }))
    updateUserAttributeValues.mockResolvedValue({})
  })

  it('submits the per-user invoice access toggle when updating a user', async () => {
    const wrapper = mount(UserEditModal, {
      props: {
        show: true,
        user: createUser({ invoice_enabled: true }),
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          UserAttributeForm: true,
          Icon: true,
        },
      },
    })

    const toggle = wrapper.get('[data-test="invoice-enabled-toggle"]')
    expect((toggle.element as HTMLInputElement).checked).toBe(true)

    await toggle.setValue(false)
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(updateUser).toHaveBeenCalledWith(
      42,
      expect.objectContaining({
        invoice_enabled: false,
      }),
    )
  })

  it('submits the per-user subscription purchase ban toggle when updating a user', async () => {
    updateUser.mockResolvedValue(createUser({ subscription_purchase_disabled: true }))

    const wrapper = mount(UserEditModal, {
      props: {
        show: true,
        user: createUser({ subscription_purchase_disabled: false }),
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          UserAttributeForm: true,
          Icon: true,
        },
      },
    })

    const toggle = wrapper.get('[data-test="subscription-purchase-disabled-toggle"]')
    expect((toggle.element as HTMLInputElement).checked).toBe(false)

    await toggle.setValue(true)
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(updateUser).toHaveBeenCalledWith(
      42,
      expect.objectContaining({
        subscription_purchase_disabled: true,
      }),
    )
  })
})
