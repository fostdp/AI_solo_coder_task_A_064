import type { TopologyNode, TopologyEdge, Alarm, KPIMetrics, PowerFlowResult, N1Result } from '@/store/useStore'

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) throw new Error(`API error: ${res.status}`)
  return res.json()
}

interface TopologyResponse {
  nodes: TopologyNode[]
  edges: TopologyEdge[]
}

export async function fetchTopology(): Promise<TopologyResponse> {
  return request<TopologyResponse>('/api/topology')
}

interface TelemetryResponse {
  device_id: string
  load_rate: number
  voltage: number
  current: number
  power: number
  temperature: number
}

export async function fetchDeviceTelemetry(
  deviceId: string,
  range?: string
): Promise<TelemetryResponse> {
  const params = range ? `?range=${range}` : ''
  return request<TelemetryResponse>(`/api/devices/${deviceId}/telemetry${params}`)
}

interface HistoryPoint {
  timestamp: string
  voltage: number
  current: number
  power: number
  temperature: number
}

export async function fetchDeviceHistory(deviceId: string): Promise<HistoryPoint[]> {
  return request<HistoryPoint[]>(`/api/devices/${deviceId}/history`)
}

export async function triggerPowerFlow(): Promise<PowerFlowResult> {
  return request<PowerFlowResult>('/api/simulation/powerflow', { method: 'POST' })
}

export async function triggerN1Analysis(): Promise<N1Result[]> {
  return request<N1Result[]>('/api/simulation/n1', { method: 'POST' })
}

interface AlarmsResponse {
  alarms: Alarm[]
}

export async function fetchAlarms(acknowledged?: boolean): Promise<Alarm[]> {
  const params = acknowledged !== undefined ? `?acknowledged=${acknowledged}` : ''
  const res = await request<AlarmsResponse>(`/api/alarms${params}`)
  return res.alarms
}

export async function acknowledgeAlarm(id: string): Promise<void> {
  await request(`/api/alarms/${id}/acknowledge`, { method: 'PUT' })
}

export async function fetchKPIMetrics(): Promise<KPIMetrics> {
  return request<KPIMetrics>('/api/kpi')
}
