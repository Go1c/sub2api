import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const viewPath = resolve(dirname(fileURLToPath(import.meta.url)), '../InvoicesView.vue')

describe('admin InvoicesView source contract', () => {
  it('lists invoice records, uploads completed invoices with a tax rate, and supports failure/download actions', () => {
    const source = readFileSync(viewPath, 'utf8')

    expect(source).toContain('adminInvoicesAPI.list')
    expect(source).toContain('adminInvoicesAPI.complete')
    expect(source).toContain('taxRate: 0.01')
    expect(source).toContain('selectedFile')
    expect(source).toContain('adminInvoicesAPI.fail')
    expect(source).toContain('adminInvoicesAPI.download')
    expect(source).toContain('downloadAdminInvoiceWorkbook')
    expect(source).toContain("t('admin.invoice.exportAll')")
    expect(source).toContain("t('admin.invoice.exportProcessing')")
    expect(source).toContain("status: scope === 'processing' ? 'processing' : ''")
  })
})
