import { api } from '@/lib/api'
import type { OpsMonitorResponse } from './types'

export async function getOpsMonitor(params: {
  range_seconds?: number
  start_timestamp?: number
  end_timestamp?: number
  channel?: number
  model_name?: string
  group?: string
}): Promise<OpsMonitorResponse> {
  const res = await api.get('/api/ops/monitor', {
    params,
    disableDuplicate: true,
  })
  return res.data
}
