package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Dataset struct {
	ID            string    `db:"id" json:"id"`
	Name          string    `db:"name" json:"name"`
	PrimaryKeyCol string    `db:"primary_key_col" json:"primary_key_col"`
	CreatedBy     string    `db:"created_by" json:"created_by"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

type CreateDatasetParams struct {
	Name          string
	PrimaryKeyCol string
	CreatedBy     string
}

func CreateDataset(ctx context.Context, pool *pgxpool.Pool, p CreateDatasetParams) (*Dataset, error) {
	query := `
        INSERT INTO datasets (name, primary_key_col, created_by)
        VALUES ($1, $2, $3)
        RETURNING id, name, primary_key_col, created_by, created_at
    `
	var d Dataset
	err := pool.QueryRow(ctx, query, p.Name, p.PrimaryKeyCol, p.CreatedBy).Scan(
		&d.ID, &d.Name, &d.PrimaryKeyCol, &d.CreatedBy, &d.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateDataset: %w", err)
	}
	return &d, nil
}
