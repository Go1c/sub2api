import { describe, expect, it } from 'vitest'

import {
  buildAdminInvoiceExportFileName,
  buildAdminInvoiceExportRows,
} from '@/utils/adminInvoiceExport'
import type { InvoiceRequest } from '@/api/invoices'

describe('admin invoice export utilities', () => {
  it('maps invoice requests to stable Excel row headers', () => {
    const rows = buildAdminInvoiceExportRows([
      {
        id: 5,
        order_no: 'INV00000005',
        user_id: 25,
        user_email: 'casterscz@163.com',
        title: '超天才技术开发(北京)有限责任公司',
        tax_number: '91110101MA0000000X',
        amount: 1050,
        recipient_email: 'finance@example.com',
        status: 'processing',
        file_name: '',
        file_size: 0,
        has_file: false,
        tax_rate: 0,
        tax_amount: 0,
        user_completed_invoice_amount: 12.5,
        user_total_recharged: 1050,
        failure_reason: undefined,
        created_at: '2026-05-15T17:14:00+08:00',
        updated_at: '2026-05-15T17:14:00+08:00',
        completed_at: undefined,
      } satisfies InvoiceRequest,
      {
        id: 1,
        order_no: 'INV00000001',
        user_id: 2,
        user_email: '18032145@qq.com',
        title: '测是',
        tax_number: '',
        amount: 10,
        recipient_email: '18032145@qq.com',
        status: 'completed',
        file_name: 'INV00000001.pdf',
        file_size: 128,
        has_file: true,
        tax_rate: 0.01,
        tax_amount: 0.1,
        user_completed_invoice_amount: 10,
        user_total_recharged: 11.2,
        failure_reason: '',
        created_at: '2026-05-13T21:14:00+08:00',
        updated_at: '2026-05-13T21:30:00+08:00',
        completed_at: '2026-05-13T21:30:00+08:00',
      } satisfies InvoiceRequest,
    ])

    expect(rows).toEqual([
      {
        申请单号: 'INV00000005',
        用户邮箱: 'casterscz@163.com',
        用户ID: 25,
        发票抬头: '超天才技术开发(北京)有限责任公司',
        税号: '91110101MA0000000X',
        开票金额: 1050,
        历史开票金额: 12.5,
        历史充值金额: 1050,
        税点扣除: 0,
        状态: '正在开票',
        接收邮箱: 'finance@example.com',
        申请时间: '2026-05-15T17:14:00+08:00',
        完成时间: '',
        失败原因: '',
      },
      {
        申请单号: 'INV00000001',
        用户邮箱: '18032145@qq.com',
        用户ID: 2,
        发票抬头: '测是',
        税号: '',
        开票金额: 10,
        历史开票金额: 10,
        历史充值金额: 11.2,
        税点扣除: 0.1,
        状态: '已完成',
        接收邮箱: '18032145@qq.com',
        申请时间: '2026-05-13T21:14:00+08:00',
        完成时间: '2026-05-13T21:30:00+08:00',
        失败原因: '',
      },
    ])
  })

  it('builds deterministic xlsx file names', () => {
    const date = new Date('2026-05-16T12:34:56+08:00')

    expect(buildAdminInvoiceExportFileName('all', date)).toBe('admin-invoices-all-20260516-043456.xlsx')
    expect(buildAdminInvoiceExportFileName('processing', date)).toBe('admin-invoices-processing-20260516-043456.xlsx')
  })
})
