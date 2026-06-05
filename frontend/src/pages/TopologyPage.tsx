import Topology3D from '@/components/Topology3D'
import KPIBar from '@/components/KPIBar'
import { useStore } from '@/store/useStore'
import { useWebSocket } from '@/hooks/useWebSocket'

export default function TopologyPage() {
  const wsConnected = useStore((s) => s.wsConnected)
  useWebSocket()

  return (
    <div className="flex flex-col h-full w-full relative">
      <KPIBar />
      <div className="flex-1 relative">
        <Topology3D />
        <div className="absolute top-4 right-4 z-30 flex items-center gap-2">
          <div className={`flex items-center gap-2 px-3 py-1.5 rounded-full bg-brand-dark-2/80 backdrop-blur-sm border border-brand-accent/20 text-xs`}>
            <div className={`w-2 h-2 rounded-full ${wsConnected ? 'bg-brand-green' : 'bg-brand-red'} ${wsConnected ? 'animate-pulse' : 'animate-flash-red'}`} />
            <span className={wsConnected ? 'text-brand-green' : 'text-brand-red'}>
              {wsConnected ? 'WS已连接' : 'WS断开'}
            </span>
          </div>
        </div>
        <div className="absolute bottom-4 left-4 z-30 flex items-center gap-4 text-xs text-gray-400 bg-brand-dark-2/70 backdrop-blur-sm rounded-lg px-4 py-2 border border-brand-accent/10">
          <span className="flex items-center gap-1.5"><span className="w-3 h-3 rounded-sm bg-brand-green/60" />{'<60%'}</span>
          <span className="flex items-center gap-1.5"><span className="w-3 h-3 rounded-sm bg-brand-yellow/60" />{'60-80%'}</span>
          <span className="flex items-center gap-1.5"><span className="w-3 h-3 rounded-sm bg-brand-red/60" />{'>80%'}</span>
          <span className="flex items-center gap-1.5"><span className="w-3 h-1 rounded-sm bg-brand-red animate-flash-red" />N-1故障</span>
        </div>
      </div>
    </div>
  )
}
