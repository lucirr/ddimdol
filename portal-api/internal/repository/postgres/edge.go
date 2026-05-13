package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/didimdol/portal-api/internal/domain"
	"github.com/didimdol/portal-api/internal/repository"
)

type EdgeRepository struct {
	db *sqlx.DB
}

func NewEdgeRepository(db *sqlx.DB) *EdgeRepository {
	return &EdgeRepository{db: db}
}

func (r *EdgeRepository) FindAll(ctx context.Context) ([]*domain.EdgeNode, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, region, tenant_id, status, last_heartbeat_at,
		       agent_version, k8s_version, capabilities, labels, public_key,
		       created_at, updated_at
		FROM edge_nodes
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("edge FindAll: %w", err)
	}
	defer rows.Close()

	edges := make([]*domain.EdgeNode, 0)
	for rows.Next() {
		e, err := scanEdge(rows)
		if err != nil {
			return nil, fmt.Errorf("edge FindAll scan: %w", err)
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

func (r *EdgeRepository) FindPaged(ctx context.Context, page, limit int) (repository.PageResult[domain.EdgeNode], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	rows, err := r.db.QueryContext(ctx, `
		SELECT COUNT(*) OVER() AS total,
		       id, name, region, tenant_id, status, last_heartbeat_at,
		       agent_version, k8s_version, capabilities, labels, public_key,
		       created_at, updated_at
		FROM edge_nodes
		ORDER BY name
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return repository.PageResult[domain.EdgeNode]{}, fmt.Errorf("edge FindPaged: %w", err)
	}
	defer rows.Close()

	var total int
	items := make([]*domain.EdgeNode, 0)
	for rows.Next() {
		var e domain.EdgeNode
		var caps, labels []byte
		if err := rows.Scan(
			&total,
			&e.ID, &e.Name, &e.Region, &e.TenantID, &e.Status,
			&e.LastHeartbeatAt, &e.AgentVersion, &e.K8sVersion,
			&caps, &labels, &e.PublicKey,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return repository.PageResult[domain.EdgeNode]{}, fmt.Errorf("edge FindPaged scan: %w", err)
		}
		if len(caps) > 0 {
			if err := json.Unmarshal(caps, &e.Capabilities); err != nil {
				return repository.PageResult[domain.EdgeNode]{}, fmt.Errorf("edge FindPaged unmarshal capabilities: %w", err)
			}
		}
		if len(labels) > 0 {
			if err := json.Unmarshal(labels, &e.Labels); err != nil {
				return repository.PageResult[domain.EdgeNode]{}, fmt.Errorf("edge FindPaged unmarshal labels: %w", err)
			}
		}
		items = append(items, &e)
	}
	return repository.PageResult[domain.EdgeNode]{Items: items, Total: total}, rows.Err()
}

func (r *EdgeRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.EdgeNode, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, region, tenant_id, status, last_heartbeat_at,
		       agent_version, k8s_version, capabilities, labels, public_key,
		       created_at, updated_at
		FROM edge_nodes
		WHERE id = $1
	`, id)

	e, err := scanEdge(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("edge %s not found: %w", id, err)
		}
		return nil, fmt.Errorf("edge FindByID: %w", err)
	}
	return e, nil
}

func (r *EdgeRepository) Save(ctx context.Context, edge *domain.EdgeNode) error {
	caps, err := json.Marshal(edge.Capabilities)
	if err != nil {
		return fmt.Errorf("edge Save marshal capabilities: %w", err)
	}
	labels, err := json.Marshal(edge.Labels)
	if err != nil {
		return fmt.Errorf("edge Save marshal labels: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO edge_nodes
			(id, name, region, tenant_id, status, last_heartbeat_at,
			 agent_version, k8s_version, capabilities, labels, public_key,
			 created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (id) DO UPDATE SET
			name             = EXCLUDED.name,
			region           = EXCLUDED.region,
			status           = EXCLUDED.status,
			last_heartbeat_at = EXCLUDED.last_heartbeat_at,
			agent_version    = EXCLUDED.agent_version,
			k8s_version      = EXCLUDED.k8s_version,
			capabilities     = EXCLUDED.capabilities,
			labels           = EXCLUDED.labels,
			public_key       = EXCLUDED.public_key,
			updated_at       = EXCLUDED.updated_at
	`,
		edge.ID, edge.Name, edge.Region, edge.TenantID, edge.Status,
		edge.LastHeartbeatAt, edge.AgentVersion, edge.K8sVersion,
		caps, labels, edge.PublicKey,
		edge.CreatedAt, edge.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("edge Save: %w", err)
	}
	return nil
}

func (r *EdgeRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.EdgeStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE edge_nodes SET status = $1, updated_at = $2 WHERE id = $3`,
		status, time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("edge UpdateStatus: %w", err)
	}
	return nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanEdge(s scanner) (*domain.EdgeNode, error) {
	var e domain.EdgeNode
	var caps, labels []byte
	err := s.Scan(
		&e.ID, &e.Name, &e.Region, &e.TenantID, &e.Status,
		&e.LastHeartbeatAt, &e.AgentVersion, &e.K8sVersion,
		&caps, &labels, &e.PublicKey,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(caps) > 0 {
		if err := json.Unmarshal(caps, &e.Capabilities); err != nil {
			return nil, fmt.Errorf("unmarshal capabilities: %w", err)
		}
	}
	if len(labels) > 0 {
		if err := json.Unmarshal(labels, &e.Labels); err != nil {
			return nil, fmt.Errorf("unmarshal labels: %w", err)
		}
	}
	return &e, nil
}
