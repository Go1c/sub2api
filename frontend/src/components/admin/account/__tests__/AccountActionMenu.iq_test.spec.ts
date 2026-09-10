import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountActionMenu from '../AccountActionMenu.vue'
import type { Account } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function makeAccount(overrides: Partial<Account>): Account {
  return {
    id: 1,
    name: 'test-account',
    platform: 'openai',
    type: 'oauth',
    proxy_id: null,
    concurrency: 3,
    priority: 50,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: false,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    ...overrides
  }
}

const position = { top: 100, left: 100 }
const getBodyText = () => document.body.textContent ?? ''
const getBodyButtons = () => Array.from(document.body.querySelectorAll('button'))

describe('AccountActionMenu — IQ test visibility', () => {
  it('shows IQ test for OpenAI OAuth accounts', () => {
    const wrapper = mount(AccountActionMenu, {
      props: { show: true, account: makeAccount({ platform: 'openai', type: 'oauth' }), position },
      attachTo: document.body
    })
    expect(getBodyText()).toContain('admin.accounts.iqTest')
    wrapper.unmount()
  })

  it('shows IQ test for OpenAI API key accounts', () => {
    const wrapper = mount(AccountActionMenu, {
      props: { show: true, account: makeAccount({ platform: 'openai', type: 'apikey' }), position },
      attachTo: document.body
    })
    expect(getBodyText()).toContain('admin.accounts.iqTest')
    wrapper.unmount()
  })

  it('hides IQ test for Claude accounts', () => {
    const wrapper = mount(AccountActionMenu, {
      props: { show: true, account: makeAccount({ platform: 'anthropic', type: 'apikey' }), position },
      attachTo: document.body
    })
    expect(getBodyText()).not.toContain('admin.accounts.iqTest')
    wrapper.unmount()
  })

  it('emits iq-test with the account', async () => {
    const account = makeAccount({ platform: 'openai', type: 'oauth' })
    const wrapper = mount(AccountActionMenu, {
      props: { show: true, account, position },
      attachTo: document.body
    })
    const button = getBodyButtons().find((btn) => btn.textContent?.includes('admin.accounts.iqTest'))
    expect(button).toBeDefined()
    button!.click()
    await wrapper.vm.$nextTick()
    expect(wrapper.emitted('iq-test')?.[0][0]).toMatchObject({ id: account.id, platform: 'openai' })
    wrapper.unmount()
  })
})
