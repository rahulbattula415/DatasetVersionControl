package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rahulbattula415/DatasetVersionControl/internal/handler"
)

func main() {
	loadEnv(".env")

	ctx := context.Background()

	dbURL := mustEnv("DATABASE_URL")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("Connected to database")

	minioEndpoint := mustEnv("MINIO_ENDPOINT")
	minioAccess := mustEnv("MINIO_ACCESS_KEY")
	minioSecret := mustEnv("MINIO_SECRET_KEY")
	minioBucket := mustEnv("MINIO_BUCKET")

	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioAccess, minioSecret, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("cannot create MinIO client: %v", err)
	}

	// Ensure bucket exists
	exists, err := minioClient.BucketExists(ctx, minioBucket)
	if err != nil {
		log.Fatalf("MinIO bucket check failed: %v", err)
	}
	if !exists {
		if err := minioClient.MakeBucket(ctx, minioBucket, minio.MakeBucketOptions{}); err != nil {
			log.Fatalf("cannot create MinIO bucket: %v", err)
		}
		log.Printf("Created MinIO bucket %q", minioBucket)
	}
	log.Println("Connected to MinIO")

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Datasets
	mux.HandleFunc("POST /datasets", handler.CreateDatasetHandler(pool))
	mux.HandleFunc("GET /datasets", handler.ListDatasetsHandler(pool))
	mux.HandleFunc("GET /datasets/{id}", handler.GetDatasetHandler(pool))

	// Snapshots
	mux.HandleFunc("POST /datasets/{id}/snapshots", handler.CreateSnapshotHandler(pool, minioClient, minioBucket))
	mux.HandleFunc("GET /datasets/{id}/snapshots", handler.ListSnapshotsHandler(pool))

	// Branches
	mux.HandleFunc("POST /datasets/{id}/branches", handler.CreateBranchHandler(pool))
	mux.HandleFunc("GET /datasets/{id}/branches", handler.ListBranchesHandler(pool))
	mux.HandleFunc("PATCH /branches/{id}", handler.UpdateBranchHandler(pool))
	mux.HandleFunc("POST /branches/{id}/merge", handler.MergeBranchHandler(pool))

	// Diff + column history
	mux.HandleFunc("GET /snapshots/{a}/diff/{b}", handler.DiffHandler(pool, minioClient, minioBucket))
	mux.HandleFunc("GET /datasets/{id}/columns/{col}/history", handler.ColumnHistoryHandler(pool))

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", cors(mux)))
}

// cors adds permissive CORS headers for the SvelteKit dev server.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loadEnv reads a simple KEY=VALUE file and sets env vars that aren't already set.
func loadEnv(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
