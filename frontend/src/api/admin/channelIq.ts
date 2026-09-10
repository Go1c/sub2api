import { apiClient } from '../client'

export type ChannelIQStatus = 'idle' | 'running' | 'success' | 'failed'

export interface ChannelIQSettings {
  group_ids: number[]
  excluded_account_ids?: number[]
  auto_enabled: boolean
  interval_seconds: number
  prompt: string
  model: string
  updated_at?: string
}

export interface ChannelIQItem {
  account_id: number
  name: string
  status: ChannelIQStatus
  test_count: number
  model: string
  reasoning_effort: string
  duration_ms: number
  total_tokens: number
  svg: string
  error?: string
  last_run_at?: string | null
}

export interface ChannelIQExcludedAccount {
  account_id: number
  name: string
}

export interface ChannelIQOverview {
  settings: ChannelIQSettings
  items: ChannelIQItem[]
  excluded?: ChannelIQExcludedAccount[]
}

export type ChannelIQSettingsPayload = Partial<
  Omit<ChannelIQSettings, 'group_ids' | 'updated_at'>
> & {
  group_ids?: number[]
}

export async function getOverview(options?: { signal?: AbortSignal }): Promise<ChannelIQOverview> {
  const { data } = await apiClient.get<ChannelIQOverview>('/admin/channel-iq', {
    signal: options?.signal
  })
  return data
}

export async function updateSettings(payload: ChannelIQSettingsPayload): Promise<ChannelIQSettings> {
  const { data } = await apiClient.put<ChannelIQSettings>('/admin/channel-iq/settings', payload)
  return data
}

export async function runAll(): Promise<{ started: boolean }> {
  const { data } = await apiClient.post<{ started: boolean }>('/admin/channel-iq/run')
  return data
}

export async function runOne(accountId: number): Promise<{ started: boolean }> {
  const { data } = await apiClient.post<{ started: boolean }>(`/admin/channel-iq/accounts/${accountId}/run`)
  return data
}

export async function excludeAccount(accountId: number): Promise<{ excluded: boolean }> {
  const { data } = await apiClient.post<{ excluded: boolean }>(
    `/admin/channel-iq/accounts/${accountId}/exclude`
  )
  return data
}

export async function restoreAccount(accountId: number): Promise<{ restored: boolean }> {
  const { data } = await apiClient.post<{ restored: boolean }>(
    `/admin/channel-iq/accounts/${accountId}/restore`
  )
  return data
}

const channelIqAPI = {
  getOverview,
  updateSettings,
  runAll,
  runOne,
  excludeAccount,
  restoreAccount
}

export default channelIqAPI
