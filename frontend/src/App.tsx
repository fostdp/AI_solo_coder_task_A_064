import { Routes, Route, NavLink } from 'react-router-dom'
import TopologyPage from './pages/TopologyPage'
import AlarmPage from './pages/AlarmPage'
import SimulationPage from './pages/SimulationPage'
import AlarmBadge from './components/AlarmBadge'

function Sidebar() {
  return (
    <div className="fixed left-0 top-0 h-full w-16 bg-brand-dark-2 border-r border-brand-accent/20 flex flex-col items-center py-6 gap-2 z-50">
      <div className="w-10 h-10 mb-6 flex items-center justify-center">
        <svg viewBox="0 0 40 40" className="w-10 h-10">
          <rect x="4" y="4" width="32" height="32" rx="4" fill="none" stroke="#00D4FF" strokeWidth="2" />
          <path d="M12 28 L20 12 L28 28" fill="none" stroke="#00D4FF" strokeWidth="2" />
          <line x1="15" y1="22" x2="25" y2="22" stroke="#00D4FF" strokeWidth="2" />
        </svg>
      </div>

      <NavLink
        to="/"
        className={({ isActive }) =>
          `w-12 h-12 flex items-center justify-center rounded-lg transition-all duration-200 ${
            isActive
              ? 'bg-brand-accent/20 text-brand-accent shadow-[0_0_15px_rgba(0,212,255,0.3)]'
              : 'text-gray-400 hover:text-brand-accent hover:bg-brand-dark-3'
          }`
        }
      >
        <svg viewBox="0 0 24 24" className="w-6 h-6" fill="none" stroke="currentColor" strokeWidth="1.5">
          <circle cx="6" cy="6" r="2" />
          <circle cx="18" cy="6" r="2" />
          <circle cx="6" cy="18" r="2" />
          <circle cx="18" cy="18" r="2" />
          <circle cx="12" cy="12" r="2" />
          <line x1="8" y1="6" x2="10" y2="12" />
          <line x1="14" y1="12" x2="16" y2="6" />
          <line x1="8" y1="18" x2="10" y2="12" />
          <line x1="14" y1="12" x2="16" y2="18" />
        </svg>
      </NavLink>

      <NavLink
        to="/alarms"
        className={({ isActive }) =>
          `w-12 h-12 flex items-center justify-center rounded-lg transition-all duration-200 relative ${
            isActive
              ? 'bg-brand-accent/20 text-brand-accent shadow-[0_0_15px_rgba(0,212,255,0.3)]'
              : 'text-gray-400 hover:text-brand-accent hover:bg-brand-dark-3'
          }`
        }
      >
        <svg viewBox="0 0 24 24" className="w-6 h-6" fill="none" stroke="currentColor" strokeWidth="1.5">
          <path d="M12 2L3 7v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V7l-9-5z" />
          <line x1="12" y1="9" x2="12" y2="13" />
          <circle cx="12" cy="16" r="0.5" fill="currentColor" />
        </svg>
        <AlarmBadge />
      </NavLink>

      <NavLink
        to="/simulation"
        className={({ isActive }) =>
          `w-12 h-12 flex items-center justify-center rounded-lg transition-all duration-200 ${
            isActive
              ? 'bg-brand-accent/20 text-brand-accent shadow-[0_0_15px_rgba(0,212,255,0.3)]'
              : 'text-gray-400 hover:text-brand-accent hover:bg-brand-dark-3'
          }`
        }
      >
        <svg viewBox="0 0 24 24" className="w-6 h-6" fill="none" stroke="currentColor" strokeWidth="1.5">
          <rect x="3" y="3" width="7" height="7" rx="1" />
          <rect x="14" y="3" width="7" height="7" rx="1" />
          <rect x="3" y="14" width="7" height="7" rx="1" />
          <rect x="14" y="14" width="7" height="7" rx="1" />
          <line x1="10" y1="6.5" x2="14" y2="6.5" />
          <line x1="6.5" y1="10" x2="6.5" y2="14" />
          <line x1="17.5" y1="10" x2="17.5" y2="14" />
          <line x1="10" y1="17.5" x2="14" y2="17.5" />
        </svg>
      </NavLink>

      <div className="flex-1" />

      <div className="w-10 h-10 flex items-center justify-center text-gray-500 text-xs font-heading">
        v1.0
      </div>
    </div>
  )
}

export default function App() {
  return (
    <div className="flex h-screen w-screen bg-brand-dark overflow-hidden">
      <Sidebar />
      <main className="flex-1 ml-16 h-full">
        <Routes>
          <Route path="/" element={<TopologyPage />} />
          <Route path="/alarms" element={<AlarmPage />} />
          <Route path="/simulation" element={<SimulationPage />} />
        </Routes>
      </main>
    </div>
  )
}
