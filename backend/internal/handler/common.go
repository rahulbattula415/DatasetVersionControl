package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rahulbattula415/DatasetVersionControl/internal/db"
	"github.com/rahulbattula415/DatasetVersionControl/internal/middleware"
)

func httpError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func jsonResponse(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func queryInt(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return def
	}
	return v
}

// userID extracts the authenticated user's ID from the request context.
func userID(r *http.Request) string {
	v, _ := r.Context().Value(middleware.ContextKeyUserID).(string)
	return v
}

// requireDatasetOwner fetches the dataset and returns it only if it belongs to
// the calling user. Returns nil and writes a 404 otherwise (no leaking existence).
func requireDatasetOwner(ctx context.Context, pool *pgxpool.Pool, w http.ResponseWriter, datasetID, callerID string) *db.Dataset {
	dataset, err := db.GetDataset(ctx, pool, datasetID)
	if err != nil || dataset == nil || dataset.CreatedBy != callerID {
		httpError(w, "dataset not found", http.StatusNotFound)
		return nil
	}
	return dataset
}

// requireSnapshotOwner checks snapshot existence and that the owning dataset
// belongs to callerID. Returns nil and writes a 404 on any failure.
func requireSnapshotOwner(ctx context.Context, pool *pgxpool.Pool, w http.ResponseWriter, snapshotID, callerID string) *db.SnapshotWithColumns {
	snap, err := db.GetSnapshotByID(ctx, pool, snapshotID)
	if err != nil || snap == nil {
		httpError(w, fmt.Sprintf("snapshot %s not found", snapshotID), http.StatusNotFound)
		return nil
	}
	if requireDatasetOwner(ctx, pool, w, snap.DatasetID, callerID) == nil {
		return nil
	}
	return snap
}
