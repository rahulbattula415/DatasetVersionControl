package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DiffCacheEntry struct {
	SnapshotAHash string          `json:"snapshot_a_hash"`
	SnapshotBHash string          `json:"snapshot_b_hash"`
	DiffJSON      json.RawMessage `json:"diff"`
	ComputedMs    int             `json:"computed_ms"`
	ComputedAt    time.Time       `json:"computed_at"`
}

// GetDiffCache looks up a cached diff. Returns nil, nil if not found.
func GetDiffCache(ctx context.Context, pool *pgxpool.Pool, hashA, hashB string) (*DiffCacheEntry, error) {
	var e DiffCacheEntry
	err := pool.QueryRow(ctx, `
		SELECT snapshot_a_hash, snapshot_b_hash, diff_json, computed_ms, computed_at
		FROM diff_cache WHERE snapshot_a_hash = $1 AND snapshot_b_hash = $2
	`, hashA, hashB).Scan(
		&e.SnapshotAHash, &e.SnapshotBHash, &e.DiffJSON, &e.ComputedMs, &e.ComputedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetDiffCache: %w", err)
	}
	return &e, nil
}

// StoreDiffCache persists a diff result for future reuse.
func StoreDiffCache(ctx context.Context, pool *pgxpool.Pool, hashA, hashB string, diffJSON json.RawMessage, computedMs int) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO diff_cache (snapshot_a_hash, snapshot_b_hash, diff_json, computed_ms)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (snapshot_a_hash, snapshot_b_hash) DO NOTHING
	`, hashA, hashB, diffJSON, computedMs)
	if err != nil {
		return fmt.Errorf("StoreDiffCache: %w", err)
	}
	return nil
}
