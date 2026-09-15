import { apiClient } from '../client'
import type { ProxyIPGroup } from '@/types'

export interface CreateProxyIPGroupRequest {
  name: string
  per_ip_concurrency?: number
  proxy_ids?: number[]
}

export interface UpdateProxyIPGroupRequest {
  name?: string
  per_ip_concurrency?: number
}

export async function list(): Promise<ProxyIPGroup[]> {
  const { data } = await apiClient.get<ProxyIPGroup[]>('/admin/proxy-ip-groups')
  return data
}

export async function getById(id: number): Promise<ProxyIPGroup> {
  const { data } = await apiClient.get<ProxyIPGroup>(`/admin/proxy-ip-groups/${id}`)
  return data
}

export async function create(payload: CreateProxyIPGroupRequest): Promise<ProxyIPGroup> {
  const { data } = await apiClient.post<ProxyIPGroup>('/admin/proxy-ip-groups', payload)
  return data
}

export async function update(id: number, payload: UpdateProxyIPGroupRequest): Promise<ProxyIPGroup> {
  const { data } = await apiClient.put<ProxyIPGroup>(`/admin/proxy-ip-groups/${id}`, payload)
  return data
}

export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/proxy-ip-groups/${id}`)
}

export async function setMembers(id: number, proxyIds: number[]): Promise<ProxyIPGroup> {
  const { data } = await apiClient.put<ProxyIPGroup>(`/admin/proxy-ip-groups/${id}/members`, {
    proxy_ids: proxyIds
  })
  return data
}

const proxyIpGroupsAPI = {
  list,
  getById,
  create,
  update,
  delete: remove,
  setMembers
}

export default proxyIpGroupsAPI
