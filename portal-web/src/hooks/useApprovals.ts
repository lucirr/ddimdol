import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { ApprovalRequest, ApprovalStatus } from '@/types/approval'

interface ApprovalPage {
  data: ApprovalRequest[]
  meta: { total: number; page: number; limit: number }
}

export function useApprovals(page = 1, limit = 20, status?: ApprovalStatus) {
  return useQuery<ApprovalPage>({
    queryKey: ['approvals', page, limit, status],
    queryFn: async () => {
      const { data } = await api.get('/approvals', { params: { page, limit, status } })
      if (data.meta) return data
      return { data: data.data ?? data, meta: { total: (data.data ?? data).length, page, limit } }
    },
  })
}

export function useCreateApproval() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload: { release_id: string; edge_id: string; idempotency_key?: string }) =>
      api.post('/approvals', payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['approvals'] }),
  })
}

export function useApproveRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      api.post(`/approvals/${id}/approve`, { reason }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['approvals'] }),
  })
}

export function useRejectRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      api.post(`/approvals/${id}/reject`, { reason }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['approvals'] }),
  })
}
