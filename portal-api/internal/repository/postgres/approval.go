package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/didimdol/portal-api/internal/domain"
	"github.com/didimdol/portal-api/internal/repository"
)

type ApprovalRepository struct {
	db *sqlx.DB
}

func NewApprovalRepository(db *sqlx.DB) *ApprovalRepository {
	return &ApprovalRepository{db: db}
}

func (r *ApprovalRepository) FindAll(ctx context.Context) ([]*domain.ApprovalRequest, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, release_id, edge_id, requested_by, status,
		       decision_by, decision_reason, scheduled_at, deferred_until,
		       idempotency_key, version, created_at, updated_at
		FROM approval_requests
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("approval FindAll: %w", err)
	}
	defer rows.Close()

	approvals := make([]*domain.ApprovalRequest, 0)
	for rows.Next() {
		a, err := scanApproval(rows)
		if err != nil {
			return nil, fmt.Errorf("approval FindAll scan: %w", err)
		}
		approvals = append(approvals, a)
	}
	return approvals, rows.Err()
}

func (r *ApprovalRepository) FindPaged(ctx context.Context, page, limit int) (repository.PageResult[domain.ApprovalRequest], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	rows, err := r.db.QueryContext(ctx, `
		SELECT COUNT(*) OVER() AS total,
		       id, release_id, edge_id, requested_by, status,
		       decision_by, decision_reason, scheduled_at, deferred_until,
		       idempotency_key, version, created_at, updated_at
		FROM approval_requests
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return repository.PageResult[domain.ApprovalRequest]{}, fmt.Errorf("approval FindPaged: %w", err)
	}
	defer rows.Close()

	var total int
	items := make([]*domain.ApprovalRequest, 0)
	for rows.Next() {
		var a domain.ApprovalRequest
		if err := rows.Scan(
			&total,
			&a.ID, &a.ReleaseID, &a.EdgeID, &a.RequestedBy, &a.Status,
			&a.DecisionBy, &a.DecisionReason, &a.ScheduledAt, &a.DeferredUntil,
			&a.IdempotencyKey, &a.Version, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return repository.PageResult[domain.ApprovalRequest]{}, fmt.Errorf("approval FindPaged scan: %w", err)
		}
		items = append(items, &a)
	}
	return repository.PageResult[domain.ApprovalRequest]{Items: items, Total: total}, rows.Err()
}

func (r *ApprovalRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.ApprovalRequest, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, release_id, edge_id, requested_by, status,
		       decision_by, decision_reason, scheduled_at, deferred_until,
		       idempotency_key, version, created_at, updated_at
		FROM approval_requests
		WHERE id = $1
	`, id)

	a, err := scanApproval(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("approval %s not found: %w", id, err)
		}
		return nil, fmt.Errorf("approval FindByID: %w", err)
	}
	return a, nil
}

func (r *ApprovalRepository) FindByIdempotencyKey(ctx context.Context, key string) (*domain.ApprovalRequest, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, release_id, edge_id, requested_by, status,
		       decision_by, decision_reason, scheduled_at, deferred_until,
		       idempotency_key, version, created_at, updated_at
		FROM approval_requests
		WHERE idempotency_key = $1
	`, key)
	a, err := scanApproval(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("approval FindByIdempotencyKey: %w", err)
	}
	return a, nil
}

func (r *ApprovalRepository) Save(ctx context.Context, req *domain.ApprovalRequest) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO approval_requests
			(id, release_id, edge_id, requested_by, status,
			 decision_by, decision_reason, scheduled_at, deferred_until,
			 idempotency_key, version, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (id) DO UPDATE SET
			status          = EXCLUDED.status,
			decision_by     = EXCLUDED.decision_by,
			decision_reason = EXCLUDED.decision_reason,
			scheduled_at    = EXCLUDED.scheduled_at,
			deferred_until  = EXCLUDED.deferred_until,
			version         = EXCLUDED.version,
			updated_at      = EXCLUDED.updated_at
	`,
		req.ID, req.ReleaseID, req.EdgeID, req.RequestedBy, req.Status,
		req.DecisionBy, req.DecisionReason, req.ScheduledAt, req.DeferredUntil,
		req.IdempotencyKey, req.Version, req.CreatedAt, req.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("approval Save: %w", err)
	}
	return nil
}

// UpdateStatus applies an optimistic-lock update using the provided version.
// It increments version on success and returns an error if no row was matched.
func (r *ApprovalRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ApprovalStatus, reason string, version int) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE approval_requests
		SET status = $1, decision_reason = $2, version = version + 1, updated_at = NOW()
		WHERE id = $3 AND version = $4
	`, status, reason, id, version)
	if err != nil {
		return fmt.Errorf("approval UpdateStatus: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("approval UpdateStatus rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("approval %s not found or version conflict (expected version %d)", id, version)
	}
	return nil
}

func scanApproval(s scanner) (*domain.ApprovalRequest, error) {
	var a domain.ApprovalRequest
	err := s.Scan(
		&a.ID, &a.ReleaseID, &a.EdgeID, &a.RequestedBy, &a.Status,
		&a.DecisionBy, &a.DecisionReason, &a.ScheduledAt, &a.DeferredUntil,
		&a.IdempotencyKey, &a.Version, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}
