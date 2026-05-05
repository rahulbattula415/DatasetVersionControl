package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/rahulbattula415/DatasetVersionControl/internal/db"
	"github.com/rahulbattula415/DatasetVersionControl/internal/service"
)

// DiffHandler handles GET /snapshots/{a}/diff/{b}
// Query params: page (default 1), page_size (default 100)
func DiffHandler(pool *pgxpool.Pool, minioClient *minio.Client, bucket string) http.HandlerFunc {
	svc := service.NewDiffService(pool, minioClient, bucket)
	return func(w http.ResponseWriter, r *http.Request) {
		idA := r.PathValue("a")
		idB := r.PathValue("b")

		snapA, err := db.GetSnapshotByID(r.Context(), pool, idA)
		if err != nil || snapA == nil {
			httpError(w, "snapshot A not found", http.StatusNotFound)
			return
		}
		snapB, err := db.GetSnapshotByID(r.Context(), pool, idB)
		if err != nil || snapB == nil {
			httpError(w, "snapshot B not found", http.StatusNotFound)
			return
		}
		if snapA.DatasetID != snapB.DatasetID {
			httpError(w, "snapshots belong to different datasets", http.StatusBadRequest)
			return
		}

		dataset, err := db.GetDataset(r.Context(), pool, snapA.DatasetID)
		if err != nil {
			httpError(w, "dataset not found", http.StatusInternalServerError)
			return
		}

		page := queryInt(r, "page", 1)
		pageSize := queryInt(r, "page_size", 100)

		result, err := svc.Diff(r.Context(), snapA, snapB, dataset.PrimaryKeyCol, page, pageSize)
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, result)
	}
}

// ColumnHistoryHandler handles GET /datasets/{id}/columns/{col}/history
// Query params: branch_id (optional, defaults to main)
func ColumnHistoryHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		datasetID := r.PathValue("id")
		col := r.PathValue("col")
		branchID := r.URL.Query().Get("branch_id")

		history, err := db.GetColumnHistory(r.Context(), pool, datasetID, branchID, col)
		if err != nil {
			httpError(w, "failed to retrieve column history", http.StatusInternalServerError)
			return
		}
		if history == nil {
			history = []db.ColumnStatHistory{}
		}
		jsonResponse(w, http.StatusOK, history)
	}
}
