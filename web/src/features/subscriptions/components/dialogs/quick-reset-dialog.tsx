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
import { Switch } from '@/components/ui/switch'
import { cn } from '@/lib/utils'

import {
  getAdminPlans,
  resetAllActiveSubscriptions,
  resetPlanSubscriptions,
  resetUserSubscriptionById,
  scheduleReset,
} from '../../api'
import type { LightcodeResetTarget } from '../../types'
import { useSubscriptions } from '../subscriptions-provider'
import { SubscriptionUserPicker } from './subscription-user-picker'

/** Which subscriptions a one-shot reset addresses. */
type ResetScope = 'subscription' | 'plan' | 'all'

const SCOPES: readonly ResetScope[] = ['subscription', 'plan', 'all']

/**
 * One dialog for the three reset scopes: a single subscription row, every
 * subscription of one plan, or every active subscription of every plan.
 * @returns the reset dialog.
 */
export function QuickResetDialog() {
  const { t } = useTranslation()
  const { open, setOpen, triggerRefresh } = useSubscriptions()
  const [scope, setScope] = useState<ResetScope>('plan')
  const [planId, setPlanId] = useState<number>(0)
  const [subscriptionId, setSubscriptionId] = useState('')
  const [advanceResetTime, setAdvanceResetTime] = useState(true)
  const [resetting, setResetting] = useState(false)
  // Scheduling runs the same reset later; a subscription-scoped schedule targets its user.
  const [scheduled, setScheduled] = useState(false)
  const [runAt, setRunAt] = useState('')
  const [pickedUserId, setPickedUserId] = useState<number | null>(null)
  const isOpen = open === 'quick-reset'

  const { data: planRows } = useQuery({
    queryKey: ['admin-subscription-plans', 'quick-reset'],
    queryFn: async () => (await getAdminPlans()).data || [],
    enabled: isOpen,
  })
  const plans = useMemo(
    () => (planRows ?? []).filter((row) => row.plan?.id),
    [planRows]
  )

  useEffect(() => {
    if (!isOpen) return
    setScope('plan')
    setSubscriptionId('')
    setAdvanceResetTime(true)
    setScheduled(false)
    setRunAt('')
    setPickedUserId(null)
  }, [isOpen])

  useEffect(() => {
    if (planId === 0 && plans[0]?.plan.id) setPlanId(plans[0].plan.id)
  }, [planId, plans])

  const parsedSubscriptionId = Number(subscriptionId.trim())
  const runAtSeconds = runAt ? Math.floor(new Date(runAt).getTime() / 1000) : 0
  const scheduleReady = runAtSeconds > Math.floor(Date.now() / 1000)
  const ready = (() => {
    if (scheduled) {
      if (!scheduleReady) return false
      if (scope === 'all') return true
      if (scope === 'plan') return planId > 0
      return pickedUserId !== null
    }
    if (scope === 'all') return true
    if (scope === 'plan') return planId > 0
    return Number.isInteger(parsedSubscriptionId) && parsedSubscriptionId > 0
  })()

  const handleSchedule = async () => {
    let targetType: LightcodeResetTarget = 'all'
    let targetId = 0
    if (scope === 'plan') {
      targetType = 'plan'
      targetId = planId
    } else if (scope === 'subscription') {
      targetType = 'user'
      targetId = pickedUserId ?? 0
    }
    const res = await scheduleReset({
      target_type: targetType,
      target_id: targetId,
      run_at: runAtSeconds,
      advance_reset_time: advanceResetTime,
    })
    if (res.success) {
      toast.success(t('Reset scheduled'))
      setOpen(null)
    } else {
      toast.error(res.message || t('Operation failed'))
    }
  }

  const handleConfirm = async () => {
    if (!ready) return
    setResetting(true)
    try {
      if (scheduled) {
        await handleSchedule()
        return
      }
      const payload = { advance_reset_time: advanceResetTime }
      let res
      if (scope === 'all') {
        res = await resetAllActiveSubscriptions(payload)
      } else if (scope === 'plan') {
        res = await resetPlanSubscriptions(planId, payload)
      } else {
        res = await resetUserSubscriptionById(parsedSubscriptionId, payload)
      }
      if (res.success) {
        toast.success(
          t('Reset {{count}} subscriptions for {{users}} users', {
            count: res.data?.reset_count || 0,
            users: res.data?.user_count || 0,
          })
        )
        triggerRefresh()
        setOpen(null)
      }
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setResetting(false)
    }
  }

  const scopeLabel = (value: ResetScope): string => {
    if (value === 'subscription') return t('One subscription')
    if (value === 'plan') return t('One plan')
    return t('All active subscriptions')
  }

  return (
    <ConfirmDialog
      open={isOpen}
      onOpenChange={(nextOpen) => !nextOpen && setOpen(null)}
      title={t('Reset 5-hour and 7-day usage')}
      desc={t(
        'Clears the 5-hour and 7-day windows of the selected subscriptions. Plan and renewal dates are unchanged.'
      )}
      confirmText={t('Reset usage')}
      handleConfirm={handleConfirm}
      disabled={!ready}
      isLoading={resetting}
    >
      <div className='space-y-3'>
        <div
          role='radiogroup'
          aria-label={t('Reset scope')}
          className='grid gap-2'
        >
          {SCOPES.map((value) => (
            <Button
              key={value}
              type='button'
              role='radio'
              aria-checked={scope === value}
              variant={scope === value ? 'secondary' : 'outline'}
              className={cn(
                'h-auto justify-start px-3 py-2 text-sm font-normal',
                scope === value && 'ring-1 ring-inset'
              )}
              onClick={() => setScope(value)}
            >
              {scopeLabel(value)}
            </Button>
          ))}
        </div>

        {scope === 'plan' ? (
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

        {scope === 'subscription' ? (
          <div className='space-y-2'>
            <SubscriptionUserPicker
              subscriptionId={subscriptionId}
              onPickSubscription={setSubscriptionId}
              onPickUser={scheduled ? setPickedUserId : undefined}
            />
            {scheduled ? null : (
              <label className='grid gap-1 text-sm'>
                <span>{t('Subscription ID')}</span>
                <Input
                  inputMode='numeric'
                  placeholder={t('Numeric subscription id')}
                  value={subscriptionId}
                  onChange={(event) => setSubscriptionId(event.target.value)}
                />
              </label>
            )}
          </div>
        ) : null}

        <label className='flex items-center justify-between gap-3 rounded-md border px-3 py-2 text-sm'>
          <span>{t('Schedule for later')}</span>
          <Switch
            checked={scheduled}
            onCheckedChange={(checked) => setScheduled(!!checked)}
            aria-label={t('Schedule for later')}
          />
        </label>
        {scheduled ? (
          <label className='grid gap-1 text-sm'>
            <span>{t('Reset time')}</span>
            <Input
              type='datetime-local'
              value={runAt}
              onChange={(event) => setRunAt(event.target.value)}
              aria-invalid={runAt !== '' && !scheduleReady}
            />
            <span className='text-muted-foreground text-xs'>
              {scope === 'subscription'
                ? t('A scheduled reset clears every active subscription of the picked user.')
                : t('Users see the scheduled time on their usage page.')}
            </span>
          </label>
        ) : null}

        <label className='flex items-center justify-between gap-3 rounded-md border px-3 py-2 text-sm'>
          <span>{t('Advance next reset time')}</span>
          <Switch
            checked={advanceResetTime}
            onCheckedChange={(checked) => setAdvanceResetTime(!!checked)}
            aria-label={t('Advance next reset time')}
          />
        </label>
      </div>
    </ConfirmDialog>
  )
}
