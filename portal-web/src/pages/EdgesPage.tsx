import { useState } from 'react'
import { useEdges, useCreateEdge } from '@/hooks/useEdges'
import { EdgeStatusBadge } from '@/components/EdgeStatusBadge'
import { Button } from '@/components/ui/button'
import { Dialog } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Pagination } from '@/components/ui/pagination'

const LIMIT = 20

export default function EdgesPage() {
  const [page, setPage] = useState(1)
  const { data, isLoading, isError } = useEdges(page, LIMIT)
  const edges = data?.data ?? []
  const total = data?.meta?.total ?? 0
  const createEdge = useCreateEdge()
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState({ name: '', region: '' })
  const [formError, setFormError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError(null)
    try {
      await createEdge.mutateAsync(form)
      setOpen(false)
      setForm({ name: '', region: '' })
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : '요청 처리 중 오류가 발생했습니다.')
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
        <h2 className="text-2xl font-bold">에지 노드</h2>
        <Button onClick={() => setOpen(true)}>+ 에지 등록</Button>
      </div>

      <div className="bg-white rounded-lg border">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b">
            <tr>
              <th className="px-4 py-3 text-left font-medium text-gray-600">이름</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">리전</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">상태</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">에이전트 버전</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">마지막 하트비트</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {edges.map((edge) => (
              <tr key={edge.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 font-medium">{edge.name}</td>
                <td className="px-4 py-3 text-gray-600">{edge.region}</td>
                <td className="px-4 py-3"><EdgeStatusBadge status={edge.status} /></td>
                <td className="px-4 py-3 text-gray-600">{edge.agent_version || '-'}</td>
                <td className="px-4 py-3 text-gray-600">
                  {edge.last_heartbeat_at
                    ? new Date(edge.last_heartbeat_at).toLocaleString('ko-KR')
                    : '-'}
                </td>
              </tr>
            ))}
            {edges.length === 0 && (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-400">등록된 에지 노드가 없습니다</td></tr>
            )}
          </tbody>
        </table>
      </div>

      <Pagination page={page} limit={LIMIT} total={total} onPageChange={setPage} />

      <Dialog open={open} onClose={() => { setOpen(false); setFormError(null) }} title="에지 등록">
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">에지 이름 *</label>
            <Input
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="예: edge-daejeon-01"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">리전 *</label>
            <Input
              value={form.region}
              onChange={(e) => setForm({ ...form, region: e.target.value })}
              placeholder="예: daejeon"
              required
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="outline" onClick={() => { setOpen(false); setFormError(null) }}>취소</Button>
            <Button type="submit" disabled={createEdge.isPending}>
              {createEdge.isPending ? '등록 중...' : '등록'}
            </Button>
          </div>
          {formError && <p className="text-sm text-red-600 mt-2">{formError}</p>}
        </form>
      </Dialog>
    </div>
  )
}
