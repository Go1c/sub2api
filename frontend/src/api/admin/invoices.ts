import { apiClient } from '../client'
import type { InvoiceRequest, InvoiceStatus } from '../invoices'
import type { PaginatedResponse } from '@/types'

export interface AdminInvoiceListParams {
  page?: number
  page_size?: number
  status?: InvoiceStatus | ''
  search?: string
  user_id?: number
}

export interface CompleteInvoiceRequest {
  file: File
  tax_rate?: number
}

export async function list(
  params: AdminInvoiceListParams = {},
): Promise<PaginatedResponse<InvoiceRequest>> {
  const { data } = await apiClient.get<PaginatedResponse<InvoiceRequest>>('/admin/invoices', {
    params: {
      page: params.page ?? 1,
      page_size: params.page_size ?? 20,
      status: params.status || undefined,
      search: params.search || undefined,
      user_id: params.user_id,
    },
  })
  return data
}

export async function complete(
  id: number,
  payload: CompleteInvoiceRequest,
): Promise<InvoiceRequest> {
  const formData = new FormData()
  formData.append('file', payload.file)
  if (payload.tax_rate !== undefined) {
    formData.append('tax_rate', String(payload.tax_rate))
  }
  const { data } = await apiClient.post<InvoiceRequest>(
    `/admin/invoices/${id}/complete`,
    formData,
    { headers: { 'Content-Type': 'multipart/form-data' } },
  )
  return data
}

export async function fail(id: number, reason: string): Promise<InvoiceRequest> {
  const { data } = await apiClient.post<InvoiceRequest>(`/admin/invoices/${id}/fail`, { reason })
  return data
}

export async function download(id: number): Promise<Blob> {
  const { data } = await apiClient.get<Blob>(`/admin/invoices/${id}/download`, {
    responseType: 'blob',
  })
  return data
}

export const adminInvoicesAPI = {
  list,
  complete,
  fail,
  download,
}

export default adminInvoicesAPI
