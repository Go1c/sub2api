import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const viewPath = resolve(dirname(fileURLToPath(import.meta.url)), '../orders/AdminOrdersView.vue')

describe('AdminOrdersView manual order supplement source contract', () => {
  it('offers manual completion only for expired orders and requires an admin reason', () => {
    const source = readFileSync(viewPath, 'utf8')

    expect(source).toContain("row.status === 'EXPIRED'")
    expect(source).toContain("t('payment.admin.manualComplete')")
    expect(source).toContain('manualCompleteReason')
    expect(source).toContain('adminPaymentAPI.manualCompleteOrder')
  })
})
