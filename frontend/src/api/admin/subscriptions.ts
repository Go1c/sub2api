/**
 * Admin Subscriptions API endpoints
 * Handles user subscription management for administrators
 */

import { apiClient } from '../client'
import type {
  UserSubscription,
  SubscriptionProgress,
  AssignSubscriptionRequest,
  BulkAssignSubscriptionRequest,
  ExtendSubscriptionRequest,
  PaginatedResponse
} from '@/types'

export interface AdminSubscriptionListFilters {
  status?: 'active' | 'expired' | 'revoked' | 'suspended'
  user_id?: number
  group_id?: number
  plan_id?: number
  email?: string
  platform?: string
  created_start?: string
  created_end?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export interface SubscriptionLedgerEntry {
  id: number
  user_id: number
  subscription_id: number
  group_id?: number | null
  api_key_id?: number | null
  usage_log_id?: number | null
  order_id?: number | null
  type: string
  delta_usd: number
  balance_delta_usd: number
  remaining_after_usd: number
  reason?: string | null
  event_key?: string | null
  metadata: Record<string, unknown>
  created_at: string
}

export interface PatchSubscriptionRequest {
  status?: 'active' | 'expired' | 'revoked' | 'suspended'
  expires_at?: string | null
  quota_limit_usd?: number
  daily_limit_usd?: number | null
  weekly_limit_usd?: number | null
  reason?: string
}

export interface WasteStatsFilters {
  start?: string
  end?: string
  plan_id?: number
  user_id?: number
  window?: 'daily' | 'weekly' | 'total' | 'all'
}

export interface WasteStatsByPlan {
  plan_id?: number | null
  plan_name: string
  purchased_usd: number
  consumed_usd: number
  expired_wasted_usd: number
  window_wasted_usd: number
  total_wasted_usd: number
  waste_ratio: number
  total_wasted_ratio: number
  average_waste_ratio: number
  reset_count: number
  purchase_count: number
  average_daily_waste_ratio: number
  average_weekly_waste_ratio: number
  total_quota_wasted_ratio: number
}

export interface WasteStatsTimeBucket {
  bucket_start: string
  bucket_end: string
  window: string
  wasted_usd: number
  total_wasted_usd: number
  expired_wasted_usd: number
  window_wasted_usd: number
  average_waste_ratio: number
  daily_average_waste_ratio: number
  weekly_average_waste_ratio: number
  total_wasted_ratio: number
  reset_count: number
}

export interface WasteStatsResult {
  start_time: string
  end_time: string
  window: string
  purchased_usd: number
  consumed_usd: number
  expired_wasted_usd: number
  window_wasted_usd: number
  total_wasted_usd: number
  waste_ratio: number
  total_wasted_ratio: number
  average_waste_ratio: number
  reset_count: number
  total_subscriptions_purchased: number
  total_quota_purchased_usd: number
  total_quota_consumed_usd: number
  total_quota_wasted_usd: number
  daily_reset_count: number
  daily_average_waste_ratio: number
  daily_total_wasted_usd: number
  weekly_reset_count: number
  weekly_average_waste_ratio: number
  weekly_total_wasted_usd: number
  by_plan: WasteStatsByPlan[]
  time_series: WasteStatsTimeBucket[]
}

/**
 * List all subscriptions with pagination
 * @param page - Page number (default: 1)
 * @param pageSize - Items per page (default: 20)
 * @param filters - Optional filters
 * @returns Paginated list of subscriptions
 */
export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: AdminSubscriptionListFilters,
  options?: {
    signal?: AbortSignal
  }
): Promise<PaginatedResponse<UserSubscription>> {
  const { data } = await apiClient.get<PaginatedResponse<UserSubscription>>(
    '/admin/subscriptions',
    {
      params: {
        page,
        page_size: pageSize,
        ...filters
      },
      signal: options?.signal
    }
  )
  return data
}

/**
 * Get subscription by ID
 * @param id - Subscription ID
 * @returns Subscription details
 */
export async function getById(id: number): Promise<UserSubscription> {
  const { data } = await apiClient.get<UserSubscription>(`/admin/subscriptions/${id}`)
  return data
}

/**
 * Get subscription progress
 * @param id - Subscription ID
 * @returns Subscription progress with usage stats
 */
export async function getProgress(id: number): Promise<SubscriptionProgress> {
  const { data } = await apiClient.get<SubscriptionProgress>(`/admin/subscriptions/${id}/progress`)
  return data
}

/**
 * Get subscription credit ledger entries.
 */
