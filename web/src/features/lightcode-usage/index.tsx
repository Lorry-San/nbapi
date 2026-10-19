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
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import {
  getLightcodeProfile,
  type LightcodeProfileData,
  type LightcodeUsageWindow,
} from '@/features/lightcode/api'
import { toIntlLocale } from '@/i18n/languages'
import { cn } from '@/lib/utils'

import {
  avatarHue,
  buildHeatmap,
  formatTokens,
  initials,
  type ActivityMode,
} from './lib/activity'

const LEVEL_ALPHA = [0, 0.3, 0.5, 0.75, 1]

function formatTime(seconds: number): string {
  return seconds > 0 ? new Date(seconds * 1000).toLocaleString() : '-'
}

function formatWindowLabel(seconds: number, t: (k: string) => string): string {
  if (seconds === 18000) return t('5 hour window')
  if (seconds === 604800) return t('7 day window')
  return seconds % 86400 === 0
    ? `${seconds / 86400}d`
    : `${Math.round(seconds / 3600)}h`
}

function WindowCard(props: { window: LightcodeUsageWindow | null }) {
  const { t } = useTranslation()
  if (!props.window) return null
  const percent = Math.min(100, Math.max(0, props.window.used_percent))
  return (
    <div className='rounded-lg border p-4'>
      <div className='flex items-center justify-between text-sm'>
        <span className='font-medium'>
          {formatWindowLabel(props.window.seconds, t)}
        </span>
        <span className='tabular-nums'>
          {t('Used')} {percent}%
        </span>
      </div>
      <Progress
        value={percent}
        aria-label={formatWindowLabel(props.window.seconds, t)}
        className='mt-2'
      />
      {props.window.reset_at > 0 && (
        <p className='text-muted-foreground mt-2 text-xs'>
          {t('Resets at')} {formatTime(props.window.reset_at)}
        </p>
      )}
    </div>
  )
}

function ProfileHeader(props: { user: LightcodeProfileData['user'] }) {
  const name = props.user.display_name || props.user.username
  const hue = avatarHue(props.user.id)
  return (
    <div className='flex flex-col items-center gap-2 text-center'>
      <div
        className='flex size-20 items-center justify-center rounded-full text-2xl font-semibold text-white'
        style={{ backgroundColor: `hsl(${hue} 55% 45%)` }}
        aria-hidden='true'
      >
        {initials(name)}
      </div>
      <div className='text-xl font-semibold'>{name}</div>
      <div className='text-muted-foreground text-sm'>
        @{props.user.username}
      </div>
    </div>
  )
}

function StatTiles(props: { stats: LightcodeProfileData['stats'] }) {
  const { t } = useTranslation()
  const s = props.stats
  const tiles = [
    {
      label: t('Total tokens'),
      value: formatTokens(s.total_tokens),
      title: s.total_tokens.toLocaleString(),
    },
    {
      label: t('Peak day'),
      value: formatTokens(s.peak_day_tokens),
      title: s.peak_day
        ? `${s.peak_day} · ${s.peak_day_tokens.toLocaleString()}`
        : s.peak_day_tokens.toLocaleString(),
    },
    {
      label: t('Longest streak'),
      value: t('{{count}} days', { count: s.longest_streak_days }),
    },
    {
      label: t('Current streak'),
      value: t('{{count}} days', { count: s.current_streak_days }),
    },
  ]
  return (
    <div className='grid grid-cols-2 divide-y rounded-xl border sm:grid-cols-4 sm:divide-x sm:divide-y-0'>
      {tiles.map((tile) => (
        <div key={tile.label} className='p-4 text-center' title={tile.title}>
          <div className='text-xl font-semibold tabular-nums'>{tile.value}</div>
          <div className='text-muted-foreground mt-1 text-xs'>{tile.label}</div>
        </div>
      ))}
    </div>
  )
}

function PlanCard(props: { sub: LightcodeProfileData['subscription'] }) {
  const { t } = useTranslation()
  const sub = props.sub
  return (
    <div className='rounded-xl border p-4'>
      {sub ? (
        <div className='flex flex-col gap-3'>
          <div className='flex flex-wrap items-baseline justify-between gap-2'>
            <span className='font-semibold'>{sub.plan}</span>
            <span className='text-muted-foreground text-xs'>
              {t('Expires at')} {formatTime(sub.end_time)}
            </span>
          </div>
          <div className='grid gap-3 sm:grid-cols-2'>
            <WindowCard window={sub.window_short} />
            <WindowCard window={sub.window_long} />
          </div>
        </div>
      ) : (
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <p className='text-muted-foreground text-sm'>
            {t('No active Lightcode plan')}
          </p>
          <Button size='sm' render={<Link to='/lightcode/subscription' />}>
            {t('View plans')}
          </Button>
        </div>
      )}
    </div>
  )
}

const CELL = 12
const GAP = 3

