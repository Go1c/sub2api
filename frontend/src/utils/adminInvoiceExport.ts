import * as XLSX from 'xlsx'

import type { InvoiceRequest, InvoiceStatus } from '@/api/invoices'

export type AdminInvoiceExportScope = 'all' | 'processing'

export interface AdminInvoiceExportRow {
  申请单号: string
  用户邮箱: string
  用户ID: number
  发票抬头: string
  税号: string
  开票金额: number
  历史开票金额: number
  历史充值金额: number
  税点扣除: number
  状态: string
  接收邮箱: string
  申请时间: string
  完成时间: string
  失败原因: string
}

const statusLabels: Record<InvoiceStatus, string> = {
  processing: '正在开票',
  completed: '已完成',
  failed: '失败',
}

export function buildAdminInvoiceExportRows(items: InvoiceRequest[]): AdminInvoiceExportRow[] {
  return items.map((item) => ({
    申请单号: item.order_no,
    用户邮箱: item.user_email,
    用户ID: item.user_id,
    发票抬头: item.title,
    税号: item.tax_number,
    开票金额: item.amount,
    历史开票金额: item.user_completed_invoice_amount ?? 0,
    历史充值金额: item.user?.total_recharged ?? item.user_total_recharged ?? 0,
    税点扣除: item.tax_amount ?? 0,
    状态: statusLabels[item.status] ?? item.status,
    接收邮箱: item.recipient_email,
    申请时间: item.created_at,
    完成时间: item.completed_at ?? '',
    失败原因: item.failure_reason ?? '',
  }))
}

export function buildAdminInvoiceExportFileName(scope: AdminInvoiceExportScope, date = new Date()): string {
  const stamp = date.toISOString().replace(/[-:]/g, '').replace(/\.\d{3}Z$/, '').replace('T', '-')
  return `admin-invoices-${scope}-${stamp}.xlsx`
}

export function downloadAdminInvoiceWorkbook(
  items: InvoiceRequest[],
  scope: AdminInvoiceExportScope,
  fileName = buildAdminInvoiceExportFileName(scope),
): void {
  const worksheet = XLSX.utils.json_to_sheet(buildAdminInvoiceExportRows(items))
  const workbook = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(workbook, worksheet, '开票记录')
  XLSX.writeFile(workbook, fileName, { bookType: 'xlsx' })
}
