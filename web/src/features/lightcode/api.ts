/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { api } from '@/lib/api'

export interface LightcodeAuthorizeRequest {
  client_id: string
  redirect_uri: string
  code_challenge: string
  code_challenge_method: string
  state: string
  response_type: string
}

export interface LightcodeDevice {
  id: string
  client_id: string
  device: string
  ip: string
  created_at: number
  last_active_at: number
  expires_at: number
  revoked_at: number
}

interface ApiResponse<T> {
  success: boolean
  message?: string
  data?: T
}

/** Approve the request; the server binds the code to the signed-in user. */
export async function approveLightcode(
  request: LightcodeAuthorizeRequest
): Promise<{ redirect_uri: string }> {
  const res = await api.post('/oauth/authorize', request)
  return res.data
}

export async function getLightcodeDevices(): Promise<
  ApiResponse<LightcodeDevice[]>
> {
  const res = await api.get('/api/lightcode/devices')
  return res.data
}

export async function revokeLightcodeDevice(
  id: string
): Promise<ApiResponse<null>> {
  const res = await api.delete(
    `/api/lightcode/devices/${encodeURIComponent(id)}`
  )
  return res.data
}

export interface LightcodeUsageWindow {
  seconds: number
  used_percent: number
  reset_at: number
}

export interface LightcodeUsageData {
  days: {
    date: string
    requests: number
    input_tokens: number
    output_tokens: number
    cache_tokens: number
    total_tokens: number
  }[]
  model_days: { date: string; model: string; total_tokens: number }[]
  models: { model: string; requests: number; total_tokens: number }[]
  totals: {
    requests: number
    input_tokens: number
    output_tokens: number
    cache_tokens: number
    total_tokens: number
  }
  subscription: {
    plan: string
    end_time: number
    window_short: LightcodeUsageWindow | null
    window_long: LightcodeUsageWindow | null
  } | null
}

export interface LightcodeSubscriptionStatus {
  plan: string
  end_time: number
  window_short: LightcodeUsageWindow | null
  window_long: LightcodeUsageWindow | null
}

export interface LightcodeProfileData {
  user: { id: number; username: string; display_name: string }
  subscription: LightcodeSubscriptionStatus | null
  stats: {
    total_tokens: number
    requests: number
    active_days: number
    peak_day_tokens: number
    peak_day: string
    longest_streak_days: number
    current_streak_days: number
  }
  /** 371 entries, oldest first, last is today in the requested timezone. */
  days: { date: string; requests: number; total_tokens: number }[]
  models: { model: string; requests: number; total_tokens: number }[]
  truncated: boolean
}

export async function getLightcodeProfile(): Promise<
  ApiResponse<LightcodeProfileData>
> {
  const res = await api.get('/api/lightcode/profile', {
    params: { tz_offset_minutes: -new Date().getTimezoneOffset() },
  })
  return res.data
}

/** Token and request counts only; the endpoint never returns money fields. */
export async function getLightcodeUsage(
  days: number
): Promise<ApiResponse<LightcodeUsageData>> {
  const res = await api.get('/api/lightcode/usage', {
    params: { days, tz_offset_minutes: -new Date().getTimezoneOffset() },
  })
  return res.data
}
