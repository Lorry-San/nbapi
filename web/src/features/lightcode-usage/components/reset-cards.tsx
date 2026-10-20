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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { type LightcodeResetCard, useLightcodeResetCard } from '@/features/lightcode/api'

function formatDate(seconds: number): string {
  return new Date(seconds * 1000).toLocaleDateString()
}

/**
 * The user's reset cards, grouped by kind. Spending a card clears the matching windows of
 * the active subscription right away, so the button asks for confirmation first.
 * @returns the card section, or nothing when the user holds no card.
 */
export function ResetCards(props: { cards: LightcodeResetCard[]; onUsed: () => void }) {
  const { t } = useTranslation()
  const [pendingId, setPendingId] = useState<number | null>(null)
  const [confirmingId, setConfirmingId] = useState<number | null>(null)
  if (props.cards.length === 0) return null

  const use = async (card: LightcodeResetCard) => {
    setPendingId(card.id)
    try {
      const res = await useLightcodeResetCard(card.id)
      if (res.success) {
        toast.success(card.kind === 'full' ? t('5-hour and 7-day windows reset') : t('5-hour window reset'))
        props.onUsed()
      } else {
        toast.error(res.message || t('Request failed'))
      }
    } catch {
      toast.error(t('Request failed'))
    } finally {
      setPendingId(null)
      setConfirmingId(null)
    }
  }

  return (
    <div className='rounded-lg border p-4 sm:col-span-2' data-slot='lightcode-reset-card'>
      <p className='text-sm font-medium'>
        {t('You have {{count}} reset cards', { count: props.cards.length })}
      </p>
      <ul className='mt-3 flex flex-col gap-2'>
        {props.cards.map((card) => (
          <li key={card.id} className='flex flex-wrap items-center justify-between gap-2 text-sm'>
            <span>
              {card.kind === 'full' ? t('Resets 5-hour and 7-day windows') : t('Resets the 5-hour window')}
              <span className='text-muted-foreground ml-2 text-xs'>
                {t('Valid until {{date}}', { date: formatDate(card.expires_at) })}
              </span>
            </span>
            {confirmingId === card.id ? (
              <span className='flex gap-2'>
                <Button size='sm' variant='outline' onClick={() => setConfirmingId(null)} disabled={pendingId !== null}>
                  {t('Cancel')}
                </Button>
                <Button size='sm' onClick={() => use(card)} disabled={pendingId !== null}>
                  {t('Confirm reset')}
                </Button>
              </span>
            ) : (
              <Button size='sm' variant='outline' onClick={() => setConfirmingId(card.id)} disabled={pendingId !== null}>
                {t('Use card')}
              </Button>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}

/** Shows an operator reset that will clear the windows at a known time. */
export function ScheduledResetNotice(props: { runAt: number }) {
  const { t } = useTranslation()
  return (
    <p className='bg-muted/50 rounded-md px-3 py-2 text-xs sm:col-span-2'>
      {t('Usage will be reset at {{time}}', { time: new Date(props.runAt * 1000).toLocaleString() })}
    </p>
  )
}
