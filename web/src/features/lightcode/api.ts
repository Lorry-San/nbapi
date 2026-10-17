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
