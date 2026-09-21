import { apiClient } from '../client'

export type TurnStateProbeStatus = 'idle' | 'running' | 'holding' | 'failed'

export interface TurnStateProbeDynamicExit {
  host: string
  username: string
  password?: string
  password_set: boolean
  region: string
  session_minutes: number
}

export interface TurnStateProbePolicy {
  enabled: boolean
  proxy_ids: number[]
  dynamic: TurnStateProbeDynamicExit
  model: string
  length_filter_enabled: boolean
  min_state_length: number
  question: string
  answer: string
  fuzzy_match: boolean
  overload_threshold: number
  recheck_minutes: number
  rpm: number
  revision?: number
  updated_at?: string
}

export interface TurnStateProbeAccountItem {
 sticky_proxy_id?: number
 sticky_until?: string | null
  account_id: number
  name: string
  enabled: boolean
  status: TurnStateProbeStatus | string
  state_hash?: string
  state_length?: number
  model?: string
  policy_revision?: number
  recheck_at?: string | null
  last_error?: string
  last_probed_at?: string | null
}

export interface TurnStateProbeOverview {
  policy: TurnStateProbePolicy
  accounts: TurnStateProbeAccountItem[]
}

export const TURN_STATE_PROBE_DEFAULT_MODEL = 'gpt-6-astra'
export const TURN_STATE_PROBE_DEFAULT_MIN_LENGTH = 160
export const TURN_STATE_PROBE_DEFAULT_RECHECK_MINUTES = 20
export const TURN_STATE_PROBE_DEFAULT_RPM = 6
export const TURN_STATE_PROBE_DEFAULT_SESSION_MINUTES = 5
export const TURN_STATE_PROBE_DEFAULT_REGION = 'Random'
export const TURN_STATE_PROBE_DEFAULT_ANSWER = '21'
export const TURN_STATE_PROBE_DEFAULT_QUESTION =
  '黑色袋子中有苹果味、桃子味、西瓜味糖果;每种分为圆形和五角星形，可用手感区分形状。圆形依次有7、9、8颗;五角星形依次有7、6、4颗。事先决定摸出的数量，最少取多少颗，才能保证拿到不同形状的苹果味和桃子味糖果?'

export function defaultTurnStateProbePolicy(): TurnStateProbePolicy {
  return {
    enabled: false,
    proxy_ids: [],
    dynamic: {
      host: '',
      username: '',
      password_set: false,
      region: TURN_STATE_PROBE_DEFAULT_REGION,
      session_minutes: TURN_STATE_PROBE_DEFAULT_SESSION_MINUTES
    },
    model: TURN_STATE_PROBE_DEFAULT_MODEL,
    length_filter_enabled: true,
    min_state_length: TURN_STATE_PROBE_DEFAULT_MIN_LENGTH,
    question: TURN_STATE_PROBE_DEFAULT_QUESTION,
    answer: TURN_STATE_PROBE_DEFAULT_ANSWER,
    fuzzy_match: true,
    recheck_minutes: TURN_STATE_PROBE_DEFAULT_RECHECK_MINUTES,
    overload_threshold: 3,
    rpm: TURN_STATE_PROBE_DEFAULT_RPM
  }
}

export type TurnStateProbePolicyPayload = Partial<
  Omit<TurnStateProbePolicy, 'dynamic' | 'revision' | 'updated_at'>
> & {
  proxy_ids?: number[]
  dynamic?: Partial<TurnStateProbeDynamicExit>
}

function asOverview(data: TurnStateProbeOverview | undefined): TurnStateProbeOverview {
  const fallback = defaultTurnStateProbePolicy()
  return {
    policy: data?.policy ? { ...fallback, ...data.policy, dynamic: { ...fallback.dynamic, ...data.policy.dynamic } } : fallback,
    accounts: data?.accounts || []
  }
}

export async function getOverview(options?: { signal?: AbortSignal }): Promise<TurnStateProbeOverview> {
  const { data } = await apiClient.get<TurnStateProbeOverview>('/admin/channels/turn-state-probe', {
    signal: options?.signal
  })
  return asOverview(data)
}

export async function updatePolicy(payload: TurnStateProbePolicyPayload): Promise<TurnStateProbePolicy> {
  const { data } = await apiClient.put<TurnStateProbePolicy>('/admin/channels/turn-state-probe', payload)
  return data
}

export async function setAccountEnabled(
  accountId: number,
  enabled: boolean
): Promise<{ enabled: boolean }> {
  const { data } = await apiClient.post<{ enabled: boolean }>(
    `/admin/accounts/${accountId}/turn-state-probe/enabled`,
    { enabled }
  )
  return data
}

export async function runOne(accountId: number): Promise<{ started: boolean }> {
  const { data } = await apiClient.post<{ started: boolean }>(
    `/admin/accounts/${accountId}/turn-state-probe/run`
  )
  return data
}

export async function clearTicket(accountId: number): Promise<{ cleared: boolean }> {
  const { data } = await apiClient.delete<{ cleared: boolean }>(
    `/admin/accounts/${accountId}/turn-state-probe`
  )
  return data
}

const turnStateProbeAPI = {
  getOverview,
  updatePolicy,
  setAccountEnabled,
  runOne,
  clearTicket
}

export default turnStateProbeAPI
