import { useEffect, useState, useMemo } from 'react'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'

interface TrendChartProps {
  deviceId: string
  data?: Array<{ time: string; voltage: number; current: number; power: number; temperature: number }>
}

function generateDemoData() {
  const now = Date.now()
  const data = []
  for (let i = 120; i >= 0; i--) {
    const t = now - i * 60 * 1000
    data.push({
      time: new Date(t).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
      voltage: 33 + Math.sin(i * 0.1) * 1.5 + Math.random() * 0.5,
      current: 400 + Math.sin(i * 0.08) * 150 + Math.random() * 30,
      power: 13 + Math.sin(i * 0.12) * 5 + Math.random() * 2,
      temperature: 45 + Math.sin(i * 0.06) * 10 + Math.random() * 3,
    })
  }
  return data
}

export default function TrendChart({ deviceId, data: externalData }: TrendChartProps) {
  const [demoData, setDemoData] = useState(generateDemoData)

  useEffect(() => {
    setDemoData(generateDemoData())
  }, [deviceId])

  const chartData = useMemo(() => {
    if (externalData && externalData.length > 0) return externalData
    return demoData
  }, [externalData, demoData])

  return (
    <ResponsiveContainer width="100%" height={220}>
      <LineChart data={chartData} margin={{ top: 5, right: 10, left: -10, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="#1a3a5c" />
        <XAxis
          dataKey="time"
          stroke="#5a7a9c"
          tick={{ fill: '#5a7a9c', fontSize: 10 }}
          interval={19}
        />
        <YAxis stroke="#5a7a9c" tick={{ fill: '#5a7a9c', fontSize: 10 }} />
        <Tooltip
          contentStyle={{
            backgroundColor: '#0F2035',
            border: '1px solid #1a3a5c',
            borderRadius: '6px',
            color: '#fff',
            fontSize: 12,
          }}
        />
        <Legend
          wrapperStyle={{ fontSize: 11, color: '#8aa0b8' }}
        />
        <Line
          type="monotone"
          dataKey="voltage"
          stroke="#00D4FF"
          dot={false}
          strokeWidth={1.5}
          name="电压(kV)"
        />
        <Line
          type="monotone"
          dataKey="current"
          stroke="#FFB800"
          dot={false}
          strokeWidth={1.5}
          name="电流(A)"
        />
        <Line
          type="monotone"
          dataKey="power"
          stroke="#00FF88"
          dot={false}
          strokeWidth={1.5}
          name="功率(MW)"
        />
        <Line
          type="monotone"
          dataKey="temperature"
          stroke="#FF3344"
          dot={false}
          strokeWidth={1.5}
          name="温度(°C)"
        />
      </LineChart>
    </ResponsiveContainer>
  )
}
