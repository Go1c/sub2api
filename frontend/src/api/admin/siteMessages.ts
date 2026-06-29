/**
 * Admin Site Messages API endpoints
 */

import { apiClient } from '../client'
import type {
  AdminSendCompensationBatchRequest,
  AdminSendSiteMessageRequest,
  PaginatedResponse,
  SiteMessage,
  SiteMessageCompensationBatch,
  SiteMessageRecipient
} from '@/types'

async function sendToUser(userId: number, request: AdminSendSiteMessageRequest): Promise<SiteMessage> {
  const { data } = await apiClient.post<SiteMessage>(`/admin/site-messages/users/${userId}`, request)
  return data
}

async function searchRecipients(query: string, limit: number = 20): Promise<SiteMessageRecipient[]> {
  const { data } = await apiClient.get<SiteMessageRecipient[]>('/admin/site-messages/recipients', {
    params: { query, limit }
  })
  return data
}

async function sendCompensationBatch(request: AdminSendCompensationBatchRequest): Promise<SiteMessageCompensationBatch> {
  const { data } = await apiClient.post<SiteMessageCompensationBatch>('/admin/site-messages/compensation-batches', request)
  return data
}

async function listCompensationBatches(page: number = 1, pageSize: number = 100): Promise<PaginatedResponse<SiteMessageCompensationBatch>> {
  const { data } = await apiClient.get<PaginatedResponse<SiteMessageCompensationBatch>>('/admin/site-messages/compensation-batches', {
    params: {
      page,
      page_size: pageSize
    }
  })
  return data
}

export const adminSiteMessagesAPI = {
  sendToUser,
  searchRecipients,
  sendCompensationBatch,
  listCompensationBatches,
}

export default adminSiteMessagesAPI
