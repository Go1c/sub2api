import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { defineComponent } from 'vue'
import PoolAutoInspectDialog from '../PoolAutoInspectDialog.vue'

const { getPoolAutoInspectConfig, updatePoolAutoInspectConfig, getPoolAutoInspectLog, runPoolAutoInspect, showSuccess, showError } = vi.hoisted(() => ({
  getPoolAutoInspectConfig: vi.fn(),
  updatePoolAutoInspectConfig: vi.fn(),
  getPoolAutoInspectLog: vi.fn(),
  runPoolAutoInspect: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getPoolAutoInspectConfig,
      updatePoolAutoInspectConfig,
      getPoolAutoInspectLog,
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

const defaultQuestion =
  '黑色袋子中有苹果味、桃子味、西瓜味糖果;每种分为圆形和五角星形，可用手感区分形状。圆形依次有7、9、8颗;五角星形依次有7、6、4颗。事先决定摸出的数量，最少取多少颗，才能保证拿到不同形状的苹果味和桃子味糖果?'

function storedConfig(overrides: Record<string, unknown> = {}) {
  return {
    enabled: false,
    interval_minutes: 10,
    jitter_seconds: 60,
    model: 'gpt-6-astra',
    question: defaultQuestion,
    answer: '21',
    fuzzy_match: true,
    correct_group_id: 0,
    incorrect_group_id: 0,
    pause_minutes: 1,
    disable_first_import_on_incorrect: false,
    ...overrides
  }
}

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
        })
      }
    }
  })
}

describe('PoolAutoInspectDialog', () => {
  beforeEach(() => {
    getPoolAutoInspectConfig.mockReset().mockResolvedValue(storedConfig())
    updatePoolAutoInspectConfig.mockReset().mockResolvedValue(
      storedConfig({ enabled: true, correct_group_id: 1, incorrect_group_id: 7 })
    )
    getPoolAutoInspectLog.mockReset().mockResolvedValue([])
    runPoolAutoInspect.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
  })

  it('does not load config while closed', async () => {
    mountDialog(false)
    await flushPromises()
    expect(getPoolAutoInspectConfig).not.toHaveBeenCalled()
  })

  it('saves interval, jitter, groups and answer', async () => {
    const wrapper = mountDialog(true)
    await flushPromises()
    expect(getPoolAutoInspectConfig).toHaveBeenCalledTimes(1)

    await wrapper.get('[data-testid="pool-auto-inspect-interval"]').setValue(10)
    await wrapper.get('[data-testid="pool-auto-inspect-jitter"]').setValue(60)
    await wrapper.get('[data-testid="pool-auto-inspect-correct-group"]').setValue('1')
    await wrapper.get('[data-testid="pool-auto-inspect-incorrect-group"]').setValue('7')
    await wrapper.get('[data-testid="pool-auto-inspect-answer"]').setValue('21')
    await wrapper.get('[data-testid="pool-auto-inspect-save"]').trigger('click')
    await flushPromises()

    expect(updatePoolAutoInspectConfig).toHaveBeenCalledWith(
      expect.objectContaining({
        interval_minutes: 10,
        jitter_seconds: 60,
        model: 'gpt-6-astra',
        answer: '21',
        fuzzy_match: true,
        correct_group_id: 1,
        incorrect_group_id: 7,
        pause_minutes: 1
      })
    )
    expect(showSuccess).toHaveBeenCalled()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('runs with the current form and shows the quiz summary', async () => {
    runPoolAutoInspect.mockResolvedValue(
      storedConfig({
        enabled: true,
        correct_group_id: 1,
        incorrect_group_id: 7,
        pause_minutes: 2,
        last_result: 'asked=1 correct=1 incorrect=0 untestable=0 moved=1'
      })
    )

    const wrapper = mountDialog(true)
    await flushPromises()

    await wrapper.get('[data-testid="pool-auto-inspect-correct-group"]').setValue('1')
    await wrapper.get('[data-testid="pool-auto-inspect-incorrect-group"]').setValue('7')
    await wrapper.get('[data-testid="pool-auto-inspect-pause"]').setValue(2)
    await wrapper.get('[data-testid="pool-auto-inspect-run"]').trigger('click')
    await flushPromises()

    expect(updatePoolAutoInspectConfig).toHaveBeenCalledWith(
      expect.objectContaining({
        correct_group_id: 1,
        incorrect_group_id: 7,
        pause_minutes: 2,
        fuzzy_match: true
      })
    )
    expect(runPoolAutoInspect).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="pool-auto-inspect-status"]').text()).toContain('moved=1')
    expect(wrapper.emitted('close')).toBeUndefined()
  })

  it('defaults fuzzy match on and can turn it off', async () => {
    const wrapper = mountDialog(true)
    await flushPromises()

    expect(wrapper.get('[data-testid="pool-auto-inspect-toggle-fuzzy"]').attributes('aria-checked')).toBe('true')
    await wrapper.get('[data-testid="pool-auto-inspect-toggle-fuzzy"]').trigger('click')
    await wrapper.get('[data-testid="pool-auto-inspect-save"]').trigger('click')
    await flushPromises()

    expect(updatePoolAutoInspectConfig).toHaveBeenCalledWith(expect.objectContaining({ fuzzy_match: false }))
  })

  it('can close a first import that answers wrong', async () => {
    const wrapper = mountDialog(true)
    await flushPromises()

    expect(wrapper.get('[data-testid="pool-auto-inspect-toggle-first-import"]').attributes('aria-checked')).toBe('false')
    await wrapper.get('[data-testid="pool-auto-inspect-toggle-first-import"]').trigger('click')
    await wrapper.get('[data-testid="pool-auto-inspect-save"]').trigger('click')
    await flushPromises()

    expect(updatePoolAutoInspectConfig).toHaveBeenCalledWith(
      expect.objectContaining({ disable_first_import_on_incorrect: true })
    )
  })

  it('shows each account group change', async () => {
    getPoolAutoInspectLog.mockResolvedValue([
      {
        at: '2026-09-29T02:00:00Z',
        account_id: 12,
        account_name: 'A账号',
        action: 'moved',
        from_group_name: 'Codex',
        to_group_name: '降智分组'
      },
      {
        at: '2026-09-29T02:01:00Z',
        account_id: 13,
        account_name: 'B账号',
        action: 'moved',
        from_group_name: '降智分组',
        to_group_name: 'Codex'
      }
    ])

    const wrapper = mountDialog(true)
    await flushPromises()

    const log = wrapper.get('[data-testid="pool-auto-inspect-log"]').text()
    expect(log).toContain('A账号')
    expect(log).toContain('Codex')
    expect(log).toContain('降智分组')
    expect(log).toContain('B账号')
  })
})
