package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Dataset struct {
	ID            string    `db:"id"              json:"id"`
	Name          string    `db:"name"            json:"name"`
	PrimaryKeyCol string    `db:"primary_key_col" json:"primary_key_col"`
	CreatedBy     string    `db:"created_by"      json:"created_by"`
	CreatedAt     time.Time `db:"created_at"      json:"created_at"`
}

type DatasetSummary struct {
	Dataset
	SnapshotCount int        `json:"snapshot_count"`
	LastUpdated   *time.Time `json:"last_updated"`
}

type CreateDatasetParams struct {
	Name          string
	PrimaryKeyCol string
	CreatedBy     string
}

func CreateDataset(ctx context.Context, pool *pgxpool.Pool, p CreateDatasetParams) (*Dataset, error) {
	var d Dataset
	err := pool.QueryRow(ctx, `
		INSERT INTO datasets (name, primary_key_col, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, name, primary_key_col, created_by, created_at
	`, p.Name, p.PrimaryKeyCol, p.CreatedBy).Scan(
		&d.ID, &d.Name, &d.PrimaryKeyCol, &d.CreatedBy, &d.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateDataset: %w", err)
	}
	return &d, nil
}

func GetDataset(ctx context.Context, pool *pgxpool.Pool, id string) (*Dataset, error) {
	var d Dataset
	err := pool.QueryRow(ctx, `
		SELECT id, name, primary_key_col, created_by, created_at
		FROM datasets WHERE id = $1
	`, id).Scan(&d.ID, &d.Name, &d.PrimaryKeyCol, &d.CreatedBy, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("GetDataset: %w", err)
	}
	return &d, nil
}

func ListDatasets(ctx context.Context, pool *pgxpool.Pool, ownerID string) ([]DatasetSummary, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			d.id, d.name, d.primary_key_col, d.created_by, d.created_at,
			COUNT(s.id)             AS snapshot_count,
			MAX(s.created_at)       AS last_updated
		FROM datasets d
		LEFT JOIN snapshots s ON s.dataset_id = d.id
		WHERE d.created_by = $1
		GROUP BY d.id
		ORDER BY d.created_at DESC
	`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("ListDatasets: %w", err)
	}
	defer rows.Close()

	var out []DatasetSummary
	for rows.Next() {
		var ds DatasetSummary
		if err := rows.Scan(
			&ds.ID, &ds.Name, &ds.PrimaryKeyCol, &ds.CreatedBy, &ds.CreatedAt,
			&ds.SnapshotCount, &ds.LastUpdated,
		); err != nil {
			return nil, err
		}
		out = append(out, ds)
	}
	return out, rows.Err()
}
