package handler_test

// Integration tests for the full dataset → snapshot → branch → merge flow.
// Requires a running Postgres + MinIO. Skip automatically when DATABASE_URL is not set.
//
// Run with:
//   DATABASE_URL=postgresql://root:secret@localhost:5432/dvc \
//   MINIO_ENDPOINT=localhost:9000 MINIO_ACCESS_KEY=minioadmin MINIO_SECRET_KEY=minioadmin \
//   go test ./internal/handler/ -v -run Integration -count=1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rahulbattula415/DatasetVersionControl/internal/handler"
)

// ─── Infrastructure helpers ───────────────────────────────────────────────────

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("pool.Ping: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func integrationMinio(t *testing.T) (*minio.Client, string) {
	t.Helper()
	endpoint := envOr("MINIO_ENDPOINT", "localhost:9000")
	access := envOr("MINIO_ACCESS_KEY", "minioadmin")
	secret := envOr("MINIO_SECRET_KEY", "minioadmin")
	bucket := "datasets-test"

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(access, secret, ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("minio.New: %v", err)
	}
	ctx := context.Background()
	exists, _ := client.BucketExists(ctx, bucket)
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			t.Fatalf("MakeBucket: %v", err)
		}
	}
	return client, bucket
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ─── Test helpers ─────────────────────────────────────────────────────────────

func buildCSVUpload(t *testing.T, csv, message, branchID string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "data.csv")
	io.WriteString(fw, csv)
	if message != "" {
		mw.WriteField("message", message)
	}
	if branchID != "" {
		mw.WriteField("branch_id", branchID)
	}
	mw.Close()
	return &buf, mw.FormDataContentType()
}

func postJSON(t *testing.T, handler http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func getJSON(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, rr.Body.String())
	}
}

// ─── Full-flow integration test ───────────────────────────────────────────────

