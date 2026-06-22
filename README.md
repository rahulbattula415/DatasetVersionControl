# DatasetVC

A Git-like version control system for CSV datasets. Track schema changes, diff any two versions row-by-row, branch and merge datasets, and explore per-column statistics over time — all through a clean web UI and REST API.

**Live demo:** deployed with Vercel (frontend) + Render (backend) + Supabase (Postgres) + Cloudflare R2 (storage).

---

## Why it's hard

CSV datasets change in ways that file-diff tools can't handle:
- **Row order is meaningless** — a sort is not a change.
- **Schema evolves** — columns get added, renamed, or dropped.
- **Identity matters** — a modified row is not a delete + add; it's the same entity with changed fields.
- **Scale** — large datasets need fast diffs without hitting the database per row.

DatasetVC solves this with content-addressed, immutable snapshots and a primary-key-aware diff engine.

---

## Architecture

```
Browser (Vercel)  ──►  Go API (Render)  ──►  PostgreSQL (Supabase)
                                         ──►  Cloudflare R2 (CSV storage)
```

| Layer | Technology | Hosting |
|-------|-----------|---------|
| Frontend | SvelteKit 2 + Svelte 5 + Tailwind CSS | Vercel |
| Backend | Go 1.25, `net/http` | Render (Docker) |
| Database | PostgreSQL (pgx/v5) | Supabase (session pooler) |
| Object storage | Cloudflare R2 (S3-compatible via `minio-go`) | Cloudflare |
| Auth | JWT (HS256, 7-day expiry) + bcrypt | — |

---

## Features

- **Snapshots** — every CSV upload is an immutable, content-addressed commit
- **Branching** — create branches from any snapshot; fast-forward-only merges
- **Row-level diffing** — compare any two snapshots by primary key (added / deleted / modified rows), with diff caching
- **Column Explorer** — min/max/mean/nulls/unique stats per column, with trend charts across snapshots
- **Authentication** — per-user datasets; all data is private and scoped to the authenticated user

---

## Data model

```
users             (id, email, password_hash)
datasets          (id, name, primary_key_col, created_by)
  └── snapshots   (id, dataset_id, parent_id, snapshot_hash, file_hash, row_count, message)
        └── snapshot_columns       (snapshot_id, column_name, column_type, column_index)
        └── snapshot_column_stats  (snapshot_id, column_name, min, max, mean, nulls, uniques)
  └── branches    (id, dataset_id, name, head_snapshot_id)
  └── merges      (id, source_branch_id, target_branch_id, snapshot_id, merged_by)

diff_cache        (snapshot_a_hash, snapshot_b_hash, diff_json)
```

### Snapshot immutability

Every snapshot is identified by a **deterministic hash**:

```
snapshot_hash = sha256(file_hash | sorted_schema_json | parent_snapshot_id_or_ROOT)
```

- `file_hash` — SHA-256 of the raw CSV bytes (deduplicates unchanged files).
- `sorted_schema_json` — sorted column names + types (catches schema-only changes).
- `parent_snapshot_id` — embeds lineage so the same file on a different branch produces a different hash.

The `snapshot_hash` column has a `UNIQUE` constraint, so inserting a duplicate is a DB-level no-op. The `snapshots` table is append-only.

---

## API reference

All routes except `/auth/*` require `Authorization: Bearer <token>`.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/auth/register` | Create account, returns JWT |
| `POST` | `/auth/login` | Sign in, returns JWT |
| `GET`  | `/datasets` | List authenticated user's datasets |
| `POST` | `/datasets` | Create dataset (`name`, `primary_key_col`) |
| `GET`  | `/datasets/:id` | Get single dataset |
| `POST` | `/datasets/:id/snapshots` | Commit a CSV (`file`, `message?`, `branch_id?`) |
| `GET`  | `/datasets/:id/snapshots` | List snapshots (newest first) |
| `POST` | `/datasets/:id/branches` | Create branch (`name`, `head_snapshot_id?`) |
| `GET`  | `/datasets/:id/branches` | List branches |
| `PATCH`| `/branches/:id` | Fast-forward head (`head_snapshot_id` must be a descendant) |
| `POST` | `/branches/:id/merge` | Merge source branch into target (fast-forward only) |
| `GET`  | `/snapshots/:a/diff/:b` | Row-level diff (`page`, `page_size`) |
| `GET`  | `/datasets/:id/columns/:col/history` | Column stats over time (`branch_id?`) |

### Diff response shape

```json
{
  "summary":     { "added": 2, "deleted": 1, "modified": 3 },
  "added":       [ { "id": "11", "name": "Dave", "revenue": "4000" } ],
  "deleted":     [ { "id": "5",  "name": "Eve",  "revenue": "900"  } ],
  "modified":    [ { "key": "1", "changes": [{ "column": "revenue", "old": "1000", "new": "1500" }] } ],
  "schema_match": true,
  "cached":       false,
  "computed_ms":  42,
  "page": 1, "page_size": 100, "total_rows": 6
}
```

---

## Local development

### Prerequisites

- Docker & Docker Compose
- Go 1.25+
- Node.js 18+

### 1. Start infrastructure

```bash
cd docker-compose-pg
docker compose -f docker-compose-pg.yaml up -d
```

Starts Postgres on port `5433` and MinIO on port `9000/9001`. The schema is applied automatically on server startup.

### 2. Start the backend

```bash
cd backend
go run ./cmd/server
```

Reads config from `backend/.env`. The defaults match the Docker setup above:

```env
DATABASE_URL=postgresql://root:secret@localhost:5433/dvc?sslmode=disable
JWT_SECRET=change-me-in-production-use-32-plus-random-bytes
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=datasets
# MINIO_SECURE=true   ← set this in production
```

### 3. Start the frontend

```bash
cd frontend
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173). Create an account, then start uploading CSVs.