export async function getLedger(
  id: number,
  params?: { type?: string; page?: number; page_size?: number }
): Promise<PaginatedResponse<SubscriptionLedgerEntry> | SubscriptionLedgerEntry[]> {
  const { data } = await apiClient.get<
    PaginatedResponse<SubscriptionLedgerEntry> | SubscriptionLedgerEntry[]
  >(`/admin/subscriptions/${id}/ledger`, { params })
  return data
}

/**
 * Patch credit-pool subscription fields when backend support is enabled.
 */
export async function patch(
  id: number,
  request: PatchSubscriptionRequest
): Promise<UserSubscription> {
  const { data } = await apiClient.patch<UserSubscription>(`/admin/subscriptions/${id}`, request)
  return data
}

/**
 * Get subscription credit-pool waste statistics.
 */
export async function getWasteStats(filters?: WasteStatsFilters): Promise<WasteStatsResult> {
  const { data } = await apiClient.get<WasteStatsResult>('/admin/subscriptions/waste-stats', {
    params: filters
  })
  return data
}

/**
 * Assign subscription to user
 * @param request - Assignment request
 * @returns Created subscription
 */
export async function assign(request: AssignSubscriptionRequest): Promise<UserSubscription> {
  const { data } = await apiClient.post<UserSubscription>('/admin/subscriptions/assign', request)
  return data
}

/**
 * Bulk assign subscriptions to multiple users
 * @param request - Bulk assignment request
 * @returns Created subscriptions
 */
export async function bulkAssign(
  request: BulkAssignSubscriptionRequest
): Promise<UserSubscription[]> {
  const { data } = await apiClient.post<UserSubscription[]>(
    '/admin/subscriptions/bulk-assign',
    request
  )
  return data
}

/**
 * Extend subscription validity
 * @param id - Subscription ID
 * @param request - Extension request with days
 * @returns Updated subscription
 */
export async function extend(
  id: number,
  request: ExtendSubscriptionRequest
): Promise<UserSubscription> {
  const { data } = await apiClient.post<UserSubscription>(
    `/admin/subscriptions/${id}/extend`,
    request
  )
  return data
}

/**
 * Revoke subscription
 * @param id - Subscription ID
 * @returns Success confirmation
 */
export async function revoke(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/subscriptions/${id}`)
  return data
}

/**
 * Reset daily, weekly, and/or monthly usage quota for a subscription
 * @param id - Subscription ID
 * @param options - Which windows to reset
 * @returns Updated subscription
 */
export async function resetQuota(
  id: number,
  options: { daily: boolean; weekly: boolean; monthly: boolean }
): Promise<UserSubscription> {
  const { data } = await apiClient.post<UserSubscription>(
    `/admin/subscriptions/${id}/reset-quota`,
    options
  )
  return data
}

/**
 * Reset only the current weekly usage window for a subscription.
 * @param id - Subscription ID
 * @returns Updated subscription
 */
export async function resetWeeklyLimit(id: number): Promise<UserSubscription> {
  const { data } = await apiClient.post<UserSubscription>(
    `/admin/subscriptions/${id}/reset-quota`,
    { daily: false, weekly: true, monthly: false }
  )
  return data
}

/**
 * List subscriptions by group
 * @param groupId - Group ID
 * @param page - Page number
 * @param pageSize - Items per page
 * @returns Paginated list of subscriptions in the group
 */
export async function listByGroup(
  groupId: number,
  page: number = 1,
  pageSize: number = 20
): Promise<PaginatedResponse<UserSubscription>> {
  const { data } = await apiClient.get<PaginatedResponse<UserSubscription>>(
    `/admin/groups/${groupId}/subscriptions`,
    {
      params: { page, page_size: pageSize }
    }
  )
  return data
}

/**
 * List subscriptions by user
 * @param userId - User ID
 * @param page - Page number
 * @param pageSize - Items per page
 * @returns Paginated list of user's subscriptions
 */
export async function listByUser(
  userId: number,
  page: number = 1,
  pageSize: number = 20
): Promise<PaginatedResponse<UserSubscription>> {
  const { data } = await apiClient.get<PaginatedResponse<UserSubscription>>(
    `/admin/users/${userId}/subscriptions`,
    {
      params: { page, page_size: pageSize }
    }
  )
  return data
}

export const subscriptionsAPI = {
  list,
  getById,
  getProgress,
  getLedger,
  patch,
  getWasteStats,
  assign,
  bulkAssign,
  extend,
  revoke,
  resetQuota,
  resetWeeklyLimit,
  listByGroup,
  listByUser
}

export default subscriptionsAPI
