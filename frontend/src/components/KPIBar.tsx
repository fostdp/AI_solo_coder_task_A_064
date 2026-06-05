import { useEffect, useState } from 'react'
import { useStore } from '@/store/useStore'
import { fetchKPIMetrics } from '@/utils/api'

function AnimatedNumber({ value, decimals = 0, suffix = '' }: { value: number; decimals?: number; suffix?: string }) {
  const [display, setDisplay] = useState(0)

  useEffect(() => {
    const duration = 1000
    const startTime = Date.now()
    const startVal = display
    const targetVal = value

    const animate = () => {
      const elapsed = Date.now() - startTime
      const progress = Math.min(elapsed / duration, 1)
      const eased = 1 - Math.pow(1 - progress, 3)
      setDisplay(startVal + (targetVal - startVal) * eased)

      if (progress < 1) {
        requestAnimationFrame(animate)
      }
    }

    requestAnimationFrame(animate)
  }, [value])

  return (
    <span className="text-3xl font-heading font-bold text-white">
      {display.toFixed(decimals)}{suffix}
    </span>
  )
}

export default function KPIBar() {
  const kpiMetrics = useStore((s) => s.kpiMetrics)
  const setKPIMetrics = useStore((s) => s.setKPIMetrics)

  useEffect(() => {
    fetchKPIMetrics().then(setKPIMetrics).catch(() => {
      setKPIMetrics({
        totalPower: 156.8,
        lineLoss: 3.24,
        voltageQualificationRate: 98.6,
        onlineDevices: 58,
        totalDevices: 60,
        activeAlarms: 3,
      })
    })
  }, [setKPIMetrics])

  if (!kpiMetrics) return null

  const cards = [
    {
      label: '全网功率总加',
      value: kpiMetrics.totalPower,
      decimals: 1,
      suffix: ' MW',
      icon: (
        <svg viewBox="0 0 24 24" className="w-5 h-5" fill="none" stroke="#00D4FF" strokeWidth="1.5">
          <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
        </svg>
      ),
      borderColor: 'border-brand-accent/40',
      glowColor: 'shadow-[0_2px_12px_rgba(0,212,255,0.15)]',
    },
    {
      label: '线路损耗',
      value: kpiMetrics.lineLoss,
      decimals: 2,
      suffix: ' MW',
      icon: (
        <svg viewBox="0 0 24 24" className="w-5 h-5" fill="none" stroke="#FFB800" strokeWidth="1.5">
          <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
        </svg>
      ),
      borderColor: 'border-brand-yellow/40',
      glowColor: 'shadow-[0_2px_12px_rgba(255,184,0,0.15)]',
    },
    {
      label: '电压合格率',
      value: kpiMetrics.voltageQualificationRate,
      decimals: 1,
      suffix: '%',
      icon: (
        <svg viewBox="0 0 24 24" className="w-5 h-5" fill="none" stroke="#00FF88" strokeWidth="1.5">
          <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
        </svg>
      ),
      borderColor: 'border-brand-green/40',
      glowColor: 'shadow-[0_2px_12px_rgba(0,255,136,0.15)]',
    },
  ]

  return (
    <div className="flex items-center gap-4 px-6 py-3 bg-brand-dark-2/80 backdrop-blur-sm border-b border-brand-accent/10">
      {cards.map((card) => (
        <div
          key={card.label}
          className={`flex-1 flex items-center gap-4 bg-brand-dark-3/50 rounded-xl px-5 py-3 border-b-2 ${card.borderColor} ${card.glowColor}`}
        >
          <div className="flex-shrink-0">{card.icon}</div>
          <div>
            <div className="text-xs text-gray-400 font-body">{card.label}</div>
            <AnimatedNumber value={card.value} decimals={card.decimals} suffix={card.suffix} />
          </div>
        </div>
      ))}
    </div>
  )
}
