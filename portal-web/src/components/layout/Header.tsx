import { logout, parseTokenClaims } from '@/lib/auth'

export function Header() {
  const claims = parseTokenClaims()
  const displayName = claims?.preferred_username || claims?.name || claims?.email || '관리자'

  const handleLogout = () => {
    logout()
  }

  return (
    <header className="h-14 border-b bg-white flex items-center justify-between px-6">
      <span className="text-sm text-gray-500">중앙 에지 관제 시스템</span>
      <div className="flex items-center gap-3">
        <span className="text-sm font-medium text-gray-700">{displayName}</span>
        <button
          type="button"
          className="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
          onClick={handleLogout}
        >
          로그아웃
        </button>
      </div>
    </header>
  )
}
