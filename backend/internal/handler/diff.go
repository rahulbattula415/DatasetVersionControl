package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/rahulbattula415/DatasetVersionControl/internal/db"
	"github.com/rahulbattula415/DatasetVersionControl/internal/service"
)

func DiffHandler(pool *pgxpool.Pool, minioClient *minio.Client, bucket string) http.HandlerFunc {
	svc := service.NewDiffService(pool, minioClient, bucket)
	return func(w http.ResponseWriter, r *http.Request) {
		uid := userID(r)

		snapA := requireSnapshotOwner(r.Context(), pool, w, r.PathValue("a"), uid)
		if snapA == nil {
			return
		}
		snapB := requireSnapshotOwner(r.Context(), pool, w, r.PathValue("b"), uid)
		if snapB == nil {
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

		result, err := svc.Diff(r.Context(), snapA, snapB, dataset.PrimaryKeyCol,
			queryInt(r, "page", 1), queryInt(r, "page_size", 100))
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, http.StatusOK, result)
	}
}

func ColumnHistoryHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		datasetID := r.PathValue("id")
		if requireDatasetOwner(r.Context(), pool, w, datasetID, userID(r)) == nil {
			return
		}
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
