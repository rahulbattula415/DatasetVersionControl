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
		uid := userID(r)

		if requireDatasetOwner(r.Context(), pool, w, datasetID, uid) == nil {
			return
		}

		var req struct {
			Name           string  `json:"name"`
			HeadSnapshotID *string `json:"head_snapshot_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			httpError(w, "name is required", http.StatusBadRequest)
			return
		}

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
		if requireDatasetOwner(r.Context(), pool, w, datasetID, userID(r)) == nil {
			return
		}
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

func UpdateBranchHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		branchID := r.PathValue("id")
		uid := userID(r)

		branch, err := db.GetBranch(r.Context(), pool, branchID)
		if err != nil || branch == nil {
			httpError(w, "branch not found", http.StatusNotFound)
			return
		}
		if requireDatasetOwner(r.Context(), pool, w, branch.DatasetID, uid) == nil {
			return
		}

		var req struct {
			HeadSnapshotID string `json:"head_snapshot_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.HeadSnapshotID == "" {
			httpError(w, "head_snapshot_id is required", http.StatusBadRequest)
			return
		}

		updated, err := db.AdvanceBranchHead(r.Context(), pool, branchID, req.HeadSnapshotID)
		if err != nil {
			httpError(w, err.Error(), http.StatusConflict)
			return
		}
		jsonResponse(w, http.StatusOK, updated)
	}
}

func MergeBranchHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetBranchID := r.PathValue("id")
		uid := userID(r)

		target, err := db.GetBranch(r.Context(), pool, targetBranchID)
		if err != nil || target == nil {
			httpError(w, "target branch not found", http.StatusNotFound)
			return
		}
		if requireDatasetOwner(r.Context(), pool, w, target.DatasetID, uid) == nil {
			return
		}

		var req struct {
			SourceBranchID string `json:"source_branch_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SourceBranchID == "" {
			httpError(w, "source_branch_id is required", http.StatusBadRequest)
			return
		}

		source, err := db.GetBranch(r.Context(), pool, req.SourceBranchID)
		if err != nil || source == nil || source.DatasetID != target.DatasetID {
			httpError(w, "source branch not found", http.StatusNotFound)
			return
		}
		if source.HeadSnapshotID == nil {
			httpError(w, "source branch has no commits", http.StatusBadRequest)
			return
		}

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

		if err := db.SetBranchHead(r.Context(), pool, target.ID, *source.HeadSnapshotID); err != nil {
			httpError(w, "failed to advance branch head", http.StatusInternalServerError)
			return
		}
		merge, _ := db.RecordMerge(r.Context(), pool, source.ID, target.ID, *source.HeadSnapshotID, uid)
		target.HeadSnapshotID = source.HeadSnapshotID
		jsonResponse(w, http.StatusOK, map[string]any{
			"merged":        true,
			"target_branch": target,
			"merge_event":   merge,
		})
	}
}
