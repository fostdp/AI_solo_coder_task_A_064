import { useState, useEffect } from 'react'
import { useStore } from '@/store/useStore'
import { fetchAlarms, acknowledgeAlarm } from '@/utils/api'

type LevelFilter = 'all' | '1' | '2' | '3'
type AckFilter = 'all' | 'unacked' | 'acked'

export default function AlarmPage() {
  const alarms = useStore((s) => s.alarms)
  const storeAck = useStore((s) => s.acknowledgeAlarm)
  const [selectedAlarm, setSelectedAlarm] = useState<string | null>(null)
  const [levelFilter, setLevelFilter] = useState<LevelFilter>('all')
  const [ackFilter, setAckFilter] = useState<AckFilter>('unacked')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    setLoading(true)
    const acked = ackFilter === 'acked' ? true : ackFilter === 'unacked' ? false : undefined
    fetchAlarms(acked)
      .catch(() => [])
      .finally(() => setLoading(false))
  }, [ackFilter])

  const filteredAlarms = alarms.filter((a) => {
    if (levelFilter !== 'all' && a.level !== Number(levelFilter)) return false
    if (ackFilter === 'unacked' && a.acknowledged) return false
    if (ackFilter === 'acked' && !a.acknowledged) return false
    return true
  })

  const selected = alarms.find((a) => a.id === selectedAlarm)

  const handleAck = async (id: string) => {
    try {
      await acknowledgeAlarm(id)
    } catch {
      // ignore
    }
    storeAck(id)
  }

  const levelColor = (level: number) => {
    if (level === 1) return 'border-brand-red/60 bg-brand-red/5'
    if (level === 2) return 'border-brand-yellow/60 bg-brand-yellow/5'
    return 'border-brand-accent/40 bg-brand-accent/5'
  }

  const levelBadge = (level: number) => {
    if (level === 1) return 'bg-brand-red/20 text-brand-red'
    if (level === 2) return 'bg-brand-yellow/20 text-brand-yellow'
    return 'bg-brand-accent/20 text-brand-accent'
  }

  return (
    <div className="flex h-full w-full">
      <div className="w-[420px] flex-shrink-0 border-r border-brand-accent/10 flex flex-col bg-brand-dark-2/50">
        <div className="px-5 py-4 border-b border-brand-accent/10">
          <h2 className="text-xl font-heading font-semibold text-white mb-3">告警管理</h2>
          <div className="flex gap-2 mb-2">
            {(['all', '1', '2', '3'] as LevelFilter[]).map((l) => (
              <button
                key={l}
                onClick={() => setLevelFilter(l)}
                className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors ${
                  levelFilter === l
                    ? 'bg-brand-accent/20 text-brand-accent'
                    : 'bg-brand-dark-3 text-gray-400 hover:text-white'
                }`}
              >
                {l === 'all' ? '全部' : `${l}级`}
              </button>
            ))}
          </div>
          <div className="flex gap-2">
            {(['all', 'unacked', 'acked'] as AckFilter[]).map((a) => (
              <button
                key={a}
                onClick={() => setAckFilter(a)}
                className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors ${
                  ackFilter === a
                    ? 'bg-brand-accent/20 text-brand-accent'
                    : 'bg-brand-dark-3 text-gray-400 hover:text-white'
                }`}
              >
                {a === 'all' ? '全部' : a === 'unacked' ? '未确认' : '已确认'}
              </button>
            ))}
          </div>
        </div>

        <div className="flex-1 overflow-y-auto">
          {loading && <div className="text-center text-gray-500 py-8 text-sm">加载中...</div>}
          {!loading && filteredAlarms.length === 0 && (
            <div className="text-center text-gray-500 py-8 text-sm">暂无告警</div>
          )}
          {filteredAlarms.map((alarm) => (
            <div
              key={alarm.id}
              onClick={() => setSelectedAlarm(alarm.id)}
              className={`px-5 py-3 border-l-4 cursor-pointer transition-colors hover:bg-brand-dark-3/50 ${
                levelColor(alarm.level)
              } ${selectedAlarm === alarm.id ? 'bg-brand-dark-3/80' : ''}`}
            >
              <div className="flex items-center gap-2 mb-1">
                <span className={`px-2 py-0.5 rounded text-[10px] font-heading font-semibold ${levelBadge(alarm.level)}`}>
                  L{alarm.level}
                </span>
                <span className="text-xs text-gray-400 font-mono">{alarm.timestamp}</span>
                {!alarm.acknowledged && (
                  <span className="ml-auto w-2 h-2 rounded-full bg-brand-red animate-flash-red" />
                )}
              </div>
              <div className="text-sm text-white truncate">{alarm.message}</div>
              <div className="text-xs text-gray-400 mt-0.5">{alarm.deviceName}</div>
            </div>
          ))}
        </div>
      </div>

      <div className="flex-1 flex flex-col bg-brand-dark">
        {selected ? (
          <div className="flex-1 flex flex-col p-6">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-lg font-heading font-semibold text-white">告警详情</h3>
              <span className={`px-3 py-1 rounded-lg text-xs font-heading font-semibold ${levelBadge(selected.level)}`}>
                {selected.level}级告警
              </span>
            </div>

            <div className="bg-brand-dark-2 rounded-xl p-5 border border-brand-accent/10 mb-6">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-gray-400">设备名称</span>
                  <div className="text-white font-heading mt-1">{selected.deviceName}</div>
                </div>
                <div>
                  <span className="text-gray-400">告警类型</span>
                  <div className="text-white font-heading mt-1">{selected.type}</div>
                </div>
                <div>
                  <span className="text-gray-400">发生时间</span>
                  <div className="text-white font-mono mt-1">{selected.timestamp}</div>
                </div>
                <div>
                  <span className="text-gray-400">确认状态</span>
                  <div className={`mt-1 font-heading ${selected.acknowledged ? 'text-brand-green' : 'text-brand-red'}`}>
                    {selected.acknowledged ? '已确认' : '未确认'}
                  </div>
                </div>
              </div>
            </div>

            <div className="bg-brand-dark-2 rounded-xl p-5 border border-brand-accent/10 mb-6">
              <h4 className="text-sm text-gray-400 mb-2">告警内容</h4>
              <p className="text-white text-base">{selected.message}</p>
            </div>

            {!selected.acknowledged && (
              <button
                onClick={() => handleAck(selected.id)}
                className="mt-auto px-6 py-3 rounded-xl bg-brand-accent/20 text-brand-accent font-heading font-semibold hover:bg-brand-accent/30 transition-colors border border-brand-accent/30"
              >
                确认告警
              </button>
            )}
          </div>
        ) : (
          <div className="flex-1 flex items-center justify-center text-gray-500 text-sm">
            点击左侧告警查看详情
          </div>
        )}
      </div>
    </div>
  )
}
