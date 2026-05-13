package domain

import (
	"time"

	"github.com/google/uuid"
)

type ApprovalStatus string

const (
	ApprovalStatusPending    ApprovalStatus = "PENDING"
	ApprovalStatusApproved   ApprovalStatus = "APPROVED"
	ApprovalStatusRejected   ApprovalStatus = "REJECTED"
	ApprovalStatusDeferred   ApprovalStatus = "DEFERRED"
	ApprovalStatusApplied    ApprovalStatus = "APPLIED"
	ApprovalStatusRolledBack ApprovalStatus = "ROLLED_BACK"
	ApprovalStatusExpired    ApprovalStatus = "EXPIRED"
)

type ApprovalRequest struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	ReleaseID      uuid.UUID      `json:"release_id" db:"release_id"`
	EdgeID         uuid.UUID      `json:"edge_id" db:"edge_id"`
	RequestedBy    uuid.UUID      `json:"requested_by" db:"requested_by"`
	Status         ApprovalStatus `json:"status" db:"status"`
	DecisionBy     *uuid.UUID     `json:"decision_by" db:"decision_by"`
	DecisionReason string         `json:"decision_reason" db:"decision_reason"`
	ScheduledAt    *time.Time     `json:"scheduled_at" db:"scheduled_at"`
	DeferredUntil  *time.Time     `json:"deferred_until" db:"deferred_until"`
	IdempotencyKey string         `json:"idempotency_key" db:"idempotency_key"`
	Version        int            `json:"version" db:"version"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`
}
