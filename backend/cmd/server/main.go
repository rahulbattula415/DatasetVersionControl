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
	"github.com/rahulbattula415/DatasetVersionControl/internal/middleware"
)

func main() {
	loadEnv(".env")

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, mustEnv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("Connected to database")

	minioClient, err := minio.New(mustEnv("MINIO_ENDPOINT"), &minio.Options{
		Creds:  credentials.NewStaticV4(mustEnv("MINIO_ACCESS_KEY"), mustEnv("MINIO_SECRET_KEY"), ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("cannot create MinIO client: %v", err)
	}
	minioBucket := mustEnv("MINIO_BUCKET")
	if exists, _ := minioClient.BucketExists(ctx, minioBucket); !exists {
		if err := minioClient.MakeBucket(ctx, minioBucket, minio.MakeBucketOptions{}); err != nil {
			log.Fatalf("cannot create MinIO bucket: %v", err)
		}
	}
	log.Println("Connected to MinIO")

	jwtSecret := mustEnv("JWT_SECRET")
	auth := middleware.Require(jwtSecret)

	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("POST /auth/register", handler.RegisterHandler(pool, jwtSecret))
	mux.HandleFunc("POST /auth/login", handler.LoginHandler(pool, jwtSecret))

	// Protected — all routes below require a valid JWT
	mux.Handle("POST /datasets",     auth(handler.CreateDatasetHandler(pool)))
	mux.Handle("GET /datasets",      auth(handler.ListDatasetsHandler(pool)))
	mux.Handle("GET /datasets/{id}", auth(handler.GetDatasetHandler(pool)))

	mux.Handle("POST /datasets/{id}/snapshots", auth(handler.CreateSnapshotHandler(pool, minioClient, minioBucket)))
	mux.Handle("GET /datasets/{id}/snapshots",  auth(handler.ListSnapshotsHandler(pool)))

	mux.Handle("POST /datasets/{id}/branches", auth(handler.CreateBranchHandler(pool)))
	mux.Handle("GET /datasets/{id}/branches",  auth(handler.ListBranchesHandler(pool)))
	mux.Handle("PATCH /branches/{id}",         auth(handler.UpdateBranchHandler(pool)))
	mux.Handle("POST /branches/{id}/merge",    auth(handler.MergeBranchHandler(pool)))

	mux.Handle("GET /snapshots/{a}/diff/{b}",              auth(handler.DiffHandler(pool, minioClient, minioBucket)))
	mux.Handle("GET /datasets/{id}/columns/{col}/history", auth(handler.ColumnHistoryHandler(pool)))

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", cors(mux)))
}

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
		key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
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
