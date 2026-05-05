package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rahulbattula415/DatasetVersionControl/internal/db"
)

type CreateDatasetRequest struct {
	Name          string `json:"name"`
	PrimaryKeyCol string `json:"primary_key_col"`
}

func CreateDatasetHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateDatasetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Name) == "" {
			httpError(w, "name is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.PrimaryKeyCol) == "" {
			httpError(w, "primary_key_col is required", http.StatusBadRequest)
			return
		}

		dataset, err := db.CreateDataset(r.Context(), pool, db.CreateDatasetParams{
			Name:          req.Name,
			PrimaryKeyCol: req.PrimaryKeyCol,
			CreatedBy:     systemUser,
		})
		if err != nil {
			httpError(w, "failed to create dataset", http.StatusInternalServerError)
			return
		}

		// Auto-create main branch with no head
		if _, err := db.CreateBranch(r.Context(), pool, dataset.ID, "main", nil); err != nil {
			httpError(w, "failed to create main branch", http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusCreated, dataset)
	}
}

func GetDatasetHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		dataset, err := db.GetDataset(r.Context(), pool, id)
		if err != nil {
			httpError(w, "dataset not found", http.StatusNotFound)
			return
		}
		jsonResponse(w, http.StatusOK, dataset)
	}
}

func ListDatasetsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		datasets, err := db.ListDatasets(r.Context(), pool)
		if err != nil {
			httpError(w, "failed to list datasets", http.StatusInternalServerError)
			return
		}
		if datasets == nil {
			datasets = []db.DatasetSummary{}
		}
		jsonResponse(w, http.StatusOK, datasets)
	}
}
