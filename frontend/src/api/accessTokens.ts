/**
 * User long-lived Access Token management (profile / JWT session only).
 * Opaque tokens (uat_*) for programmatic key management APIs.
 */

import { apiClient } from './client'

export type UserAccessTokenStatus = 'active' | 'revoked' | 'expired'

export interface UserAccessToken {
  id: number
  name: string
  /** Plaintext — only present on create response */
  token?: string
  token_prefix: string
  expires_at: string
  last_used_at?: string | null
  revoked_at?: string | null
  created_at: string
  status: UserAccessTokenStatus
}

export interface CreateUserAccessTokenRequest {
  name: string
  expires_in_days?: number
}

export async function listAccessTokens(): Promise<UserAccessToken[]> {
  const { data } = await apiClient.get<UserAccessToken[]>('/user/access-tokens')
  return data ?? []
}

export async function createAccessToken(
  payload: CreateUserAccessTokenRequest
): Promise<UserAccessToken> {
  const { data } = await apiClient.post<UserAccessToken>('/user/access-tokens', payload)
  return data
}

export async function revokeAccessToken(id: number): Promise<void> {
  await apiClient.delete(`/user/access-tokens/${id}`)
}

export const accessTokensAPI = {
  list: listAccessTokens,
  create: createAccessToken,
  revoke: revokeAccessToken
}

export default accessTokensAPI
