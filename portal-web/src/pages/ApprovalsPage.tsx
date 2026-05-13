import { useState } from 'react'
import { useApprovals, useCreateApproval, useApproveRequest, useRejectRequest } from '@/hooks/useApprovals'
import { useEdges } from '@/hooks/useEdges'
import { useReleases } from '@/hooks/useReleases'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog } from '@/components/ui/dialog'
import { Textarea } from '@/components/ui/textarea'
import { Pagination } from '@/components/ui/pagination'
import type { ApprovalRequest, ApprovalStatus } from '@/types/approval'

const statusVariant: Record<ApprovalStatus, 'success' | 'danger' | 'warning' | 'muted' | 'default'> = {
  PENDING: 'warning',
  APPROVED: 'success',
  REJECTED: 'danger',
  DEFERRED: 'muted',
  APPLIED: 'success',
  ROLLED_BACK: 'danger',
  EXPIRED: 'muted',
}

type ActionDialog = { type: 'approve' | 'reject'; approval: ApprovalRequest } | null

const LIMIT = 20

export default function ApprovalsPage() {
  const [page, setPage] = useState(1)
  const { data, isLoading, isError } = useApprovals(page, LIMIT)
  const approvals = data?.data ?? []
  const total = data?.meta?.total ?? 0

  // Full lists for dialog dropdowns
  const { data: edgesData } = useEdges(1, 1000)
  const { data: releasesData } = useReleases(1, 1000)
  const edges = edgesData?.data ?? []
  const releases = releasesData?.data ?? []

  const createApproval = useCreateApproval()
  const approve = useApproveRequest()
  const reject = useRejectRequest()

  const [createOpen, setCreateOpen] = useState(false)
  const [actionDialog, setActionDialog] = useState<ActionDialog>(null)
  const [reason, setReason] = useState('')
  const [form, setForm] = useState({ release_id: '', edge_id: '' })
  const [formError, setFormError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError(null)
    try {
      await createApproval.mutateAsync(form)
      setCreateOpen(false)
      setForm({ release_id: '', edge_id: '' })
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : '요청 처리 중 오류가 발생했습니다.')
    }
  }

  const handleAction = async () => {
    if (!actionDialog) return
    setActionError(null)
    try {
      if (actionDialog.type === 'approve') {
        await approve.mutateAsync({ id: actionDialog.approval.id, reason })
      } else {
        await reject.mutateAsync({ id: actionDialog.approval.id, reason })
      }
      setActionDialog(null)
      setReason('')
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : '요청 처리 중 오류가 발생했습니다.')
    }
  }

  if (isLoading) return <div className="text-gray-500">로딩 중...</div>

  if (isError) return (
    <div className="flex items-center justify-center h-64 text-red-500">
      <p>데이터를 불러오는 중 오류가 발생했습니다. 페이지를 새로고침 해주세요.</p>
    </div>
  )

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">승인 관리</h2>
        <Button onClick={() => setCreateOpen(true)}>+ 승인 요청</Button>
      </div>

      <div className="bg-white rounded-lg border">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b">
            <tr>
              <th className="px-4 py-3 text-left font-medium text-gray-600">ID</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">릴리즈</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">이미지</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">에지</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">상태</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">사유</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">생성일</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">액션</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {approvals.map((req) => {
              const release = releases.find((r) => r.id === req.release_id)
              const edge = edges.find((e) => e.id === req.edge_id)
              const showReason = req.status === 'APPROVED' || req.status === 'REJECTED'
              return (
                <tr key={req.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 font-mono text-xs">{req.id.slice(0, 8)}...</td>
                  <td className="px-4 py-3 text-sm">
                    {release ? `${release.package_name} v${release.version}` : <span className="font-mono text-xs text-gray-400">{req.release_id.slice(0, 8)}...</span>}
                  </td>
                  <td className="px-4 py-3 text-xs font-mono text-gray-500">
                    {release?.image_ref ?? '-'}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    {edge ? `${edge.name} (${edge.region})` : <span className="font-mono text-xs text-gray-400">{req.edge_id.slice(0, 8)}...</span>}
                  </td>
                  <td className="px-4 py-3">
                    <Badge variant={statusVariant[req.status]}>{req.status}</Badge>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">
                    {showReason && req.decision_reason ? req.decision_reason : '-'}
                  </td>
                  <td className="px-4 py-3 text-gray-600">
                    {new Date(req.created_at).toLocaleString('ko-KR')}
                  </td>
                  <td className="px-4 py-3">
                    {req.status === 'PENDING' && (
                      <div className="flex gap-2">
                        <Button size="sm" onClick={() => setActionDialog({ type: 'approve', approval: req })}>
                          승인
                        </Button>
                        <Button size="sm" variant="destructive" onClick={() => setActionDialog({ type: 'reject', approval: req })}>
                          거부
                        </Button>
                      </div>
                    )}
                  </td>
                </tr>
              )
            })}
            {approvals.length === 0 && (
              <tr><td colSpan={8} className="px-4 py-8 text-center text-gray-400">승인 요청이 없습니다</td></tr>
            )}
          </tbody>
        </table>
      </div>

      <Pagination page={page} limit={LIMIT} total={total} onPageChange={setPage} />

      {/* 승인 요청 생성 다이얼로그 */}
      <Dialog open={createOpen} onClose={() => { setCreateOpen(false); setFormError(null) }} title="승인 요청 생성">
        <form onSubmit={handleCreate} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">릴리즈 *</label>
            <select
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-gray-400"
              value={form.release_id}
              onChange={(e) => setForm({ ...form, release_id: e.target.value })}
              required
            >
              <option value="">릴리즈 선택...</option>
              {releases.map((r) => (
                <option key={r.id} value={r.id}>{r.package_name} v{r.version}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">에지 노드 *</label>
            <select
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-gray-400"
              value={form.edge_id}
              onChange={(e) => setForm({ ...form, edge_id: e.target.value })}
              required
            >
              <option value="">에지 선택...</option>
              {edges.map((e) => (
                <option key={e.id} value={e.id}>{e.name} ({e.region})</option>
              ))}
            </select>
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="outline" onClick={() => { setCreateOpen(false); setFormError(null) }}>취소</Button>
            <Button type="submit" disabled={createApproval.isPending}>
              {createApproval.isPending ? '요청 중...' : '요청'}
            </Button>
          </div>
          {formError && <p className="text-sm text-red-600 mt-2">{formError}</p>}
        </form>
      </Dialog>

      {/* 승인/거부 다이얼로그 */}
      <Dialog
        open={actionDialog !== null}
        onClose={() => { setActionDialog(null); setReason(''); setActionError(null) }}
        title={actionDialog?.type === 'approve' ? '승인 확인' : '거부 확인'}
      >
        <div className="space-y-4">
          <p className="text-sm text-gray-600">
            {actionDialog?.type === 'approve'
              ? '이 배포 요청을 승인하시겠습니까?'
              : '이 배포 요청을 거부하시겠습니까?'}
          </p>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">사유 *</label>
            <Textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="사유를 입력하세요..."
              rows={3}
              required
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="outline" onClick={() => { setActionDialog(null); setReason(''); setActionError(null) }}>취소</Button>
            <Button
              variant={actionDialog?.type === 'reject' ? 'destructive' : 'default'}
              onClick={handleAction}
              disabled={!reason || approve.isPending || reject.isPending}
            >
              {actionDialog?.type === 'approve' ? '승인' : '거부'}
            </Button>
          </div>
          {actionError && <p className="text-sm text-red-600 mt-2">{actionError}</p>}
        </div>
      </Dialog>
    </div>
  )
}