The frontend reads `PUBLIC_API_BASE` from `frontend/.env` (defaults to `http://localhost:8080`).

---

## Deployment

### Backend → Render

1. Create a new **Web Service** on Render, connect the repo, set **Root Directory** to `backend`, **Runtime** to Docker.
2. Provision the database on **Supabase** (see [Database → Supabase](#database--supabase) below) and set its **session pooler** connection string as `DATABASE_URL`.
3. Set the remaining environment variables:

| Variable | Value |
|----------|-------|
| `DATABASE_URL` | Supabase **session pooler** URI, e.g. `postgresql://postgres.<ref>:<pw>@aws-0-<region>.pooler.supabase.com:5432/postgres?sslmode=require` |
| `JWT_SECRET` | Any long random string |
| `MINIO_ENDPOINT` | `<account-id>.r2.cloudflarestorage.com` (no scheme) |
| `MINIO_ACCESS_KEY` | Cloudflare R2 token **Access Key ID** |
| `MINIO_SECRET_KEY` | Cloudflare R2 token **Secret Access Key** |
| `MINIO_BUCKET` | Your R2 bucket name (exact match, case-sensitive) |
| `MINIO_SECURE` | `true` |

The schema runs automatically on every startup — no separate migration step needed.

### Database → Supabase

1. Create a project at [supabase.com](https://supabase.com) and save the database password (use letters/numbers only to avoid URL-encoding the connection string).
2. **Connect** → **Session pooler** (port `5432`) and copy the URI. Use the session pooler, **not** the direct connection — the direct host is IPv6-only on the free tier and is unreachable from Render's IPv4 network.
3. Append `?sslmode=require` and set the result as `DATABASE_URL` in Render.

The schema (`backend/internal/db/schema.sql`) is applied automatically on startup; no manual SQL is needed in Supabase.

### Storage → Cloudflare R2

1. Create a bucket in Cloudflare R2 with **public access disabled**.
2. Go to **R2 → API Tokens → Create Account API Token**.
3. Set permissions to **Object Read & Write**. If you scope it to specific buckets, the scope **must include the bucket named in `MINIO_BUCKET`** — a mismatch returns `Access Denied` on upload.
4. Copy the resulting **Access Key ID** and **Secret Access Key** as a matched pair into `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY` (the secret is shown only once). The S3 endpoint shown on the same screen goes into `MINIO_ENDPOINT` with `https://` stripped.

### Frontend → Vercel

1. Import the repo on Vercel. Set **Root Directory** to `frontend`.
2. Add one environment variable: `PUBLIC_API_BASE` = your Render service URL (no trailing slash).
3. Deploy. Vercel auto-detects SvelteKit via `frontend/vercel.json`.

---

## Key design decisions

| Decision | Rationale |
|----------|-----------|
| Append-only `snapshots` table | Immutability enforced at DB level via `UNIQUE(snapshot_hash)` — no UPDATE ever touches it |
| Cloudflare R2 for CSV blobs | Keeps Postgres lean; S3-compatible API means the same `minio-go` client works locally and in production |
| Fast-forward-only merges | Avoids 3-way merge complexity; ancestry verified via recursive CTE |
| DB diff cache | Avoids re-reading large files on repeat requests; keyed by `(hash_a, hash_b)` |
| Column stats at commit time | Cheap at write time, instant at read time — no full table scan needed for the history chart |
| Schema migration on startup | All DDL uses `IF NOT EXISTS`; running on every boot is safe and removes a manual deploy step |
| `ssr = false` on frontend | All pages use `localStorage` for auth; disabling SSR avoids server-side rendering errors on Vercel |
