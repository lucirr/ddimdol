export type ApprovalStatus =
  | 'PENDING'
  | 'APPROVED'
  | 'REJECTED'
  | 'DEFERRED'
  | 'APPLIED'
  | 'ROLLED_BACK'
  | 'EXPIRED'

export interface ApprovalRequest {
  id: string
  release_id: string
  edge_id: string
  requested_by: string
  status: ApprovalStatus
  decision_by: string | null
  decision_reason: string
  scheduled_at: string | null
  deferred_until: string | null
  idempotency_key: string
  version: number
  created_at: string
  updated_at: string
}
