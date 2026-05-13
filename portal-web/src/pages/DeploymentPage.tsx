import { useState } from 'react'
import { useDeployments } from '@/hooks/useDeployments'
import { useEdges } from '@/hooks/useEdges'
import { useReleases } from '@/hooks/useReleases'
import { Badge } from '@/components/ui/badge'
import { Pagination } from '@/components/ui/pagination'
import type { ApprovalStatus } from '@/types/approval'

const LIMIT = 20

const statusVariant: Record<ApprovalStatus, 'success' | 'danger' | 'warning' | 'muted' | 'default'> = {
  PENDING: 'warning',
  APPROVED: 'success',
  REJECTED: 'danger',
  DEFERRED: 'muted',
  APPLIED: 'success',
  ROLLED_BACK: 'danger',
  EXPIRED: 'muted',
}

const statusLabel: Record<ApprovalStatus, string> = {
  PENDING: '대기',
  APPROVED: '승인됨',
  REJECTED: '거부됨',
  DEFERRED: '지연됨',
  APPLIED: '배포 완료',
  ROLLED_BACK: '롤백됨',
  EXPIRED: '만료됨',
}

const getVariant = (status: string) => statusVariant[status as ApprovalStatus] ?? 'default'
const getLabel = (status: string) => statusLabel[status as ApprovalStatus] ?? status

export default function DeploymentPage() {
  const [page, setPage] = useState(1)
  const { data: deployments = [], meta, isLoading, isError } = useDeployments(page, LIMIT)
  const { data: edgesData } = useEdges(1, 1000)
  const { data: releasesData } = useReleases(1, 1000)
  const edges = edgesData?.data ?? []
  const releases = releasesData?.data ?? []
  const total = meta?.total ?? deployments.length

  const edgeMap = Object.fromEntries(edges.map((e) => [e.id, e.name]))
  const releaseMap = Object.fromEntries(
    releases.map((r) => [r.id, `${r.package_name} v${r.version}`])
  )

  if (isLoading) return <div className="text-gray-500">로딩 중...</div>

  if (isError) return (
    <div className="flex items-center justify-center h-64 text-red-500">
      <p>데이터를 불러오는 중 오류가 발생했습니다. 페이지를 새로고침 해주세요.</p>
    </div>
  )

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">배포 이력</h2>
      </div>

      <div className="bg-white rounded-lg border">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b">
            <tr>
              <th className="px-4 py-3 text-left font-medium text-gray-600">릴리즈</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">에지</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">상태</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">요청자</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">처리자</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">요청일</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">처리일</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {deployments.map((dep) => (
              <tr key={dep.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-xs font-mono text-gray-700">
                  {releaseMap[dep.release_id] ?? dep.release_id.slice(0, 8) + '...'}
                </td>
                <td className="px-4 py-3 text-xs text-gray-700">
                  {edgeMap[dep.edge_id] ?? dep.edge_id.slice(0, 8) + '...'}
                </td>
                <td className="px-4 py-3">
                  <Badge variant={getVariant(dep.status)}>
                    {getLabel(dep.status)}
                  </Badge>
                </td>
                <td className="px-4 py-3 text-xs text-gray-600">{dep.requested_by}</td>
                <td className="px-4 py-3 text-xs text-gray-600">{dep.decision_by ?? '-'}</td>
                <td className="px-4 py-3 text-gray-600">
                  {new Date(dep.requested_at).toLocaleString('ko-KR')}
                </td>
                <td className="px-4 py-3 text-gray-600">
                  {dep.decided_at ? new Date(dep.decided_at).toLocaleString('ko-KR') : '-'}
                </td>
              </tr>
            ))}
            {deployments.length === 0 && (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-gray-400">
                  배포 이력이 없습니다
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <Pagination page={page} limit={LIMIT} total={total} onPageChange={setPage} />
    </div>
  )
}
