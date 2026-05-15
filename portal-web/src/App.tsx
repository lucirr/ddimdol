import type { ReactNode } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { Sidebar } from '@/components/layout/Sidebar'
import { Header } from '@/components/layout/Header'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import AuthCallbackPage from '@/pages/AuthCallbackPage'
import LoginPage from '@/pages/LoginPage'
import DashboardPage from '@/pages/DashboardPage'
import EdgesPage from '@/pages/EdgesPage'
import ApprovalsPage from '@/pages/ApprovalsPage'
import ReleasesPage from '@/pages/ReleasesPage'
import DeploymentPage from '@/pages/DeploymentPage'
import { hasAccessToken } from '@/lib/auth'

function RequireAuth({ children }: { children: ReactNode }) {
  if (!hasAccessToken()) {
    return <Navigate to="/login" replace />
  }

  return children
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/auth/callback" element={<AuthCallbackPage />} />
      <Route
        path="/*"
        element={
          <RequireAuth>
            <div className="flex h-screen bg-gray-50">
              <Sidebar />
              <div className="flex flex-col flex-1 overflow-hidden">
                <Header />
                <main className="flex-1 overflow-y-auto p-6">
                  <ErrorBoundary>
                    <Routes>
                      <Route path="/" element={<Navigate to="/dashboard" replace />} />
                      <Route path="/dashboard" element={<DashboardPage />} />
                      <Route path="/edges" element={<EdgesPage />} />
                      <Route path="/approvals" element={<ApprovalsPage />} />
                      <Route path="/releases" element={<ReleasesPage />} />
                      <Route path="/deployments" element={<DeploymentPage />} />
                    </Routes>
                  </ErrorBoundary>
                </main>
              </div>
            </div>
          </RequireAuth>
        }
      />
    </Routes>
  )
}
