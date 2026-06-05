import { useEffect, useRef, useCallback } from 'react'
import { useStore } from '@/store/useStore'

interface TelemetryUpdate {
  type: 'telemetry_update'
  device_id: string
  load_rate: number
  voltage: number
  current: number
  power: number
  temperature: number
}

interface AlarmMessage {
  type: 'alarm'
  id: string
  device_id: string
  device_name: string
  level: 1 | 2 | 3
  alarm_type: string
  message: string
  timestamp: string
}

interface PowerFlowMessage {
  type: 'powerflow_result'
  converged: boolean
  iterations: number
  total_loss: number
  branch_powers: { branch: string; power: number }[]
}

interface N1Message {
  type: 'n1_result'
  results: {
    fault_branch: string
    fault_branch_name: string
    overloaded_branches: string[]
    overloaded_branch_names: string[]
    transfer_suggestions: string[]
    max_load_rate: number
  }[]
}

interface KPIUpdateMessage {
  type: 'kpi_update'
  total_power: number
  line_loss: number
  voltage_qualification_rate: number
  online_devices: number
  total_devices: number
  active_alarms: number
}

type WSMessage = TelemetryUpdate | AlarmMessage | PowerFlowMessage | N1Message | KPIUpdateMessage

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const setWSConnected = useStore((s) => s.setWSConnected)
  const updateNodeTelemetry = useStore((s) => s.updateNodeTelemetry)
  const addAlarm = useStore((s) => s.addAlarm)
  const setPowerFlowResult = useStore((s) => s.setPowerFlowResult)
  const setN1Results = useStore((s) => s.setN1Results)
  const setKPIMetrics = useStore((s) => s.setKPIMetrics)

  const handleMessage = useCallback(
    (event: MessageEvent) => {
      try {
        const msg: WSMessage = JSON.parse(event.data)

        switch (msg.type) {
          case 'telemetry_update':
            updateNodeTelemetry(msg.device_id, {
              loadRate: msg.load_rate,
              voltage: msg.voltage,
              current: msg.current,
              power: msg.power,
              temperature: msg.temperature,
              status:
                msg.load_rate > 80
                  ? 'fault'
                  : msg.load_rate > 60
                  ? 'warning'
                  : 'normal',
            })
            break

          case 'alarm':
            addAlarm({
              id: msg.id,
              deviceId: msg.device_id,
              deviceName: msg.device_name,
              level: msg.level,
              type: msg.alarm_type,
              message: msg.message,
              timestamp: msg.timestamp,
              acknowledged: false,
            })
            break

          case 'powerflow_result':
            setPowerFlowResult({
              converged: msg.converged,
              iterations: msg.iterations,
              totalLoss: msg.total_loss,
              branchPowers: msg.branch_powers,
            })
            break

          case 'n1_result':
            setN1Results(
              msg.results.map((r) => ({
                faultBranch: r.fault_branch,
                faultBranchName: r.fault_branch_name,
                overloadedBranches: r.overloaded_branches,
                overloadedBranchNames: r.overloaded_branch_names,
                transferSuggestions: r.transfer_suggestions,
                maxLoadRate: r.max_load_rate,
              }))
            )
            break

          case 'kpi_update':
            setKPIMetrics({
              totalPower: msg.total_power,
              lineLoss: msg.line_loss,
              voltageQualificationRate: msg.voltage_qualification_rate,
              onlineDevices: msg.online_devices,
              totalDevices: msg.total_devices,
              activeAlarms: msg.active_alarms,
            })
            break
        }
      } catch {
        // ignore parse errors
      }
    },
    [updateNodeTelemetry, addAlarm, setPowerFlowResult, setN1Results, setKPIMetrics]
  )

  const connect = useCallback(() => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.hostname}:${window.location.port}/ws`

    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => {
      setWSConnected(true)
    }

    ws.onclose = () => {
      setWSConnected(false)
      reconnectTimerRef.current = setTimeout(() => {
        connect()
      }, 3000)
    }

    ws.onerror = () => {
      ws.close()
    }

    ws.onmessage = handleMessage
  }, [setWSConnected, handleMessage])

  useEffect(() => {
    connect()

    return () => {
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [connect])

  const connected = useStore((s) => s.wsConnected)
  return { connected }
}
