export type ReleaseStatus = 'DRAFT' | 'SCANNED' | 'SIGNED' | 'PUBLISHED' | 'DEPRECATED'

export interface Release {
  id: string
  package_name: string
  version: string
  artifact_digest: string
  image_ref: string
  sbom_uri: string
  cve_report: { critical: number; high: number; medium: number; low: number }
  signature: string
  signed_by: string
  status: ReleaseStatus
  published_at: string | null
  created_at: string
}
