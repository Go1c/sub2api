/**
 * User Subscription API
 * API for regular users to view their own subscriptions and progress
 */

import { apiClient } from './client'
import type { UserSubscription, SubscriptionProgress } from '@/types'

export const subscriptionCreditErrorMessages: Record<string, string> = {
  SUBSCRIPTION_CREDIT_EXHAUSTED: 'Subscription credit has been exhausted. Please renew or buy another plan.',
  SUBSCRIPTION_EXPIRED: 'Subscription has expired. Please renew or buy another plan.',
  SUBSCRIPTION_DAILY_LIMIT_REACHED: 'Daily subscription credit limit reached. Please try again after the daily reset.',
  SUBSCRIPTION_WEEKLY_LIMIT_REACHED: 'Weekly subscription credit limit reached. Please try again after the weekly reset.',
  SUBSCRIPTION_RENEWAL_NOT_ALLOWED: 'This subscription cannot be renewed right now.',
  DAILY_LIMIT_EXCEEDED: 'Daily subscription credit limit reached. Please try again after the daily reset.',
  WEEKLY_LIMIT_EXCEEDED: 'Weekly subscription credit limit reached. Please try again after the weekly reset.',
  SUBSCRIPTION_WEEKLY_LIMIT_RESET_EXHAUSTED:
    'You have already used the weekly limit reset for this subscription period.',
  SUBSCRIPTION_NO_WEEKLY_LIMIT: 'This subscription has no weekly limit to reset.',
  SUBSCRIPTION_NOT_USABLE: 'This subscription is not currently usable.',
  SUBSCRIPTION_NOT_FOUND: 'Subscription not found.',
}

/**
 * Subscription summary for user dashboard
 */
export interface SubscriptionSummary {
  active_count: number
  subscriptions: Array<{
    id: number
    group_name: string
    status: string
    daily_progress: number | null
    weekly_progress: number | null
    monthly_progress: number | null
    expires_at: string | null
    days_remaining: number | null
  }>
}

/**
 * Get list of current user's subscriptions
 */
export async function getMySubscriptions(): Promise<UserSubscription[]> {
  const response = await apiClient.get<UserSubscription[]>('/subscriptions')
  return response.data
}

/**
 * Get current user's active subscriptions
 */
export async function getActiveSubscriptions(): Promise<UserSubscription[]> {
  const response = await apiClient.get<UserSubscription[]>('/subscriptions/active')
  return response.data
}

/**
 * Get progress for all user's active subscriptions
 */
export async function getSubscriptionsProgress(): Promise<SubscriptionProgress[]> {
  const response = await apiClient.get<SubscriptionProgress[]>('/subscriptions/progress')
  return response.data
}

/**
 * Get subscription summary for dashboard display
 */
export async function getSubscriptionSummary(): Promise<SubscriptionSummary> {
  const response = await apiClient.get<SubscriptionSummary>('/subscriptions/summary')
  return response.data
}

/**
 * Get progress for a specific subscription
 */
export async function getSubscriptionProgress(
  subscriptionId: number
): Promise<SubscriptionProgress> {
  const response = await apiClient.get<SubscriptionProgress>(
    `/subscriptions/${subscriptionId}/progress`
  )
  return response.data
}

/**
 * User self-service: reset only the current weekly usage window once per subscription period.
 */
export async function resetWeeklyLimit(id: number): Promise<UserSubscription> {
  const response = await apiClient.post<UserSubscription>(
    `/subscriptions/${id}/reset-weekly-limit`
  )
  return response.data
}

export default {
  getMySubscriptions,
  getActiveSubscriptions,
  getSubscriptionsProgress,
  getSubscriptionSummary,
  getSubscriptionProgress,
  resetWeeklyLimit,
}
