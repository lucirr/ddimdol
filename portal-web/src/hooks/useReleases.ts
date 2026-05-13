import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { Release } from '@/types/release'

interface ReleasePage {
  data: Release[]
  meta: { total: number; page: number; limit: number }
}

export function useReleases(page = 1, limit = 20) {
  return useQuery<ReleasePage>({
    queryKey: ['releases', page, limit],
    queryFn: async () => {
      const { data } = await api.get('/releases', { params: { page, limit } })
      if (data.meta) return data
      return { data: data.data ?? data, meta: { total: (data.data ?? data).length, page, limit } }
    },
  })
}

export function useCreateRelease() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload: {
      package_name: string
      version: string
      artifact_digest: string
      image_ref?: string
      sbom_uri?: string
      signed_by?: string
    }) => api.post('/releases', payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['releases'] }),
  })
}
