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
import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { getUsers, searchUsers } from '@/features/users/api'
import type { User } from '@/features/users/types'
import { cn } from '@/lib/utils'

import { getUserSubscriptions } from '../../api'
import type { UserSubscriptionRecord } from '../../types'

/** Debounce before a keystroke becomes a request. */
const SEARCH_DEBOUNCE_MS = 300

interface SubscriptionUserPickerProps {
  /** The subscription id currently selected, or an empty string. */
  subscriptionId: string
  /** Called with the subscription id of the picked user's subscription. */
  onPickSubscription: (subscriptionId: string) => void
  /** Called with the picked user id; when set, the subscription list is not shown. */
  onPickUser?: (userId: number | null) => void
}

/**
 * Search users by name, display name, or email, then pick one of their
 * subscriptions. Picking fills the reset target so an operator never has to look
 * up a subscription id first.
 * @returns the picker.
 */
export function SubscriptionUserPicker(props: SubscriptionUserPickerProps) {
  const { t } = useTranslation()
  // The field is uncontrolled between searches, so a picked user's name survives typing.
  const [query, setQuery] = useState('')
  const [debounced, setDebounced] = useState('')
  const [pickedUserId, setPickedUserId] = useState<number | null>(null)

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim()), SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [query])

  // An empty query browses the first page, so the picker is usable without typing.
  const { data: userPage, isFetching } = useQuery({
    queryKey: ['subscription-reset-users', debounced],
    queryFn: async () => {
      if (debounced === '') {
        return (await getUsers({ p: 1, page_size: 10 })).data
      }
      return (
        await searchUsers({ keyword: debounced, p: 1, page_size: 10 })
      ).data
    },
  })
  const users: User[] = (userPage?.items as User[] | undefined) ?? []

  const { data: subscriptions, isFetching: loadingSubscriptions } = useQuery({
    queryKey: ['subscription-reset-user-subs', pickedUserId],
    queryFn: async () => {
      if (pickedUserId === null) return []
      const res = await getUserSubscriptions(pickedUserId)
      return res.data ?? []
    },
    enabled: pickedUserId !== null,
  })
  const rows: UserSubscriptionRecord[] = subscriptions ?? []

  const pickedUser = users.find((user) => user.id === pickedUserId)

  return (
    <div className='space-y-2'>
      <label className='grid gap-1 text-sm'>
        <span>{t('Search users')}</span>
        <input
          className='border-input bg-background h-9 rounded-md border px-2 text-sm'
          type='search'
          placeholder={t('Name or email, any part of it')}
          value={query}
          onChange={(event) => {
            setQuery(event.target.value)
            setPickedUserId(null)
            props.onPickUser?.(null)
          }}
        />
      </label>

      <div
        className='max-h-40 overflow-y-auto rounded-md border'
        role='group'
        aria-label={t('Matching users')}
        aria-busy={isFetching}
      >
        {users.length === 0 ? (
          <p className='text-muted-foreground px-3 py-2 text-sm'>
            {isFetching ? t('Searching...') : t('No matching users')}
          </p>
        ) : (
          users.map((user) => (
            <button
              key={user.id}
              type='button'
              role='button'
              aria-pressed={pickedUserId === user.id}
              className={cn(
                'hover:bg-accent flex w-full items-baseline justify-between gap-2 px-3 py-2 text-left text-sm',
                pickedUserId === user.id && 'bg-accent'
              )}
              onClick={() => {
                setPickedUserId(user.id)
                props.onPickUser?.(user.id)
              }}
            >
              <span className='truncate'>{user.username}</span>
              <span className='text-muted-foreground truncate text-xs'>
                {user.email}
              </span>
            </button>
          ))
        )}
      </div>

      {pickedUserId !== null && !props.onPickUser ? (
        <div className='grid gap-1 text-sm'>
          <span>{t('Subscriptions of {{name}}', { name: pickedUser?.username ?? `#${pickedUserId}` })}</span>
          <div
            className='max-h-40 overflow-y-auto rounded-md border'
            role='group'
            aria-label={t('Subscriptions of the selected user')}
            aria-busy={loadingSubscriptions}
          >
            {rows.length === 0 ? (
              <p className='text-muted-foreground px-3 py-2 text-sm'>
                {loadingSubscriptions ? t('Loading...') : t('No subscriptions')}
              </p>
            ) : (
              rows.map((row) => {
                const value = String(row.subscription.id)
                const active = props.subscriptionId.trim() === value
                return (
                  <button
                    key={row.subscription.id}
                    type='button'
                    role='button'
                    aria-pressed={active}
                    className={cn(
                      'hover:bg-accent flex w-full items-baseline justify-between gap-2 px-3 py-2 text-left text-sm',
                      active && 'bg-accent'
                    )}
                    onClick={() => props.onPickSubscription(value)}
                  >
                    <span>
                      #{row.subscription.id} · {t('Plan')} {row.subscription.plan_id}
                    </span>
                    <span className='text-muted-foreground text-xs'>
                      {row.subscription.status}
                    </span>
                  </button>
                )
              })
            )}
          </div>
        </div>
      ) : null}
    </div>
  )
}
