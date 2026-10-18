import { useTranslation } from 'react-i18next'

import { Progress } from '@/components/ui/progress'

import type { LightcodeWindow } from '../types'

/** Only utilization and reset time are shown for a subscription window. */
export function LightcodeWindowUsage(props: { label: string; window: LightcodeWindow; active: boolean }) {
  const { t } = useTranslation()
  return (
    <div>
      <div className='text-muted-foreground flex flex-wrap justify-between gap-2'>
        <span>{props.label}</span>
        <span>{t('Used')} {props.window.used_percent}%</span>
      </div>
      {props.active && <Progress value={props.window.used_percent} aria-label={props.label} className='mt-1 h-1.5' />}
      {props.window.reset_at > 0 && (
        <p className='text-muted-foreground mt-1'>
          {t('Resets at')} {new Date(props.window.reset_at * 1000).toLocaleString()}
        </p>
      )}
    </div>
  )
}
