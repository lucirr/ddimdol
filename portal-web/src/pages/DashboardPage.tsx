import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { useEdges } from '@/hooks/useEdges'
import { useEdgeWebSocket } from '@/hooks/useEdgeWebSocket'

export default function DashboardPage() {
  useEdgeWebSocket()
  const { data, isLoading, isError } = useEdges(1, 1000)
  const edges = data?.data ?? []

  const stats = {
    total: data?.meta?.total ?? edges.length,
    up: edges.filter((e) => e.status === 'UP').length,
    down: edges.filter((e) => e.status === 'DOWN').length,
    degraded: edges.filter((e) => e.status === 'DEGRADED').length,
  }

  if (isLoading) return <div className="text-gray-500">로딩 중...</div>

  if (isError) return (
    <div className="flex items-center justify-center h-64 text-red-500">
      <p>데이터를 불러오는 중 오류가 발생했습니다. 페이지를 새로고침 해주세요.</p>
    </div>
  )

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">대시보드</h2>
      <div className="grid grid-cols-4 gap-4">
        <Card>
          <CardHeader><CardTitle>전체 에지</CardTitle></CardHeader>
          <CardContent><p className="text-3xl font-bold">{stats.total}</p></CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-green-700">정상</CardTitle></CardHeader>
          <CardContent><p className="text-3xl font-bold text-green-600">{stats.up}</p></CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-red-700">장애</CardTitle></CardHeader>
          <CardContent><p className="text-3xl font-bold text-red-600">{stats.down}</p></CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="text-yellow-700">성능 저하</CardTitle></CardHeader>
          <CardContent><p className="text-3xl font-bold text-yellow-600">{stats.degraded}</p></CardContent>
        </Card>
      </div>

      <div className="bg-white rounded-lg border mt-4">
        <div className="px-4 py-3 border-b">
          <h3 className="font-semibold text-gray-700">에지 노드 현황</h3>
        </div>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-2 text-left text-gray-600">이름</th>
              <th className="px-4 py-2 text-left text-gray-600">지역</th>
              <th className="px-4 py-2 text-left text-gray-600">상태</th>
              <th className="px-4 py-2 text-left text-gray-600">마지막 Heartbeat</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {edges.map((e) => (
              <tr key={e.id}>
                <td className="px-4 py-2 font-medium">{e.name}</td>
                <td className="px-4 py-2 text-gray-600">{e.region}</td>
                <td className="px-4 py-2">
                  <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${
                    e.status === 'UP' ? 'bg-green-100 text-green-700' :
                    e.status === 'DOWN' ? 'bg-red-100 text-red-700' :
                    'bg-yellow-100 text-yellow-700'
                  }`}>{e.status}</span>
                </td>
                <td className="px-4 py-2 text-gray-500 text-xs">
                  {e.last_heartbeat_at ? new Date(e.last_heartbeat_at).toLocaleString('ko-KR') : '-'}
                </td>
              </tr>
            ))}
            {edges.length === 0 && (
              <tr><td colSpan={4} className="px-4 py-8 text-center text-gray-400">에지 노드가 없습니다</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
