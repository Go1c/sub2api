import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import RequestHealthLines from '../RequestHealthLines.vue'
import type { AccountHealthRow } from '../requestHealth'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function line(ip: string): AccountHealthRow['lines'][number] {
  return { ip, health: { outcomes: [{ slot: 'ok' }], current: 0, max: 10 } }
}

describe('RequestHealthLines', () => {
  it('scrolls the IP group list after more than two lines', () => {
    const row: AccountHealthRow = {
      accountId: 1,
      mode: 'ip_group',
      lines: [line('1.2.*.*'), line('1.3.*.*'), line('1.4.*.*')]
    }
    const wrapper = mount(RequestHealthLines, {
      props: { row, windowSize: 12, now: Date.now() },
      global: {
        mocks: { $t: (key: string) => key },
        stubs: { RequestHealthBar: true, RequestHealthStatus: true, Icon: true }
      }
    })
    expect(wrapper.html()).toContain('max-h-[7.5rem]')
    expect(wrapper.html()).toContain('overflow-y-auto')
  })

  it('keeps a single-IP row as one bar', () => {
    const row: AccountHealthRow = {
      accountId: 2,
      mode: 'single',
      lines: [line('8.8.*.*')]
    }
    const wrapper = mount(RequestHealthLines, {
      props: { row, windowSize: 12, now: Date.now() },
      global: {
        mocks: { $t: (key: string) => key },
        stubs: { RequestHealthBar: true, RequestHealthStatus: true, Icon: true }
      }
    })
    expect(wrapper.html()).not.toContain('max-h-[7.5rem]')
  })
})
