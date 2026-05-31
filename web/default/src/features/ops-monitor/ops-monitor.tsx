import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { VChart } from '@visactor/react-vchart'
import {
  Activity,
  AlertTriangle,
  Bell,
  Gauge,
  Plus,
  RadioTower,
  RefreshCw,
  Timer,
  Trash2,
  Zap,
} from 'lucide-react'
import { VCHART_OPTION } from '@/lib/vchart'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import {
  createOpsAlertRule,
  deleteOpsAlertRule,
  evaluateOpsAlerts,
  getOpsAlertEvents,
  getOpsAlertRules,
  getOpsMonitor,
} from './api'
import type {
  OpsAlertEvent,
  OpsAlertMetric,
  OpsAlertRule,
  OpsAlertRulePayload,
  OpsMonitorData,
} from './types'

const RANGE_OPTIONS = [
  { label: '近1小时', value: '3600' },
  { label: '近6小时', value: '21600' },
  { label: '近24小时', value: '86400' },
  { label: '近7天', value: '604800' },
]

const EVENT_RANGE_OPTIONS = [
  { label: '近24小时', value: '86400' },
  { label: '近7天', value: '604800' },
  { label: '近30天', value: '2592000' },
]

const LEVEL_OPTIONS = ['P0', 'P1', 'P2', 'P3']
const COMPARATOR_OPTIONS = ['>', '>=', '<', '<=']
const ALERT_RULE_INPUT_CLASS = 'bg-background text-foreground'
const ALERT_RULE_SELECT_TRIGGER_CLASS = 'w-full bg-background text-foreground'

const emptyRule: OpsAlertRulePayload = {
  name: '',
  description: '',
  metric: 'error_rate',
  comparator: '>',
  threshold: 20,
  level: 'P1',
  enabled: true,
  window_seconds: 300,
  duration_seconds: 300,
  cooldown_seconds: 1800,
  notify_email: true,
  scope: 'overall',
  channel_id: 0,
  model_name: '',
  group: '',
}

function formatNumber(value?: number) {
  const num = Number(value || 0)
  if (num >= 100000000) return `${(num / 100000000).toFixed(2)}亿`
  if (num >= 10000) return `${(num / 10000).toFixed(2)}万`
  return num.toLocaleString()
}

function formatPercent(value?: number) {
  return `${Number(value || 0).toFixed(3)}%`
}

function formatTime(ts?: number) {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString()
}

function formatDuration(seconds?: number) {
  const value = Number(seconds || 0)
  if (value <= 0) return '-'
  if (value % 3600 === 0) return `${value / 3600}h`
  if (value % 60 === 0) return `${value / 60}m`
  return `${value}s`
}

function buildLineSpec(
  points: unknown[],
  fields: Array<{ key: string; label: string }>
) {
  const values = points.flatMap((point) =>
    fields.map((field) => ({
      time: new Date(
        Number(readPointValue(point, 'ts')) * 1000
      ).toLocaleTimeString([], {
        hour: '2-digit',
        minute: '2-digit',
      }),
      value: Number(readPointValue(point, field.key) || 0),
      type: field.label,
    }))
  )

  return {
    type: 'line',
    data: [{ id: 'data', values }],
    xField: 'time',
    yField: 'value',
    seriesField: 'type',
    legends: { visible: true, orient: 'top', position: 'end' },
    axes: [
      { orient: 'bottom', label: { visible: true } },
      { orient: 'left', grid: { visible: true } },
    ],
    line: { style: { lineWidth: 2 } },
    point: { visible: false },
    tooltip: { visible: true },
    padding: { left: 8, right: 16, top: 8, bottom: 6 },
  }
}

function readPointValue(point: unknown, key: string): unknown {
  if (!point || typeof point !== 'object') return undefined
  return (point as Record<string, unknown>)[key]
}

function metricLabel(metrics: OpsAlertMetric[], key: string) {
  return metrics.find((item) => item.key === key)?.label || key
}

