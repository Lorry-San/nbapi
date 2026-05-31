import React, { useEffect, useMemo, useState } from 'react';
import { VChart } from '@visactor/react-vchart';
import { API } from '../../helpers/api';
import {
  Button,
  Card,
  Col,
  Form,
  Modal,
  Popconfirm,
  Row,
  Select,
  Spin,
  Switch,
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
  Trash2,
  Zap,
} from 'lucide-react';

const { Text, Title } = Typography;

const RANGE_OPTIONS = [
  { label: '近1小时', value: 3600 },
  { label: '近6小时', value: 21600 },
  { label: '近24小时', value: 86400 },
  { label: '近7天', value: 604800 },
];

const EVENT_RANGE_OPTIONS = [
  { label: '近24小时', value: 86400 },
  { label: '近7天', value: 604800 },
  { label: '近30天', value: 2592000 },
];

const LEVEL_OPTIONS = ['P0', 'P1', 'P2', 'P3'];
const COMPARATOR_OPTIONS = ['>', '>=', '<', '<='];

const defaultRuleForm = {
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
};

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

function formatDuration(seconds) {
  const value = Number(seconds || 0);
  if (value <= 0) return '-';
  if (value % 3600 === 0) return `${value / 3600}h`;
  if (value % 60 === 0) return `${value / 60}m`;
  return `${value}s`;
}

