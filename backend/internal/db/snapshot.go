package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDuplicateSnapshot = errors.New("snapshot with this hash already exists")

type Snapshot struct {
	ID           string    `db:"id"            json:"id"`
	DatasetID    string    `db:"dataset_id"    json:"dataset_id"`
	ParentID     *string   `db:"parent_id"     json:"parent_id"`
	SnapshotHash string    `db:"snapshot_hash" json:"snapshot_hash"`
	FileHash     string    `db:"file_hash"     json:"file_hash"`
	RowCount     int       `db:"row_count"     json:"row_count"`
	Message      *string   `db:"message"       json:"message"`
	CreatedBy    string    `db:"created_by"    json:"created_by"`
	CreatedAt    time.Time `db:"created_at"    json:"created_at"`
}

type SnapshotWithColumns struct {
	Snapshot
	Columns []SnapshotColumn `json:"columns"`
}

type SnapshotColumn struct {
	ColumnName  string `json:"column_name"`
	ColumnType  string `json:"column_type"`
	ColumnIndex int    `json:"column_index"`
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

// GetSnapshotByHash returns an existing snapshot if one exists with the given hash.
// Returns nil, nil if not found.
func GetSnapshotByHash(ctx context.Context, pool *pgxpool.Pool, hash string) (*Snapshot, error) {
	var s Snapshot
	err := pool.QueryRow(ctx, `
		SELECT id, dataset_id, parent_id, snapshot_hash, file_hash, row_count, message, created_by, created_at
		FROM snapshots WHERE snapshot_hash = $1
	`, hash).Scan(
		&s.ID, &s.DatasetID, &s.ParentID, &s.SnapshotHash, &s.FileHash,
		&s.RowCount, &s.Message, &s.CreatedBy, &s.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetSnapshotByHash: %w", err)
	}
	return &s, nil
}

func GetSnapshotByID(ctx context.Context, pool *pgxpool.Pool, id string) (*SnapshotWithColumns, error) {
	var s SnapshotWithColumns
	err := pool.QueryRow(ctx, `
		SELECT id, dataset_id, parent_id, snapshot_hash, file_hash, row_count, message, created_by, created_at
		FROM snapshots WHERE id = $1
	`, id).Scan(
		&s.ID, &s.DatasetID, &s.ParentID, &s.SnapshotHash, &s.FileHash,
		&s.RowCount, &s.Message, &s.CreatedBy, &s.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetSnapshotByID: %w", err)
	}

	cols, err := listSnapshotColumns(ctx, pool, s.ID)
	if err != nil {
		return nil, err
	}
	s.Columns = cols
	return &s, nil
}

// CreateSnapshot inserts a new snapshot and its columns atomically.
// It enforces immutability: if snapshot_hash already exists it returns ErrDuplicateSnapshot.
func CreateSnapshot(ctx context.Context, pool *pgxpool.Pool, p CreateSnapshotParams) (*Snapshot, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

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
		// Unique constraint on snapshot_hash → duplicate content
		return nil, fmt.Errorf("insert snapshot: %w", err)
	}

	for _, col := range p.Columns {
		if _, err := tx.Exec(ctx, `
			INSERT INTO snapshot_columns (snapshot_id, column_name, column_type, column_index)
			VALUES ($1, $2, $3, $4)
		`, s.ID, col.ColumnName, col.ColumnType, col.ColumnIndex); err != nil {
			return nil, fmt.Errorf("insert column %s: %w", col.ColumnName, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &s, nil
}

func ListSnapshots(ctx context.Context, pool *pgxpool.Pool, datasetID string) ([]SnapshotWithColumns, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, dataset_id, parent_id, snapshot_hash, file_hash, row_count, message, created_by, created_at
		FROM snapshots
		WHERE dataset_id = $1
		ORDER BY created_at DESC
	`, datasetID)
	if err != nil {
		return nil, fmt.Errorf("ListSnapshots: %w", err)
	}
	defer rows.Close()

	var out []SnapshotWithColumns
	for rows.Next() {
		var s SnapshotWithColumns
		if err := rows.Scan(
			&s.ID, &s.DatasetID, &s.ParentID, &s.SnapshotHash, &s.FileHash,
			&s.RowCount, &s.Message, &s.CreatedBy, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Attach columns for each snapshot
	for i := range out {
		cols, err := listSnapshotColumns(ctx, pool, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Columns = cols
	}
	return out, nil
}

func listSnapshotColumns(ctx context.Context, pool *pgxpool.Pool, snapshotID string) ([]SnapshotColumn, error) {
	rows, err := pool.Query(ctx, `
		SELECT column_name, column_type, column_index
		FROM snapshot_columns
		WHERE snapshot_id = $1
		ORDER BY column_index
	`, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("listSnapshotColumns: %w", err)
	}
	defer rows.Close()

	var cols []SnapshotColumn
	for rows.Next() {
		var c SnapshotColumn
		if err := rows.Scan(&c.ColumnName, &c.ColumnType, &c.ColumnIndex); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

// IsAncestor returns true if candidateID is an ancestor (or equal) of descendantID.
// Used to verify fast-forward eligibility.
func IsAncestor(ctx context.Context, pool *pgxpool.Pool, candidateID, descendantID string) (bool, error) {
	if candidateID == descendantID {
		return true, nil
	}
	var found bool
	err := pool.QueryRow(ctx, `
		WITH RECURSIVE lineage AS (
			SELECT id, parent_id FROM snapshots WHERE id = $2
			UNION ALL
			SELECT s.id, s.parent_id
			FROM snapshots s JOIN lineage l ON s.id = l.parent_id
		)
		SELECT EXISTS(SELECT 1 FROM lineage WHERE id = $1)
	`, candidateID, descendantID).Scan(&found)
	if err != nil {
		return false, fmt.Errorf("IsAncestor: %w", err)
	}
	return found, nil
}
