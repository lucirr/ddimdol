import { Routes, Route, Navigate } from 'react-router-dom'
import { Sidebar } from '@/components/layout/Sidebar'
import { Header } from '@/components/layout/Header'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import DashboardPage from '@/pages/DashboardPage'
import EdgesPage from '@/pages/EdgesPage'
import ApprovalsPage from '@/pages/ApprovalsPage'
import ReleasesPage from '@/pages/ReleasesPage'
import DeploymentPage from '@/pages/DeploymentPage'

export default function App() {
  return (
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
  )
}
