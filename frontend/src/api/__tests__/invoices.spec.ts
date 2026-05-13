import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
  },
}))

import { invoicesAPI, type CreateInvoiceRequest } from '@/api/invoices'
import { adminInvoicesAPI } from '@/api/admin/invoices'

describe('invoices api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    get.mockResolvedValue({ data: {} })
    post.mockResolvedValue({ data: {} })
  })

  it('uses the user invoice endpoints for overview, list, create, and download', async () => {
    const payload: CreateInvoiceRequest = {
      title: 'Lumio API',
      tax_number: '91310000MA1K00000X',
      amount: 128.5,
      recipient_email: 'billing@example.com',
    }

    await invoicesAPI.getOverview()
    await invoicesAPI.list(2, 10)
    await invoicesAPI.create(payload)
    await invoicesAPI.download(9)

    expect(get).toHaveBeenNthCalledWith(1, '/invoices/overview')
    expect(get).toHaveBeenNthCalledWith(2, '/invoices', {
      params: { page: 2, page_size: 10 },
    })
    expect(post).toHaveBeenCalledWith('/invoices', payload)
    expect(get).toHaveBeenNthCalledWith(3, '/invoices/9/download', {
      responseType: 'blob',
    })
  })

  it('uses the admin invoice endpoints and sends completion as multipart form data', async () => {
    const file = new File(['invoice'], 'invoice.pdf', { type: 'application/pdf' })

    await adminInvoicesAPI.list({ page: 1, page_size: 20, status: 'processing', search: 'INV00000009' })
    await adminInvoicesAPI.complete(9, { file, tax_rate: 0.02 })
    await adminInvoicesAPI.fail(10, '资料不完整')
    await adminInvoicesAPI.download(11)

    expect(get).toHaveBeenNthCalledWith(1, '/admin/invoices', {
      params: {
        page: 1,
        page_size: 20,
        status: 'processing',
        search: 'INV00000009',
        user_id: undefined,
      },
    })
    expect(post).toHaveBeenNthCalledWith(
      1,
      '/admin/invoices/9/complete',
      expect.any(FormData),
      { headers: { 'Content-Type': 'multipart/form-data' } },
    )
    const formData = post.mock.calls[0][1] as FormData
    expect(formData.get('file')).toBe(file)
    expect(formData.get('tax_rate')).toBe('0.02')
    expect(post).toHaveBeenNthCalledWith(2, '/admin/invoices/10/fail', { reason: '资料不完整' })
    expect(get).toHaveBeenNthCalledWith(2, '/admin/invoices/11/download', {
      responseType: 'blob',
    })
  })
})
