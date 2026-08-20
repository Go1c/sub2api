import type { PaginatedResponse } from '@/types'

export type CheckinRecordStatus = 'awarded' | 'budget_exhausted'

export interface CheckinMilestone { day: number; bonus: string }

export interface CheckinSettingsRequest {
  enabled: boolean
  min_reward: string
  max_reward: string
  timezone: string
  daily_cap: string
  milestones: CheckinMilestone[]
}

export interface CheckinSettings extends CheckinSettingsRequest {
  maximum_single_reward: string
  updated_at: string
}

export interface CheckinRecord {
  id: number
  user_id?: number
  user_email: string
  username: string
  business_date: string
  checked_at: string
  timezone: string
  streak_days: number
  cycle_day: number
  milestone_day?: number
  base_reward: string
  milestone_bonus: string
  actual_reward: string
  status: CheckinRecordStatus
  balance_after: string
  client_ip?: string
  user_agent?: string
}

export interface CheckinStatus {
  enabled: boolean
  checked_in_today: boolean
  total_checkins: number
  total_reward: string
  current_streak: number
  cycle_day: number
  next_milestone: { day: number; bonus: string; days_until: number } | null
  balance: string
  today_record: CheckinRecord | null
  recent_records: CheckinRecord[]
}

export interface CheckinResult extends CheckinRecord { already_checked_in: boolean }

export type AdminCheckinStatsPeriod = 'day' | 'week' | 'month' | 'all'

export interface AdminCheckinStats {
  period: AdminCheckinStatsPeriod
  timezone: string
  from?: string
  to?: string
  unique_users: number
  checkin_count: number
  total_amount: string
  avg_amount: string
  p50_amount: string
  p90_amount: string
  max_amount: string
}

export interface AdminCheckinStatsParams {
  period?: AdminCheckinStatsPeriod
  user_id?: number
  search?: string
  status?: CheckinRecordStatus
}

export interface AdminCheckinListParams {
  page?: number
  page_size?: number
  user_id?: number
  search?: string
  business_date?: string
  status?: CheckinRecordStatus
  sort_by?: 'business_date' | 'checked_at' | 'streak_days' | 'actual_reward' | 'balance_after'
  sort_order?: 'asc' | 'desc'
}

export type AdminCheckinPage = PaginatedResponse<CheckinRecord>
