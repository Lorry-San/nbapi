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
import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

import { getAdminPlans, giftResetCards } from '../../api'
import type { LightcodeResetCardKind, LightcodeResetTarget } from '../../types'
import { useSubscriptions } from '../subscriptions-provider'
import { SubscriptionUserPicker } from './subscription-user-picker'

const TARGETS: readonly LightcodeResetTarget[] = ['user', 'plan', 'all']
const KINDS: readonly LightcodeResetCardKind[] = ['short', 'full']

/**
 * Gift reset cards to one user, every subscriber of a plan, or every subscriber. Each card
 * lasts a month and is spent by the user from their usage page.
 * @returns the gift dialog.
 */
export function GiftResetCardsDialog() {
  const { t } = useTranslation()
  const { open, setOpen } = useSubscriptions()
  const isOpen = open === 'gift-reset-cards'
  const [target, setTarget] = useState<LightcodeResetTarget>('user')
  const [kind, setKind] = useState<LightcodeResetCardKind>('short')
  const [count, setCount] = useState('1')
  const [planId, setPlanId] = useState(0)
  const [userId, setUserId] = useState<number | null>(null)
  const [sending, setSending] = useState(false)

  const { data: planRows } = useQuery({
    queryKey: ['admin-subscription-plans', 'gift-reset-cards'],
    queryFn: async () => (await getAdminPlans()).data || [],
    enabled: isOpen,
  })
  const plans = useMemo(() => (planRows ?? []).filter((row) => row.plan?.id), [planRows])

  useEffect(() => {
    if (!isOpen) return
    setTarget('user')
    setKind('short')
    setCount('1')
    setUserId(null)
  }, [isOpen])

  useEffect(() => {
    if (planId === 0 && plans[0]?.plan.id) setPlanId(plans[0].plan.id)
  }, [planId, plans])

  const parsedCount = Number(count)
  const countValid = Number.isInteger(parsedCount) && parsedCount >= 1 && parsedCount <= 100
  const ready = (() => {
    if (!countValid) return false
    if (target === 'user') return userId !== null
    if (target === 'plan') return planId > 0
    return true
  })()

  const targetLabel = (value: LightcodeResetTarget): string => {
    if (value === 'user') return t('One user')
    if (value === 'plan') return t('One plan')
    return t('All subscribers')
  }
  const kindLabel = (value: LightcodeResetCardKind): string =>
    value === 'short' ? t('5-hour window only') : t('5-hour and 7-day windows')

  const handleConfirm = async () => {
    if (!ready) return
    setSending(true)
    try {
      let targetId = 0
      if (target === 'user') targetId = userId ?? 0
      if (target === 'plan') targetId = planId
      const res = await giftResetCards({ target_type: target, target_id: targetId, kind, count: parsedCount })
      if (res.success) {
        toast.success(
          t('Gave {{cards}} reset cards to {{users}} users', {
            cards: res.data?.card_count || 0,
            users: res.data?.user_count || 0,
          })
        )
        setOpen(null)
      } else {
        toast.error(res.message || t('Operation failed'))
      }
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setSending(false)
    }
  }

  return (
    <ConfirmDialog
      open={isOpen}
      onOpenChange={(nextOpen) => !nextOpen && setOpen(null)}
      title={t('Gift reset cards')}
      desc={t('Each card lasts one month. Users spend it from their usage page.')}
      confirmText={t('Gift cards')}
      handleConfirm={handleConfirm}
      disabled={!ready}
      isLoading={sending}
    >
      <div className='space-y-3'>
        <div role='radiogroup' aria-label={t('Recipients')} className='grid grid-cols-3 gap-2'>
          {TARGETS.map((value) => (
            <Button
              key={value}
              type='button'
              role='radio'
              aria-checked={target === value}
              variant={target === value ? 'secondary' : 'outline'}
              className={cn('h-auto px-3 py-2 text-sm font-normal', target === value && 'ring-1 ring-inset')}
              onClick={() => setTarget(value)}
            >
              {targetLabel(value)}
            </Button>
          ))}
        </div>

        {target === 'user' ? (
          <SubscriptionUserPicker subscriptionId='' onPickSubscription={() => {}} onPickUser={setUserId} />
        ) : null}

        {target === 'plan' ? (
          <label className='grid gap-1 text-sm'>
            <span>{t('Plan')}</span>
            <select
              className='border-input bg-background h-9 rounded-md border px-2 text-sm'
              value={planId}
              onChange={(event) => setPlanId(Number(event.target.value))}
            >
              {plans.map((row) => (
                <option key={row.plan.id} value={row.plan.id}>
                  {row.plan.title || `#${row.plan.id}`}
                </option>
              ))}
            </select>
          </label>
        ) : null}

        <div role='radiogroup' aria-label={t('Card type')} className='grid gap-2'>
          {KINDS.map((value) => (
            <Button
              key={value}
              type='button'
              role='radio'
              aria-checked={kind === value}
              variant={kind === value ? 'secondary' : 'outline'}
              className={cn('h-auto justify-start px-3 py-2 text-sm font-normal', kind === value && 'ring-1 ring-inset')}
              onClick={() => setKind(value)}
            >
              {kindLabel(value)}
            </Button>
          ))}
        </div>

        <label className='grid gap-1 text-sm'>
          <span>{t('Cards per user')}</span>
          <Input
            type='number'
            min={1}
            max={100}
            inputMode='numeric'
            value={count}
            onChange={(event) => setCount(event.target.value)}
            aria-invalid={!countValid}
          />
        </label>
      </div>
    </ConfirmDialog>
  )
}
