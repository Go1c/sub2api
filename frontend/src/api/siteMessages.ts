/**
 * User Site Messages API endpoints
 */

import { apiClient } from './client'
import type {
  BasePaginationResponse,
  CreateSiteMessageRequest,
  ReplySiteMessageRequest,
  SiteMessage,
  SiteMessageRecipient
} from '@/types'

export interface SiteMessageListParams {
  page?: number
  page_size?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export interface SiteMessageUnreadCount {
  count: number
}

async function listInbox(params: SiteMessageListParams = {}): Promise<BasePaginationResponse<SiteMessage>> {
  const { data } = await apiClient.get<BasePaginationResponse<SiteMessage>>('/site-messages/inbox', {
    params
  })
  return data
}

async function listSent(params: SiteMessageListParams = {}): Promise<BasePaginationResponse<SiteMessage>> {
  const { data } = await apiClient.get<BasePaginationResponse<SiteMessage>>('/site-messages/sent', {
    params
  })
  return data
}

async function getById(id: number): Promise<SiteMessage> {
  const { data } = await apiClient.get<SiteMessage>(`/site-messages/${id}`)
  return data
}

async function send(request: CreateSiteMessageRequest): Promise<SiteMessage> {
  const { data } = await apiClient.post<SiteMessage>('/site-messages', request)
  return data
}

async function reply(id: number, request: ReplySiteMessageRequest): Promise<SiteMessage> {
  const { data } = await apiClient.post<SiteMessage>(`/site-messages/${id}/reply`, request)
  return data
}

async function markRead(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>(`/site-messages/${id}/read`)
  return data
}

async function resolveRecipient(query: string): Promise<SiteMessageRecipient> {
  const { data } = await apiClient.get<SiteMessageRecipient>('/site-messages/recipient/resolve', {
    params: { query }
  })
  return data
}

async function getUnreadCount(): Promise<SiteMessageUnreadCount> {
  const { data } = await apiClient.get<SiteMessageUnreadCount>('/site-messages/unread-count')
  return data
}

export const siteMessagesAPI = {
  listInbox,
  listSent,
  getById,
  send,
  reply,
  markRead,
  resolveRecipient,
  getUnreadCount,
}

export default siteMessagesAPI
