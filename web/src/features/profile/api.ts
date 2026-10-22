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
import type { LoginSession } from '@/stores/auth-store'

import type {
  ApiResponse,
  UserProfile,
  UpdateUserRequest,
  UpdateUserSettingsRequest,
  DeleteAccountRequest,
  CheckinStatusResponse,
  CheckinResponse,
  LightcodeProfileData,
  UpdateDisplayNameResponse,
  UploadAvatarResponse,
} from './types'

// ============================================================================
// User Profile APIs
// ============================================================================

/**
 * Get current user profile
 */
export async function getUserProfile(): Promise<ApiResponse<UserProfile>> {
  const res = await api.get('/api/user/self')
  return res.data
}

/**
 * Update user profile
 */
export async function updateUserProfile(
  data: UpdateUserRequest
): Promise<ApiResponse> {
  const res = await api.put('/api/user/self', data, {
    acceptAuthRotation: Boolean(data.password),
  })
  return res.data
}

/**
 * Update user settings
 */
export async function updateUserSettings(
  data: UpdateUserSettingsRequest
): Promise<ApiResponse> {
  const res = await api.put('/api/user/setting', data)
  return res.data
}

/**
 * Update interface language preference
 */
export async function updateUserLanguage(
  language: string
): Promise<ApiResponse> {
  const res = await api.put('/api/user/self', { language })
  return res.data
}

/**
 * Delete user account
 */
export async function deleteUserAccount(
  data?: DeleteAccountRequest
): Promise<ApiResponse> {
  const res = await api.delete('/api/user/self', { data })
  return res.data
}

/**
 * Generate/regenerate system access token
 */
export async function generateAccessToken(): Promise<ApiResponse<string>> {
  const res = await api.get('/api/user/token')
  return res.data
}

// ============================================================================
// Identity APIs (Lightcode profile)
// ============================================================================

/**
 * Get the identity profile (display name and avatar) from the gateway.
 * The dashboard user payload has no avatar, so `avatar_url` only exists here.
 * Business errors are surfaced by the caller, hence `skipBusinessError`.
 */
export async function getLightcodeProfile(): Promise<
  ApiResponse<LightcodeProfileData>
> {
  const res = await api.get('/api/lightcode/profile', {
    skipBusinessError: true,
    skipErrorHandler: true,
  })
  return res.data
}

/**
 * Update the display name (1-20 characters, trimmed by the server).
 * The server falls back to the username only for display, so the response
 * carries the stored value.
 */
export async function updateLightcodeDisplayName(
  displayName: string
): Promise<ApiResponse<UpdateDisplayNameResponse>> {
  const res = await api.put(
    '/api/lightcode/profile/display-name',
    { display_name: displayName },
    { skipBusinessError: true }
  )
  return res.data
}

/**
 * Upload an avatar as multipart/form-data under the field name `avatar`.
 * The server sniffs the bytes and accepts PNG/JPEG/WebP/GIF up to 256KB.
 */
export async function uploadLightcodeAvatar(
  file: File
): Promise<ApiResponse<UploadAvatarResponse>> {
  const form = new FormData()
  form.append('avatar', file)
  const res = await api.put('/api/lightcode/profile/avatar', form, {
    skipBusinessError: true,
  })
  return res.data
}

/**
 * Delete the current user's avatar; the next profile fetch returns an empty URL.
 */
export async function deleteLightcodeAvatar(): Promise<ApiResponse<null>> {
  const res = await api.delete('/api/lightcode/profile/avatar')
  return res.data
}

// ============================================================================
// Account Binding APIs
// ============================================================================

/**
 * Send email verification code
 */
export async function sendEmailVerification(
  email: string,
  turnstileToken?: string
): Promise<ApiResponse> {
  const params = new URLSearchParams({ email })
  if (turnstileToken) {
    params.append('turnstile', turnstileToken)
  }
  const res = await api.get(`/api/verification?${params}`)
  return res.data
}

/**
 * Bind email account
 */
export async function bindEmail(
  email: string,
  code: string
): Promise<ApiResponse> {
  const res = await api.post('/api/oauth/email/bind', {
    email,
    code,
  })
  return res.data
}

/**
 * Bind WeChat account
 */
export async function bindWeChat(code: string): Promise<ApiResponse> {
  const res = await api.post(
    '/api/oauth/wechat/bind',
    { code },
    { skipBusinessError: true, skipErrorHandler: true }
  )
  return res.data
}

export interface TelegramBindFlow {
  flow_token: string
  callback_url: string
  expires_at: number
}

export async function startTelegramBind(): Promise<
  ApiResponse<TelegramBindFlow>
> {
  const res = await api.post('/api/oauth/telegram/bind/start')
  return res.data
}

// ============================================================================
// Login Session APIs
// ============================================================================

export async function getLoginSessions(): Promise<ApiResponse<LoginSession[]>> {
  const res = await api.get('/api/user/sessions')
  return res.data
}

export async function revokeLoginSession(sid: string): Promise<ApiResponse> {
  const res = await api.delete(`/api/user/sessions/${encodeURIComponent(sid)}`)
  return res.data
}

export async function revokeOtherLoginSessions(): Promise<ApiResponse> {
  const res = await api.post('/api/user/sessions/revoke-others')
  return res.data
}

/**
 * Bind Mofang account
 */
export async function bindMofang(jwt: string): Promise<ApiResponse> {
  const res = await api.post('/api/oauth/mofang/bind', { jwt })
  return res.data
}

// ============================================================================
// Custom OAuth Binding APIs
// ============================================================================

export interface CustomOAuthBinding {
  provider_id: string
  provider_name: string
  external_id?: string
}

/**
 * Get current user's custom OAuth bindings
 */
export async function getSelfOAuthBindings(): Promise<
  ApiResponse<CustomOAuthBinding[]>
> {
  const res = await api.get('/api/user/oauth/bindings')
  return res.data
}

/**
 * Unbind a custom OAuth provider for current user
 */
export async function unbindCustomOAuth(
  providerId: string
): Promise<ApiResponse> {
  const res = await api.delete(`/api/user/oauth/bindings/${providerId}`)
  return res.data
}

// ============================================================================
// Checkin APIs
// ============================================================================

/**
 * Get checkin status for a specific month
 */
export async function getCheckinStatus(
  month: string
): Promise<ApiResponse<CheckinStatusResponse>> {
  const res = await api.get(`/api/user/checkin?month=${month}`)
  return res.data
}

/**
 * Perform daily checkin
 */
export async function performCheckin(
  turnstileToken?: string
): Promise<ApiResponse<CheckinResponse>> {
  const url = turnstileToken
    ? `/api/user/checkin?turnstile=${encodeURIComponent(turnstileToken)}`
    : '/api/user/checkin'
  const res = await api.post(url)
  return res.data
}
