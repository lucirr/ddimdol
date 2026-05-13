package domain

import (
	"time"

	"github.com/google/uuid"
)

type ReleaseStatus string

const (
	ReleaseStatusDraft      ReleaseStatus = "DRAFT"
	ReleaseStatusScanned    ReleaseStatus = "SCANNED"
	ReleaseStatusSigned     ReleaseStatus = "SIGNED"
	ReleaseStatusPublished  ReleaseStatus = "PUBLISHED"
	ReleaseStatusDeprecated ReleaseStatus = "DEPRECATED"
)

type Release struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	PackageName    string         `json:"package_name" db:"package_name"`
	Version        string         `json:"version" db:"version"`
	ArtifactDigest string         `json:"artifact_digest" db:"artifact_digest"`
	ImageRef       string         `json:"image_ref" db:"image_ref"`
	SbomURI        string         `json:"sbom_uri" db:"sbom_uri"`
	CveReport      map[string]any `json:"cve_report" db:"cve_report"`
	Signature      string         `json:"signature" db:"signature"`
	SignedBy       string         `json:"signed_by" db:"signed_by"`
	Status         ReleaseStatus  `json:"status" db:"status"`
	PublishedAt    *time.Time     `json:"published_at" db:"published_at"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
}
