import { useEffect, useRef, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { clearAuth, completeLogin } from '@/lib/auth'

export default function AuthCallbackPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [error, setError] = useState<string | null>(null)
  const handledRef = useRef(false)

  useEffect(() => {
    if (handledRef.current) return
    handledRef.current = true

    const code = searchParams.get('code')
    const state = searchParams.get('state')
    const oidcError = searchParams.get('error_description') || searchParams.get('error')

    if (oidcError) {
      setError(oidcError)
      return
    }
    if (!code || !state) {
      setError('Keycloak callback에 code 또는 state가 없습니다.')
      return
    }

    completeLogin(code, state)
      .then(() => navigate('/dashboard', { replace: true }))
      .catch((err) => {
        clearAuth()
        const message = err && typeof err === 'object' && 'message' in err
          ? String(err.message)
          : '로그인 처리 중 오류가 발생했습니다.'
        setError(message)
      })
  }, [navigate, searchParams])

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center px-4">
      <div className="w-full max-w-md rounded-lg border bg-white p-6 text-center shadow-sm">
        {!error ? (
          <div className="space-y-3">
            <Loader2 className="mx-auto animate-spin text-gray-700" size={28} />
            <h1 className="text-lg font-bold text-gray-900">로그인 처리 중</h1>
            <p className="text-sm text-gray-500">Keycloak 인증 결과를 확인하고 있습니다.</p>
          </div>
        ) : (
          <div className="space-y-4">
            <h1 className="text-lg font-bold text-gray-900">로그인 실패</h1>
            <p className="text-sm text-red-600">{error}</p>
            <Button type="button" className="w-full" onClick={() => navigate('/login', { replace: true })}>
              로그인으로 돌아가기
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}
