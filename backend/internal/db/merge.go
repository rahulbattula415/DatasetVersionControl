package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Merge struct {
	ID             string    `json:"id"`
	SourceBranchID string    `json:"source_branch_id"`
	TargetBranchID string    `json:"target_branch_id"`
	SnapshotID     string    `json:"snapshot_id"`
	MergedBy       string    `json:"merged_by"`
	MergedAt       time.Time `json:"merged_at"`
}

func RecordMerge(ctx context.Context, pool *pgxpool.Pool, sourceBranchID, targetBranchID, snapshotID, mergedBy string) (*Merge, error) {
	var m Merge
	err := pool.QueryRow(ctx, `
		INSERT INTO merges (source_branch_id, target_branch_id, snapshot_id, merged_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, source_branch_id, target_branch_id, snapshot_id, merged_by, merged_at
	`, sourceBranchID, targetBranchID, snapshotID, mergedBy).Scan(
		&m.ID, &m.SourceBranchID, &m.TargetBranchID, &m.SnapshotID, &m.MergedBy, &m.MergedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("RecordMerge: %w", err)
	}
	return &m, nil
}