func TestIntegration_FullFlow(t *testing.T) {
	pool := integrationPool(t)
	minioClient, bucket := integrationMinio(t)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /datasets", handler.CreateDatasetHandler(pool))
	mux.HandleFunc("GET /datasets", handler.ListDatasetsHandler(pool))
	mux.HandleFunc("GET /datasets/{id}", handler.GetDatasetHandler(pool))
	mux.HandleFunc("POST /datasets/{id}/snapshots", handler.CreateSnapshotHandler(pool, minioClient, bucket))
	mux.HandleFunc("GET /datasets/{id}/snapshots", handler.ListSnapshotsHandler(pool))
	mux.HandleFunc("POST /datasets/{id}/branches", handler.CreateBranchHandler(pool))
	mux.HandleFunc("GET /datasets/{id}/branches", handler.ListBranchesHandler(pool))
	mux.HandleFunc("PATCH /branches/{id}", handler.UpdateBranchHandler(pool))
	mux.HandleFunc("POST /branches/{id}/merge", handler.MergeBranchHandler(pool))
	mux.HandleFunc("GET /snapshots/{a}/diff/{b}", handler.DiffHandler(pool, minioClient, bucket))

	// ── 1. Create dataset ──────────────────────────────────────────────────────
	rr := postJSON(t, mux, "/datasets", map[string]string{
		"name":            "sales_" + randomSuffix(),
		"primary_key_col": "id",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create dataset: got %d, body: %s", rr.Code, rr.Body)
	}
	var dataset map[string]any
	decodeJSON(t, rr, &dataset)
	datasetID := dataset["id"].(string)
	t.Logf("Dataset ID: %s", datasetID)

	// ── 2. Verify main branch auto-created ────────────────────────────────────
	rr = getJSON(t, mux, "/datasets/"+datasetID+"/branches")
	if rr.Code != http.StatusOK {
		t.Fatalf("list branches: got %d", rr.Code)
	}
	var branches []map[string]any
	decodeJSON(t, rr, &branches)
	if len(branches) != 1 || branches[0]["name"] != "main" {
		t.Fatalf("expected 1 branch named 'main', got %v", branches)
	}
	mainBranchID := branches[0]["id"].(string)

	// ── 3. Upload first CSV snapshot ──────────────────────────────────────────
	csv1 := "id,name,revenue\n1,Alice,1000\n2,Bob,2000\n3,Carol,3000\n"
	body, ct := buildCSVUpload(t, csv1, "initial load", "")
	req := httptest.NewRequest(http.MethodPost, "/datasets/"+datasetID+"/snapshots", body)
	req.Header.Set("Content-Type", ct)
	req.SetPathValue("id", datasetID)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("snapshot 1: got %d, body: %s", rr.Code, rr.Body)
	}
	var snap1 map[string]any
	decodeJSON(t, rr, &snap1)
	snap1ID := snap1["id"].(string)
	snap1Hash := snap1["snapshot_hash"].(string)
	t.Logf("Snapshot 1: %s hash=%s", snap1ID, snap1Hash)

	// Parent should be nil (first commit)
	if snap1["parent_id"] != nil {
		t.Errorf("first snapshot should have nil parent, got %v", snap1["parent_id"])
	}

	// ── 4. Upload identical CSV → must reuse same snapshot ───────────────────
	body, ct = buildCSVUpload(t, csv1, "duplicate", "")
	req = httptest.NewRequest(http.MethodPost, "/datasets/"+datasetID+"/snapshots", body)
	req.Header.Set("Content-Type", ct)
	req.SetPathValue("id", datasetID)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	// 200 = reused, 201 = new (both are acceptable but hash must match)
	if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
		t.Fatalf("duplicate upload: got %d", rr.Code)
	}
	var snap1dup map[string]any
	decodeJSON(t, rr, &snap1dup)
	if snap1dup["snapshot_hash"].(string) != snap1Hash {
		t.Errorf("duplicate upload returned different hash: %s vs %s",
			snap1dup["snapshot_hash"], snap1Hash)
	}

	// ── 5. Upload second CSV snapshot (different data) ────────────────────────
	csv2 := "id,name,revenue\n1,Alice,1500\n2,Bob,2000\n4,Dave,4000\n"
	body, ct = buildCSVUpload(t, csv2, "update Q2", "")
	req = httptest.NewRequest(http.MethodPost, "/datasets/"+datasetID+"/snapshots", body)
	req.Header.Set("Content-Type", ct)
	req.SetPathValue("id", datasetID)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("snapshot 2: got %d, body: %s", rr.Code, rr.Body)
	}
	var snap2 map[string]any
	decodeJSON(t, rr, &snap2)
	snap2ID := snap2["id"].(string)
	t.Logf("Snapshot 2: %s parent=%v", snap2ID, snap2["parent_id"])

	// Verify lineage: snapshot 2 parent = snapshot 1
	if snap2["parent_id"] == nil {
		t.Error("snapshot 2 should have parent_id set")
	}
	if snap2["parent_id"].(string) != snap1ID {
		t.Errorf("expected parent %s, got %s", snap1ID, snap2["parent_id"])
	}
	if snap2["snapshot_hash"].(string) == snap1Hash {
		t.Error("second snapshot must have different hash from first")
	}

	// ── 6. List snapshots ─────────────────────────────────────────────────────
	rr = getJSON(t, mux, "/datasets/"+datasetID+"/snapshots")
	if rr.Code != http.StatusOK {
		t.Fatalf("list snapshots: got %d", rr.Code)
	}
	var snaps []map[string]any
	decodeJSON(t, rr, &snaps)
	if len(snaps) != 2 {
		t.Errorf("expected 2 snapshots, got %d", len(snaps))
	}

	// ── 7. Diff the two snapshots ─────────────────────────────────────────────
	rr = getJSON(t, mux, fmt.Sprintf("/snapshots/%s/diff/%s", snap1ID, snap2ID))
	if rr.Code != http.StatusOK {
		t.Fatalf("diff: got %d, body: %s", rr.Code, rr.Body)
	}
	var diff map[string]any
	decodeJSON(t, rr, &diff)
	summary := diff["summary"].(map[string]any)
	t.Logf("Diff summary: added=%v deleted=%v modified=%v cached=%v",
		summary["added"], summary["deleted"], summary["modified"], diff["cached"])

	if summary["added"].(float64) != 1 { // Dave is new
		t.Errorf("expected 1 added row, got %v", summary["added"])
	}
	if summary["deleted"].(float64) != 1 { // Carol removed
		t.Errorf("expected 1 deleted row, got %v", summary["deleted"])
	}
	if summary["modified"].(float64) != 1 { // Alice's revenue changed
		t.Errorf("expected 1 modified row, got %v", summary["modified"])
	}

	// ── 8. Diff is cached on second call ─────────────────────────────────────
	rr = getJSON(t, mux, fmt.Sprintf("/snapshots/%s/diff/%s", snap1ID, snap2ID))
	var diff2 map[string]any
	decodeJSON(t, rr, &diff2)
	if diff2["cached"].(bool) != true {
		t.Error("expected diff to be cached on second call")
	}

	// ── 9. Create a feature branch from snapshot 1 ───────────────────────────
	rr = postJSON(t, mux, "/datasets/"+datasetID+"/branches", map[string]any{
		"name":             "feature",
		"head_snapshot_id": snap1ID,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create branch: got %d, body: %s", rr.Code, rr.Body)
	}
	var featureBranch map[string]any
	decodeJSON(t, rr, &featureBranch)
	featureBranchID := featureBranch["id"].(string)

	// ── 10. Fast-forward feature branch to snapshot 2 ────────────────────────
	body2 := bytes.NewBufferString(`{"head_snapshot_id":"` + snap2ID + `"}`)
	req = httptest.NewRequest(http.MethodPatch, "/branches/"+featureBranchID, body2)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", featureBranchID)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("fast-forward branch: got %d, body: %s", rr.Code, rr.Body)
	}

	// ── 11. Prevent non-fast-forward rewind ───────────────────────────────────
	body3 := bytes.NewBufferString(`{"head_snapshot_id":"` + snap1ID + `"}`)
	req = httptest.NewRequest(http.MethodPatch, "/branches/"+featureBranchID, body3)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", featureBranchID)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409 for non-fast-forward, got %d", rr.Code)
	}

	// ── 12. Merge feature branch into main ────────────────────────────────────
	rr = postJSON(t, mux, "/branches/"+mainBranchID+"/merge", map[string]string{
		"source_branch_id": featureBranchID,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("merge: got %d, body: %s", rr.Code, rr.Body)
	}
	var mergeResp map[string]any
	decodeJSON(t, rr, &mergeResp)
	if mergeResp["merged"].(bool) != true {
		t.Error("expected merged=true")
	}
	t.Log("Full integration test passed")
}

func randomSuffix() string {
	f, _ := os.CreateTemp("", "")
	name := f.Name()
	f.Close()
	os.Remove(f.Name())
	parts := strings.Split(name, string(os.PathSeparator))
	last := parts[len(parts)-1]
	if len(last) > 8 {
		return last[len(last)-8:]
	}
	return last
}
