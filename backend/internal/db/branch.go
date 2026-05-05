package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Branch struct {
	ID             string    `db:"id"               json:"id"`
	DatasetID      string    `db:"dataset_id"       json:"dataset_id"`
	Name           string    `db:"name"             json:"name"`
	HeadSnapshotID *string   `db:"head_snapshot_id" json:"head_snapshot_id"`
	CreatedAt      time.Time `db:"created_at"       json:"created_at"`
}

func CreateBranch(ctx context.Context, pool *pgxpool.Pool, datasetID, name string, headSnapshotID *string) (*Branch, error) {
	var b Branch
	err := pool.QueryRow(ctx, `
		INSERT INTO branches (dataset_id, name, head_snapshot_id)
		VALUES ($1, $2, $3)
		RETURNING id, dataset_id, name, head_snapshot_id, created_at
	`, datasetID, name, headSnapshotID).Scan(
		&b.ID, &b.DatasetID, &b.Name, &b.HeadSnapshotID, &b.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateBranch: %w", err)
	}
	return &b, nil
}

func GetBranch(ctx context.Context, pool *pgxpool.Pool, id string) (*Branch, error) {
	var b Branch
	err := pool.QueryRow(ctx, `
		SELECT id, dataset_id, name, head_snapshot_id, created_at
		FROM branches WHERE id = $1
	`, id).Scan(&b.ID, &b.DatasetID, &b.Name, &b.HeadSnapshotID, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetBranch: %w", err)
	}
	return &b, nil
}

func GetBranchByName(ctx context.Context, pool *pgxpool.Pool, datasetID, name string) (*Branch, error) {
	var b Branch
	err := pool.QueryRow(ctx, `
		SELECT id, dataset_id, name, head_snapshot_id, created_at
		FROM branches WHERE dataset_id = $1 AND name = $2
	`, datasetID, name).Scan(&b.ID, &b.DatasetID, &b.Name, &b.HeadSnapshotID, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetBranchByName: %w", err)
	}
	return &b, nil
}

func ListBranches(ctx context.Context, pool *pgxpool.Pool, datasetID string) ([]Branch, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, dataset_id, name, head_snapshot_id, created_at
		FROM branches WHERE dataset_id = $1
		ORDER BY created_at ASC
	`, datasetID)
	if err != nil {
		return nil, fmt.Errorf("ListBranches: %w", err)
	}
	defer rows.Close()

	var out []Branch
	for rows.Next() {
		var b Branch
		if err := rows.Scan(&b.ID, &b.DatasetID, &b.Name, &b.HeadSnapshotID, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// AdvanceBranchHead updates a branch's head to newHeadID only if newHeadID is a
// descendant of the current head (fast-forward only). Prevents branch rewrites.
func AdvanceBranchHead(ctx context.Context, pool *pgxpool.Pool, branchID, newHeadID string) (*Branch, error) {
	branch, err := GetBranch(ctx, pool, branchID)
	if err != nil {
		return nil, err
	}
	if branch == nil {
		return nil, fmt.Errorf("branch %s not found", branchID)
	}

	// If branch has no head yet (empty), any snapshot is valid
	if branch.HeadSnapshotID != nil {
		ok, err := IsAncestor(ctx, pool, *branch.HeadSnapshotID, newHeadID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("not a fast-forward: new head is not a descendant of current head")
		}
	}

	var b Branch
	err = pool.QueryRow(ctx, `
		UPDATE branches SET head_snapshot_id = $1
		WHERE id = $2
		RETURNING id, dataset_id, name, head_snapshot_id, created_at
	`, newHeadID, branchID).Scan(&b.ID, &b.DatasetID, &b.Name, &b.HeadSnapshotID, &b.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("AdvanceBranchHead: %w", err)
	}
	return &b, nil
}

// SetBranchHead unconditionally sets a branch head (used internally during snapshot commit).
func SetBranchHead(ctx context.Context, pool *pgxpool.Pool, branchID, snapshotID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE branches SET head_snapshot_id = $1 WHERE id = $2
	`, snapshotID, branchID)
	return err
}
