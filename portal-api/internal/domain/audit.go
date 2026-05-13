package domain

import (
	"net"
	"time"

	"github.com/google/uuid"
)

type ActorType string

const (
	ActorTypeUser   ActorType = "USER"
	ActorTypeAgent  ActorType = "AGENT"
	ActorTypeSystem ActorType = "SYSTEM"
)

type AuditOutcome string

const (
	AuditOutcomeSuccess AuditOutcome = "SUCCESS"
	AuditOutcomeFailure AuditOutcome = "FAILURE"
)

type AuditLog struct {
	ID           int64          `json:"id" db:"id"`
	Ts           time.Time      `json:"ts" db:"ts"`
	ActorID      *uuid.UUID     `json:"actor_id" db:"actor_id"`
	ActorType    ActorType      `json:"actor_type" db:"actor_type"`
	Action       string         `json:"action" db:"action"`
	ResourceType string         `json:"resource_type" db:"resource_type"`
	ResourceID   string         `json:"resource_id" db:"resource_id"`
	Outcome      AuditOutcome   `json:"outcome" db:"outcome"`
	RequestID    string         `json:"request_id" db:"request_id"`
	ClientIP     *net.IP        `json:"client_ip" db:"client_ip"`
	Metadata     map[string]any `json:"metadata" db:"metadata"`
	HashPrev     []byte         `json:"hash_prev" db:"hash_prev"`
	HashSelf     []byte         `json:"hash_self" db:"hash_self"`
}