function ActivityHeatmap(props: { days: LightcodeProfileData['days'] }) {
  const { t, i18n } = useTranslation()
  const [mode, setMode] = useState<ActivityMode>('daily')
  const map = useMemo(() => buildHeatmap(props.days, mode), [props.days, mode])
  const monthFmt = useMemo(
    () =>
      new Intl.DateTimeFormat(toIntlLocale(i18n.language), {
        month: 'short',
        timeZone: 'UTC',
      }),
    [i18n.language]
  )
  const modes: [ActivityMode, string][] = [
    ['daily', t('Daily')],
    ['weekly', t('Weekly')],
    ['cumulative', t('Cumulative')],
  ]
  const width = map.columns.length * (CELL + GAP) - GAP
  const cells = map.columns.flatMap((col, ci) =>
    col.map((cell, ri) => ({ cell, id: cell?.date ?? `pad-${ci}-${ri}` }))
  )

  return (
    <div className='rounded-xl border p-4'>
      <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
        <h2 className='text-sm font-semibold'>{t('Token activity')}</h2>
        <div
          className='bg-muted flex rounded-md p-0.5'
          role='group'
          aria-label={t('Token activity')}
        >
          {modes.map(([value, label]) => (
            <button
              key={value}
              type='button'
              aria-pressed={mode === value}
              onClick={() => setMode(value)}
              className={cn(
                'rounded px-2.5 py-1 text-xs transition-colors',
                mode === value
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              {label}
            </button>
          ))}
        </div>
      </div>
      <div className='overflow-x-auto pb-1'>
        <div style={{ width }} className='mx-auto'>
          <div
            role='img'
            aria-label={t('Token activity over the last year')}
            className='grid grid-flow-col'
            style={{
              gridTemplateRows: `repeat(7, ${CELL}px)`,
              gridAutoColumns: `${CELL}px`,
              gap: GAP,
            }}
          >
            {cells.map(({ cell, id }) =>
              cell ? (
                <div
                  key={id}
                  title={`${cell.date} · ${cell.tokens.toLocaleString()} ${t('tokens')}${
                    mode === 'daily'
                      ? ''
                      : ` (${modes.find((m) => m[0] === mode)?.[1]}: ${cell.value.toLocaleString()})`
                  }`}
                  className={cn(
                    'rounded-[3px]',
                    cell.level === 0 && 'bg-muted'
                  )}
                  style={
                    cell.level > 0
                      ? {
                          backgroundColor: `rgb(103 153 254 / ${LEVEL_ALPHA[cell.level]})`,
                        }
                      : undefined
                  }
                />
              ) : (
                <div key={id} />
              )
            )}
          </div>
          <div className='text-muted-foreground relative mt-1 h-4 text-[10px]'>
            {map.months.map((m) => {
              const cell = map.columns[m.column].find((c) => c !== null)
              return (
                <span
                  key={`${m.column}`}
                  className='absolute'
                  style={{ left: m.column * (CELL + GAP) }}
                >
                  {cell
                    ? monthFmt.format(new Date(`${cell.date}T00:00:00Z`))
                    : ''}
                </span>
              )
            })}
          </div>
        </div>
      </div>
      <div className='text-muted-foreground mt-2 flex items-center justify-end gap-1 text-[10px]'>
        <span>{t('Less')}</span>
        {LEVEL_ALPHA.map((a, i) => (
          <span
            key={a}
            className={cn('size-2.5 rounded-[2px]', i === 0 && 'bg-muted')}
            style={
              i > 0 ? { backgroundColor: `rgb(103 153 254 / ${a})` } : undefined
            }
          />
        ))}
        <span>{t('More')}</span>
      </div>
    </div>
  )
}

function TopModels(props: { data: LightcodeProfileData }) {
  const { t } = useTranslation()
  const total = props.data.models.reduce((s, m) => s + m.total_tokens, 0)
  return (
    <div className='rounded-xl border p-4'>
      <h2 className='mb-3 text-sm font-semibold'>{t('Top models')}</h2>
      <ul className='flex flex-col gap-3'>
        {props.data.models.map((m) => {
          const share = total > 0 ? (m.total_tokens / total) * 100 : 0
          return (
            <li key={m.model} className='text-sm'>
              <div className='flex items-center justify-between gap-2'>
                <span className='truncate font-medium'>{m.model}</span>
                <span
                  className='text-muted-foreground shrink-0 tabular-nums'
                  title={m.total_tokens.toLocaleString()}
                >
                  {formatTokens(m.total_tokens)} · {share.toFixed(1)}%
                </span>
              </div>
              <Progress value={share} aria-label={m.model} className='mt-1' />
            </li>
          )
        })}
      </ul>
    </div>
  )
}

export function LightcodeUsage() {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['lightcode', 'profile'],
    queryFn: getLightcodeProfile,
  })
  const data = query.data?.success ? query.data.data : undefined

  let content: ReactNode
  if (query.isLoading) {
    content = (
      <>
        <div className='flex flex-col items-center gap-2'>
          <Skeleton className='size-20 rounded-full' />
          <Skeleton className='h-5 w-32' />
        </div>
        <Skeleton className='h-20 w-full' />
        <Skeleton className='h-28 w-full' />
        <Skeleton className='h-44 w-full' />
      </>
    )
  } else if (query.isError || !data) {
    content = (
      <div className='flex flex-col items-start gap-3'>
        <p className='text-muted-foreground text-sm'>
          {query.data?.message || t('Request failed')}
        </p>
        <Button variant='outline' size='sm' onClick={() => query.refetch()}>
          {t('Retry')}
        </Button>
      </div>
    )
  } else {
    content = (
      <>
        <ProfileHeader user={data.user} />
        <StatTiles stats={data.stats} />
        <PlanCard sub={data.subscription} />
        {data.stats.requests === 0 ? (
          <p className='text-muted-foreground py-6 text-center text-sm'>
            {t('No Lightcode usage yet')}
          </p>
        ) : (
          <>
            <ActivityHeatmap days={data.days} />
            {data.models.length > 0 && <TopModels data={data} />}
          </>
        )}
        {data.truncated && (
          <p className='text-muted-foreground text-xs'>
            {t('Some older usage records were not counted.')}
          </p>
        )}
      </>
    )
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Usage Dashboard')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='mx-auto flex w-full max-w-[736px] flex-col gap-5'>
          {content}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
