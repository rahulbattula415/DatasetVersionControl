package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Snapshot struct {
	ID           string    `db:"id" json:"id"`
	DatasetID    string    `db:"dataset_id" json:"dataset_id"`
	ParentID     *string   `db:"parent_id" json:"parent_id"` // pointer = nullable
	SnapshotHash string    `db:"snapshot_hash" json:"snapshot_hash"`
	FileHash     string    `db:"file_hash" json:"file_hash"`
	RowCount     int       `db:"row_count" json:"row_count"`
	Message      *string   `db:"message" json:"message"`
	CreatedBy    string    `db:"created_by" json:"created_by"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type CreateSnapshotParams struct {
	DatasetID    string
	ParentID     *string
	SnapshotHash string
	FileHash     string
	RowCount     int
	Message      *string
	CreatedBy    string
	Columns      []SnapshotColumn
}

type SnapshotColumn struct {
	ColumnName  string
	ColumnType  string
	ColumnIndex int
}

func CreateSnapshot(ctx context.Context, pool *pgxpool.Pool, p CreateSnapshotParams) (*Snapshot, error) {
	// Use a transaction so snapshot + columns are inserted atomically.
	// If the columns insert fails, the snapshot row is also rolled back.
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // no-op if tx.Commit() succeeds

	var s Snapshot
	err = tx.QueryRow(ctx, `
        INSERT INTO snapshots (dataset_id, parent_id, snapshot_hash, file_hash, row_count, message, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id, dataset_id, parent_id, snapshot_hash, file_hash, row_count, message, created_by, created_at
    `, p.DatasetID, p.ParentID, p.SnapshotHash, p.FileHash, p.RowCount, p.Message, p.CreatedBy).Scan(
		&s.ID, &s.DatasetID, &s.ParentID, &s.SnapshotHash, &s.FileHash,
		&s.RowCount, &s.Message, &s.CreatedBy, &s.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert snapshot: %w", err)
	}

	for _, col := range p.Columns {
		_, err := tx.Exec(ctx, `
            INSERT INTO snapshot_columns (snapshot_id, column_name, column_type, column_index)
            VALUES ($1, $2, $3, $4)
        `, s.ID, col.ColumnName, col.ColumnType, col.ColumnIndex)
		if err != nil {
			return nil, fmt.Errorf("insert column %s: %w", col.ColumnName, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &s, nil
}
