package domain

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentPhase string

const (
	DeploymentPhaseDownloading  DeploymentPhase = "DOWNLOADING"
	DeploymentPhaseApplying     DeploymentPhase = "APPLYING"
	DeploymentPhaseHealthcheck  DeploymentPhase = "HEALTHCHECK"
	DeploymentPhaseCompleted    DeploymentPhase = "COMPLETED"
	DeploymentPhaseFailed       DeploymentPhase = "FAILED"
	DeploymentPhaseRolledBack   DeploymentPhase = "ROLLED_BACK"
)

type DeploymentRecord struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	ApprovalID   uuid.UUID       `json:"approval_id" db:"approval_id"`
	EdgeID       uuid.UUID       `json:"edge_id" db:"edge_id"`
	ReleaseID    uuid.UUID       `json:"release_id" db:"release_id"`
	Phase        DeploymentPhase `json:"phase" db:"phase"`
	ProgressPct  int16           `json:"progress_pct" db:"progress_pct"`
	ErrorCode    string          `json:"error_code" db:"error_code"`
	ErrorMessage string          `json:"error_message" db:"error_message"`
	StartedAt    time.Time       `json:"started_at" db:"started_at"`
	FinishedAt   *time.Time      `json:"finished_at" db:"finished_at"`
}
