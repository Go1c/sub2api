import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { defineComponent } from 'vue'
import PoolAutoInspectDialog from '../PoolAutoInspectDialog.vue'

const { getPoolAutoInspectConfig, updatePoolAutoInspectConfig, runPoolAutoInspect, showSuccess, showError } = vi.hoisted(() => ({
  getPoolAutoInspectConfig: vi.fn(),
  updatePoolAutoInspectConfig: vi.fn(),
  runPoolAutoInspect: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getPoolAutoInspectConfig,
      updatePoolAutoInspectConfig,
      runPoolAutoInspect
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess, showError })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const groups = [
  { id: 1, name: 'Codex', platform: 'openai', subscription_type: '', rate_multiplier: 1 },
  { id: 7, name: '降智分组', platform: 'openai', subscription_type: '', rate_multiplier: 1 }
]

function mountDialog(show = true) {
  return mount(PoolAutoInspectDialog, {
    props: { show, groups },
    global: {
      stubs: {
        BaseDialog: defineComponent({
          props: ['show', 'title'],
          template: '<div v-if="show"><h1>{{ title }}</h1><slot /><slot name="footer" /></div>'
        }),
        Toggle: defineComponent({
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<button type="button" :aria-checked="String(modelValue)" @click="$emit(\'update:modelValue\', !modelValue)" />'
        }),
        GroupBadge: true,
        Icon: true
      }
    }
  })
}

describe('PoolAutoInspectDialog', () => {
  beforeEach(() => {
    getPoolAutoInspectConfig.mockReset().mockResolvedValue({
      enabled: false,
      interval_minutes: 5,
      success_rate_threshold: 50,
      min_samples: 4,
      add_group_ids: [],
      remove_models: [],
      notify_oauth_401: false,
      oauth_401_cooldown_minutes: 60,
      close_429_exemption_on_degrade: true
    })
    updatePoolAutoInspectConfig.mockReset().mockResolvedValue({
      enabled: true,
      interval_minutes: 5,
      success_rate_threshold: 50,
      min_samples: 4,
      add_group_ids: [1, 7],
      remove_models: ['gpt-6-astra'],
      notify_oauth_401: true,
      oauth_401_cooldown_minutes: 60,
      close_429_exemption_on_degrade: true
    })
    runPoolAutoInspect.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
  })

  it('does not load config while closed', async () => {
    mountDialog(false)
    await flushPromises()
    expect(getPoolAutoInspectConfig).not.toHaveBeenCalled()
  })

  it('saves interval, groups, models and 401 notify', async () => {
    const wrapper = mountDialog(true)
    await flushPromises()
    expect(getPoolAutoInspectConfig).toHaveBeenCalledTimes(1)

    await wrapper.get('[data-testid="pool-auto-inspect-interval"]').setValue(5)
    await wrapper.get('[data-testid="pool-auto-inspect-group-7"]').setValue(true)
    await wrapper.get('[data-testid="pool-auto-inspect-group-1"]').setValue(true)
    await wrapper.get('[data-testid="pool-auto-inspect-model-input"]').setValue('gpt-6-astra')
    await wrapper.get('[data-testid="pool-auto-inspect-model-input"]').trigger('keydown.enter')
    await wrapper.get('[data-testid="pool-auto-inspect-save"]').trigger('click')
    await flushPromises()

    expect(updatePoolAutoInspectConfig).toHaveBeenCalledWith(
      expect.objectContaining({
        interval_minutes: 5,
        add_group_ids: [7, 1],
        remove_models: ['gpt-6-astra']
      })
    )
    expect(showSuccess).toHaveBeenCalled()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('shows 401 cooldown and runs with the current form', async () => {
    runPoolAutoInspect.mockResolvedValue({
      enabled: true,
      interval_minutes: 5,
      success_rate_threshold: 50,
      min_samples: 4,
      add_group_ids: [7],
      remove_models: ['gpt-6-astra'],
      notify_oauth_401: true,
      oauth_401_cooldown_minutes: 90,
      last_result: 'accounts=1 degraded=1 oauth401=0'
    })

    const wrapper = mountDialog(true)
    await flushPromises()

    await wrapper.get('[data-testid="pool-auto-inspect-toggle-notify401"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="pool-auto-inspect-cooldown"]').setValue(90)
    await wrapper.get('[data-testid="pool-auto-inspect-group-7"]').setValue(true)
    await wrapper.get('[data-testid="pool-auto-inspect-model-input"]').setValue('gpt-6-astra')
    await wrapper.get('[data-testid="pool-auto-inspect-model-input"]').trigger('keydown.enter')
    await wrapper.get('[data-testid="pool-auto-inspect-run"]').trigger('click')
    await flushPromises()

    expect(updatePoolAutoInspectConfig).toHaveBeenCalledWith(
      expect.objectContaining({
        notify_oauth_401: true,
        oauth_401_cooldown_minutes: 90,
        close_429_exemption_on_degrade: true,
        add_group_ids: [7],
        remove_models: ['gpt-6-astra']
      })
    )
    expect(runPoolAutoInspect).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="pool-auto-inspect-status"]').text()).toContain('degraded=1')
    expect(wrapper.emitted('close')).toBeUndefined()
  })

  it('defaults close-429-exemption on and can be turned off', async () => {
    const wrapper = mountDialog(true)
    await flushPromises()

    expect(wrapper.get('[data-testid="pool-auto-inspect-toggle-close429"]').attributes('aria-checked')).toBe('true')
    await wrapper.get('[data-testid="pool-auto-inspect-toggle-close429"]').trigger('click')
    await wrapper.get('[data-testid="pool-auto-inspect-save"]').trigger('click')
    await flushPromises()

    expect(updatePoolAutoInspectConfig).toHaveBeenCalledWith(
      expect.objectContaining({ close_429_exemption_on_degrade: false })
    )
  })
})
