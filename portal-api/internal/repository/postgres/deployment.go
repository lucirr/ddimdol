package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/didimdol/portal-api/internal/domain"
)

type DeploymentRepository struct {
	db *sqlx.DB
}

func NewDeploymentRepository(db *sqlx.DB) *DeploymentRepository {
	return &DeploymentRepository{db: db}
}

func (r *DeploymentRepository) FindByApprovalID(ctx context.Context, approvalID uuid.UUID) ([]*domain.DeploymentRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, approval_id, edge_id, release_id, phase, progress_pct,
		       error_code, error_message, started_at, finished_at
		FROM deployment_records
		WHERE approval_id = $1
		ORDER BY started_at DESC
	`, approvalID)
	if err != nil {
		return nil, fmt.Errorf("deployment FindByApprovalID: %w", err)
	}
	defer rows.Close()

	records := make([]*domain.DeploymentRecord, 0)
	for rows.Next() {
		var d domain.DeploymentRecord
		if err := rows.Scan(
			&d.ID, &d.ApprovalID, &d.EdgeID, &d.ReleaseID, &d.Phase, &d.ProgressPct,
			&d.ErrorCode, &d.ErrorMessage, &d.StartedAt, &d.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("deployment scan: %w", err)
		}
		records = append(records, &d)
	}
	return records, rows.Err()
}

func (r *DeploymentRepository) Save(ctx context.Context, rec *domain.DeploymentRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO deployment_records
			(id, approval_id, edge_id, release_id, phase, progress_pct,
			 error_code, error_message, started_at, finished_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO UPDATE SET
			phase         = EXCLUDED.phase,
			progress_pct  = EXCLUDED.progress_pct,
			error_code    = EXCLUDED.error_code,
			error_message = EXCLUDED.error_message,
			finished_at   = EXCLUDED.finished_at
	`,
		rec.ID, rec.ApprovalID, rec.EdgeID, rec.ReleaseID, rec.Phase, rec.ProgressPct,
		rec.ErrorCode, rec.ErrorMessage, rec.StartedAt, rec.FinishedAt,
	)
	if err != nil {
		return fmt.Errorf("deployment Save: %w", err)
	}
	return nil
}

func (r *DeploymentRepository) UpdatePhase(ctx context.Context, id uuid.UUID, phase domain.DeploymentPhase, progressPct int16) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE deployment_records SET phase = $1, progress_pct = $2 WHERE id = $3
	`, phase, progressPct, id)
	if err != nil {
		return fmt.Errorf("deployment UpdatePhase: %w", err)
	}
	return nil
}
