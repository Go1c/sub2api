import { buildApiUrl } from '@/api/client'
import type { ApiResponse, AuthResponse } from '@/types'

export const AUTH_BRIDGE_DEFAULT_REDIRECT = '/purchase'

export function sanitizeAuthBridgeRedirect(value: string | null | undefined): string {
  if (value == null || value === '') {
    return AUTH_BRIDGE_DEFAULT_REDIRECT
  }

  let decoded = value
  try {
    decoded = decodeURIComponent(value)
  } catch {
    return AUTH_BRIDGE_DEFAULT_REDIRECT
  }

  if (!decoded.startsWith('/') || decoded.includes('//')) {
    return AUTH_BRIDGE_DEFAULT_REDIRECT
  }

  return decoded
}

export function parseAuthBridgeHash(hash: string): { token: string; redirect: string } {
  const raw = hash.startsWith('#') ? hash.slice(1) : hash
  const params = new URLSearchParams(raw)
  return {
    token: (params.get('t') ?? '').trim(),
    redirect: sanitizeAuthBridgeRedirect(params.get('r')),
  }
}

export async function exchangeAuthBridge(token: string): Promise<AuthResponse> {
  let response: Response
  try {
    response = await fetch(buildApiUrl('/auth/bridge'), {
      method: 'POST',
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        Authorization: `Bearer ${token}`,
      },
    })
  } catch {
    throw {
      status: 0,
      message: 'Network error. Please check your connection.',
    }
  }

  let envelope: Partial<ApiResponse<AuthResponse>>
  try {
    envelope = await response.json() as Partial<ApiResponse<AuthResponse>>
  } catch {
    throw {
      status: response.status,
      message: 'Invalid server response',
    }
  }

  if (!response.ok || envelope.code !== 0 || !envelope.data?.access_token) {
    throw {
      status: response.status,
      message: envelope.message || 'Auth bridge failed',
    }
  }

  return envelope.data
}
