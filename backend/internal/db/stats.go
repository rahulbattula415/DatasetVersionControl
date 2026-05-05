package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ColumnStat struct {
	SnapshotID  string    `json:"snapshot_id"`
	ColumnName  string    `json:"column_name"`
	MinValue    *string   `json:"min_value"`
	MaxValue    *string   `json:"max_value"`
	MeanValue   *float64  `json:"mean_value"`
	NullCount   int       `json:"null_count"`
	UniqueCount int       `json:"unique_count"`
}

type ColumnStatHistory struct {
	SnapshotID string    `json:"snapshot_id"`
	CreatedAt  time.Time `json:"created_at"`
	Message    *string   `json:"message"`
	MinValue   *string   `json:"min_value"`
	MaxValue   *string   `json:"max_value"`
	MeanValue  *float64  `json:"mean_value"`
	NullCount  int       `json:"null_count"`
	UniqueCount int      `json:"unique_count"`
}

func UpsertColumnStats(ctx context.Context, pool *pgxpool.Pool, snapshotID string, stats []ColumnStat) error {
	for _, s := range stats {
		_, err := pool.Exec(ctx, `
			INSERT INTO snapshot_column_stats
				(snapshot_id, column_name, min_value, max_value, mean_value, null_count, unique_count)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (snapshot_id, column_name) DO UPDATE SET
				min_value    = EXCLUDED.min_value,
				max_value    = EXCLUDED.max_value,
				mean_value   = EXCLUDED.mean_value,
				null_count   = EXCLUDED.null_count,
				unique_count = EXCLUDED.unique_count
		`, snapshotID, s.ColumnName, s.MinValue, s.MaxValue, s.MeanValue, s.NullCount, s.UniqueCount)
		if err != nil {
			return fmt.Errorf("UpsertColumnStats %s: %w", s.ColumnName, err)
		}
	}
	return nil
}

// GetColumnHistory returns time-series stats for a column by traversing the snapshot
// lineage from the branch head, ordered oldest-to-newest.
func GetColumnHistory(ctx context.Context, pool *pgxpool.Pool, datasetID, branchID, columnName string) ([]ColumnStatHistory, error) {
	// branchID="" → default to main branch
	branchClause := "b.name = 'main'"
	args := []any{datasetID, columnName}
	if branchID != "" {
		branchClause = "b.id = $3"
		args = append(args, branchID)
	}

	query := fmt.Sprintf(`
		WITH RECURSIVE lineage AS (
			SELECT s.id, s.parent_id, s.created_at, s.message
			FROM   snapshots s
			JOIN   branches  b ON b.head_snapshot_id = s.id AND b.dataset_id = $1
			WHERE  %s
			UNION ALL
			SELECT s.id, s.parent_id, s.created_at, s.message
			FROM   snapshots s JOIN lineage l ON s.id = l.parent_id
		)
		SELECT
			l.id, l.created_at, l.message,
			cs.min_value, cs.max_value, cs.mean_value,
			cs.null_count, cs.unique_count
		FROM lineage l
		JOIN snapshot_column_stats cs ON cs.snapshot_id = l.id AND cs.column_name = $2
		ORDER BY l.created_at ASC
	`, branchClause)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("GetColumnHistory: %w", err)
	}
	defer rows.Close()

	var out []ColumnStatHistory
	for rows.Next() {
		var h ColumnStatHistory
		if err := rows.Scan(
			&h.SnapshotID, &h.CreatedAt, &h.Message,
			&h.MinValue, &h.MaxValue, &h.MeanValue,
			&h.NullCount, &h.UniqueCount,
		); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
