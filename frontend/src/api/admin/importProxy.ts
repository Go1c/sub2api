import { apiClient } from '../client'

export interface ImportProxyDefault {
  proxy_id: number | null
  proxy_ip_group_id: number | null
  mode: 'dynamic' | 'group' | 'single' | 'none'
}

export async function getImportProxyDefault(): Promise<ImportProxyDefault> {
  const { data } = await apiClient.get<ImportProxyDefault>('/admin/accounts/import-proxy-default')
  return data
}
