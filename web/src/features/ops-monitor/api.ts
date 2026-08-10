import { api } from '@/lib/api'
import type {
  OpsAlertEventsResponse,
  OpsAlertRulePayload,
  OpsAlertRulesResponse,
  OpsMonitorResponse,
} from './types'

export async function getOpsMonitor(params: {
  range_seconds?: number
  start_timestamp?: number
  end_timestamp?: number
  channel?: number
  model_name?: string
  group?: string
}): Promise<OpsMonitorResponse> {
  const res = await api.get('/api/ops/monitor', { params })
  return res.data
}

export async function getOpsAlertRules(): Promise<OpsAlertRulesResponse> {
  const res = await api.get('/api/ops/alerts/rules')
  return res.data
}

export async function createOpsAlertRule(payload: OpsAlertRulePayload) {
  const res = await api.post('/api/ops/alerts/rules', payload)
  return res.data
}

export async function updateOpsAlertRule(
  id: number,
  payload: OpsAlertRulePayload
) {
  const res = await api.put(`/api/ops/alerts/rules/${id}`, payload)
  return res.data
}

export async function deleteOpsAlertRule(id: number) {
  const res = await api.delete(`/api/ops/alerts/rules/${id}`)
  return res.data
}

export async function getOpsAlertEvents(params: {
  range_seconds?: number
  level?: string
  status?: string
  rule_id?: number
  limit?: number
}): Promise<OpsAlertEventsResponse> {
  const res = await api.get('/api/ops/alerts/events', { params })
  return res.data
}

export async function evaluateOpsAlerts() {
  const res = await api.post('/api/ops/alerts/evaluate')
  return res.data
}
