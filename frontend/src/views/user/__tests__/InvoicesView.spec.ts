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
    expect(source).toContain('MIN_INVOICE_AMOUNT')
    expect(source).toContain('form.amount < MIN_INVOICE_AMOUNT')
    expect(source).toContain("t('invoice.minimumAmountRule'")
    expect(source).toContain("key: 'tax_amount'")
    expect(source).toContain("#cell-tax_amount")
    expect(source).toContain('invoicesAPI.create')
    expect(source).toContain('invoicesAPI.download')
    expect(source).toContain("t('invoice.mailDeliveryHint')")
  })

  it('supports one-click fill from the last completed invoice without overwriting amount', () => {
    const source = readFileSync(viewPath, 'utf8')

    expect(source).toContain("t('invoice.fillLastSuccess')")
    expect(source).toContain('fillFromLastSuccess')
    expect(source).toContain("item.status === 'completed'")
    expect(source).toContain('form.title = last.title')
    expect(source).toContain('form.tax_number = last.tax_number')
    expect(source).toContain('form.recipient_email = last.recipient_email')
    // Amount must stay user-controlled — do not copy last.amount into the form.
    expect(source).not.toContain('form.amount = last.amount')
    expect(source).not.toContain('form.amount = last?.amount')
  })
})
