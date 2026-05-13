import { useState } from 'react'
import { useReleases, useCreateRelease } from '@/hooks/useReleases'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Pagination } from '@/components/ui/pagination'
import type { ReleaseStatus } from '@/types/release'

const statusVariant: Record<ReleaseStatus, 'success' | 'warning' | 'muted' | 'default' | 'danger'> = {
  DRAFT: 'muted',
  SCANNED: 'warning',
  SIGNED: 'default',
  PUBLISHED: 'success',
  DEPRECATED: 'danger',
}

const LIMIT = 20

export default function ReleasesPage() {
  const [page, setPage] = useState(1)
  const { data, isLoading, isError } = useReleases(page, LIMIT)
  const releases = data?.data ?? []
  const total = data?.meta?.total ?? 0
  const createRelease = useCreateRelease()
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState({ package_name: '', version: '', artifact_digest: '', image_ref: '', signed_by: '' })
  const [formError, setFormError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError(null)
    try {
      await createRelease.mutateAsync(form)
      setOpen(false)
      setForm({ package_name: '', version: '', artifact_digest: '', image_ref: '', signed_by: '' })
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
        <h2 className="text-2xl font-bold">릴리즈</h2>
        <Button onClick={() => setOpen(true)}>+ 릴리즈 등록</Button>
      </div>

      <div className="bg-white rounded-lg border">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b">
            <tr>
              <th className="px-4 py-3 text-left font-medium text-gray-600">패키지</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">버전</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">이미지</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">상태</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">CVE</th>
              <th className="px-4 py-3 text-left font-medium text-gray-600">발행일</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {releases.map((rel) => (
              <tr key={rel.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 font-medium">{rel.package_name}</td>
                <td className="px-4 py-3 font-mono">{rel.version}</td>
                <td className="px-4 py-3 font-mono text-xs max-w-[200px] truncate" title={rel.image_ref}>{rel.image_ref || '-'}</td>
                <td className="px-4 py-3">
                  <Badge variant={statusVariant[rel.status]}>{rel.status}</Badge>
                </td>
                <td className="px-4 py-3 text-xs">
                  <span className="text-red-600">C:{rel.cve_report?.critical ?? 0}</span>
                  {' '}<span className="text-orange-600">H:{rel.cve_report?.high ?? 0}</span>
                  {' '}<span className="text-yellow-600">M:{rel.cve_report?.medium ?? 0}</span>
                </td>
                <td className="px-4 py-3 text-gray-600">
                  {rel.published_at ? new Date(rel.published_at).toLocaleString('ko-KR') : '-'}
                </td>
              </tr>
            ))}
            {releases.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-400">릴리즈가 없습니다</td></tr>
            )}
          </tbody>
        </table>
      </div>

      <Pagination page={page} limit={LIMIT} total={total} onPageChange={setPage} />

      <Dialog open={open} onClose={() => { setOpen(false); setFormError(null) }} title="릴리즈 등록">
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">패키지명 *</label>
            <Input
              value={form.package_name}
              onChange={(e) => setForm({ ...form, package_name: e.target.value })}
              placeholder="예: edge-app"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">버전 *</label>
            <Input
              value={form.version}
              onChange={(e) => setForm({ ...form, version: e.target.value })}
              placeholder="예: 1.0.0"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">아티팩트 다이제스트 *</label>
            <Input
              value={form.artifact_digest}
              onChange={(e) => setForm({ ...form, artifact_digest: e.target.value })}
              placeholder="예: sha256:abc123..."
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">이미지 참조</label>
            <Input
              value={form.image_ref}
              onChange={(e) => setForm({ ...form, image_ref: e.target.value })}
              placeholder="예: harbor.internal/myapp:v1.0.0"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">서명자</label>
            <Input
              value={form.signed_by}
              onChange={(e) => setForm({ ...form, signed_by: e.target.value })}
              placeholder="예: ci-pipeline"
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="outline" onClick={() => { setOpen(false); setFormError(null) }}>취소</Button>
            <Button type="submit" disabled={createRelease.isPending}>
              {createRelease.isPending ? '등록 중...' : '등록'}
            </Button>
          </div>
          {formError && <p className="text-sm text-red-600 mt-2">{formError}</p>}
        </form>
      </Dialog>
    </div>
  )
}
