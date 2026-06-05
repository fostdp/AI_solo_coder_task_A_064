import { create } from 'zustand'

export interface TopologyNode {
  id: string
  name: string
  type: 'substation' | 'transformer' | 'bus'
  lineId: string
  lineName: string
  x: number
  y: number
  z: number
  loadRate: number
  voltage: number
  current: number
  power: number
  temperature: number
  status: 'normal' | 'warning' | 'fault'
}

export interface TopologyEdge {
  id: string
  source: string
  target: string
  type: 'feeder' | 'tie'
  loadRate: number
  power: number
  fault: boolean
}

export interface Alarm {
  id: string
  deviceId: string
  deviceName: string
  level: 1 | 2 | 3
  type: string
  message: string
  timestamp: string
  acknowledged: boolean
}

export interface KPIMetrics {
  totalPower: number
  lineLoss: number
  voltageQualificationRate: number
  onlineDevices: number
  totalDevices: number
  activeAlarms: number
}

export interface PowerFlowResult {
  converged: boolean
  iterations: number
  totalLoss: number
  branchPowers: { branch: string; power: number }[]
}

export interface N1Result {
  faultBranch: string
  faultBranchName: string
  overloadedBranches: string[]
  overloadedBranchNames: string[]
  transferSuggestions: string[]
  maxLoadRate: number
}

interface StoreState {
  topologyNodes: TopologyNode[]
  topologyEdges: TopologyEdge[]
  selectedDevice: string | null
  alarms: Alarm[]
  kpiMetrics: KPIMetrics | null
  powerFlowResult: PowerFlowResult | null
  n1Results: N1Result[]
  wsConnected: boolean
  setTopology: (nodes: TopologyNode[], edges: TopologyEdge[]) => void
  setSelectedDevice: (id: string | null) => void
  addAlarm: (alarm: Alarm) => void
  setKPIMetrics: (metrics: KPIMetrics) => void
  setPowerFlowResult: (result: PowerFlowResult) => void
  setN1Results: (results: N1Result[]) => void
  setWSConnected: (connected: boolean) => void
  updateNodeTelemetry: (deviceId: string, data: Partial<TopologyNode>) => void
  acknowledgeAlarm: (id: string) => void
}

export const useStore = create<StoreState>((set) => ({
  topologyNodes: [],
  topologyEdges: [],
  selectedDevice: null,
  alarms: [],
  kpiMetrics: null,
  powerFlowResult: null,
  n1Results: [],
  wsConnected: false,

  setTopology: (nodes, edges) =>
    set({ topologyNodes: nodes, topologyEdges: edges }),

  setSelectedDevice: (id) =>
    set({ selectedDevice: id }),

  addAlarm: (alarm) =>
    set((state) => ({ alarms: [alarm, ...state.alarms] })),

  setKPIMetrics: (metrics) =>
    set({ kpiMetrics: metrics }),

  setPowerFlowResult: (result) =>
    set({ powerFlowResult: result }),

  setN1Results: (results) =>
    set({ n1Results: results }),

  setWSConnected: (connected) =>
    set({ wsConnected: connected }),

  updateNodeTelemetry: (deviceId, data) =>
    set((state) => ({
      topologyNodes: state.topologyNodes.map((node) =>
        node.id === deviceId ? { ...node, ...data } : node
      ),
    })),

  acknowledgeAlarm: (id) =>
    set((state) => ({
      alarms: state.alarms.map((alarm) =>
        alarm.id === id ? { ...alarm, acknowledged: true } : alarm
      ),
    })),
}))

export function getLoadRateColor(loadRate: number): string {
  if (loadRate < 60) return '#00FF88'
  if (loadRate < 80) return '#FFB800'
  return '#FF3344'
}

export function getLoadRateColorClass(loadRate: number): string {
  if (loadRate < 60) return 'text-brand-green'
  if (loadRate < 80) return 'text-brand-yellow'
  return 'text-brand-red'
}
