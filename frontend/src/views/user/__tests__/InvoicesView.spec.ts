import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const viewPath = resolve(dirname(fileURLToPath(import.meta.url)), '../InvoicesView.vue')

describe('user InvoicesView source contract', () => {
  it('loads invoice overview, blocks over-quota requests, creates invoices, and supports downloads', () => {
    const source = readFileSync(viewPath, 'utf8')

    expect(source).toContain('invoicesAPI.getOverview')
    expect(source).toContain('form.amount > overview.value.remaining_amount')
    expect(source).toContain('invoicesAPI.create')
    expect(source).toContain('invoicesAPI.download')
    expect(source).toContain("t('invoice.mailDeliveryHint')")
  })
})
