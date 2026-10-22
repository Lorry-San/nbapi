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
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  deleteLightcodeAvatar,
  getLightcodeProfile,
  updateLightcodeDisplayName,
  uploadLightcodeAvatar,
} from '../api'
import { validateAvatarFile } from '../lib'
import type { LightcodeProfileUser } from '../types'

// ============================================================================
// Profile Identity Hook
// ============================================================================

/**
 * Owns the avatar and display name shown on the profile page.
 *
 * The dashboard user payload has no avatar field, so identity data is loaded
 * from GET /api/lightcode/profile and kept in local state. Callers merge
 * `identity` into their own profile so the header shows one source of truth.
 */
export function useProfileIdentity() {
  const { t } = useTranslation()
  const [identity, setIdentity] = useState<LightcodeProfileUser | null>(null)
  const [saving, setSaving] = useState(false)
  const mountedRef = useRef(true)

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const refreshIdentity = useCallback(async () => {
    try {
      const response = await getLightcodeProfile()
      if (response.success && response.data?.user && mountedRef.current) {
        setIdentity(response.data.user)
      }
    } catch {
      // The identity card degrades to the dashboard profile values on failure.
    }
  }, [])

  useEffect(() => {
    void refreshIdentity()
  }, [refreshIdentity])

  const saveDisplayName = useCallback(
    async (displayName: string): Promise<boolean> => {
      setSaving(true)
      try {
        const response = await updateLightcodeDisplayName(displayName)
        const saved = response.data
        if (!response.success || !saved) {
          toast.error(response.message || t('Failed to update display name'))
          return false
        }
        if (mountedRef.current) {
          setIdentity((current) =>
            current ? { ...current, display_name: saved.display_name } : current
          )
        }
        toast.success(t('Display name updated'))
        return true
      } catch {
        toast.error(t('Failed to update display name'))
        return false
      } finally {
        if (mountedRef.current) setSaving(false)
      }
    },
    [t]
  )

  const uploadAvatar = useCallback(
    async (file: File | undefined): Promise<boolean> => {
      if (!file) return false
      const rejection = validateAvatarFile(file)
      if (rejection) {
        toast.error(t(rejection))
        return false
      }

      setSaving(true)
      try {
        const response = await uploadLightcodeAvatar(file)
        const saved = response.data
        if (!response.success || !saved) {
          toast.error(response.message || t('Failed to update avatar'))
          return false
        }
        // The server returns the cache-busted URL, so show the new image now
        // instead of waiting for a refetch that would still be cached.
        if (mountedRef.current) {
          setIdentity((current) =>
            current ? { ...current, avatar_url: saved.avatar_url } : current
          )
        }
        toast.success(t('Avatar updated'))
        return true
      } catch {
        toast.error(t('Failed to update avatar'))
        return false
      } finally {
        if (mountedRef.current) setSaving(false)
      }
    },
    [t]
  )

  const removeAvatar = useCallback(async (): Promise<boolean> => {
    setSaving(true)
    try {
      const response = await deleteLightcodeAvatar()
      if (!response.success) {
        toast.error(response.message || t('Failed to update avatar'))
        return false
      }
      if (mountedRef.current) {
        setIdentity((current) =>
          current ? { ...current, avatar_url: '' } : current
        )
      }
      toast.success(t('Avatar removed'))
      return true
    } catch {
      toast.error(t('Failed to update avatar'))
      return false
    } finally {
      if (mountedRef.current) setSaving(false)
    }
  }, [t])

  return {
    identity,
    saving,
    refreshIdentity,
    saveDisplayName,
    uploadAvatar,
    removeAvatar,
  }
}
