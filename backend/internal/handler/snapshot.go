package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/rahulbattula415/DatasetVersionControl/internal/csvutil"
	"github.com/rahulbattula415/DatasetVersionControl/internal/db"
	"github.com/rahulbattula415/DatasetVersionControl/internal/service"
)

const maxUploadSize = 100 << 20

func CreateSnapshotHandler(pool *pgxpool.Pool, minioClient *minio.Client, bucket string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		datasetID := r.PathValue("id")
		uid := userID(r)

		if requireDatasetOwner(r.Context(), pool, w, datasetID, uid) == nil {
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

		parsed, err := csvutil.Parse(bytes.NewReader(fileBytes))
		if err != nil {
			httpError(w, fmt.Sprintf("invalid CSV: %s", err), http.StatusBadRequest)
			return
		}

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

		parentID := branch.HeadSnapshotID

		hashBytes := sha256.Sum256(fileBytes)
		fileHash := hex.EncodeToString(hashBytes[:])
		snapshotHash := computeSnapshotHash(fileHash, parsed.Headers, parsed.ColTypes, parentID)

		existing, err := db.GetSnapshotByHash(r.Context(), pool, snapshotHash)
		if err != nil {
			httpError(w, "database error", http.StatusInternalServerError)
			return
		}
		if existing != nil {
			if branch.HeadSnapshotID == nil || *branch.HeadSnapshotID != existing.ID {
				if err := db.SetBranchHead(r.Context(), pool, branch.ID, existing.ID); err != nil {
					httpError(w, "failed to update branch", http.StatusInternalServerError)
					return
				}
			}
			jsonResponse(w, http.StatusOK, existing)
			return
		}

		objectKey := fmt.Sprintf("%s/%s.csv", datasetID, snapshotHash)
		_, err = minioClient.PutObject(
			context.Background(), bucket, objectKey,
			bytes.NewReader(fileBytes), int64(len(fileBytes)),
			minio.PutObjectOptions{ContentType: "text/csv"},
		)
		if err != nil {
			log.Printf("MinIO PutObject error: %v", err)
			httpError(w, "failed to store file", http.StatusInternalServerError)
			return
		}

		columns := make([]db.SnapshotColumn, len(parsed.Headers))
		for i, h := range parsed.Headers {
			columns[i] = db.SnapshotColumn{
				ColumnName:  h,
				ColumnType:  parsed.ColTypes[h],
				ColumnIndex: i,
			}
		}

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
			CreatedBy:    uid,
			Columns:      columns,
		})
		if err != nil {
			httpError(w, "failed to save snapshot", http.StatusInternalServerError)
			return
		}

		stats := service.ComputeColumnStats(snapshot.ID, parsed.Headers, parsed.Rows)
		_ = db.UpsertColumnStats(r.Context(), pool, snapshot.ID, stats)

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
		if requireDatasetOwner(r.Context(), pool, w, datasetID, userID(r)) == nil {
			return
		}
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
