export type ActorType = 'USER' | 'AGENT' | 'SYSTEM'
export type AuditOutcome = 'SUCCESS' | 'FAILURE'

export interface AuditLog {
  id: number
  ts: string
  actor_id: string | null
  actor_type: ActorType
  action: string
  resource_type: string
  resource_id: string
  outcome: AuditOutcome
  request_id: string
  client_ip: string | null
  metadata: Record<string, unknown>
}
