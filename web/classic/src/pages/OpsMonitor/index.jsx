import React, { useEffect, useMemo, useState } from 'react';
import { VChart } from '@visactor/react-vchart';
import { API } from '../../helpers/api';
import {
  Card,
  Col,
  Row,
  Select,
  Spin,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  Activity,
  AlertTriangle,
  Gauge,
  RadioTower,
  RefreshCw,
  Timer,
  Zap,
} from 'lucide-react';

const { Text, Title } = Typography;

const RANGE_OPTIONS = [
  { label: '近1小时', value: 3600 },
  { label: '近6小时', value: 21600 },
  { label: '近24小时', value: 86400 },
  { label: '近7天', value: 604800 },
];

const chartOption = {
  mode: 'desktop-browser',
  animation: false,
};

function formatNumber(value) {
  const num = Number(value || 0);
  if (num >= 100000000) return `${(num / 100000000).toFixed(2)}亿`;
  if (num >= 10000) return `${(num / 10000).toFixed(2)}万`;
  return num.toLocaleString();
}

function formatPercent(value) {
  return `${Number(value || 0).toFixed(3)}%`;
}

function formatTime(ts) {
  if (!ts) return '-';
  return new Date(ts * 1000).toLocaleString();
}

function lineSpec(data, fields) {
  const rows = [];
  data.forEach((point) => {
    fields.forEach((field) => {
      rows.push({
        time: new Date(point.ts * 1000).toLocaleTimeString([], {
          hour: '2-digit',
          minute: '2-digit',
        }),
        value: point[field.key] || 0,
        type: field.label,
      });
    });
  });
  return {
    type: 'line',
    data: [{ id: 'data', values: rows }],
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
    padding: { left: 10, right: 16, top: 8, bottom: 6 },
  };
}

function StatCard({ icon: Icon, title, value, sub, tone = 'blue' }) {
  const color =
    tone === 'green' ? '#16a34a' : tone === 'red' ? '#dc2626' : '#2563eb';
  return (
    <Card className='h-full rounded-lg border border-slate-200 shadow-sm'>
      <div className='flex items-start justify-between gap-3'>
        <div>
          <Text type='tertiary' size='small'>
            {title}
          </Text>
          <div className='mt-2 text-2xl font-bold text-slate-900'>{value}</div>
          {sub && <div className='mt-1 text-xs text-slate-500'>{sub}</div>}
        </div>
        <div
          className='flex h-9 w-9 items-center justify-center rounded-md'
          style={{ background: `${color}16`, color }}
        >
          <Icon size={18} />
        </div>
      </div>
    </Card>
  );
}

