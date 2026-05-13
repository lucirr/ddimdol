package domain

import (
	"time"

	"github.com/google/uuid"
)

type SessionStatus string

const (
	SessionStatusPendingApproval SessionStatus = "PENDING_APPROVAL"
	SessionStatusActive          SessionStatus = "ACTIVE"
	SessionStatusExpired         SessionStatus = "EXPIRED"
	SessionStatusTerminated      SessionStatus = "TERMINATED"
)

type RemoteSession struct {
	ID               uuid.UUID      `json:"id" db:"id"`
	EdgeID           uuid.UUID      `json:"edge_id" db:"edge_id"`
	OperatorID       uuid.UUID      `json:"operator_id" db:"operator_id"`
	Reason           string         `json:"reason" db:"reason"`
	TicketRef        string         `json:"ticket_ref" db:"ticket_ref"`
	Status           SessionStatus  `json:"status" db:"status"`
	ApprovedBy       *uuid.UUID     `json:"approved_by" db:"approved_by"`
	TokenJTI         *string        `json:"token_jti" db:"token_jti"`
	TTLSeconds       int            `json:"ttl_seconds" db:"ttl_seconds"`
	WhitelistEntries []any          `json:"whitelist_entries" db:"whitelist_entries"`
	RecordingURI     string         `json:"recording_uri" db:"recording_uri"`
	ActivatedAt      *time.Time     `json:"activated_at" db:"activated_at"`
	ExpiresAt        *time.Time     `json:"expires_at" db:"expires_at"`
	TerminatedAt     *time.Time     `json:"terminated_at" db:"terminated_at"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
}
