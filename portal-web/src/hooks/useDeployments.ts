import { useApprovals } from '@/hooks/useApprovals'
import type { ApprovalRequest, ApprovalStatus } from '@/types/approval'

const DEPLOYMENT_STATUSES: ApprovalStatus[] = ['APPLIED', 'ROLLED_BACK', 'REJECTED']

export interface DeploymentRow {
  id: string
  approval_id: string
  release_id: string
  edge_id: string
  status: ApprovalStatus
  requested_by: string
  decision_by: string | null
  decision_reason: string
  requested_at: string
  decided_at: string
}

function toDeploymentRow(approval: ApprovalRequest): DeploymentRow {
  return {
    id: approval.id,
    approval_id: approval.id,
    release_id: approval.release_id,
    edge_id: approval.edge_id,
    status: approval.status,
    requested_by: approval.requested_by,
    decision_by: approval.decision_by,
    decision_reason: approval.decision_reason,
    requested_at: approval.created_at,
    decided_at: approval.updated_at,
  }
}

export function useDeployments(page = 1, limit = 20) {
  const query = useApprovals(page, limit)

  // TODO: Replace client-side filtering with a dedicated deployments API endpoint.
  // Currently we fetch all approvals and filter to deployment statuses, which means
  // the total count reflects approvals, not deployments. This is a temporary workaround.
  const deployments = (query.data?.data ?? [])
    .filter((a) => DEPLOYMENT_STATUSES.includes(a.status))
    .map(toDeploymentRow)
    .sort((a, b) => new Date(b.decided_at).getTime() - new Date(a.decided_at).getTime())

  return {
    ...query,
    data: deployments,
    meta: {
      ...query.data?.meta,
      total: deployments.length,  // use filtered count, not approval total
    },
  }
}
