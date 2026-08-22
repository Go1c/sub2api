import { describe, expect, it } from 'vitest'

import en from '../locales/en/admin/accounts'
import zh from '../locales/zh/admin/accounts'

describe('account status locale', () => {
  it('includes expired for proxy/group list cells that interpolate status', () => {
    expect(en.accounts.status.expired).toBe('Expired')
    expect(zh.accounts.status.expired).toBe('已过期')
  })
})
