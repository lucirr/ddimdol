import { useState } from 'react'
import { KeyRound, LogIn, ShieldCheck } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { authConfig, startLogin } from '@/lib/auth'

export default function LoginPage() {
  const [isSigningIn, setIsSigningIn] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleLogin = async () => {
    setIsSigningIn(true)
    setError(null)

    try {
      await startLogin()
    } catch (err) {
      setIsSigningIn(false)
      setError(err instanceof Error ? err.message : '로그인을 시작하지 못했습니다.')
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center px-4">
      <div className="w-full max-w-md rounded-lg border bg-white p-6 shadow-sm">
        <div className="mb-6 flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-md bg-gray-900 text-white">
            <ShieldCheck size={20} />
          </div>
          <div>
            <h1 className="text-lg font-bold text-gray-900">Edge DIP Portal</h1>
            <p className="text-sm text-gray-500">Keycloak SSO 로그인</p>
          </div>
        </div>

        <div className="space-y-4">
          <div className="rounded-md border border-gray-200 bg-gray-50 p-3 text-sm text-gray-600">
            <div className="flex items-center gap-2 font-medium text-gray-800">
              <KeyRound size={16} />
              {authConfig.realm} / {authConfig.clientId}
            </div>
            <p className="mt-1 text-xs text-gray-500">{authConfig.keycloakUrl}</p>
          </div>

          <Button type="button" className="w-full gap-2" onClick={handleLogin} disabled={isSigningIn}>
            <LogIn size={16} />
            {isSigningIn ? 'Keycloak로 이동 중...' : 'Keycloak로 로그인'}
          </Button>

          {error && <p className="text-sm text-red-600">{error}</p>}
        </div>
      </div>
    </div>
  )
}
