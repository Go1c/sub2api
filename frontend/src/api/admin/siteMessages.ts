/**
 * Admin Site Messages API endpoints
 */

import { apiClient } from '../client'
import type {
  AdminSendCompensationBatchRequest,
  AdminSendSiteMessageRequest,
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

export const adminSiteMessagesAPI = {
  sendToUser,
  searchRecipients,
  sendCompensationBatch,
}

export default adminSiteMessagesAPI
