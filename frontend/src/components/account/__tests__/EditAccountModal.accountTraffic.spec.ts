import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'

const { updateAccountMock, showErrorMock, authIsSimpleMode } = vi.hoisted(() => ({
  updateAccountMock: vi.fn(),
  showErrorMock: vi.fn(),
  authIsSimpleMode: { value: true }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: showErrorMock, showSuccess: vi.fn(), showInfo: vi.fn() })
}))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ get isSimpleMode() { return authIsSimpleMode.value } }) }))
vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      update: updateAccountMock,
      checkMixedChannelRisk: vi.fn().mockResolvedValue({ has_risk: false })
    },
    settings: {
      getWebSearchEmulationConfig: vi.fn().mockResolvedValue({ enabled: false, providers: [] }),
      getSettings: vi.fn().mockResolvedValue({})
    },
    tlsFingerprintProfiles: { list: vi.fn().mockResolvedValue([]) },
    proxyIpGroups: { list: vi.fn().mockResolvedValue([]) }
  }
}))
vi.mock('@/api/admin/accounts', () => ({ getAntigravityDefaultModelMapping: vi.fn() }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

import EditAccountModal from '../EditAccountModal.vue'

const BaseDialogStub = defineComponent({
  props: { show: { type: Boolean, default: false } },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const account = (extra: Record<string,unknown> = {}) => ({
  id: 12, name: 'OpenAI', notes: '', platform: 'openai', type: 'oauth',
  credentials: { expires_at: '2027-01-01T00:00:00Z', token_type: 'Bearer' },
  credentials_status: { has_access_token: true, has_refresh_token: true }, extra,
  proxy_id: null, concurrency: 8, priority: 1, rate_multiplier: 1, status: 'active',
  group_ids: [], expires_at: null, auto_pause_on_expired: false
})

function mountModal(value = account()) {
  return mount(EditAccountModal, {
    props: { show: true, account: value, proxies: [], groups: [] },
    global: { stubs: {
      BaseDialog: BaseDialogStub, Select: true, Icon: true, ProxySelector: true, OpenAIAccountProxyFields: true,
      GroupSelector: true, ModelWhitelistSelector: true
    } }
  })
}

const strictToggle = (wrapper: ReturnType<typeof mountModal>) =>
  wrapper.get('[data-testid=strict-rpm-toggle]') as unknown as { element: HTMLInputElement; setValue(v: unknown): Promise<void> }

describe('EditAccountModal embedded account traffic controls', () => {
  beforeEach(() => {
    authIsSimpleMode.value = true
    updateAccountMock.mockReset()
    showErrorMock.mockReset()
    updateAccountMock.mockResolvedValue(account())
  })

  it('loads the draft from extra and saves it with the account', async () => {
    const wrapper = mountModal(account({ account_traffic_control: { adaptive_enabled: true, adaptive_mode: 'observe' } }))
    expect((strictToggle(wrapper).element as HTMLInputElement).checked).toBe(false)
    expect((wrapper.get('[data-testid=adaptive-toggle]').element as HTMLInputElement).checked).toBe(true)
    await strictToggle(wrapper).setValue(true)
    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await vi.waitFor(() => expect(updateAccountMock).toHaveBeenCalled())
    const payload = updateAccountMock.mock.calls[0][1] as Record<string, any>
    expect(payload.extra.account_traffic_control).toEqual(expect.objectContaining({
      strict_rpm_enabled: true, adaptive_enabled: true, adaptive_mode: 'observe', rpm: 60, burst: 5
    }))
  })

  it('does not resend the traffic policy when nothing changed', async () => {
    const wrapper = mountModal(account({ keep: 'me' }))
    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await vi.waitFor(() => expect(updateAccountMock).toHaveBeenCalled())
    const payload = updateAccountMock.mock.calls[0][1] as Record<string, any>
    expect(payload.extra || {}).not.toHaveProperty('account_traffic_control')
  })

  it('keeps dirty draft edits when the same account is re-synced', async () => {
    const base = account()
    const wrapper = mountModal(base)
    await strictToggle(wrapper).setValue(true)
    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true, account: { ...base, name: 'refreshed' } })
    expect((strictToggle(wrapper).element as HTMLInputElement).checked).toBe(true)
    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await vi.waitFor(() => expect(updateAccountMock).toHaveBeenCalled())
    const payload = updateAccountMock.mock.calls[0][1] as Record<string, any>
    expect(payload.extra.account_traffic_control.strict_rpm_enabled).toBe(true)
  })

  it('reloads the draft when switching to another account', async () => {
    const wrapper = mountModal(account({ account_traffic_control: { strict_rpm_enabled: true } }))
    expect((strictToggle(wrapper).element as HTMLInputElement).checked).toBe(true)
    const other = account()
    other.id = 13
    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true, account: other })
    expect((strictToggle(wrapper).element as HTMLInputElement).checked).toBe(false)
  })

  it('blocks submit when strict RPM numbers are invalid', async () => {
    const wrapper = mountModal(account())
    await strictToggle(wrapper).setValue(true)
    const numbers = wrapper.findAll('#edit-account-form details input[type=number]')
    await numbers[0].setValue(0)
    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    expect(updateAccountMock).not.toHaveBeenCalled()
    expect(showErrorMock).toHaveBeenCalledWith('RPM 必须为 1–60000 的整数，突发额度必须为正整数，突发额度不能大于 RPM')
  })
})
