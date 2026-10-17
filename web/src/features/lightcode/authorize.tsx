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

import { Button } from '@/components/ui/button'
import { getSystemName } from '@/stores/system-config-store'

import { approveLightcode, type LightcodeAuthorizeRequest } from './api'

interface Props {
  request: LightcodeAuthorizeRequest
}

/** The redirect target is re-validated by the server before any code is issued. */
function loopbackPort(redirect: string): string {
  try {
    return new URL(redirect).port
  } catch {
    return ''
  }
}

export function LightcodeAuthorize(props: Props) {
  const { t } = useTranslation()
  const [busy, setBusy] = useState(false)
  const [failed, setFailed] = useState(false)
  const [denied, setDenied] = useState(false)

  const approve = async () => {
    setBusy(true)
    setFailed(false)
    try {
      const result = await approveLightcode(props.request)
      window.location.replace(result.redirect_uri)
    } catch {
      setFailed(true)
      setBusy(false)
    }
  }

  if (denied) {
    return (
      <main className='mx-auto flex min-h-svh max-w-md flex-col justify-center gap-4 p-6'>
        <h1 className='text-xl font-semibold'>{t('Authorization cancelled')}</h1>
        <p className='text-muted-foreground text-sm'>
          {t('You can close this page and return to Lightcode.')}
        </p>
      </main>
    )
  }

  return (
    <main className='mx-auto flex min-h-svh max-w-md flex-col justify-center gap-6 p-6'>
      <div className='flex flex-col gap-2'>
        <h1 className='text-xl font-semibold'>
          {t('Sign in to Lightcode with {{name}}', { name: getSystemName() })}
        </h1>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Lightcode on this computer will use your Lightcode subscription. It cannot use your API keys or wallet balance.'
          )}
        </p>
        <p className='text-muted-foreground text-xs'>
          {t('Returns to this computer on port {{port}}.', {
            port: loopbackPort(props.request.redirect_uri),
          })}
        </p>
      </div>
      {failed ? (
        <p role='alert' className='text-destructive text-sm'>
          {t('Authorization failed. Restart sign-in from Lightcode.')}
        </p>
      ) : null}
      <div className='flex gap-3'>
        <Button onClick={approve} disabled={busy}>
          {t('Authorize')}
        </Button>
        <Button
          variant='outline'
          onClick={() => setDenied(true)}
          disabled={busy}
        >
          {t('Cancel')}
        </Button>
      </div>
    </main>
  )
}
