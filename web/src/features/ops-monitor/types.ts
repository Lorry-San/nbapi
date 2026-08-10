export interface OpsMonitorOverview {
  request_count: number
  success_count: number
  error_count: number
  excluded_errors: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  avg_qps: number
  avg_tps: number
  sla: number
  error_rate: number
  upstream_error_rate: number
  avg_latency_ms: number
  avg_ttft_ms: number
  p95_latency_ms: number
  p99_latency_ms: number
  p95_ttft_ms: number
  p99_ttft_ms: number
}

export interface OpsMonitorPoint {
  ts: number
  requests: number
  errors: number
  tokens: number
  qps: number
  tps: number
  sla: number
  avg_ttft_ms: number
}

export interface OpsMonitorChannel {
  channel_id: number
  channel_name: string
  requests: number
  errors: number
  tokens: number
  avg_latency_ms: number
  avg_ttft_ms: number
  sla: number
  queue_depth: number
  channel_switches: number
  latest_error?: string
  latest_error_at?: number
  latest_status_code?: number
}

export interface OpsMonitorError {
  id: number
  created_at: number
  channel_id: number
  channel_name: string
  model_name: string
  status_code: number
  error_code: string
  error_type: string
  request_id: string
  upstream_request_id: string
  content: string
  request_path: string
  business_like: boolean
}

export interface OpsMonitorChannelSwitchPoint {
  ts: number
  switches: number
}

export interface OpsMonitorData {
  start_timestamp: number
  end_timestamp: number
  bucket_seconds: number
  overview: OpsMonitorOverview
  trend: OpsMonitorPoint[]
  channels: OpsMonitorChannel[]
  channel_switch_trend: OpsMonitorChannelSwitchPoint[]
  errors: OpsMonitorError[]
  updated_at: number
  truncated: boolean
}

export interface OpsMonitorResponse {
  success: boolean
  message?: string
  data?: OpsMonitorData
}

export interface OpsAlertRule {
  id: number
  name: string
  description: string
  metric: string
  comparator: string
  threshold: number
  level: string
  enabled: boolean
  window_seconds: number
  duration_seconds: number
  cooldown_seconds: number
  notify_email: boolean
  scope: string
  channel_id: number
  model_name: string
  group: string
  last_state: string
  first_triggered_at: number
  last_triggered_at: number
  last_recovered_at: number
  last_notified_at: number
  last_value: number
  last_message: string
  created_at: number
  updated_at: number
}

export interface OpsAlertMetric {
  key: string
  label: string
  unit: string
  default_comparator: string
}

export interface OpsAlertEvent {
  id: number
  rule_id: number
  rule_name: string
  title: string
  message: string
  metric: string
  comparator: string
  threshold: number
  current_value: number
  level: string
  status: string
  scope: string
  channel_id: number
  channel_name: string
  model_name: string
  group: string
  window_seconds: number
  duration_seconds: number
  triggered_at: number
  resolved_at: number
  email_sent: boolean
  email_error: string
  email_recipient: string
  notification_type: string
  created_at: number
}

export interface OpsAlertRulesResponse {
  success: boolean
  message?: string
  data?: {
    rules: OpsAlertRule[]
    metrics: OpsAlertMetric[]
  }
}

export interface OpsAlertEventsResponse {
  success: boolean
  message?: string
  data?: OpsAlertEvent[]
}

export type OpsAlertRulePayload = Omit<
  OpsAlertRule,
  | 'id'
  | 'last_state'
  | 'first_triggered_at'
  | 'last_triggered_at'
  | 'last_recovered_at'
  | 'last_notified_at'
  | 'last_value'
  | 'last_message'
  | 'created_at'
  | 'updated_at'
>
