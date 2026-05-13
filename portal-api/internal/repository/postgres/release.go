package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/didimdol/portal-api/internal/domain"
	"github.com/didimdol/portal-api/internal/repository"
)

type ReleaseRepository struct {
	db *sqlx.DB
}

func NewReleaseRepository(db *sqlx.DB) *ReleaseRepository {
	return &ReleaseRepository{db: db}
}

func (r *ReleaseRepository) FindAll(ctx context.Context) ([]*domain.Release, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, package_name, version, artifact_digest, image_ref, sbom_uri,
		       cve_report, signature, signed_by, status, published_at, created_at
		FROM releases
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("release FindAll: %w", err)
	}
	defer rows.Close()

	releases := make([]*domain.Release, 0)
	for rows.Next() {
		rel, err := scanRelease(rows)
		if err != nil {
			return nil, fmt.Errorf("release FindAll scan: %w", err)
		}
		releases = append(releases, rel)
	}
	return releases, rows.Err()
}

func (r *ReleaseRepository) FindPaged(ctx context.Context, page, limit int) (repository.PageResult[domain.Release], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	rows, err := r.db.QueryContext(ctx, `
		SELECT COUNT(*) OVER() AS total,
		       id, package_name, version, artifact_digest, image_ref, sbom_uri,
		       cve_report, signature, signed_by, status, published_at, created_at
		FROM releases
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return repository.PageResult[domain.Release]{}, fmt.Errorf("release FindPaged: %w", err)
	}
	defer rows.Close()

	var total int
	items := make([]*domain.Release, 0)
	for rows.Next() {
		var rel domain.Release
		var cveReport []byte
		if err := rows.Scan(
			&total,
			&rel.ID, &rel.PackageName, &rel.Version, &rel.ArtifactDigest, &rel.ImageRef, &rel.SbomURI,
			&cveReport, &rel.Signature, &rel.SignedBy, &rel.Status, &rel.PublishedAt, &rel.CreatedAt,
		); err != nil {
			return repository.PageResult[domain.Release]{}, fmt.Errorf("release FindPaged scan: %w", err)
		}
		if len(cveReport) > 0 {
			if err := json.Unmarshal(cveReport, &rel.CveReport); err != nil {
				return repository.PageResult[domain.Release]{}, fmt.Errorf("release FindPaged unmarshal cve_report: %w", err)
			}
		}
		items = append(items, &rel)
	}
	return repository.PageResult[domain.Release]{Items: items, Total: total}, rows.Err()
}

func (r *ReleaseRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Release, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, package_name, version, artifact_digest, image_ref, sbom_uri,
		       cve_report, signature, signed_by, status, published_at, created_at
		FROM releases
		WHERE id = $1
	`, id)

	rel, err := scanRelease(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("release %s not found: %w", id, err)
		}
		return nil, fmt.Errorf("release FindByID: %w", err)
	}
	return rel, nil
}

func (r *ReleaseRepository) Save(ctx context.Context, rel *domain.Release) error {
	cveReport, err := json.Marshal(rel.CveReport)
	if err != nil {
		return fmt.Errorf("release Save marshal cve_report: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO releases
			(id, package_name, version, artifact_digest, image_ref, sbom_uri,
			 cve_report, signature, signed_by, status, published_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (id) DO UPDATE SET
			package_name    = EXCLUDED.package_name,
			version         = EXCLUDED.version,
			artifact_digest = EXCLUDED.artifact_digest,
			image_ref       = EXCLUDED.image_ref,
			sbom_uri        = EXCLUDED.sbom_uri,
			cve_report      = EXCLUDED.cve_report,
			signature       = EXCLUDED.signature,
			signed_by       = EXCLUDED.signed_by,
			status          = EXCLUDED.status,
			published_at    = EXCLUDED.published_at
	`,
		rel.ID, rel.PackageName, rel.Version, rel.ArtifactDigest, rel.ImageRef, rel.SbomURI,
		cveReport, rel.Signature, rel.SignedBy, rel.Status, rel.PublishedAt, rel.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("release Save: %w", err)
	}
	return nil
}

func (r *ReleaseRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ReleaseStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE releases SET status = $1 WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("release UpdateStatus: %w", err)
	}
	return nil
}

func scanRelease(s scanner) (*domain.Release, error) {
	var rel domain.Release
	var cveReport []byte
	err := s.Scan(
		&rel.ID, &rel.PackageName, &rel.Version, &rel.ArtifactDigest, &rel.ImageRef, &rel.SbomURI,
		&cveReport, &rel.Signature, &rel.SignedBy, &rel.Status, &rel.PublishedAt, &rel.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(cveReport) > 0 {
		if err := json.Unmarshal(cveReport, &rel.CveReport); err != nil {
			return nil, fmt.Errorf("unmarshal cve_report: %w", err)
		}
	}
	return &rel, nil
}
