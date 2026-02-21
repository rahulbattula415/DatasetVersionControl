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
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Validation
		if strings.TrimSpace(req.Name) == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.PrimaryKeyCol) == "" {
			http.Error(w, "primary_key_col is required", http.StatusBadRequest)
			return
		}

		// Hardcode a system user UUID for now, swap for auth middleware later
		const systemUser = "00000000-0000-0000-0000-000000000001"

		dataset, err := db.CreateDataset(r.Context(), pool, db.CreateDatasetParams{
			Name:          req.Name,
			PrimaryKeyCol: req.PrimaryKeyCol,
			CreatedBy:     systemUser,
		})
		if err != nil {
			http.Error(w, "failed to create dataset", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(dataset)
	}
}
