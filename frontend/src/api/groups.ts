/**
 * User Groups API endpoints (non-admin)
 * Handles group-related operations for regular users
 */

import { apiClient } from './client'
import type { ModelMarketModel } from './modelMarket'
import type { Group } from '@/types'

/**
 * Get available groups that the current user can bind to API keys
 * This returns groups based on user's permissions:
 * - Standard groups: public (non-exclusive) or explicitly allowed
 * - Subscription groups: user has active subscription
 * @returns List of available groups
 */
export async function getAvailable(): Promise<Group[]> {
  const { data } = await apiClient.get<Group[]>('/groups/available')
  return data
}

/**
 * Models on groups the current user can bind, including enterprise-exclusive groups.
 * The public model market omits those groups, so key model restriction uses this list.
 */
export async function getAvailableModels(): Promise<ModelMarketModel[]> {
  const { data } = await apiClient.get<ModelMarketModel[]>('/groups/available-models')
  return data
}

/**
 * Get current user's custom group rate multipliers
 * @returns Map of group_id to custom rate_multiplier
 */
export async function getUserGroupRates(): Promise<Record<number, number>> {
  const { data } = await apiClient.get<Record<number, number> | null>('/groups/rates')
  return data || {}
}

export const userGroupsAPI = {
  getAvailable,
  getAvailableModels,
  getUserGroupRates
}

export default userGroupsAPI
