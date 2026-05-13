package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/didimdol/portal-api/internal/domain"
	"github.com/didimdol/portal-api/internal/repository"
)

type EdgeService struct {
	edges repository.EdgeRepository
}

func NewEdgeService(edges repository.EdgeRepository) *EdgeService {
	return &EdgeService{edges: edges}
}

// RecordHeartbeat updates the edge status and last_heartbeat_at timestamp.
func (s *EdgeService) RecordHeartbeat(ctx context.Context, id uuid.UUID) error {
	edge, err := s.edges.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find edge: %w", err)
	}

	now := time.Now().UTC()
	updated := *edge
	updated.Status = domain.EdgeStatusUp
	updated.LastHeartbeatAt = &now
	updated.UpdatedAt = now

	if err := s.edges.Save(ctx, &updated); err != nil {
		return fmt.Errorf("failed to save edge heartbeat: %w", err)
	}
	return nil
}

// MarkDown sets the edge status to DOWN.
func (s *EdgeService) MarkDown(ctx context.Context, id uuid.UUID) error {
	if err := s.edges.UpdateStatus(ctx, id, domain.EdgeStatusDown); err != nil {
		return fmt.Errorf("failed to mark edge down: %w", err)
	}
	return nil
}
