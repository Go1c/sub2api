import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import RequestHealthBar from '../RequestHealthBar.vue'
import type { RequestOutcome } from '../requestHealth'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('RequestHealthBar', () => {
  it('opens the error dialog when a red fail slot is clicked', async () => {
    const outcomes: RequestOutcome[] = [
      { slot: 'ok' },
      { slot: 'fail', error: { statusCode: 500, message: 'upstream exploded', occurredAt: '2026-01-01T00:00:00Z', model: 'gpt-5.4' } }
    ]
    const wrapper = mount(RequestHealthBar, {
      props: { outcomes, windowSize: 8, sourceLabel: '1.2.*.*' },
      global: {
        mocks: { $t: (key: string) => key },
        stubs: {
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>'
          }
        }
      }
    })

    const failButton = wrapper.get('button')
    await failButton.trigger('click')
    expect(wrapper.text()).toContain('upstream exploded')
    expect(wrapper.text()).toContain('500')
  })
})
