import { Badge } from '@/components/ui/badge'
import type { EdgeStatus } from '@/types/edge'

const statusConfig: Record<EdgeStatus, { label: string; variant: 'success' | 'danger' | 'warning' | 'muted' }> = {
  UP: { label: 'UP', variant: 'success' },
  DOWN: { label: 'DOWN', variant: 'danger' },
  DEGRADED: { label: 'DEGRADED', variant: 'warning' },
  UNKNOWN: { label: 'UNKNOWN', variant: 'muted' },
}

export function EdgeStatusBadge({ status }: { status: EdgeStatus }) {
  const { label, variant } = statusConfig[status]
  return <Badge variant={variant}>{label}</Badge>
}
