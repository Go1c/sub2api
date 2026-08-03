import { describe, expect, it } from 'vitest'

import { deepMergeMessages } from '../index'
import monothithicZh from '../locales/zh'
import modularAccounts from '../locales/zh/admin/accounts'

describe('modular locale merge', () => {
  it('deep-merges modular account keys over monothithic admin.accounts', () => {
    const merged = deepMergeMessages(monothithicZh, { admin: modularAccounts })
    // Keys present only in modular pack (missing from monothithic zh.ts).
    expect(merged.admin.accounts.columns.id).toBe('账号ID')
    expect(merged.admin.accounts.columns.schedulerScore).toBe('调度权值')
    expect(merged.admin.accounts.duplicateAccount).toBe('复制账号')
    // Keys already in monothithic stay translated (overlay may refine).
    expect(merged.admin.accounts.columns.platformType).toBeTruthy()
    expect(merged.admin.accounts.title).toBeTruthy()
  })

  it('does not clobber sibling admin sections when merging accounts pack', () => {
    const base = {
      admin: {
        accounts: { title: 'old' },
        users: { title: '用户' }
      }
    }
    const merged = deepMergeMessages(base, {
      admin: { accounts: { title: '账号管理', columns: { id: '账号ID' } } }
    })
    expect(merged.admin.users.title).toBe('用户')
    expect(merged.admin.accounts.title).toBe('账号管理')
    expect(merged.admin.accounts.columns.id).toBe('账号ID')
  })
})
