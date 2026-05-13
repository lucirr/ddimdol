export type DeploymentPhase = 'PENDING' | 'PULLING' | 'DEPLOYING' | 'COMPLETED' | 'FAILED' | 'ROLLED_BACK'

export interface DeploymentRecord {
  id: string
  approval_id: string
  edge_id: string
  release_id: string
  phase: DeploymentPhase
  progress_pct: number
  image_ref: string
  error_code?: string
  error_message?: string
  started_at: string
  completed_at?: string
}
