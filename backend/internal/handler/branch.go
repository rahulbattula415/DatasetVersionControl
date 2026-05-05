package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rahulbattula415/DatasetVersionControl/internal/db"
)

func CreateBranchHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		datasetID := r.PathValue("id")

		var req struct {
			Name           string  `json:"name"`
			HeadSnapshotID *string `json:"head_snapshot_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			httpError(w, "name is required", http.StatusBadRequest)
			return
		}

		// Verify snapshot belongs to dataset if provided
		if req.HeadSnapshotID != nil {
			snap, err := db.GetSnapshotByID(r.Context(), pool, *req.HeadSnapshotID)
			if err != nil || snap == nil || snap.DatasetID != datasetID {
				httpError(w, "snapshot not found in this dataset", http.StatusBadRequest)
				return
			}
		}

		branch, err := db.CreateBranch(r.Context(), pool, datasetID, req.Name, req.HeadSnapshotID)
		if err != nil {
			httpError(w, "failed to create branch (name may already exist)", http.StatusConflict)
			return
		}
		jsonResponse(w, http.StatusCreated, branch)
	}
}

func ListBranchesHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		datasetID := r.PathValue("id")
		branches, err := db.ListBranches(r.Context(), pool, datasetID)
		if err != nil {
			httpError(w, "failed to list branches", http.StatusInternalServerError)
			return
		}
		if branches == nil {
			branches = []db.Branch{}
		}
		jsonResponse(w, http.StatusOK, branches)
	}
}

// UpdateBranchHandler handles PATCH /branches/{id}
// Only allows fast-forward head advancement — no rewrites.
func UpdateBranchHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		branchID := r.PathValue("id")

		var req struct {
			HeadSnapshotID string `json:"head_snapshot_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.HeadSnapshotID == "" {
			httpError(w, "head_snapshot_id is required", http.StatusBadRequest)
			return
		}

		branch, err := db.AdvanceBranchHead(r.Context(), pool, branchID, req.HeadSnapshotID)
		if err != nil {
			httpError(w, err.Error(), http.StatusConflict)
			return
		}
		jsonResponse(w, http.StatusOK, branch)
	}
}

// MergeBranchHandler handles POST /branches/{id}/merge
// Fast-forward only: if source head is a descendant of target head, advance target.
func MergeBranchHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetBranchID := r.PathValue("id")

		var req struct {
			SourceBranchID string `json:"source_branch_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SourceBranchID == "" {
			httpError(w, "source_branch_id is required", http.StatusBadRequest)
			return
		}

		target, err := db.GetBranch(r.Context(), pool, targetBranchID)
		if err != nil || target == nil {
			httpError(w, "target branch not found", http.StatusNotFound)
			return
		}
		source, err := db.GetBranch(r.Context(), pool, req.SourceBranchID)
		if err != nil || source == nil {
			httpError(w, "source branch not found", http.StatusNotFound)
			return
		}
		if source.DatasetID != target.DatasetID {
			httpError(w, "branches belong to different datasets", http.StatusBadRequest)
			return
		}
		if source.HeadSnapshotID == nil {
			httpError(w, "source branch has no commits", http.StatusBadRequest)
			return
		}

		// Check fast-forwardability
		canFF := true
		if target.HeadSnapshotID != nil {
			canFF, err = db.IsAncestor(r.Context(), pool, *target.HeadSnapshotID, *source.HeadSnapshotID)
			if err != nil {
				httpError(w, "ancestry check failed", http.StatusInternalServerError)
				return
			}
		}

		if !canFF {
			jsonResponse(w, http.StatusConflict, map[string]string{
				"error":  "cannot merge",
				"reason": "not fast-forwardable: target has commits not in source",
				"hint":   "rebase the source branch onto the target before merging",
			})
			return
		}

		// Fast-forward target head
		if err := db.SetBranchHead(r.Context(), pool, target.ID, *source.HeadSnapshotID); err != nil {
			httpError(w, "failed to advance branch head", http.StatusInternalServerError)
			return
		}

		// Record merge event
		merge, err := db.RecordMerge(r.Context(), pool, source.ID, target.ID, *source.HeadSnapshotID, systemUser)
		if err != nil {
			// Non-fatal — head is already advanced
		}

		target.HeadSnapshotID = source.HeadSnapshotID
		jsonResponse(w, http.StatusOK, map[string]any{
			"merged":        true,
			"target_branch": target,
			"merge_event":   merge,
		})
	}
}
