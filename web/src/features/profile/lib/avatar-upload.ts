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
// ============================================================================
// Avatar Upload Validation
// ============================================================================

/** Mirrors model.LightcodeAvatarMaxBytes so oversized files never hit the network. */
export const AVATAR_MAX_BYTES = 256 * 1024

/** Mirrors the types the backend accepts after sniffing the bytes. */
export const AVATAR_ALLOWED_TYPES = [
  'image/png',
  'image/jpeg',
  'image/webp',
  'image/gif',
] as const

export type AvatarAllowedType = (typeof AVATAR_ALLOWED_TYPES)[number]

/** i18n keys; the caller renders them with t(). */
export type AvatarRejection =
  | 'Avatar must be 256KB or smaller'
  | 'Only PNG, JPEG, WebP or GIF images are supported'

/**
 * Check a picked file against the server limits. Returns the i18n key of the
 * reason the file was rejected, or null when it may be uploaded.
 * The declared MIME type is only a fast path: the server sniffs the bytes, so
 * a renamed file still gets rejected there.
 */
export function validateAvatarFile(file: File): AvatarRejection | null {
  if (!AVATAR_ALLOWED_TYPES.includes(file.type as AvatarAllowedType)) {
    return 'Only PNG, JPEG, WebP or GIF images are supported'
  }
  if (file.size > AVATAR_MAX_BYTES) {
    return 'Avatar must be 256KB or smaller'
  }
  return null
}
