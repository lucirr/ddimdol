package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/didimdol/portal-api/internal/domain"
	"github.com/didimdol/portal-api/internal/repository"
)

type AuditRepository struct {
	db *sqlx.DB
}

func NewAuditRepository(db *sqlx.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Save persists an audit log entry.
// It computes hash_self = SHA256(ts + action + outcome) before inserting.
func (r *AuditRepository) Save(ctx context.Context, log *domain.AuditLog) error {
	h := sha256.New()
	h.Write([]byte(log.Ts.String() + log.Action + string(log.Outcome)))
	log.HashSelf = h.Sum(nil)

	metadata, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf("audit Save marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO audit_logs
			(ts, actor_id, actor_type, action, resource_type, resource_id,
			 outcome, request_id, client_ip, metadata, hash_prev, hash_self)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`,
		log.Ts, log.ActorID, log.ActorType, log.Action,
		log.ResourceType, log.ResourceID, log.Outcome,
		log.RequestID, log.ClientIP, metadata,
		log.HashPrev, log.HashSelf,
	)
	if err != nil {
		return fmt.Errorf("audit Save: %w", err)
	}
	return nil
}

func (r *AuditRepository) FindAll(ctx context.Context, filter repository.AuditFilter) ([]*domain.AuditLog, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 500
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, ts, actor_id, actor_type, action, resource_type, resource_id,
		       outcome, request_id, client_ip, metadata, hash_prev, hash_self
		FROM audit_logs
		WHERE ($1::text IS NULL OR actor_id::text = $1)
		  AND ($2::text IS NULL OR resource_type = $2)
		  AND ($3::text IS NULL OR resource_id = $3)
		ORDER BY ts DESC
		LIMIT $4 OFFSET $5
	`, filter.ActorID, filter.ResourceType, filter.ResourceID, limit, filter.Offset)
	if err != nil {
		return nil, fmt.Errorf("audit FindAll: %w", err)
	}
	defer rows.Close()

	logs := make([]*domain.AuditLog, 0)
	for rows.Next() {
		l, err := scanAuditLog(rows)
		if err != nil {
			return nil, fmt.Errorf("audit FindAll scan: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func scanAuditLog(s scanner) (*domain.AuditLog, error) {
	var l domain.AuditLog
	var metadata []byte
	err := s.Scan(
		&l.ID, &l.Ts, &l.ActorID, &l.ActorType, &l.Action,
		&l.ResourceType, &l.ResourceID, &l.Outcome,
		&l.RequestID, &l.ClientIP, &metadata,
		&l.HashPrev, &l.HashSelf,
	)
	if err != nil {
		return nil, err
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &l.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	return &l, nil
}
