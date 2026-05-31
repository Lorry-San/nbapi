import { useEffect, useMemo, useState } from 'react'
import { VChart } from '@visactor/react-vchart'
import {
  Activity,
  AlertTriangle,
  Gauge,
  RadioTower,
  RefreshCw,
  Timer,
  Zap,
} from 'lucide-react'
import { VCHART_OPTION } from '@/lib/vchart'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { getOpsMonitor } from './api'
import type { OpsMonitorData } from './types'

const RANGE_OPTIONS = [
  { label: '近1小时', value: '3600' },
  { label: '近6小时', value: '21600' },
  { label: '近24小时', value: '86400' },
  { label: '近7天', value: '604800' },
]

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

export function OpsMonitor() {
  const [range, setRange] = useState('3600')
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<OpsMonitorData | null>(null)

  const fetchData = async () => {
    setLoading(true)
    try {
      const res = await getOpsMonitor({ range_seconds: Number(range) })
      if (res.success && res.data) setData(res.data)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchData()
  }, [range])

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
    <div className='space-y-4 p-4 md:p-6'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div>
          <h1 className='text-xl font-semibold'>运维监控</h1>
          <div className='text-muted-foreground mt-1 text-sm'>
            就绪 · 刷新：{formatTime(data?.updated_at)}
          </div>
        </div>
        <div className='flex items-center gap-2'>
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
          <Button variant='outline' size='sm' onClick={() => void fetchData()}>
            <RefreshCw className={loading ? 'size-4 animate-spin' : 'size-4'} />
            刷新
          </Button>
        </div>
      </div>

      <div className='grid gap-4 sm:grid-cols-2 xl:grid-cols-6'>
        <StatCard
          icon={Activity}
          title='请求数'
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
          title='首 token'
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
          <CardContent>
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
          <CardContent>
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
