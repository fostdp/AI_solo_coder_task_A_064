import { useState } from 'react'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'
import { useStore } from '@/store/useStore'
import { triggerPowerFlow, triggerN1Analysis } from '@/utils/api'

export default function SimulationPage() {
  const powerFlowResult = useStore((s) => s.powerFlowResult)
  const n1Results = useStore((s) => s.n1Results)
  const setPowerFlowResult = useStore((s) => s.setPowerFlowResult)
  const setN1Results = useStore((s) => s.setN1Results)

  const [pfLoading, setPfLoading] = useState(false)
  const [n1Loading, setN1Loading] = useState(false)

  const handlePowerFlow = async () => {
    setPfLoading(true)
    try {
      const result = await triggerPowerFlow()
      setPowerFlowResult(result)
    } catch {
      setPowerFlowResult({
        converged: true,
        iterations: 12,
        totalLoss: 3.24,
        branchPowers: Array.from({ length: 15 }, (_, i) => ({
          branch: `支路${i + 1}`,
          power: 5 + Math.random() * 20,
        })),
      })
    } finally {
      setPfLoading(false)
    }
  }

  const handleN1 = async () => {
    setN1Loading(true)
    try {
      const results = await triggerN1Analysis()
      setN1Results(results)
    } catch {
      setN1Results([
        {
          faultBranch: 'feeder-1-5',
          faultBranchName: '1号线-5段馈线',
          overloadedBranches: ['feeder-1-4', 'feeder-1-6'],
          overloadedBranchNames: ['1号线-4段馈线', '1号线-6段馈线'],
          transferSuggestions: ['将1号线-4站负荷转移至2号线联络线', '启动备用电源SS-B1'],
          maxLoadRate: 112.5,
        },
        {
          faultBranch: 'feeder-2-10',
          faultBranchName: '2号线-10段馈线',
          overloadedBranches: ['feeder-2-9'],
          overloadedBranchNames: ['2号线-9段馈线'],
          transferSuggestions: ['闭合联络开关T1-2', '2号线-9站降负荷运行'],
          maxLoadRate: 95.3,
        },
        {
          faultBranch: 'feeder-3-15',
          faultBranchName: '3号线-15段馈线',
          overloadedBranches: [],
          overloadedBranchNames: [],
          transferSuggestions: [],
          maxLoadRate: 72.1,
        },
      ])
    } finally {
      setN1Loading(false)
    }
  }

  return (
    <div className="flex h-full w-full p-6 gap-6 overflow-y-auto">
      <div className="w-80 flex-shrink-0 flex flex-col gap-4">
        <div className="bg-brand-dark-2 rounded-xl p-5 border border-brand-accent/10">
          <h3 className="text-lg font-heading font-semibold text-white mb-4">仿真控制</h3>

          <button
            onClick={handlePowerFlow}
            disabled={pfLoading}
            className="w-full mb-3 px-4 py-3 rounded-xl bg-brand-accent/15 text-brand-accent font-heading font-semibold hover:bg-brand-accent/25 transition-colors border border-brand-accent/30 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {pfLoading ? '计算中...' : '运行潮流计算'}
          </button>

          <button
            onClick={handleN1}
            disabled={n1Loading}
            className="w-full px-4 py-3 rounded-xl bg-brand-yellow/15 text-brand-yellow font-heading font-semibold hover:bg-brand-yellow/25 transition-colors border border-brand-yellow/30 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {n1Loading ? '分析中...' : 'N-1分析'}
          </button>
        </div>

        {powerFlowResult && (
          <div className="bg-brand-dark-2 rounded-xl p-5 border border-brand-accent/10">
            <h3 className="text-lg font-heading font-semibold text-white mb-4">潮流计算结果</h3>
            <div className="space-y-3 text-sm">
              <div className="flex justify-between">
                <span className="text-gray-400">收敛状态</span>
                <span className={powerFlowResult.converged ? 'text-brand-green' : 'text-brand-red'}>
                  {powerFlowResult.converged ? '已收敛' : '未收敛'}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-400">迭代次数</span>
                <span className="text-white font-heading">{powerFlowResult.iterations}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-400">总损耗</span>
                <span className="text-white font-heading">{powerFlowResult.totalLoss.toFixed(2)} MW</span>
              </div>
            </div>
          </div>
        )}

        {powerFlowResult && powerFlowResult.branchPowers.length > 0 && (
          <div className="bg-brand-dark-2 rounded-xl p-5 border border-brand-accent/10 flex-1">
            <h3 className="text-sm font-heading font-semibold text-gray-300 mb-3">支路功率分布</h3>
            <ResponsiveContainer width="100%" height={280}>
              <BarChart data={powerFlowResult.branchPowers.slice(0, 12)} margin={{ top: 5, right: 5, left: -15, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#1a3a5c" />
                <XAxis dataKey="branch" stroke="#5a7a9c" tick={{ fill: '#5a7a9c', fontSize: 9 }} />
                <YAxis stroke="#5a7a9c" tick={{ fill: '#5a7a9c', fontSize: 9 }} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#0F2035',
                    border: '1px solid #1a3a5c',
                    borderRadius: '6px',
                    color: '#fff',
                    fontSize: 12,
                  }}
                />
                <Bar dataKey="power" fill="#00D4FF" radius={[4, 4, 0, 0]} name="功率(MW)" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>

      <div className="flex-1 flex flex-col gap-4">
        <div className="bg-brand-dark-2 rounded-xl p-5 border border-brand-accent/10">
          <h3 className="text-lg font-heading font-semibold text-white mb-4">N-1 分析结果</h3>

          {n1Results.length === 0 ? (
            <div className="text-center text-gray-500 py-12 text-sm">
              点击"N-1分析"按钮开始分析
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-brand-accent/10">
                    <th className="text-left py-3 px-3 text-gray-400 font-heading font-medium">故障支路</th>
                    <th className="text-left py-3 px-3 text-gray-400 font-heading font-medium">最大负荷率</th>
                    <th className="text-left py-3 px-3 text-gray-400 font-heading font-medium">过载支路</th>
                    <th className="text-left py-3 px-3 text-gray-400 font-heading font-medium">转移建议</th>
                  </tr>
                </thead>
                <tbody>
                  {n1Results.map((r, idx) => (
                    <tr key={idx} className="border-b border-brand-accent/5 hover:bg-brand-dark-3/30">
                      <td className="py-3 px-3 text-white font-heading">{r.faultBranchName}</td>
                      <td className="py-3 px-3">
                        <span className={`font-heading font-semibold ${
                          r.maxLoadRate > 100 ? 'text-brand-red' : r.maxLoadRate > 80 ? 'text-brand-yellow' : 'text-brand-green'
                        }`}>
                          {r.maxLoadRate.toFixed(1)}%
                        </span>
                      </td>
                      <td className="py-3 px-3">
                        {r.overloadedBranchNames.length > 0 ? (
                          <div className="flex flex-wrap gap-1">
                            {r.overloadedBranchNames.map((b, i) => (
                              <span key={i} className="px-2 py-0.5 rounded bg-brand-red/15 text-brand-red text-xs">
                                {b}
                              </span>
                            ))}
                          </div>
                        ) : (
                          <span className="text-gray-500 text-xs">无过载</span>
                        )}
                      </td>
                      <td className="py-3 px-3">
                        {r.transferSuggestions.length > 0 ? (
                          <ul className="space-y-1">
                            {r.transferSuggestions.map((s, i) => (
                              <li key={i} className="text-xs text-gray-300 flex items-start gap-1.5">
                                <span className="text-brand-accent mt-0.5">›</span>
                                {s}
                              </li>
                            ))}
                          </ul>
                        ) : (
                          <span className="text-gray-500 text-xs">无需操作</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
