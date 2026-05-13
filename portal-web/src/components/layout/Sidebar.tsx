import { NavLink } from 'react-router-dom'
import { LayoutDashboard, Server, CheckSquare, Package, History } from 'lucide-react'
import { cn } from '@/lib/utils'

const navItems = [
  { to: '/dashboard', icon: LayoutDashboard, label: '대시보드' },
  { to: '/edges', icon: Server, label: '에지 노드' },
  { to: '/approvals', icon: CheckSquare, label: '승인 관리' },
  { to: '/releases', icon: Package, label: '릴리즈' },
  { to: '/deployments', icon: History, label: '배포 이력' },
]

export function Sidebar() {
  return (
    <aside className="w-60 bg-gray-900 text-white flex flex-col">
      <div className="px-6 py-5 border-b border-gray-700">
        <h1 className="text-lg font-bold">Edge DIP Portal</h1>
      </div>
      <nav className="flex-1 px-3 py-4 space-y-1">
        {navItems.map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors',
                isActive ? 'bg-gray-700 text-white' : 'text-gray-300 hover:bg-gray-800',
              )
            }
          >
            <Icon size={16} />
            {label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
