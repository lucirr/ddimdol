package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/didimdol/portal-api/internal/domain"
	"github.com/didimdol/portal-api/internal/repository"
)

// heartbeatDebounceWindow is the minimum interval between DB writes per edge.
// Heartbeats arriving within this window update only the in-memory timestamp.
const heartbeatDebounceWindow = 2 * time.Minute

type EdgeService struct {
	edges repository.EdgeRepository

	mu            sync.Mutex
	lastDBWrite   map[uuid.UUID]time.Time
}

func NewEdgeService(edges repository.EdgeRepository) *EdgeService {
	return &EdgeService{
		edges:       edges,
		lastDBWrite: make(map[uuid.UUID]time.Time),
	}
}

// RecordHeartbeat updates last_heartbeat_at in the DB at most once per
// heartbeatDebounceWindow per edge. Calls within the window are counted
// as alive but skipped for DB writes to reduce write pressure.
func (s *EdgeService) RecordHeartbeat(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()

	s.mu.Lock()
	last, seen := s.lastDBWrite[id]
	if seen && now.Sub(last) < heartbeatDebounceWindow {
		s.mu.Unlock()
		return nil
	}
	s.lastDBWrite[id] = now
	s.mu.Unlock()

	edge, err := s.edges.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find edge: %w", err)
	}

	updated := *edge
	updated.Status = domain.EdgeStatusUp
	updated.LastHeartbeatAt = &now
	updated.UpdatedAt = now

	if err := s.edges.Save(ctx, &updated); err != nil {
		// Roll back the in-memory timestamp so the next heartbeat retries.
		s.mu.Lock()
		if t, ok := s.lastDBWrite[id]; ok && t.Equal(now) {
			delete(s.lastDBWrite, id)
		}
		s.mu.Unlock()
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
