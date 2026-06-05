import { useStore } from '@/store/useStore'
import { useNavigate } from 'react-router-dom'

export default function AlarmBadge() {
  const alarms = useStore((s) => s.alarms)
  const unackCount = alarms.filter((a) => !a.acknowledged).length
  const navigate = useNavigate()

  if (unackCount === 0) return null

  return (
    <button
      onClick={(e) => {
        e.preventDefault()
        e.stopPropagation()
        navigate('/alarms')
      }}
      className="absolute -top-1 -right-1 min-w-[18px] h-[18px] flex items-center justify-center rounded-full bg-brand-red text-white text-[10px] font-heading font-bold animate-flash-red"
    >
      {unackCount > 99 ? '99+' : unackCount}
    </button>
  )
}
