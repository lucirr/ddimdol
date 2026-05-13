package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/didimdol/portal-api/internal/domain"
	"github.com/didimdol/portal-api/internal/repository"
)

type ApprovalService struct {
	approvals repository.ApprovalRepository
}

func NewApprovalService(approvals repository.ApprovalRepository) *ApprovalService {
	return &ApprovalService{approvals: approvals}
}

// Approve transitions an approval from PENDING to APPROVED.
// TODO: Implement optimistic locking via version field.
func (s *ApprovalService) Approve(ctx context.Context, id uuid.UUID, decisionBy uuid.UUID, reason string) error {
	approval, err := s.approvals.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find approval: %w", err)
	}
	if approval.Status != domain.ApprovalStatusPending {
		return fmt.Errorf("approval %s is not in PENDING state", id)
	}
	return s.approvals.UpdateStatus(ctx, id, domain.ApprovalStatusApproved, reason, approval.Version)
}

// Reject transitions an approval from PENDING to REJECTED.
func (s *ApprovalService) Reject(ctx context.Context, id uuid.UUID, decisionBy uuid.UUID, reason string) error {
	approval, err := s.approvals.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find approval: %w", err)
	}
	if approval.Status != domain.ApprovalStatusPending {
		return fmt.Errorf("approval %s is not in PENDING state", id)
	}
	return s.approvals.UpdateStatus(ctx, id, domain.ApprovalStatusRejected, reason, approval.Version)
}

// Defer transitions an approval from PENDING to DEFERRED.
func (s *ApprovalService) Defer(ctx context.Context, id uuid.UUID) error {
	approval, err := s.approvals.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find approval: %w", err)
	}
	if approval.Status != domain.ApprovalStatusPending {
		return fmt.Errorf("approval %s is not in PENDING state", id)
	}
	return s.approvals.UpdateStatus(ctx, id, domain.ApprovalStatusDeferred, "", approval.Version)
}
