package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/rahulbattula415/DatasetVersionControl/internal/csvutil"
	"github.com/rahulbattula415/DatasetVersionControl/internal/db"
	"github.com/rahulbattula415/DatasetVersionControl/internal/service"
)

const maxUploadSize = 100 << 20 // 100 MB

// CreateSnapshotHandler handles POST /datasets/{id}/snapshots
// Multipart form fields:
//   file       – CSV file (required)
//   message    – commit message (optional)
//   branch_id  – branch to commit to (optional, defaults to "main")
func CreateSnapshotHandler(pool *pgxpool.Pool, minioClient *minio.Client, bucket string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		datasetID := r.PathValue("id")
		if datasetID == "" {
			httpError(w, "missing dataset id", http.StatusBadRequest)
			return
		}

		if _, err := db.GetDataset(r.Context(), pool, datasetID); err != nil {
			httpError(w, "dataset not found", http.StatusNotFound)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			httpError(w, "file too large or invalid form", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			httpError(w, "missing file field", http.StatusBadRequest)
			return
		}
		defer file.Close()

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			httpError(w, "failed to read file", http.StatusInternalServerError)
			return
		}

		// Parse CSV
		parsed, err := csvutil.Parse(bytes.NewReader(fileBytes))
		if err != nil {
			httpError(w, fmt.Sprintf("invalid CSV: %s", err), http.StatusBadRequest)
			return
		}

		// Resolve branch (default → main)
		branchID := r.FormValue("branch_id")
		var branch *db.Branch
		if branchID != "" {
			branch, err = db.GetBranch(r.Context(), pool, branchID)
		} else {
			branch, err = db.GetBranchByName(r.Context(), pool, datasetID, "main")
		}
		if err != nil || branch == nil {
			httpError(w, "branch not found", http.StatusNotFound)
			return
		}

		// Parent = current branch head
		parentID := branch.HeadSnapshotID

		// Compute file hash
		hashBytes := sha256.Sum256(fileBytes)
		fileHash := hex.EncodeToString(hashBytes[:])

		// Compute snapshot hash: sha256(file_hash | sorted_schema_json | parent_hash)
		snapshotHash := computeSnapshotHash(fileHash, parsed.Headers, parsed.ColTypes, parentID)

		// Immutability check: reuse existing snapshot if hash already exists
		existing, err := db.GetSnapshotByHash(r.Context(), pool, snapshotHash)
		if err != nil {
			httpError(w, "database error", http.StatusInternalServerError)
			return
		}
		if existing != nil {
			// Advance branch head if it isn't already pointing here
			if branch.HeadSnapshotID == nil || *branch.HeadSnapshotID != existing.ID {
				if err := db.SetBranchHead(r.Context(), pool, branch.ID, existing.ID); err != nil {
					httpError(w, "failed to update branch", http.StatusInternalServerError)
					return
				}
			}
			jsonResponse(w, http.StatusOK, existing)
			return
		}

		// Upload CSV to MinIO
		objectKey := fmt.Sprintf("%s/%s.csv", datasetID, snapshotHash)
		_, err = minioClient.PutObject(
			context.Background(), bucket, objectKey,
			bytes.NewReader(fileBytes), int64(len(fileBytes)),
			minio.PutObjectOptions{ContentType: "text/csv"},
		)
		if err != nil {
			httpError(w, "failed to store file", http.StatusInternalServerError)
			return
		}

		// Build column list
		columns := make([]db.SnapshotColumn, len(parsed.Headers))
		for i, h := range parsed.Headers {
			columns[i] = db.SnapshotColumn{
				ColumnName:  h,
				ColumnType:  parsed.ColTypes[h],
				ColumnIndex: i,
			}
		}

		// Optional commit message
		var msg *string
		if m := r.FormValue("message"); m != "" {
			msg = &m
		}

		snapshot, err := db.CreateSnapshot(r.Context(), pool, db.CreateSnapshotParams{
			DatasetID:    datasetID,
			ParentID:     parentID,
			SnapshotHash: snapshotHash,
			FileHash:     fileHash,
			RowCount:     parsed.RowCount,
			Message:      msg,
			CreatedBy:    systemUser,
			Columns:      columns,
		})
		if err != nil {
			httpError(w, "failed to save snapshot", http.StatusInternalServerError)
			return
		}

		// Compute and store column stats asynchronously (best-effort)
		stats := service.ComputeColumnStats(snapshot.ID, parsed.Headers, parsed.Rows)
		_ = db.UpsertColumnStats(r.Context(), pool, snapshot.ID, stats)

		// Advance branch head
		if err := db.SetBranchHead(r.Context(), pool, branch.ID, snapshot.ID); err != nil {
			httpError(w, "failed to update branch head", http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusCreated, snapshot)
	}
}

func ListSnapshotsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		datasetID := r.PathValue("id")
		snapshots, err := db.ListSnapshots(r.Context(), pool, datasetID)
		if err != nil {
			httpError(w, "failed to list snapshots", http.StatusInternalServerError)
			return
		}
		if snapshots == nil {
			snapshots = []db.SnapshotWithColumns{}
		}
		jsonResponse(w, http.StatusOK, snapshots)
	}
}

// computeSnapshotHash produces a deterministic hash from file content, schema, and parent.
// snapshot_hash = sha256(file_hash "|" schema_json "|" parent_hash_or_ROOT)
func computeSnapshotHash(fileHash string, headers []string, colTypes map[string]string, parentID *string) string {
	type schemaEntry struct {
		Name string `json:"n"`
		Type string `json:"t"`
	}
	schema := make([]schemaEntry, len(headers))
	for i, h := range headers {
		schema[i] = schemaEntry{Name: h, Type: colTypes[h]}
	}
	sort.Slice(schema, func(i, j int) bool { return schema[i].Name < schema[j].Name })
	schemaJSON, _ := json.Marshal(schema)

	parentHash := "ROOT"
	if parentID != nil {
		parentHash = *parentID
	}

	combined := fileHash + "|" + string(schemaJSON) + "|" + parentHash
	raw := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(raw[:])
}
