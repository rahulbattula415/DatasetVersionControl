package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/rahulbattula415/DatasetVersionControl/internal/csvutil"
	"github.com/rahulbattula415/DatasetVersionControl/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Types ────────────────────────────────────────────────────────────────────

type ColumnChange struct {
	Column string `json:"column"`
	Old    string `json:"old"`
	New    string `json:"new"`
}

type ModifiedRow struct {
	Key     string         `json:"key"`
	Changes []ColumnChange `json:"changes"`
}

type DiffSummary struct {
	Added    int `json:"added"`
	Deleted  int `json:"deleted"`
	Modified int `json:"modified"`
}

type DiffResult struct {
	Summary      DiffSummary            `json:"summary"`
	Added        []map[string]string    `json:"added"`
	Deleted      []map[string]string    `json:"deleted"`
	Modified     []ModifiedRow          `json:"modified"`
	SchemaA      []string               `json:"schema_a"`
	SchemaB      []string               `json:"schema_b"`
	SchemaMatch  bool                   `json:"schema_match"`
	SchemaDiff   *SchemaDiff            `json:"schema_diff,omitempty"`
	ComputedMs   int                    `json:"computed_ms"`
	Cached       bool                   `json:"cached"`
}

type SchemaDiff struct {
	AddedColumns   []string `json:"added_columns"`
	RemovedColumns []string `json:"removed_columns"`
}

type PagedDiffResult struct {
	DiffResult
	Page      int `json:"page"`
	PageSize  int `json:"page_size"`
	TotalRows int `json:"total_rows"`
}

// ─── DiffService ─────────────────────────────────────────────────────────────

type DiffService struct {
	pool   *pgxpool.Pool
	minio  *minio.Client
	bucket string
}

func NewDiffService(pool *pgxpool.Pool, minioClient *minio.Client, bucket string) *DiffService {
	return &DiffService{pool: pool, minio: minioClient, bucket: bucket}
}

// Diff returns the diff between two snapshots, using the DB cache when available.
// Results are paginated: page starts at 1.
func (svc *DiffService) Diff(ctx context.Context, snapshotA, snapshotB *db.SnapshotWithColumns, primaryKeyCol string, page, pageSize int) (*PagedDiffResult, error) {
	// Check cache
	cached, err := db.GetDiffCache(ctx, svc.pool, snapshotA.SnapshotHash, snapshotB.SnapshotHash)
	if err != nil {
		return nil, err
	}

	var result DiffResult
	if cached != nil {
		if err := json.Unmarshal(cached.DiffJSON, &result); err != nil {
			return nil, fmt.Errorf("unmarshal cached diff: %w", err)
		}
		result.Cached = true
		result.ComputedMs = cached.ComputedMs
	} else {
		start := time.Now()
		result, err = svc.compute(ctx, snapshotA, snapshotB, primaryKeyCol)
		if err != nil {
			return nil, err
		}
		result.ComputedMs = int(time.Since(start).Milliseconds())
		result.Cached = false

		// Persist to cache
		raw, _ := json.Marshal(result)
		_ = db.StoreDiffCache(ctx, svc.pool, snapshotA.SnapshotHash, snapshotB.SnapshotHash, raw, result.ComputedMs)
	}

	return paginate(result, page, pageSize), nil
}

func (svc *DiffService) compute(ctx context.Context, snapA, snapB *db.SnapshotWithColumns, primaryKeyCol string) (DiffResult, error) {
	rowsA, headersA, err := svc.loadCSV(ctx, snapA.DatasetID, snapA.SnapshotHash)
	if err != nil {
		return DiffResult{}, fmt.Errorf("load snapshot A: %w", err)
	}
	rowsB, headersB, err := svc.loadCSV(ctx, snapB.DatasetID, snapB.SnapshotHash)
	if err != nil {
		return DiffResult{}, fmt.Errorf("load snapshot B: %w", err)
	}

	// Schema comparison
	schemaA := headersA
	schemaB := headersB
	setA := toSet(headersA)
	setB := toSet(headersB)

	var schemaDiff *SchemaDiff
	schemaMatch := equalStringSlices(headersA, headersB)
	if !schemaMatch {
		added, removed := schemaDifference(setA, setB)
		schemaDiff = &SchemaDiff{AddedColumns: added, RemovedColumns: removed}
	}

	// Build primary-key index for each snapshot
	pkIdxA := colIndex(headersA, primaryKeyCol)
	pkIdxB := colIndex(headersB, primaryKeyCol)
	if pkIdxA < 0 {
		return DiffResult{}, fmt.Errorf("primary key column %q not found in snapshot A", primaryKeyCol)
	}
	if pkIdxB < 0 {
		return DiffResult{}, fmt.Errorf("primary key column %q not found in snapshot B", primaryKeyCol)
	}

	mapA := indexByPK(rowsA, headersA, pkIdxA)
	mapB := indexByPK(rowsB, headersB, pkIdxB)

	// Columns present in both schemas (for modified row comparison)
	commonCols := intersect(headersA, headersB)

	var added, deleted []map[string]string
	var modified []ModifiedRow

	for key, rowA := range mapA {
		rowB, exists := mapB[key]
		if !exists {
			deleted = append(deleted, rowA)
			continue
		}
		changes := diffRow(rowA, rowB, commonCols)
		if len(changes) > 0 {
			modified = append(modified, ModifiedRow{Key: key, Changes: changes})
		}
	}
	for key, rowB := range mapB {
		if _, exists := mapA[key]; !exists {
			added = append(added, rowB)
		}
	}

	// Deterministic ordering for consistent pagination
	sort.Slice(added, func(i, j int) bool { return added[i][primaryKeyCol] < added[j][primaryKeyCol] })
	sort.Slice(deleted, func(i, j int) bool { return deleted[i][primaryKeyCol] < deleted[j][primaryKeyCol] })
	sort.Slice(modified, func(i, j int) bool { return modified[i].Key < modified[j].Key })

	return DiffResult{
		Summary:     DiffSummary{Added: len(added), Deleted: len(deleted), Modified: len(modified)},
		Added:       added,
		Deleted:     deleted,
		Modified:    modified,
		SchemaA:     schemaA,
		SchemaB:     schemaB,
		SchemaMatch: schemaMatch,
		SchemaDiff:  schemaDiff,
	}, nil
}

