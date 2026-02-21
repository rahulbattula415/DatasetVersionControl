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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/rahulbattula415/DatasetVersionControl/internal/csvutil"
	"github.com/rahulbattula415/DatasetVersionControl/internal/db"
)

const systemUser = "00000000-0000-0000-0000-000000000001"
const maxUploadSize = 100 << 20 // 100 MB

func UploadCSVHandler(pool *pgxpool.Pool, minioClient *minio.Client, bucket string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Get dataset ID from URL path
		datasetID := r.PathValue("id") // requires Go 1.22+
		if datasetID == "" {
			http.Error(w, "missing dataset id", http.StatusBadRequest)
			return
		}

		// 2. Parse the multipart form (the CSV comes in as a file field)
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "file too large or invalid form", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file") // "file" is the form field name
		if err != nil {
			http.Error(w, "missing file field", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// 3. Read entire file into memory so we can hash it + parse it
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "failed to read file", http.StatusInternalServerError)
			return
		}

		// 4. Compute file hash
		hashBytes := sha256.Sum256(fileBytes)
		fileHash := hex.EncodeToString(hashBytes[:])

		// 5. Parse CSV
		parsed, err := csvutil.Parse(bytes.NewReader(fileBytes))
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid CSV: %s", err), http.StatusBadRequest)
			return
		}

		// 6. Build snapshot hash (hash of datasetID + fileHash to make it unique per dataset)
		snapshotHashRaw := sha256.Sum256([]byte(datasetID + fileHash))
		snapshotHash := hex.EncodeToString(snapshotHashRaw[:])

		// 7. Upload to MinIO
		objectKey := fmt.Sprintf("%s/%s.csv", datasetID, snapshotHash)
		_, err = minioClient.PutObject(
			context.Background(),
			bucket,
			objectKey,
			bytes.NewReader(fileBytes),
			int64(len(fileBytes)),
			minio.PutObjectOptions{ContentType: "text/csv"},
		)
		if err != nil {
			http.Error(w, "failed to store file", http.StatusInternalServerError)
			return
		}

		// 8. Build columns list for DB insert
		columns := make([]db.SnapshotColumn, len(parsed.Headers))
		for i, h := range parsed.Headers {
			columns[i] = db.SnapshotColumn{
				ColumnName:  h,
				ColumnType:  parsed.ColTypes[h],
				ColumnIndex: i,
			}
		}

		// 9. Insert snapshot into Postgres
		snapshot, err := db.CreateSnapshot(r.Context(), pool, db.CreateSnapshotParams{
			DatasetID:    datasetID,
			ParentID:     nil, // first snapshot has no parent
			SnapshotHash: snapshotHash,
			FileHash:     fileHash,
			RowCount:     parsed.RowCount,
			Message:      nil,
			CreatedBy:    systemUser,
			Columns:      columns,
		})
		if err != nil {
			http.Error(w, "failed to save snapshot", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(snapshot)
	}
}
