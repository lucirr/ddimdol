package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/didimdol/portal-api/internal/domain"
)

// PageResult holds a page of items alongside the total count across all pages.
type PageResult[T any] struct {
	Items []*T
	Total int
}

type EdgeRepository interface {
	FindAll(ctx context.Context) ([]*domain.EdgeNode, error)
	FindPaged(ctx context.Context, page, limit int) (PageResult[domain.EdgeNode], error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.EdgeNode, error)
	Save(ctx context.Context, edge *domain.EdgeNode) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.EdgeStatus) error
}

type ReleaseRepository interface {
	FindAll(ctx context.Context) ([]*domain.Release, error)
	FindPaged(ctx context.Context, page, limit int) (PageResult[domain.Release], error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Release, error)
	Save(ctx context.Context, release *domain.Release) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ReleaseStatus) error
}

type ApprovalRepository interface {
	FindAll(ctx context.Context) ([]*domain.ApprovalRequest, error)
	FindPaged(ctx context.Context, page, limit int) (PageResult[domain.ApprovalRequest], error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.ApprovalRequest, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*domain.ApprovalRequest, error)
	Save(ctx context.Context, approval *domain.ApprovalRequest) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ApprovalStatus, reason string, version int) error
}

type DeploymentRepository interface {
	FindByApprovalID(ctx context.Context, approvalID uuid.UUID) ([]*domain.DeploymentRecord, error)
	Save(ctx context.Context, record *domain.DeploymentRecord) error
	UpdatePhase(ctx context.Context, id uuid.UUID, phase domain.DeploymentPhase, progressPct int16) error
}

type SessionRepository interface {
	FindAll(ctx context.Context) ([]*domain.RemoteSession, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.RemoteSession, error)
	Save(ctx context.Context, session *domain.RemoteSession) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.SessionStatus) error
}

type AuditRepository interface {
	FindAll(ctx context.Context, filter AuditFilter) ([]*domain.AuditLog, error)
	Save(ctx context.Context, log *domain.AuditLog) error
}

type AuditFilter struct {
	ActorID      *string
	ResourceType *string
	ResourceID   *string
	Limit        int
	Offset       int
}
