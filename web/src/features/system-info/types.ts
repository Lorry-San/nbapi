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
export type SystemInstanceStatus = 'online' | 'stale'

export type SystemInstanceInfo = {
  schema_version?: number
  node?: {
    name?: string
    source?: string
    manually_configured?: boolean
    should_configure_manually?: boolean
    [key: string]: unknown
  }
  role?: {
    is_master?: boolean
    [key: string]: unknown
  }
  runtime?: {
    version?: string
    goos?: string
    goarch?: string
    started_at?: number
    [key: string]: unknown
  }
  host?: {
    hostname?: string
    [key: string]: unknown
  }
  resources?: {
    cpu?: {
      usage_percent?: number
      [key: string]: unknown
    }
    memory?: {
      usage_percent?: number
      [key: string]: unknown
    }
    storage?: {
      total_bytes?: number
      used_bytes?: number
      free_bytes?: number
      used_percent?: number
      [key: string]: unknown
    }
    [key: string]: unknown
  }
  [key: string]: unknown
}

export type SystemInstance = {
  node_name: string
  status: SystemInstanceStatus
  stale_after_seconds: number
  started_at: number
  last_seen_at: number
  info?: SystemInstanceInfo
}

export type SystemInstanceListResponse = {
  success: boolean
  message: string
  data?: SystemInstance[]
}

export type HAConfig = {
  enabled: boolean
  primary_node_name: string
  standby_node_name: string
  primary_health_url: string
  standby_health_url: string
  public_entry: string
  origin_entry: string
  primary_origin: string
  standby_origin: string
  dns_provider: 'cloudflare' | 'manual' | 'other' | string
  dns_record_name: string
  database_engine: 'postgresql' | 'mysql' | 'sqlite' | 'external' | string
  replication_mode:
    | 'external'
    | 'postgres_streaming'
    | 'mysql_replica'
    | 'managed'
    | 'manual'
    | string
  redis_mode: 'shared' | 'sentinel' | 'primary_standby' | 'disabled' | string
  failover_strategy: 'manual' | 'assisted' | string
  health_check_interval_seconds: number
  failover_threshold: number
  cutover_runbook: string
  rollback_runbook: string
  notes: string
}

export type HACurrentNode = {
  name: string
  source: string
  is_master: boolean
}

export type HACheck = {
  level: 'ok' | 'warn' | 'error'
  key: string
  message: string
}

export type HAHealthProbe = {
  target: 'primary' | 'standby' | string
  url: string
  reachable: boolean
  status_code?: number
  success?: boolean
  message?: string
  data?: Record<string, unknown>
}

export type HASnippets = {
  primary_env: string
  standby_env: string
  compose_env: string
  cutover_checklist: string
}

export type HAOverview = {
  config: HAConfig
  current_node: HACurrentNode
  instances: SystemInstance[]
  probes: HAHealthProbe[]
  checks: HACheck[]
  summary: 'ok' | 'warn' | 'error' | 'disabled'
  snippets: HASnippets
}

export type HAOverviewResponse = {
  success: boolean
  message: string
  data?: HAOverview
}

export type SystemInstanceDeleteResponse = {
  success: boolean
  message: string
  data?: {
    deleted_count: number
  }
}