func (svc *DiffService) loadCSV(ctx context.Context, datasetID, snapshotHash string) ([][]string, []string, error) {
	key := fmt.Sprintf("%s/%s.csv", datasetID, snapshotHash)
	obj, err := svc.minio.GetObject(ctx, svc.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("MinIO GetObject %s: %w", key, err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, nil, fmt.Errorf("read object: %w", err)
	}

	parsed, err := csvutil.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("parse CSV: %w", err)
	}
	return parsed.Rows, parsed.Headers, nil
}

// ─── Stats computation ────────────────────────────────────────────────────────

// ComputeColumnStats derives per-column stats from parsed CSV rows.
func ComputeColumnStats(snapshotID string, headers []string, rows [][]string) []db.ColumnStat {
	stats := make([]db.ColumnStat, len(headers))
	for i, h := range headers {
		stats[i] = computeStat(snapshotID, h, i, rows)
	}
	return stats
}

func computeStat(snapshotID, col string, idx int, rows [][]string) db.ColumnStat {
	s := db.ColumnStat{SnapshotID: snapshotID, ColumnName: col}

	var nums []float64
	seen := map[string]struct{}{}
	nulls := 0
	var minStr, maxStr *string

	for _, row := range rows {
		val := ""
		if idx < len(row) {
			val = row[idx]
		}
		if val == "" {
			nulls++
			continue
		}
		seen[val] = struct{}{}
		if minStr == nil || val < *minStr {
			v := val
			minStr = &v
		}
		if maxStr == nil || val > *maxStr {
			v := val
			maxStr = &v
		}
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			nums = append(nums, f)
		}
	}

	s.NullCount = nulls
	s.UniqueCount = len(seen)
	s.MinValue = minStr
	s.MaxValue = maxStr

	if len(nums) > 0 {
		sum := 0.0
		for _, n := range nums {
			sum += n
		}
		mean := sum / float64(len(nums))
		if !math.IsNaN(mean) && !math.IsInf(mean, 0) {
			s.MeanValue = &mean
		}
	}
	return s
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func colIndex(headers []string, col string) int {
	for i, h := range headers {
		if h == col {
			return i
		}
	}
	return -1
}

func indexByPK(rows [][]string, headers []string, pkIdx int) map[string]map[string]string {
	m := make(map[string]map[string]string, len(rows))
	for _, row := range rows {
		if pkIdx >= len(row) {
			continue
		}
		key := row[pkIdx]
		rec := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(row) {
				rec[h] = row[i]
			}
		}
		m[key] = rec
	}
	return m
}

func diffRow(a, b map[string]string, cols []string) []ColumnChange {
	var changes []ColumnChange
	for _, col := range cols {
		va, vb := a[col], b[col]
		if va != vb {
			changes = append(changes, ColumnChange{Column: col, Old: va, New: vb})
		}
	}
	return changes
}

func toSet(s []string) map[string]struct{} {
	m := make(map[string]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}

func intersect(a, b []string) []string {
	sb := toSet(b)
	var out []string
	for _, v := range a {
		if _, ok := sb[v]; ok {
			out = append(out, v)
		}
	}
	return out
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func schemaDifference(setA, setB map[string]struct{}) (addedInB, removedFromA []string) {
	for k := range setB {
		if _, ok := setA[k]; !ok {
			addedInB = append(addedInB, k)
		}
	}
	for k := range setA {
		if _, ok := setB[k]; !ok {
			removedFromA = append(removedFromA, k)
		}
	}
	sort.Strings(addedInB)
	sort.Strings(removedFromA)
	return
}

func paginate(r DiffResult, page, pageSize int) *PagedDiffResult {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 100
	}

	total := len(r.Added) + len(r.Deleted) + len(r.Modified)

	start := (page - 1) * pageSize
	end := start + pageSize

	r.Added = pageSlice(r.Added, start, end)
	r.Deleted = pageSlice(r.Deleted, start, end)
	r.Modified = pageModified(r.Modified, start, end)

	return &PagedDiffResult{
		DiffResult: r,
		Page:       page,
		PageSize:   pageSize,
		TotalRows:  total,
	}
}

func pageSlice[T any](s []T, start, end int) []T {
	if start >= len(s) {
		return []T{}
	}
	if end > len(s) {
		end = len(s)
	}
	return s[start:end]
}

func pageModified(s []ModifiedRow, start, end int) []ModifiedRow {
	if start >= len(s) {
		return []ModifiedRow{}
	}
	if end > len(s) {
		end = len(s)
	}
	return s[start:end]
}
