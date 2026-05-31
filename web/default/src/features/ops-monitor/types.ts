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
