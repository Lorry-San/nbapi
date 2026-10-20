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
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  deleteLightcodeAvatar,
  type LightcodeProfileData,
  updateLightcodeDisplayName,
  uploadLightcodeAvatar,
} from '@/features/lightcode/api'

import { avatarHue, initials } from '../lib/activity'

/** Matches the server's avatar size limit so oversized files fail before uploading. */
const AVATAR_MAX_BYTES = 256 * 1024
const DISPLAY_NAME_MAX = 20

/**
 * Avatar, display name and username, with inline editing for the first two.
 * @returns the profile header.
 */
export function ProfileHeader(props: { user: LightcodeProfileData['user']; onChanged: () => void }) {
  const { t } = useTranslation()
  const name = props.user.display_name || props.user.username
  const hue = avatarHue(props.user.id)
  const fileRef = useRef<HTMLInputElement>(null)
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(name)
  const [busy, setBusy] = useState(false)
  const draftLength = [...draft.trim()].length
  const draftValid = draftLength > 0 && draftLength <= DISPLAY_NAME_MAX

  const pickAvatar = async (file: File | undefined) => {
    if (!file) return
    if (file.size > AVATAR_MAX_BYTES) {
      toast.error(t('Avatar must be 256KB or smaller'))
      return
    }
    setBusy(true)
    try {
      const res = await uploadLightcodeAvatar(file)
      if (res.success) {
        toast.success(t('Avatar updated'))
        props.onChanged()
      } else {
        toast.error(res.message || t('Request failed'))
      }
    } catch {
      toast.error(t('Request failed'))
    } finally {
      setBusy(false)
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  const removeAvatar = async () => {
    setBusy(true)
    try {
      const res = await deleteLightcodeAvatar()
      if (res.success) props.onChanged()
      else toast.error(res.message || t('Request failed'))
    } catch {
      toast.error(t('Request failed'))
    } finally {
      setBusy(false)
    }
  }

  const saveName = async () => {
    if (!draftValid) return
    setBusy(true)
    try {
      const res = await updateLightcodeDisplayName(draft)
      if (res.success) {
        toast.success(t('Display name updated'))
        setEditing(false)
        props.onChanged()
      } else {
        toast.error(res.message || t('Request failed'))
      }
    } catch {
      toast.error(t('Request failed'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className='flex flex-col items-center gap-2 text-center'>
      <button
        type='button'
        className='group relative size-20 overflow-hidden rounded-full'
        onClick={() => fileRef.current?.click()}
        disabled={busy}
        aria-label={t('Change avatar')}
        title={t('Change avatar')}
      >
        {props.user.avatar_url ? (
          <img src={props.user.avatar_url} alt='' className='size-full object-cover' />
        ) : (
          <span
            className='flex size-full items-center justify-center text-2xl font-semibold text-white'
            style={{ backgroundColor: `hsl(${hue} 55% 45%)` }}
            aria-hidden='true'
          >
            {initials(name)}
          </span>
        )}
        <span className='absolute inset-0 flex items-center justify-center bg-black/50 text-xs text-white opacity-0 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100'>
          {t('Change')}
        </span>
      </button>
      <input
        ref={fileRef}
        type='file'
        accept='image/png,image/jpeg,image/webp,image/gif'
        className='hidden'
        onChange={(event) => pickAvatar(event.target.files?.[0])}
      />
      {props.user.avatar_url ? (
        <Button size='sm' variant='ghost' onClick={removeAvatar} disabled={busy}>
          {t('Remove avatar')}
        </Button>
      ) : null}

      {editing ? (
        <div className='flex items-center gap-2'>
          <Input
            value={draft}
            maxLength={DISPLAY_NAME_MAX}
            onChange={(event) => setDraft(event.target.value)}
            aria-label={t('Display name')}
            aria-invalid={!draftValid}
            className='h-8 w-44'
          />
          <Button size='sm' onClick={saveName} disabled={busy || !draftValid}>
            {t('Save')}
          </Button>
          <Button
            size='sm'
            variant='outline'
            onClick={() => {
              setDraft(name)
              setEditing(false)
            }}
            disabled={busy}
          >
            {t('Cancel')}
          </Button>
        </div>
      ) : (
        <button
          type='button'
          className='text-xl font-semibold hover:underline'
          onClick={() => {
            setDraft(name)
            setEditing(true)
          }}
          title={t('Edit display name')}
        >
          {name}
        </button>
      )}
      <div className='text-muted-foreground text-sm'>@{props.user.username}</div>
    </div>
  )
}
