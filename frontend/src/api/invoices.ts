import { apiClient } from './client'
import type { PaginatedResponse, User } from '@/types'

export type InvoiceStatus = 'processing' | 'completed' | 'failed'

export interface InvoiceRequest {
  id: number
  order_no: string
  user_id: number
  user_email: string
  title: string
  tax_number: string
  amount: number
  recipient_email: string
  status: InvoiceStatus
  file_name: string
  file_size: number
  has_file: boolean
  tax_rate: number
  tax_amount: number
  user_completed_invoice_amount?: number
  user_total_recharged?: number
  failure_reason?: string
  created_at: string
  updated_at: string
  completed_at?: string
  user?: User
}

export interface InvoiceOverview {
  total_recharged: number
  used_invoice_amount: number
  remaining_amount: number
  enabled: boolean
}

export interface CreateInvoiceRequest {
  title: string
  tax_number: string
  amount: number
  recipient_email: string
}

export async function getOverview(): Promise<InvoiceOverview> {
  const { data } = await apiClient.get<InvoiceOverview>('/invoices/overview')
  return data
}

export async function list(
  page: number = 1,
  pageSize: number = 20,
): Promise<PaginatedResponse<InvoiceRequest>> {
  const { data } = await apiClient.get<PaginatedResponse<InvoiceRequest>>('/invoices', {
    params: { page, page_size: pageSize },
  })
  return data
}

export async function create(payload: CreateInvoiceRequest): Promise<InvoiceRequest> {
  const { data } = await apiClient.post<InvoiceRequest>('/invoices', payload)
  return data
}

export async function download(id: number): Promise<Blob> {
  const { data } = await apiClient.get<Blob>(`/invoices/${id}/download`, {
    responseType: 'blob',
  })
  return data
}

export const invoicesAPI = {
  getOverview,
  list,
  create,
  download,
}

export default invoicesAPI
