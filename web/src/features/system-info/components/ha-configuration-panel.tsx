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
import {
  AlertTriangle,
  CheckCircle2,
  Copy,
  Database,
  Globe2,
  RefreshCw,
  Route,
  Save,
  ShieldCheck,
  XCircle,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

import { getHAOverview, updateHAConfig } from '../api'
import type { HACheck, HAConfig, HAHealthProbe, HAOverview } from '../types'

const HA_POLL_INTERVAL_MS = 30_000

const DEFAULT_CONFIG: HAConfig = {
  enabled: false,
  primary_node_name: 'nbapi-main',
  standby_node_name: 'nbapi-backup',
  primary_health_url: '',
  standby_health_url: '',
  public_entry: 'api.lcapi.online',
  origin_entry: 'o-api.lcapi.online',
  primary_origin: '',
  standby_origin: '',
  dns_provider: 'cloudflare',
  dns_record_name: '',
  database_engine: 'postgresql',
  replication_mode: 'external',
  redis_mode: 'shared',
  failover_strategy: 'manual',
  health_check_interval_seconds: 30,
  failover_threshold: 3,
  cutover_runbook: '',
  rollback_runbook: '',
  notes: '',
}

const DATABASE_OPTIONS = [
  { value: 'postgresql', label: 'PostgreSQL' },
  { value: 'mysql', label: 'MySQL' },
  { value: 'sqlite', label: 'SQLite' },
  { value: 'external', label: 'External database' },
] as const

const REPLICATION_OPTIONS = [
  { value: 'external', label: 'External or managed' },
  { value: 'postgres_streaming', label: 'PostgreSQL streaming' },
  { value: 'mysql_replica', label: 'MySQL replica' },
  { value: 'managed', label: 'Managed service' },
  { value: 'manual', label: 'Manual restore' },
] as const

const REDIS_OPTIONS = [
  { value: 'shared', label: 'Shared Redis' },
  { value: 'sentinel', label: 'Redis Sentinel' },
  { value: 'primary_standby', label: 'Primary/standby Redis' },
  { value: 'disabled', label: 'Disabled' },
] as const

const DNS_OPTIONS = [
  { value: 'cloudflare', label: 'Cloudflare' },
  { value: 'manual', label: 'Manual DNS' },
  { value: 'other', label: 'Other provider' },
] as const

const FAILOVER_OPTIONS = [
  { value: 'manual', label: 'Manual confirmation' },
  { value: 'assisted', label: 'Assisted diagnosis' },
] as const

const SUMMARY_CLASS_NAME: Record<HAOverview['summary'], string> = {
  ok: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300',
  warn: 'bg-amber-50 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300',
  error: '',
  disabled: 'bg-muted text-muted-foreground',
}

const CHECK_ICON = {
  ok: CheckCircle2,
  warn: AlertTriangle,
  error: XCircle,
} as const

function normalizeConfig(config?: HAConfig): HAConfig {
  return { ...DEFAULT_CONFIG, ...config }
}

function sameConfig(a: HAConfig, b: HAConfig) {
  return JSON.stringify(a) === JSON.stringify(b)
}

function summaryLabel(summary: HAOverview['summary']) {
  if (summary === 'ok') return 'healthy'
  if (summary === 'warn') return 'needs attention'
  if (summary === 'error') return 'misconfigured'
  return 'disabled'
}

type FieldProps = {
  label: string
  value: string
  placeholder?: string
  onChange: (value: string) => void
}

function Field(props: FieldProps) {
  return (
    <div className='space-y-1.5'>
      <Label className='text-xs'>{props.label}</Label>
      <Input
        value={props.value}
        placeholder={props.placeholder}
        onChange={(event) => props.onChange(event.target.value)}
      />
    </div>
  )
}

type SelectFieldProps = {
  label: string
  value: string
  options: readonly { value: string; label: string }[]
  onChange: (value: string) => void
}

function SelectField(props: SelectFieldProps) {
  const { t } = useTranslation()

  return (
    <div className='space-y-1.5'>
      <Label className='text-xs'>{props.label}</Label>
      <NativeSelect
        className='w-full'
        value={props.value}
        onChange={(event) => props.onChange(event.target.value)}
      >
        {props.options.map((option) => (
          <NativeSelectOption key={option.value} value={option.value}>
            {t(option.label)}
          </NativeSelectOption>
        ))}
      </NativeSelect>
    </div>
  )
}

type NumberFieldProps = {
  label: string
  value: number
  min: number
  max: number
  onChange: (value: number) => void
}

function NumberField(props: NumberFieldProps) {
  return (
    <div className='space-y-1.5'>
      <Label className='text-xs'>{props.label}</Label>
      <Input
        type='number'
        min={props.min}
        max={props.max}
        value={props.value}
        onChange={(event) => {
          const next = Number(event.target.value)
          props.onChange(Number.isNaN(next) ? props.min : next)
        }}
      />
    </div>
  )
}

type ConfigGroupProps = {
  icon: LucideIcon
  title: string
  children: ReactNode
}

function ConfigGroup(props: ConfigGroupProps) {
  const Icon = props.icon

  return (
    <div className='space-y-3 rounded-md border p-3'>
      <div className='flex items-center gap-2 text-xs font-medium'>
        <Icon className='text-muted-foreground size-3.5' aria-hidden='true' />
        {props.title}
      </div>
      {props.children}
    </div>
  )
}

type SnippetBlockProps = {
  title: string
  value: string
}

function SnippetBlock(props: SnippetBlockProps) {
  const { t } = useTranslation()

  async function copy() {
    try {
      await navigator.clipboard.writeText(props.value)
      toast.success(t('Copied'))
    } catch {
      toast.error(t('Copy failed'))
    }
  }

  return (
    <div className='rounded-md border'>
      <div className='flex items-center justify-between border-b px-3 py-2'>
        <span className='text-xs font-medium'>{props.title}</span>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          className='size-7'
          onClick={() => void copy()}
          aria-label={t('Copy')}
        >
          <Copy className='size-3.5' aria-hidden='true' />
        </Button>
      </div>
      <pre className='text-muted-foreground overflow-x-auto px-3 py-2 font-mono text-[11px] leading-relaxed'>
        {props.value}
      </pre>
    </div>
  )
}

function CheckRow({ check }: { check: HACheck }) {
  const Icon = CHECK_ICON[check.level]
  return (
    <li className='flex items-start gap-2 rounded-md border px-3 py-2'>
      <Icon
        className={cn(
          'mt-0.5 size-4 shrink-0',
          check.level === 'ok' && 'text-emerald-500',
          check.level === 'warn' && 'text-amber-500',
          check.level === 'error' && 'text-destructive',
        )}
        aria-hidden='true'
      />
      <div className='min-w-0'>
        <div className='font-mono text-[11px]'>{check.key}</div>
        <div className='text-muted-foreground text-xs'>{check.message}</div>
      </div>
    </li>
  )
}

function probeValue(probe: HAHealthProbe, key: string) {
  const value = probe.data?.[key]
  if (value === undefined || value === null || value === '') return '-'
  return String(value)
}

function ProbeRow({ probe }: { probe: HAHealthProbe }) {
  const reachable = probe.reachable
  return (
    <li className='rounded-md border px-3 py-2'>
      <div className='flex items-center justify-between gap-3'>
        <div className='min-w-0'>
          <div className='font-medium'>{probe.target}</div>
          <div className='text-muted-foreground truncate font-mono text-[11px]'>
            {probe.url}
          </div>
        </div>
        <Badge variant={reachable ? 'secondary' : 'destructive'}>
          {reachable ? probe.status_code || 'ok' : 'down'}
        </Badge>
      </div>
      <div className='text-muted-foreground mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs'>
        <span>database</span>
        <span className='font-mono'>{probeValue(probe, 'database')}</span>
        <span>role</span>
        <span className='font-mono'>{probeValue(probe, 'databaseRole')}</span>
      </div>
      {probe.message && (
        <div className='text-muted-foreground mt-2 truncate text-xs'>
          {probe.message}
        </div>
      )}
    </li>
  )
}

export function HAConfigurationPanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const overviewQuery = useQuery({
    queryKey: ['system-info', 'ha'],
    queryFn: async () => {
      const res = await getHAOverview()
      if (!res.success || !res.data) {
        throw new Error(res.message || t('We could not load HA settings.'))
      }
      return res.data
    },
    staleTime: 30 * 1000,
    retry: false,
    refetchInterval: (query) =>
      (query.state.data?.config.health_check_interval_seconds ??
        HA_POLL_INTERVAL_MS / 1000) * 1000,
  })

  const overview = overviewQuery.data
  const persistedConfig = useMemo(
    () => normalizeConfig(overview?.config),
    [overview?.config],
  )
  const [draft, setDraft] = useState<HAConfig>(DEFAULT_CONFIG)
  const [draftInitialized, setDraftInitialized] = useState(false)

  useEffect(() => {
    if (!draftInitialized && overview?.config) {
      setDraft(persistedConfig)
      setDraftInitialized(true)
    }
  }, [draftInitialized, overview?.config, persistedConfig])

  const updateMutation = useMutation({
    mutationFn: updateHAConfig,
    onSuccess: (res) => {
      if (!res.success || !res.data) {
        toast.error(res.message || t('Failed to update HA settings'))
        return
      }
      queryClient.setQueryData(['system-info', 'ha'], res.data)
      queryClient.invalidateQueries({ queryKey: ['system-options'] })
      setDraft(normalizeConfig(res.data.config))
      toast.success(t('Setting updated successfully'))
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to update HA settings'))
    },
  })

  const dirty = !sameConfig(draft, persistedConfig)
  const refreshing = overviewQuery.isFetching && !overviewQuery.isLoading

  function updateDraft<K extends keyof HAConfig>(key: K, value: HAConfig[K]) {
    setDraft((current) => ({ ...current, [key]: value }))
  }

  return (
    <Card className='rounded-lg shadow-xs'>
      <CardHeader className='border-b'>
        <div className='flex min-w-0 items-center gap-2'>
          <span className='bg-muted text-muted-foreground inline-flex size-7 items-center justify-center rounded-md'>
            <ShieldCheck className='size-4' aria-hidden='true' />
          </span>
          <div className='min-w-0'>
            <CardTitle className='text-sm'>{t('Primary/Standby')}</CardTitle>
            <CardDescription className='text-xs'>
              {t(
                'Configure the intended HA topology and compare it with live nodes.',
              )}
            </CardDescription>
          </div>
        </div>
        <CardAction className='flex items-center gap-2'>
          {overview && (
            <Badge
              variant={
                overview.summary === 'error' ? 'destructive' : 'secondary'
              }
              className={cn(SUMMARY_CLASS_NAME[overview.summary])}
            >
              {t(summaryLabel(overview.summary))}
            </Badge>
          )}
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={() => void overviewQuery.refetch()}
            disabled={overviewQuery.isFetching}
            aria-label={t('Refresh')}
          >
            <RefreshCw
              data-icon='inline-start'
              className={cn('size-3.5', refreshing && 'animate-spin')}
              aria-hidden='true'
            />
            {t('Refresh')}
          </Button>
        </CardAction>
      </CardHeader>

      <CardContent className='space-y-4'>
        {overviewQuery.isLoading ? (
          <div className='space-y-2'>
            <Skeleton className='h-9 w-full rounded-md' />
            <Skeleton className='h-32 w-full rounded-md' />
          </div>
        ) : overviewQuery.isError ? (
          <Alert variant='destructive'>
            <AlertTriangle className='size-4' aria-hidden='true' />
            <AlertTitle>{t('We could not load HA settings.')}</AlertTitle>
            <AlertDescription>
              {overviewQuery.error instanceof Error
                ? overviewQuery.error.message
                : undefined}
            </AlertDescription>
          </Alert>
        ) : (
          <>
            <div className='grid gap-3 lg:grid-cols-[1.2fr_0.8fr]'>
              <div className='space-y-3'>
                <div className='flex items-center justify-between rounded-md border px-3 py-2'>
                  <div className='min-w-0'>
                    <Label className='text-sm'>{t('Enable HA topology')}</Label>
                    <p className='text-muted-foreground mt-1 text-xs'>
                      {t('This enables diagnostics and expected-role checks.')}
                    </p>
                  </div>
                  <Switch
                    checked={draft.enabled}
                    onCheckedChange={(value) =>
                      updateDraft('enabled', Boolean(value))
                    }
                  />
                </div>

                <ConfigGroup icon={ShieldCheck} title={t('Topology')}>
                  <div className='grid gap-3 md:grid-cols-2'>
                    <Field
                      label={t('Primary node name')}
                      value={draft.primary_node_name}
                      placeholder='nbapi-main'
                      onChange={(value) =>
                        updateDraft('primary_node_name', value)
                      }
                    />
                    <Field
                      label={t('Standby node name')}
                      value={draft.standby_node_name}
                      placeholder='nbapi-backup'
                      onChange={(value) =>
                        updateDraft('standby_node_name', value)
                      }
                    />
                    <Field
                      label={t('Primary health URL')}
                      value={draft.primary_health_url}
                      placeholder='http://10.0.0.1:3000/api/ha/health'
                      onChange={(value) =>
                        updateDraft('primary_health_url', value)
                      }
                    />
                    <Field
                      label={t('Standby health URL')}
                      value={draft.standby_health_url}
                      placeholder='http://10.0.0.2:3000/api/ha/health'
                      onChange={(value) =>
                        updateDraft('standby_health_url', value)
                      }
                    />
                  </div>
                </ConfigGroup>

                <ConfigGroup icon={Globe2} title={t('Traffic and DNS')}>
                  <div className='grid gap-3 md:grid-cols-2'>
                    <Field
                      label={t('Public entry')}
                      value={draft.public_entry}
                      placeholder='api.example.com'
                      onChange={(value) => updateDraft('public_entry', value)}
                    />
                    <Field
                      label={t('Origin entry')}
                      value={draft.origin_entry}
                      placeholder='o-api.example.com'
                      onChange={(value) => updateDraft('origin_entry', value)}
                    />
                    <Field
                      label={t('Primary origin target')}
                      value={draft.primary_origin}
                      placeholder='main-tunnel.example.com'
                      onChange={(value) => updateDraft('primary_origin', value)}
                    />
                    <Field
                      label={t('Standby origin target')}
                      value={draft.standby_origin}
                      placeholder='backup-tunnel.example.com'
                      onChange={(value) => updateDraft('standby_origin', value)}
                    />
                    <SelectField
                      label={t('DNS provider')}
                      value={draft.dns_provider}
                      options={DNS_OPTIONS}
                      onChange={(value) => updateDraft('dns_provider', value)}
                    />
                    <Field
                      label={t('DNS record name')}
                      value={draft.dns_record_name}
                      placeholder='api.example.com'
                      onChange={(value) =>
                        updateDraft('dns_record_name', value)
                      }
                    />
                  </div>
                </ConfigGroup>

                <ConfigGroup icon={Database} title={t('Data layer')}>
                  <div className='grid gap-3 md:grid-cols-3'>
                    <SelectField
                      label={t('Database engine')}
                      value={draft.database_engine}
                      options={DATABASE_OPTIONS}
                      onChange={(value) =>
                        updateDraft('database_engine', value)
                      }
                    />
                    <SelectField
                      label={t('Replication mode')}
                      value={draft.replication_mode}
                      options={REPLICATION_OPTIONS}
                      onChange={(value) =>
                        updateDraft('replication_mode', value)
                      }
                    />
                    <SelectField
                      label={t('Redis mode')}
                      value={draft.redis_mode}
                      options={REDIS_OPTIONS}
                      onChange={(value) => updateDraft('redis_mode', value)}
                    />
                  </div>
                </ConfigGroup>

                <ConfigGroup icon={Route} title={t('Failover policy')}>
                  <div className='grid gap-3 md:grid-cols-3'>
                    <SelectField
                      label={t('Strategy')}
                      value={draft.failover_strategy}
                      options={FAILOVER_OPTIONS}
                      onChange={(value) =>
                        updateDraft('failover_strategy', value)
                      }
                    />
                    <NumberField
                      label={t('Health interval seconds')}
                      value={draft.health_check_interval_seconds}
                      min={5}
                      max={3600}
                      onChange={(value) =>
                        updateDraft('health_check_interval_seconds', value)
                      }
                    />
                    <NumberField
                      label={t('Failover threshold')}
                      value={draft.failover_threshold}
                      min={1}
                      max={100}
                      onChange={(value) =>
                        updateDraft('failover_threshold', value)
                      }
                    />
                  </div>
                  <div className='grid gap-3 md:grid-cols-2'>
                    <div className='space-y-1.5'>
                      <Label className='text-xs'>{t('Cutover runbook')}</Label>
                      <Textarea
                        value={draft.cutover_runbook}
                        onChange={(event) =>
                          updateDraft('cutover_runbook', event.target.value)
                        }
                        placeholder={t(
                          'Promote database, switch DNS, verify health, then reopen traffic.',
                        )}
                      />
                    </div>
                    <div className='space-y-1.5'>
                      <Label className='text-xs'>{t('Rollback runbook')}</Label>
                      <Textarea
                        value={draft.rollback_runbook}
                        onChange={(event) =>
                          updateDraft('rollback_runbook', event.target.value)
                        }
                        placeholder={t(
                          'Restore public entry, demote failed node, and resync standby.',
                        )}
                      />
                    </div>
                  </div>
                  <div className='space-y-1.5'>
                    <Label className='text-xs'>{t('Notes')}</Label>
                    <Textarea
                      value={draft.notes}
                      onChange={(event) =>
                        updateDraft('notes', event.target.value)
                      }
                      placeholder={t(
                        'Tunnel names, Cloudflare record, database host, or recovery reminders.',
                      )}
                    />
                  </div>
                </ConfigGroup>

                <div className='flex justify-end'>
                  <Button
                    type='button'
                    onClick={() => updateMutation.mutate(draft)}
                    disabled={!dirty || updateMutation.isPending}
                  >
                    <Save data-icon='inline-start' className='size-3.5' />
                    {updateMutation.isPending ? t('Saving...') : t('Save')}
                  </Button>
                </div>
              </div>

              <div className='space-y-3'>
                <div className='rounded-md border px-3 py-2'>
                  <div className='text-xs font-medium'>{t('Current node')}</div>
                  <div className='mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs'>
                    <span className='text-muted-foreground'>{t('Name')}</span>
                    <span className='font-mono'>
                      {overview?.current_node.name || '-'}
                    </span>
                    <span className='text-muted-foreground'>{t('Role')}</span>
                    <span className='font-mono'>
                      {overview?.current_node.is_master ? 'master' : 'worker'}
                    </span>
                    <span className='text-muted-foreground'>{t('Source')}</span>
                    <span className='font-mono'>
                      {overview?.current_node.source || '-'}
                    </span>
                  </div>
                </div>

                {overview?.snippets && (
                  <div className='space-y-2'>
                    <SnippetBlock
                      title={t('Primary .env')}
                      value={overview.snippets.primary_env}
                    />
                    <SnippetBlock
                      title={t('Standby .env')}
                      value={overview.snippets.standby_env}
                    />
                    <SnippetBlock
                      title={t('Compose mapping')}
                      value={overview.snippets.compose_env}
                    />
                    <SnippetBlock
                      title={t('Cutover checklist')}
                      value={overview.snippets.cutover_checklist}
                    />
                  </div>
                )}
              </div>
            </div>

            {overview?.checks && overview.checks.length > 0 && (
              <div className='space-y-2'>
                <div className='text-xs font-medium'>{t('Checks')}</div>
                <ul className='grid gap-2 md:grid-cols-2'>
                  {overview.checks.map((check) => (
                    <CheckRow
                      key={`${check.key}-${check.message}`}
                      check={check}
                    />
                  ))}
                </ul>
              </div>
            )}

            {overview?.probes && overview.probes.length > 0 && (
              <div className='space-y-2'>
                <div className='text-xs font-medium'>{t('Health probes')}</div>
                <ul className='grid gap-2 md:grid-cols-2'>
                  {overview.probes.map((probe) => (
                    <ProbeRow
                      key={`${probe.target}-${probe.url}`}
                      probe={probe}
                    />
                  ))}
                </ul>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}