export default function OpsMonitor() {
  const [range, setRange] = useState(3600);
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState(null);

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/ops/monitor', {
        params: { range_seconds: range },
        disableDuplicate: true,
      });
      if (res.data?.success) {
        setData(res.data.data);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, [range]);

  const overview = data?.overview || {};
  const trend = data?.trend || [];
  const switchTrend = data?.channel_switch_trend || [];
  const switchChartData = switchTrend.map((item) => ({
    ts: item.ts,
    switches: item.switches,
  }));

  const columns = useMemo(
    () => [
      {
        title: '渠道',
        dataIndex: 'channel_name',
        render: (text, row) => text || `渠道 #${row.channel_id}`,
      },
      {
        title: '请求',
        dataIndex: 'requests',
        render: formatNumber,
      },
      {
        title: '错误',
        dataIndex: 'errors',
        render: (value) => (
          <Tag color={value > 0 ? 'red' : 'green'}>{formatNumber(value)}</Tag>
        ),
      },
      {
        title: 'SLA',
        dataIndex: 'sla',
        render: formatPercent,
      },
      {
        title: '平均时长',
        dataIndex: 'avg_latency_ms',
        render: (value) => `${formatNumber(value)} ms`,
      },
      {
        title: '首 token',
        dataIndex: 'avg_ttft_ms',
        render: (value) => `${formatNumber(value)} ms`,
      },
      {
        title: '渠道切换',
        dataIndex: 'channel_switches',
        render: formatNumber,
      },
    ],
    [],
  );

  const errorColumns = [
    { title: '时间', dataIndex: 'created_at', render: formatTime },
    {
      title: '渠道',
      dataIndex: 'channel_name',
      render: (text, row) => text || `渠道 #${row.channel_id}`,
    },
    { title: '模型', dataIndex: 'model_name' },
    { title: '状态', dataIndex: 'status_code' },
    { title: '错误码', dataIndex: 'error_code' },
    {
      title: '错误',
      dataIndex: 'content',
      render: (text) => (
        <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 360 }}>
          {text}
        </Text>
      ),
    },
  ];

  return (
    <div className='mt-[60px] px-4 pb-8'>
      <div className='mb-4 flex flex-wrap items-center justify-between gap-3'>
        <div>
          <Title heading={3} className='!mb-1'>
            运维监控
          </Title>
          <Text type='tertiary'>
            就绪 · 刷新：{data ? formatTime(data.updated_at) : '-'}
          </Text>
        </div>
        <div className='flex items-center gap-2'>
          <Select
            value={range}
            onChange={setRange}
            optionList={RANGE_OPTIONS}
          />
          <button
            className='inline-flex h-9 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 text-sm font-medium'
            onClick={fetchData}
          >
            <RefreshCw size={15} />
            刷新
          </button>
        </div>
      </div>

      <Spin spinning={loading}>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={8} xl={4}>
            <StatCard
              icon={Activity}
              title='请求数'
              value={formatNumber(overview.request_count)}
              sub={`Token ${formatNumber(overview.total_tokens)}`}
            />
          </Col>
          <Col xs={24} md={8} xl={4}>
            <StatCard
              icon={Gauge}
              title='平均 QPS / TPS'
              value={`${overview.avg_qps || 0} / ${overview.avg_tps || 0}`}
              sub='按所选时间段平均'
            />
          </Col>
          <Col xs={24} md={8} xl={4}>
            <StatCard
              icon={Zap}
              title='SLA'
              value={formatPercent(overview.sla)}
              sub={`排除业务限制 ${formatNumber(overview.excluded_errors)}`}
              tone='green'
            />
          </Col>
          <Col xs={24} md={8} xl={4}>
            <StatCard
              icon={AlertTriangle}
              title='请求错误'
              value={formatPercent(overview.error_rate)}
              sub={`错误 ${formatNumber(overview.error_count)}`}
              tone={overview.error_count > 0 ? 'red' : 'green'}
            />
          </Col>
          <Col xs={24} md={8} xl={4}>
            <StatCard
              icon={Timer}
              title='请求时长'
              value={`${formatNumber(overview.avg_latency_ms)} ms`}
              sub={`P99 ${formatNumber(overview.p99_latency_ms)} ms`}
            />
          </Col>
          <Col xs={24} md={8} xl={4}>
            <StatCard
              icon={RadioTower}
              title='首 token'
              value={`${formatNumber(overview.avg_ttft_ms)} ms`}
              sub={`P99 ${formatNumber(overview.p99_ttft_ms)} ms`}
            />
          </Col>
        </Row>

        <Row gutter={[16, 16]} className='mt-4'>
          <Col xs={24} xl={12}>
            <Card
              title='吞吐趋势'
              className='rounded-lg border border-slate-200'
            >
              <div style={{ height: 280 }}>
                <VChart
                  spec={lineSpec(trend, [
                    { key: 'qps', label: 'QPS' },
                    { key: 'tps', label: 'TPS' },
                  ])}
                  option={chartOption}
                />
              </div>
            </Card>
          </Col>
          <Col xs={24} xl={12}>
            <Card
              title='渠道切换趋势'
              className='rounded-lg border border-slate-200'
            >
              <div style={{ height: 280 }}>
                <VChart
                  spec={lineSpec(switchChartData, [
                    { key: 'switches', label: '渠道切换' },
                  ])}
                  option={chartOption}
                />
              </div>
            </Card>
          </Col>
        </Row>

        <Row gutter={[16, 16]} className='mt-4'>
          <Col xs={24} xl={12}>
            <Card
              title='渠道并发 / 排队'
              className='rounded-lg border border-slate-200'
            >
              <Table
                size='small'
                pagination={false}
                columns={columns}
                dataSource={data?.channels || []}
                rowKey='channel_id'
              />
            </Card>
          </Col>
          <Col xs={24} xl={12}>
            <Card
              title='上游错误'
              className='rounded-lg border border-slate-200'
            >
              <Table
                size='small'
                pagination={false}
                columns={errorColumns}
                dataSource={data?.errors || []}
                rowKey='id'
              />
            </Card>
          </Col>
        </Row>
      </Spin>
    </div>
  );
}
