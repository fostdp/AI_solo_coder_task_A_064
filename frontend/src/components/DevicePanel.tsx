import { useStore, getLoadRateColorClass } from '@/store/useStore'
import TrendChart from './TrendChart'

export default function DevicePanel() {
  const selectedDevice = useStore((s) => s.selectedDevice)
  const nodes = useStore((s) => s.topologyNodes)
  const setSelectedDevice = useStore((s) => s.setSelectedDevice)

  const device = nodes.find((n) => n.id === selectedDevice)

  if (!device) return null

  const metrics = [
    { label: '电压', value: device.voltage.toFixed(2), unit: 'kV', color: 'text-brand-accent' },
    { label: '电流', value: device.current.toFixed(1), unit: 'A', color: 'text-brand-yellow' },
    { label: '功率', value: device.power.toFixed(2), unit: 'MW', color: 'text-brand-green' },
    { label: '温度', value: device.temperature.toFixed(1), unit: '°C', color: 'text-brand-red' },
  ]

  const historyItems = [
    { time: '14:32:10', event: '负荷率超过80%' },
    { time: '14:15:00', event: '电压波动告警' },
    { time: '13:45:22', event: '遥测数据更新' },
    { time: '12:00:00', event: '设备巡检完成' },
    { time: '10:30:15', event: '保护定值校核' },
  ]

  return (
    <div className="absolute right-0 top-0 h-full w-96 bg-brand-dark-2/95 backdrop-blur-xl border-l border-brand-accent/20 animate-slide-in-right z-40 flex flex-col overflow-hidden">
      <div className="flex items-center justify-between px-5 py-4 border-b border-brand-accent/10">
        <div>
          <h2 className="text-lg font-heading font-semibold text-white">{device.name}</h2>
          <span className="text-xs text-gray-400">{device.lineName} · {device.type === 'substation' ? '牵引变电所' : device.type}</span>
        </div>
        <button
          onClick={() => setSelectedDevice(null)}
          className="w-8 h-8 flex items-center justify-center rounded-lg text-gray-400 hover:text-white hover:bg-brand-dark-3 transition-colors"
        >
          <svg viewBox="0 0 24 24" className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth="2">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>

      <div className="px-5 py-4 border-b border-brand-accent/10">
        <div className="text-center">
          <span className="text-xs text-gray-400 uppercase tracking-wider">负荷率</span>
          <div className={`text-5xl font-heading font-bold ${getLoadRateColorClass(device.loadRate)} mt-1`}>
            {device.loadRate.toFixed(1)}%
          </div>
          <div className={`inline-block mt-2 px-3 py-0.5 rounded-full text-xs font-medium ${
            device.status === 'fault'
              ? 'bg-brand-red/20 text-brand-red'
              : device.status === 'warning'
              ? 'bg-brand-yellow/20 text-brand-yellow'
              : 'bg-brand-green/20 text-brand-green'
          }`}>
            {device.status === 'fault' ? '过载' : device.status === 'warning' ? '预警' : '正常'}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 px-5 py-4 border-b border-brand-accent/10">
        {metrics.map((m) => (
          <div key={m.label} className="bg-brand-dark-3/60 rounded-lg px-3 py-2.5">
            <div className="text-xs text-gray-400">{m.label}</div>
            <div className={`text-xl font-heading font-semibold ${m.color}`}>
              {m.value}
              <span className="text-xs text-gray-400 ml-1">{m.unit}</span>
            </div>
          </div>
        ))}
      </div>

      <div className="px-5 py-4 border-b border-brand-accent/10">
        <h3 className="text-sm font-heading font-semibold text-gray-300 mb-3">实时趋势 (2h)</h3>
        <TrendChart deviceId={device.id} />
      </div>

      <div className="px-5 py-4 flex-1 overflow-y-auto">
        <h3 className="text-sm font-heading font-semibold text-gray-300 mb-3">操作记录</h3>
        <div className="space-y-2">
          {historyItems.map((item, idx) => (
            <div key={idx} className="flex items-start gap-3 text-xs">
              <span className="text-gray-500 font-mono whitespace-nowrap">{item.time}</span>
              <span className="text-gray-300">{item.event}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
