import { apiClient } from '@/api/client'
import type { AdminCheckinListParams, AdminCheckinPage, CheckinResult, CheckinSettings, CheckinSettingsRequest, CheckinStatus } from './types'

const userPath = '/user/checkin'
const adminPath = '/admin/affiliates/checkins'

async function getUserStatus(): Promise<CheckinStatus> {
  const { data } = await apiClient.get<CheckinStatus>(userPath)
  return data
}

async function checkIn(): Promise<CheckinResult> {
  const { data } = await apiClient.post<CheckinResult>(userPath)
  return data
}

async function listAdminRecords(params: AdminCheckinListParams = {}): Promise<AdminCheckinPage> {
  const { data } = await apiClient.get<AdminCheckinPage>(adminPath, { params })
  return data
}

async function getSettings(): Promise<CheckinSettings> {
  const { data } = await apiClient.get<CheckinSettings>(`${adminPath}/settings`)
  return data
}

async function updateSettings(payload: CheckinSettingsRequest): Promise<CheckinSettings> {
  const { data } = await apiClient.put<CheckinSettings>(`${adminPath}/settings`, payload)
  return data
}

export const checkinAPI = { getUserStatus, checkIn, listAdminRecords, getSettings, updateSettings }
export default checkinAPI