function metricLabel(metrics, key) {
  return metrics.find((item) => item.key === key)?.label || key;
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
  const [eventRange, setEventRange] = useState(604800);
  const [eventLevel, setEventLevel] = useState('all');
  const [eventStatus, setEventStatus] = useState('all');
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState(null);
  const [rules, setRules] = useState([]);
  const [metrics, setMetrics] = useState([]);
  const [events, setEvents] = useState([]);
  const [ruleModalVisible, setRuleModalVisible] = useState(false);
  const [ruleForm, setRuleForm] = useState(defaultRuleForm);

  const fetchData = async () => {
    const res = await API.get('/api/ops/monitor', {
      params: { range_seconds: range },
      disableDuplicate: true,
    });
    if (res.data?.success) {
      setData(res.data.data);
    }
  };

  const fetchRules = async () => {
    const res = await API.get('/api/ops/alerts/rules', {
      disableDuplicate: true,
    });
    if (res.data?.success) {
      setRules(res.data.data.rules || []);
      setMetrics(res.data.data.metrics || []);
    }
  };

  const fetchEvents = async () => {
    const res = await API.get('/api/ops/alerts/events', {
      params: {
        range_seconds: eventRange,
        level: eventLevel,
        status: eventStatus,
        limit: 100,
      },
      disableDuplicate: true,
    });
    if (res.data?.success) {
      setEvents(res.data.data || []);
    }
  };

  const refreshAll = async () => {
    setLoading(true);
    try {
      await Promise.all([fetchData(), fetchRules(), fetchEvents()]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refreshAll();
  }, [range, eventRange, eventLevel, eventStatus]);
  const openRuleModal = () => {
    setRuleForm(defaultRuleForm);
    setRuleModalVisible(true);
  };

  const saveRule = async () => {
    const res = await API.post('/api/ops/alerts/rules', ruleForm);
    if (res.data?.success) {
      setRuleModalVisible(false);
      await refreshAll();
    }
  };

  const deleteRule = async (rule) => {
    const res = await API.delete(`/api/ops/alerts/rules/${rule.id}`);
    if (res.data?.success) {
      await refreshAll();
    }
  };

  const runEvaluation = async () => {
    setLoading(true);
    try {
      await API.post('/api/ops/alerts/evaluate');
      await refreshAll();
    } finally {
      setLoading(false);
    }
  };

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
        title: '首 Token',
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

  const ruleColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      render: (text, row) => (
        <div>
          <div className='font-semibold text-slate-900'>{text}</div>
          <div className='text-xs text-slate-500'>
            {row.description || row.last_message || '-'}
          </div>
          <div className='mt-1 text-xs text-slate-400'>
            {formatTime(row.updated_at)}
          </div>
        </div>
      ),
    },
    {
      title: '指标',
      dataIndex: 'metric',
      render: (text, row) => (
        <div>
          <code>
            {text} {row.comparator} {row.threshold}
          </code>
          <div className='text-xs text-slate-500'>
            {metricLabel(metrics, text)} / {formatDuration(row.duration_seconds)}
          </div>
        </div>
      ),
    },
    {
      title: '级别',
      dataIndex: 'level',
      render: (value) => (
        <Tag color={value === 'P0' ? 'red' : value === 'P1' ? 'orange' : 'blue'}>
          {value}
        </Tag>
      ),
    },
    {
      title: '启用',
      dataIndex: 'enabled',
      render: (value) => (value ? '已启用' : '已禁用'),
    },
    {
      title: '邮件',
      dataIndex: 'notify_email',
      render: (value) => (value ? '发送' : '关闭'),
    },
    {
      title: '状态',
      dataIndex: 'last_state',
      render: (value) => (
        <Tag color={value === 'firing' ? 'red' : 'green'}>
          {value === 'firing' ? '告警中' : '正常'}
        </Tag>
      ),
    },
    {
      title: '操作',
      dataIndex: 'operate',
      render: (_, row) => (
        <div className='flex gap-2'>
          <Popconfirm
            title={`删除告警规则「${row.name}」？`}
            onConfirm={() => deleteRule(row)}
          >
            <Button size='small' theme='solid' type='danger' icon={<Trash2 size={14} />}>
              删除
            </Button>
          </Popconfirm>
        </div>
      ),
    },
  ];

  const eventColumns = [
    { title: '时间', dataIndex: 'created_at', render: formatTime },
    {
      title: '级别',
      dataIndex: 'level',
      render: (value, row) => (
        <div className='flex items-center gap-2'>
          <Tag color={value === 'P0' ? 'red' : value === 'P1' ? 'orange' : 'blue'}>
            {value}
          </Tag>
          <Tag color={row.status === 'firing' ? 'red' : 'green'}>
            {row.status === 'firing' ? '触发' : '恢复'}
          </Tag>
        </div>
      ),
    },
    { title: '规则ID', dataIndex: 'rule_id', render: (value) => `#${value}` },
    {
      title: '标题',
      dataIndex: 'title',
      render: (text, row) => (
        <div>
          <div className='font-semibold text-slate-900'>{text}</div>
          <div className='text-xs text-slate-500'>
            {metricLabel(metrics, row.metric)} {row.comparator} {row.threshold}
            ，当前 {row.current_value}
          </div>
          <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 520 }}>
            {row.message}
          </Text>
        </div>
      ),
    },
    {
      title: '持续时间',
      dataIndex: 'duration_seconds',
      render: formatDuration,
    },
    {
      title: '维度',
      dataIndex: 'channel_name',
      render: (text, row) =>
        text || (row.channel_id > 0 ? `渠道 #${row.channel_id}` : '-'),
    },
    {
      title: '邮件已发送',
      dataIndex: 'email_sent',
      render: (value, row) =>
        value ? (
          <span className='text-emerald-600'>已发送</span>
        ) : (
          <span className='text-slate-500'>{row.email_error || '已忽略'}</span>
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
          <Select value={range} onChange={setRange} optionList={RANGE_OPTIONS} />
          <Button onClick={runEvaluation}>评估告警</Button>
          <Button icon={<RefreshCw size={15} />} onClick={refreshAll}>
            刷新
          </Button>
        </div>
      </div>

      <Spin spinning={loading}>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={8} xl={4}>
            <StatCard
              icon={Activity}
              title='请求数量'
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
              title='首 Token'
              value={`${formatNumber(overview.avg_ttft_ms)} ms`}
              sub={`P99 ${formatNumber(overview.p99_ttft_ms)} ms`}
            />
          </Col>
        </Row>

        <Row gutter={[16, 16]} className='mt-4'>
          <Col xs={24} xl={12}>
            <Card title='吞吐趋势' className='rounded-lg border border-slate-200'>
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
            <Card title='渠道切换趋势' className='rounded-lg border border-slate-200'>
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
            <Card title='上游错误' className='rounded-lg border border-slate-200'>
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

        <Card
          title={
            <div className='flex items-center justify-between'>
              <div>
                <div>告警规则</div>
                <div className='text-xs font-normal text-slate-500'>
                  创建与管理系统阈值告警，仅邮件通知管理员和超管
                </div>
              </div>
              <Button theme='solid' type='primary' onClick={() => openRuleModal()}>
                新建规则
              </Button>
            </div>
          }
          className='mt-4 rounded-lg border border-slate-200'
        >
          <Table
            size='small'
            pagination={false}
            columns={ruleColumns}
            dataSource={rules}
            rowKey='id'
          />
        </Card>

        <Card
          title={
            <div className='flex flex-wrap items-center justify-between gap-3'>
              <div>
                <div>告警事件</div>
                <div className='text-xs font-normal text-slate-500'>
                  最近的告警触发和恢复记录
                </div>
              </div>
              <div className='flex items-center gap-2'>
                <Select
                  value={eventRange}
                  optionList={EVENT_RANGE_OPTIONS}
                  onChange={setEventRange}
                  style={{ width: 120 }}
                />
                <Select
                  value={eventLevel}
                  optionList={[
                    { label: '全部', value: 'all' },
                    ...LEVEL_OPTIONS.map((item) => ({ label: item, value: item })),
                  ]}
                  onChange={setEventLevel}
                  style={{ width: 100 }}
                />
                <Select
                  value={eventStatus}
                  optionList={[
                    { label: '全部', value: 'all' },
                    { label: '触发', value: 'firing' },
                    { label: '恢复', value: 'resolved' },
                  ]}
                  onChange={setEventStatus}
                  style={{ width: 100 }}
                />
              </div>
            </div>
          }
          className='mt-4 rounded-lg border border-slate-200'
        >
          <Table
            size='small'
            pagination={false}
            columns={eventColumns}
            dataSource={events}
            rowKey='id'
          />
        </Card>
      </Spin>

      <Modal
        title='新建告警规则'
        visible={ruleModalVisible}
        onOk={saveRule}
        onCancel={() => setRuleModalVisible(false)}
        okText='保存'
        cancelText='取消'
        style={{ width: 720 }}
      >
        <Form labelPosition='top'>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Input
                label='名称'
                value={ruleForm.name}
                onChange={(value) => setRuleForm({ ...ruleForm, name: value })}
              />
            </Col>
            <Col span={12}>
              <Form.Select
                label='级别'
                value={ruleForm.level}
                onChange={(value) => setRuleForm({ ...ruleForm, level: value })}
              >
                {LEVEL_OPTIONS.map((item) => (
                  <Form.Select.Option key={item} value={item}>
                    {item}
                  </Form.Select.Option>
                ))}
              </Form.Select>
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Select
                label='指标'
                value={ruleForm.metric}
                onChange={(value) => {
                  const metric = metrics.find((item) => item.key === value);
                  setRuleForm({
                    ...ruleForm,
                    metric: value,
                    comparator: metric?.default_comparator || ruleForm.comparator,
                  });
                }}
              >
                {metrics.map((item) => (
                  <Form.Select.Option key={item.key} value={item.key}>
                    {item.label}
                  </Form.Select.Option>
                ))}
              </Form.Select>
            </Col>
            <Col span={6}>
              <Form.Select
                label='比较'
                value={ruleForm.comparator}
                onChange={(value) =>
                  setRuleForm({ ...ruleForm, comparator: value })
                }
              >
                {COMPARATOR_OPTIONS.map((item) => (
                  <Form.Select.Option key={item} value={item}>
                    {item}
                  </Form.Select.Option>
                ))}
              </Form.Select>
            </Col>
            <Col span={6}>
              <Form.InputNumber
                label='阈值'
                value={ruleForm.threshold}
                onChange={(value) =>
                  setRuleForm({ ...ruleForm, threshold: Number(value) })
                }
              />
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={8}>
              <Form.InputNumber
                label='统计窗口（秒）'
                value={ruleForm.window_seconds}
                onChange={(value) =>
                  setRuleForm({ ...ruleForm, window_seconds: Number(value) })
                }
              />
            </Col>
            <Col span={8}>
              <Form.InputNumber
                label='持续时间（秒）'
                value={ruleForm.duration_seconds}
                onChange={(value) =>
                  setRuleForm({ ...ruleForm, duration_seconds: Number(value) })
                }
              />
            </Col>
            <Col span={8}>
              <Form.InputNumber
                label='冷却时间（秒）'
                value={ruleForm.cooldown_seconds}
                onChange={(value) =>
                  setRuleForm({ ...ruleForm, cooldown_seconds: Number(value) })
                }
              />
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={8}>
              <Form.InputNumber
                label='渠道 ID'
                value={ruleForm.channel_id}
                onChange={(value) =>
                  setRuleForm({ ...ruleForm, channel_id: Number(value) })
                }
              />
            </Col>
            <Col span={8}>
              <Form.Input
                label='模型'
                value={ruleForm.model_name}
                placeholder='留空为全部'
                onChange={(value) =>
                  setRuleForm({ ...ruleForm, model_name: value })
                }
              />
            </Col>
            <Col span={8}>
              <Form.Input
                label='分组'
                value={ruleForm.group}
                placeholder='留空为全部'
                onChange={(value) => setRuleForm({ ...ruleForm, group: value })}
              />
            </Col>
          </Row>
          <Form.TextArea
            label='描述'
            value={ruleForm.description}
            autosize={{ minRows: 2, maxRows: 4 }}
            onChange={(value) =>
              setRuleForm({ ...ruleForm, description: value })
            }
          />
          <div className='mt-3 flex gap-8 rounded-lg border border-slate-200 p-3'>
            <label className='flex items-center gap-2'>
              <Switch
                checked={ruleForm.enabled}
                onChange={(value) =>
                  setRuleForm({ ...ruleForm, enabled: value })
                }
              />
              启用规则
            </label>
            <label className='flex items-center gap-2'>
              <Switch
                checked={ruleForm.notify_email}
                onChange={(value) =>
                  setRuleForm({ ...ruleForm, notify_email: value })
                }
              />
              邮件通知管理员/超管
            </label>
          </div>
        </Form>
      </Modal>
    </div>
  );
}
