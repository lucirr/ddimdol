export type EdgeStatus = 'UP' | 'DOWN' | 'DEGRADED' | 'UNKNOWN'

export interface EdgeNode {
  id: string
  name: string
  region: string
  tenant_id: string
  status: EdgeStatus
  last_heartbeat_at: string | null
  agent_version: string
  k8s_version: string
  capabilities: Record<string, unknown>
  labels: Record<string, unknown>
  created_at: string
  updated_at: string
}
