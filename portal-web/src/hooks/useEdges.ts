import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { EdgeNode } from '@/types/edge'

interface EdgePage {
  data: EdgeNode[]
  meta: { total: number; page: number; limit: number }
}

export function useEdges(page = 1, limit = 20) {
  return useQuery<EdgePage>({
    queryKey: ['edges', page, limit],
    queryFn: async () => {
      const { data } = await api.get('/edges', { params: { page, limit } })
      if (data.meta) return data
      return { data: data.data ?? data, meta: { total: (data.data ?? data).length, page, limit } }
    },
  })
}

export function useEdge(id: string) {
  return useQuery<EdgeNode>({
    queryKey: ['edges', id],
    queryFn: async () => {
      const { data } = await api.get(`/edges/${id}`)
      return data.data ?? data
    },
    enabled: !!id,
  })
}

export function useCreateEdge() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload: { name: string; region: string; tenant_id?: string }) =>
      api.post('/edges', payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['edges'] }),
  })
}