function StatCard(props: {
  icon: typeof Activity
  title: string
  value: string
  sub?: string
  tone?: 'blue' | 'green' | 'red'
}) {
  const Icon = props.icon
  const tone =
    props.tone === 'green'
      ? 'text-emerald-600 bg-emerald-500/10'
      : props.tone === 'red'
        ? 'text-red-600 bg-red-500/10'
        : 'text-blue-600 bg-blue-500/10'
  return (
    <Card>
      <CardContent className='flex items-start justify-between gap-3 p-4'>
        <div className='min-w-0'>
          <div className='text-muted-foreground text-xs font-medium'>
            {props.title}
          </div>
          <div className='mt-2 text-2xl font-semibold tabular-nums'>
            {props.value}
          </div>
          {props.sub ? (
            <div className='text-muted-foreground mt-1 text-xs'>
              {props.sub}
            </div>
          ) : null}
        </div>
        <div
          className={`flex size-9 shrink-0 items-center justify-center rounded-md ${tone}`}
        >
          <Icon className='size-4' />
        </div>
      </CardContent>
    </Card>
  )
}

function LevelBadge({ level }: { level: string }) {
  const className =
    level === 'P0'
      ? 'bg-red-100 text-red-700'
      : level === 'P1'
        ? 'bg-amber-100 text-amber-700'
        : 'bg-blue-100 text-blue-700'
  return <Badge className={className}>{level}</Badge>
}

