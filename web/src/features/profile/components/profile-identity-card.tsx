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
import { Loader2, Pencil, Trash2, Upload } from 'lucide-react'
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { TitledCard } from '@/components/ui/titled-card'
import { getUserAvatarFallback, getUserAvatarStyle } from '@/lib/avatar'

import { useProfileIdentity } from '../hooks'
import { DISPLAY_NAME_MAX, isValidDisplayName } from '../lib'
import type { UserProfile } from '../types'

type ProfileIdentityCardProps = {
  profile: UserProfile | null
}

/**
 * Edit the avatar and display name shown across the dashboard.
 *
 * The dashboard user record carries neither field, so this card reads and writes
 * the gateway identity endpoints and keeps one local copy; callers show the same
 * values in the header through `useProfileIdentity`.
 */
export function ProfileIdentityCard(props: ProfileIdentityCardProps) {
  const { t } = useTranslation()
  const { identity, saving, saveDisplayName, uploadAvatar, removeAvatar } =
    useProfileIdentity()
  const fileRef = useRef<HTMLInputElement>(null)
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState('')

  const displayName = identity?.display_name || props.profile?.username || ''
  const avatarUrl = identity?.avatar_url || ''
  const fallbackName = props.profile?.username || displayName
  const draftValid = isValidDisplayName(draft)

  const beginEdit = () => {
    setDraft(displayName)
    setEditing(true)
  }

  const commit = async () => {
    if (!draftValid) return
    if (await saveDisplayName(draft.trim())) setEditing(false)
  }

  return (
    <TitledCard title={t('Profile')} icon={<Upload className='h-4 w-4' />}>
      <div className='flex flex-col gap-4 sm:flex-row sm:items-start'>
        <div className='flex flex-col items-center gap-2'>
          {avatarUrl ? (
            <img
              src={avatarUrl}
              alt=''
              className='ring-background h-20 w-20 rounded-2xl object-cover ring-2'
            />
          ) : (
            <span
              className='ring-background flex h-20 w-20 items-center justify-center rounded-2xl text-2xl font-semibold text-white ring-2'
              style={getUserAvatarStyle(fallbackName)}
              aria-hidden='true'
            >
              {getUserAvatarFallback(fallbackName)}
            </span>
          )}
          <div className='flex gap-2'>
            <Button
              type='button'
              variant='outline'
              size='sm'
              disabled={saving}
              onClick={() => fileRef.current?.click()}
            >
              {saving ? <Loader2 className='h-3.5 w-3.5 animate-spin' /> : null}
              {t('Change avatar')}
            </Button>
            {avatarUrl && (
              <Button
                type='button'
                variant='ghost'
                size='sm'
                disabled={saving}
                onClick={() => {
                  void removeAvatar()
                }}
                aria-label={t('Remove avatar')}
              >
                <Trash2 className='h-3.5 w-3.5' />
              </Button>
            )}
          </div>
          <input
            ref={fileRef}
            type='file'
            accept='image/png,image/jpeg,image/webp,image/gif'
            className='hidden'
            onChange={(event) => {
              const file = event.target.files?.[0]
              if (fileRef.current) fileRef.current.value = ''
              void uploadAvatar(file)
            }}
          />
        </div>

        <div className='min-w-0 flex-1 space-y-2'>
          <div className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
            {t('Display name')}
          </div>
          {editing ? (
            <div className='flex flex-wrap items-center gap-2'>
              <Input
                value={draft}
                maxLength={DISPLAY_NAME_MAX}
                aria-label={t('Display name')}
                aria-invalid={!draftValid}
                onChange={(event) => {
                  setDraft(event.target.value)
                }}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') void commit()
                  if (event.key === 'Escape') setEditing(false)
                }}
              />
              <Button
                type='button'
                disabled={saving || !draftValid}
                onClick={() => {
                  void commit()
                }}
              >
                {t('Save')}
              </Button>
              <Button
                type='button'
                variant='outline'
                disabled={saving}
                onClick={() => {
                  setEditing(false)
                }}
              >
                {t('Cancel')}
              </Button>
            </div>
          ) : (
            <div className='flex items-center gap-2'>
              <span className='truncate text-lg font-medium'>{displayName}</span>
              <Button
                type='button'
                variant='ghost'
                size='sm'
                onClick={beginEdit}
                aria-label={t('Edit display name')}
              >
                <Pencil className='h-3.5 w-3.5' />
              </Button>
            </div>
          )}
          {props.profile?.username && (
            <p className='text-muted-foreground text-xs'>
              @{props.profile.username}
            </p>
          )}
          <p className='text-muted-foreground text-xs'>
            {t(
              'This name and avatar appear in the desktop client and on this page.'
            )}
          </p>
        </div>
      </div>
    </TitledCard>
  )
}
