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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'

import { getLightcodeDevices, revokeLightcodeDevice } from './api'

const DEVICES_KEY = ['lightcode', 'devices']

function formatTime(seconds: number): string {
  return seconds > 0 ? new Date(seconds * 1000).toLocaleString() : '-'
}

export function LightcodeDevices() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const devices = useQuery({ queryKey: DEVICES_KEY, queryFn: getLightcodeDevices })
  const revoke = useMutation({
    mutationFn: revokeLightcodeDevice,
    onSuccess: () => {
      toast.success(t('Device signed out'))
      void queryClient.invalidateQueries({ queryKey: DEVICES_KEY })
    },
    onError: () => toast.error(t('Request failed')),
  })
  const rows = devices.data?.data ?? []

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Lightcode devices')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <p className='text-muted-foreground mb-4 text-sm'>
          {t('Signing out a device revokes its Lightcode login immediately.')}
        </p>
        {rows.length === 0 && !devices.isLoading ? (
          <p className='text-muted-foreground text-sm'>{t('No Lightcode devices')}</p>
        ) : null}
        <ul className='flex flex-col gap-3'>
          {rows.map((device) => (
            <li
              key={device.id}
              className='flex items-center justify-between gap-4 rounded-md border p-3'
            >
              <div className='min-w-0 text-sm'>
                <div className='truncate font-medium'>{device.device || t('Unknown device')}</div>
                <div className='text-muted-foreground text-xs'>
                  {t('Last active')}: {formatTime(device.last_active_at)} · {device.ip || '-'}
                </div>
              </div>
              {device.revoked_at > 0 ? (
                <span className='text-muted-foreground text-xs'>{t('Signed out')}</span>
              ) : (
                <Button
                  variant='outline'
                  size='sm'
                  disabled={revoke.isPending}
                  onClick={() => revoke.mutate(device.id)}
                >
                  {t('Sign out')}
                </Button>
              )}
            </li>
          ))}
        </ul>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