function AlertRuleDialog(props: {
  open: boolean
  metrics: OpsAlertMetric[]
  onOpenChange: (open: boolean) => void
  onSaved: () => void
}) {
  const [form, setForm] = useState<OpsAlertRulePayload>(emptyRule)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (props.open) setForm(emptyRule)
  }, [props.open])

  const update = <K extends keyof OpsAlertRulePayload>(
    key: K,
    value: OpsAlertRulePayload[K]
  ) => setForm((prev) => ({ ...prev, [key]: value }))

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    try {
      const res = await createOpsAlertRule(form)
      if (res.success) {
        props.onSaved()
        props.onOpenChange(false)
      }
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[92vh] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>新建告警规则</DialogTitle>
          <DialogDescription>
            规则由后台每分钟评估，触发和恢复都会记录事件并按配置发送邮件。
          </DialogDescription>
        </DialogHeader>
        <form className='grid gap-4' onSubmit={submit}>
          <div className='grid gap-3 md:grid-cols-2'>
            <div className='space-y-2'>
              <Label>名称</Label>
              <Input
                className={ALERT_RULE_INPUT_CLASS}
                value={form.name}
                onChange={(event) => update('name', event.target.value)}
                required
              />
            </div>
            <div className='space-y-2'>
              <Label>级别</Label>
              <Select
                value={form.level}
                onValueChange={(value) => {
                  if (value) update('level', value)
                }}
              >
                <SelectTrigger className={ALERT_RULE_SELECT_TRIGGER_CLASS}>
                  <SelectValue>{form.level}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {LEVEL_OPTIONS.map((item) => (
                    <SelectItem key={item} value={item}>
                      {item}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className='grid gap-3 md:grid-cols-[1.4fr_.7fr_1fr]'>
            <div className='space-y-2'>
              <Label>指标</Label>
              <Select
                value={form.metric}
                onValueChange={(value) => {
                  if (!value) return
                  const metric = props.metrics.find((item) => item.key === value)
                  update('metric', value)
                  if (metric?.default_comparator) {
                    update('comparator', metric.default_comparator)
                  }
                }}
              >
                <SelectTrigger className={ALERT_RULE_SELECT_TRIGGER_CLASS}>
                  <SelectValue>{metricLabel(props.metrics, form.metric)}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {props.metrics.map((item) => (
                    <SelectItem key={item.key} value={item.key}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className='space-y-2'>
              <Label>比较</Label>
              <Select
                value={form.comparator}
                onValueChange={(value) => {
                  if (value) update('comparator', value)
                }}
              >
                <SelectTrigger className={ALERT_RULE_SELECT_TRIGGER_CLASS}>
                  <SelectValue>{form.comparator}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {COMPARATOR_OPTIONS.map((item) => (
                    <SelectItem key={item} value={item}>
                      {item}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className='space-y-2'>
              <Label>阈值</Label>
              <Input
                className={ALERT_RULE_INPUT_CLASS}
                type='number'
                step='0.01'
                value={form.threshold}
                onChange={(event) =>
                  update('threshold', Number(event.target.value))
                }
              />
            </div>
          </div>

          <div className='grid gap-3 md:grid-cols-3'>
            <div className='space-y-2'>
              <Label>统计窗口（秒）</Label>
              <Input
                className={ALERT_RULE_INPUT_CLASS}
                type='number'
                value={form.window_seconds}
                onChange={(event) =>
                  update('window_seconds', Number(event.target.value))
                }
              />
            </div>
            <div className='space-y-2'>
              <Label>持续时间（秒）</Label>
              <Input
                className={ALERT_RULE_INPUT_CLASS}
                type='number'
                value={form.duration_seconds}
                onChange={(event) =>
                  update('duration_seconds', Number(event.target.value))
                }
              />
            </div>
            <div className='space-y-2'>
              <Label>冷却时间（秒）</Label>
              <Input
                className={ALERT_RULE_INPUT_CLASS}
                type='number'
                value={form.cooldown_seconds}
                onChange={(event) =>
                  update('cooldown_seconds', Number(event.target.value))
                }
              />
            </div>
          </div>

          <div className='grid gap-3 md:grid-cols-3'>
            <div className='space-y-2'>
              <Label>渠道 ID</Label>
              <Input
                className={ALERT_RULE_INPUT_CLASS}
                type='number'
                value={form.channel_id}
                onChange={(event) =>
                  update('channel_id', Number(event.target.value))
                }
              />
            </div>
            <div className='space-y-2'>
              <Label>模型</Label>
              <Input
                className={ALERT_RULE_INPUT_CLASS}
                value={form.model_name}
                onChange={(event) => update('model_name', event.target.value)}
                placeholder='留空为全部'
              />
            </div>
            <div className='space-y-2'>
              <Label>分组</Label>
              <Input
                className={ALERT_RULE_INPUT_CLASS}
                value={form.group}
                onChange={(event) => update('group', event.target.value)}
                placeholder='留空为全部'
              />
            </div>
          </div>

          <div className='space-y-2'>
            <Label>描述</Label>
            <Textarea
              className={ALERT_RULE_INPUT_CLASS}
              value={form.description}
              onChange={(event) => update('description', event.target.value)}
            />
          </div>

          <div className='flex flex-wrap gap-6 rounded-lg border p-3'>
            <label className='flex items-center gap-2 text-sm'>
              <Switch
                checked={form.enabled}
                onCheckedChange={(value) => update('enabled', Boolean(value))}
              />
              启用规则
            </label>
            <label className='flex items-center gap-2 text-sm'>
              <Switch
                checked={form.notify_email}
                onCheckedChange={(value) =>
                  update('notify_email', Boolean(value))
                }
              />
              邮件通知管理员/超管
            </label>
          </div>

          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => props.onOpenChange(false)}
            >
              取消
            </Button>
            <Button type='submit' disabled={saving}>
              {saving ? '保存中...' : '保存'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function OverviewTab({ data }: { data: OpsMonitorData | null }) {
  const overview = data?.overview
  const throughputSpec = useMemo(
    () =>
      buildLineSpec(data?.trend ?? [], [
        { key: 'qps', label: 'QPS' },
        { key: 'tps', label: 'TPS' },
      ]),
    [data?.trend]
  )
  const switchSpec = useMemo(
    () =>
      buildLineSpec(data?.channel_switch_trend ?? [], [
        { key: 'switches', label: '渠道切换' },
      ]),
    [data?.channel_switch_trend]
  )

  return (
    <div className='space-y-4'>
      <div className='grid gap-4 sm:grid-cols-2 xl:grid-cols-6'>
        <StatCard
          icon={Activity}
          title='请求数量'
          value={formatNumber(overview?.request_count)}
          sub={`Token ${formatNumber(overview?.total_tokens)}`}
        />
        <StatCard
          icon={Gauge}
          title='平均 QPS / TPS'
          value={`${overview?.avg_qps ?? 0} / ${overview?.avg_tps ?? 0}`}
          sub='按所选时间段平均'
        />
        <StatCard
          icon={Zap}
          title='SLA'
          value={formatPercent(overview?.sla)}
          sub={`排除业务限制 ${formatNumber(overview?.excluded_errors)}`}
          tone='green'
        />
        <StatCard
          icon={AlertTriangle}
          title='请求错误'
          value={formatPercent(overview?.error_rate)}
          sub={`错误 ${formatNumber(overview?.error_count)}`}
          tone={(overview?.error_count ?? 0) > 0 ? 'red' : 'green'}
        />
        <StatCard
          icon={Timer}
          title='请求时长'
          value={`${formatNumber(overview?.avg_latency_ms)} ms`}
          sub={`P99 ${formatNumber(overview?.p99_latency_ms)} ms`}
        />
        <StatCard
          icon={RadioTower}
          title='首 Token'
          value={`${formatNumber(overview?.avg_ttft_ms)} ms`}
          sub={`P99 ${formatNumber(overview?.p99_ttft_ms)} ms`}
        />
      </div>

      <div className='grid gap-4 xl:grid-cols-2'>
        <Card>
          <CardHeader>
            <CardTitle>吞吐趋势</CardTitle>
          </CardHeader>
          <CardContent>
            <div className='h-72'>
              <VChart spec={throughputSpec} option={VCHART_OPTION} />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>渠道切换趋势</CardTitle>
          </CardHeader>
          <CardContent>
            <div className='h-72'>
              <VChart spec={switchSpec} option={VCHART_OPTION} />
            </div>
          </CardContent>
        </Card>
      </div>

      <div className='grid gap-4 xl:grid-cols-2'>
        <Card>
          <CardHeader>
            <CardTitle>渠道并发 / 排队</CardTitle>
          </CardHeader>
          <CardContent className='overflow-x-auto'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>渠道</TableHead>
                  <TableHead>请求</TableHead>
                  <TableHead>错误</TableHead>
                  <TableHead>SLA</TableHead>
                  <TableHead>平均时长</TableHead>
                  <TableHead>渠道切换</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(data?.channels ?? []).map((channel) => (
                  <TableRow key={channel.channel_id}>
                    <TableCell>
                      {channel.channel_name || `渠道 #${channel.channel_id}`}
                    </TableCell>
                    <TableCell>{formatNumber(channel.requests)}</TableCell>
                    <TableCell>
                      <Badge
                        variant={channel.errors > 0 ? 'destructive' : 'outline'}
                      >
                        {formatNumber(channel.errors)}
                      </Badge>
                    </TableCell>
                    <TableCell>{formatPercent(channel.sla)}</TableCell>
                    <TableCell>
                      {formatNumber(channel.avg_latency_ms)} ms
                    </TableCell>
                    <TableCell>
                      {formatNumber(channel.channel_switches)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>上游错误</CardTitle>
          </CardHeader>
          <CardContent className='overflow-x-auto'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>时间</TableHead>
                  <TableHead>渠道</TableHead>
                  <TableHead>模型</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>错误</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(data?.errors ?? []).map((error) => (
                  <TableRow key={error.id}>
                    <TableCell>{formatTime(error.created_at)}</TableCell>
                    <TableCell>
                      {error.channel_name || `渠道 #${error.channel_id}`}
                    </TableCell>
                    <TableCell>{error.model_name || '-'}</TableCell>
                    <TableCell>{error.status_code || '-'}</TableCell>
                    <TableCell className='max-w-[280px] truncate'>
                      {error.content || error.error_code || '-'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function RulesTab(props: {
  rules: OpsAlertRule[]
  metrics: OpsAlertMetric[]
  onCreate: () => void
  onDelete: (rule: OpsAlertRule) => void
}) {
  return (
    <Card>
      <CardHeader className='flex flex-row items-center justify-between gap-3'>
        <div>
          <CardTitle>告警规则</CardTitle>
          <div className='text-muted-foreground mt-1 text-sm'>
            创建与管理系统阈值告警，仅邮件通知管理员和超管
          </div>
        </div>
        <Button onClick={props.onCreate}>
          <Plus className='size-4' />
          新建规则
        </Button>
      </CardHeader>
      <CardContent>
        <div className='grid gap-3 lg:grid-cols-2'>
          {props.rules.map((rule) => (
            <div
              key={rule.id}
              className='rounded-lg border bg-muted/20 p-4 transition-colors hover:bg-muted/30'
            >
              <div className='flex items-start justify-between gap-3'>
                <div className='min-w-0'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <div className='font-medium'>{rule.name}</div>
                    <LevelBadge level={rule.level} />
                    <Badge
                      variant={
                        rule.last_state === 'firing' ? 'destructive' : 'outline'
                      }
                    >
                      {rule.last_state === 'firing' ? '告警中' : '正常'}
                    </Badge>
                  </div>
                  <div className='text-muted-foreground mt-1 text-xs'>
                    {rule.description || rule.last_message || '-'}
                  </div>
                </div>
                <Button
                  variant='destructive'
                  size='icon-sm'
                  onClick={() => props.onDelete(rule)}
                  title='删除'
                >
                  <Trash2 className='size-4' />
                </Button>
              </div>
              <div className='mt-4 grid gap-3 text-sm sm:grid-cols-3'>
                <div>
                  <div className='text-muted-foreground text-xs'>指标</div>
                  <div className='mt-1 font-mono text-xs'>
                    {rule.metric} {rule.comparator} {rule.threshold}
                  </div>
                  <div className='text-muted-foreground mt-1 text-xs'>
                    {metricLabel(props.metrics, rule.metric)}
                  </div>
                </div>
                <div>
                  <div className='text-muted-foreground text-xs'>持续/窗口</div>
                  <div className='mt-1'>
                    {formatDuration(rule.duration_seconds)} /{' '}
                    {formatDuration(rule.window_seconds)}
                  </div>
                </div>
                <div>
                  <div className='text-muted-foreground text-xs'>通知</div>
                  <div className='mt-1'>
                    {rule.enabled ? '已启用' : '已禁用'} ·{' '}
                    {rule.notify_email ? '邮件发送' : '邮件关闭'}
                  </div>
                </div>
              </div>
              <div className='text-muted-foreground mt-3 text-xs'>
                更新时间 {formatTime(rule.updated_at)}
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function EventsTab({
  events,
  metrics,
}: {
  events: OpsAlertEvent[]
  metrics: OpsAlertMetric[]
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>告警事件</CardTitle>
        <div className='text-muted-foreground mt-1 text-sm'>
          最近的告警触发和恢复记录
        </div>
      </CardHeader>
      <CardContent className='overflow-x-auto'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              <TableHead>级别</TableHead>
              <TableHead>规则ID</TableHead>
              <TableHead>标题</TableHead>
              <TableHead>持续时间</TableHead>
              <TableHead>维度</TableHead>
              <TableHead>邮件已发送</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {events.map((event) => (
              <TableRow key={event.id}>
                <TableCell>{formatTime(event.created_at)}</TableCell>
                <TableCell>
                  <div className='flex items-center gap-2'>
                    <LevelBadge level={event.level} />
                    <Badge
                      variant={
                        event.status === 'firing' ? 'destructive' : 'outline'
                      }
                    >
                      {event.status === 'firing' ? '触发' : '恢复'}
                    </Badge>
                  </div>
                </TableCell>
                <TableCell>#{event.rule_id}</TableCell>
                <TableCell className='min-w-[360px]'>
                  <div className='font-medium'>{event.title}</div>
                  <div className='text-muted-foreground text-xs'>
                    {metricLabel(metrics, event.metric)} {event.comparator}{' '}
                    {event.threshold}，当前 {event.current_value}
                  </div>
                  <div className='text-muted-foreground mt-1 text-xs'>
                    {event.message}
                  </div>
                </TableCell>
                <TableCell>{formatDuration(event.duration_seconds)}</TableCell>
                <TableCell>
                  {event.channel_name ||
                    (event.channel_id > 0 ? `渠道 #${event.channel_id}` : '-')}
                </TableCell>
                <TableCell>
                  {event.email_sent ? (
                    <span className='text-emerald-600'>已发送</span>
                  ) : (
                    <span className='text-muted-foreground'>
                      {event.email_error || '已忽略'}
                    </span>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}

export function OpsMonitor() {
  const [range, setRange] = useState('3600')
  const [eventRange, setEventRange] = useState('604800')
  const [eventLevel, setEventLevel] = useState('all')
  const [eventStatus, setEventStatus] = useState('all')
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<OpsMonitorData | null>(null)
  const [rules, setRules] = useState<OpsAlertRule[]>([])
  const [metrics, setMetrics] = useState<OpsAlertMetric[]>([])
  const [events, setEvents] = useState<OpsAlertEvent[]>([])
  const [dialogOpen, setDialogOpen] = useState(false)

  const fetchMonitor = async () => {
    const res = await getOpsMonitor({ range_seconds: Number(range) })
    if (res.success && res.data) setData(res.data)
  }

  const fetchRules = async () => {
    const res = await getOpsAlertRules()
    if (res.success && res.data) {
      setRules(res.data.rules)
      setMetrics(res.data.metrics)
    }
  }

  const fetchEvents = async () => {
    const res = await getOpsAlertEvents({
      range_seconds: Number(eventRange),
      level: eventLevel,
      status: eventStatus,
      limit: 100,
    })
    if (res.success && res.data) setEvents(res.data)
  }

  const refreshAll = async () => {
    setLoading(true)
    try {
      await Promise.all([fetchMonitor(), fetchRules(), fetchEvents()])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void refreshAll()
  }, [range, eventRange, eventLevel, eventStatus])

  const createRule = () => {
    setDialogOpen(true)
  }

  const removeRule = async (rule: OpsAlertRule) => {
    if (!window.confirm(`删除告警规则「${rule.name}」？`)) return
    const res = await deleteOpsAlertRule(rule.id)
    if (res.success) await refreshAll()
  }

  const runEvaluation = async () => {
    setLoading(true)
    try {
      await evaluateOpsAlerts()
      await refreshAll()
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className='min-h-0 flex-1 overflow-y-auto'>
      <div className='space-y-4 p-4 pb-8 md:p-6 md:pb-10'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div>
            <h1 className='text-xl font-semibold'>运维监控</h1>
            <div className='text-muted-foreground mt-1 text-sm'>
              就绪 · 刷新：{formatTime(data?.updated_at)}
            </div>
          </div>
          <div className='flex flex-wrap items-center gap-2'>
            <Select
              value={range}
              onValueChange={(value) => value && setRange(value)}
            >
              <SelectTrigger className='w-32'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {RANGE_OPTIONS.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button variant='outline' size='sm' onClick={runEvaluation}>
              <Bell className='size-4' />
              评估告警
            </Button>
            <Button variant='outline' size='sm' onClick={() => void refreshAll()}>
              <RefreshCw className={loading ? 'size-4 animate-spin' : 'size-4'} />
              刷新
            </Button>
          </div>
        </div>

        <Tabs defaultValue='overview'>
          <div className='flex flex-wrap items-center justify-between gap-3'>
            <TabsList>
              <TabsTrigger value='overview'>概览</TabsTrigger>
              <TabsTrigger value='rules'>告警规则</TabsTrigger>
              <TabsTrigger value='events'>告警事件</TabsTrigger>
            </TabsList>
            <div className='flex flex-wrap items-center gap-2'>
              <Select
                value={eventRange}
                onValueChange={(value) => value && setEventRange(value)}
              >
                <SelectTrigger className='w-32'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {EVENT_RANGE_OPTIONS.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Select
                value={eventLevel}
                onValueChange={(value) => value && setEventLevel(value)}
              >
                <SelectTrigger className='w-24'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='all'>全部</SelectItem>
                  {LEVEL_OPTIONS.map((item) => (
                    <SelectItem key={item} value={item}>
                      {item}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Select
                value={eventStatus}
                onValueChange={(value) => value && setEventStatus(value)}
              >
                <SelectTrigger className='w-28'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='all'>全部</SelectItem>
                  <SelectItem value='firing'>触发</SelectItem>
                  <SelectItem value='resolved'>恢复</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <TabsContent value='overview'>
            <OverviewTab data={data} />
          </TabsContent>
          <TabsContent value='rules'>
            <RulesTab
              rules={rules}
              metrics={metrics}
              onCreate={createRule}
              onDelete={removeRule}
            />
          </TabsContent>
          <TabsContent value='events'>
            <EventsTab events={events} metrics={metrics} />
          </TabsContent>
        </Tabs>

        <AlertRuleDialog
          open={dialogOpen}
          metrics={metrics}
          onOpenChange={setDialogOpen}
          onSaved={refreshAll}
        />
      </div>
    </div>
  )
}
