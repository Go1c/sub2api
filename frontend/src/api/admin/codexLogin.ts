import { apiClient } from '../client'

export interface CodexLoginJob {
  id: number
  email: string
  account_id: number | null
  status: 'queued' | 'running' | 'succeeded' | 'failed'
  error_message: string
  attempts: number
}

export async function jobs() {
  const { data } = await apiClient.get<{ available: boolean; jobs: CodexLoginJob[] }>('/admin/accounts/codex-2fa/jobs')
  return data
}

export async function importAccounts(payload: {
  documents: string[]
  group_ids: number[]
  proxy_id?: number
  proxy_ip_group_id?: number
}) {
  const { data } = await apiClient.post<{ job_ids: number[]; errors: { document: number; index: number; message: string }[] | null }>('/admin/accounts/codex-2fa', payload)
  return data
}

export async function retry(id: number) {
  await apiClient.post(`/admin/accounts/codex-2fa/jobs/${id}/retry`)
}
