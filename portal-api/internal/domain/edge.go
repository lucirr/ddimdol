package domain

import (
	"time"

	"github.com/google/uuid"
)

type EdgeStatus string

const (
	EdgeStatusUp       EdgeStatus = "UP"
	EdgeStatusDown     EdgeStatus = "DOWN"
	EdgeStatusDegraded EdgeStatus = "DEGRADED"
	EdgeStatusUnknown  EdgeStatus = "UNKNOWN"
)

type EdgeNode struct {
	ID              uuid.UUID      `json:"id" db:"id"`
	Name            string         `json:"name" db:"name"`
	Region          string         `json:"region" db:"region"`
	TenantID        uuid.UUID      `json:"tenant_id" db:"tenant_id"`
	Status          EdgeStatus     `json:"status" db:"status"`
	LastHeartbeatAt *time.Time     `json:"last_heartbeat_at" db:"last_heartbeat_at"`
	AgentVersion    string         `json:"agent_version" db:"agent_version"`
	K8sVersion      string         `json:"k8s_version" db:"k8s_version"`
	Capabilities    map[string]any `json:"capabilities" db:"capabilities"`
	Labels          map[string]any `json:"labels" db:"labels"`
	PublicKey       string         `json:"public_key" db:"public_key"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
}
