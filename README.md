# DatasetVC

A Git-like version control system for CSV datasets. Track schema changes, diff any two versions row-by-row, branch and merge datasets, and explore per-column statistics over time — all through a clean web UI and REST API.

---

## Why it's hard

CSV datasets change in ways that file-diff tools can't handle:
- **Row order is meaningless** — a sort is not a change.
- **Schema evolves** — columns get added, renamed, or dropped.
- **Identity matters** — a modified row is not a delete + add; it's the same entity with changed fields.
- **Scale** — 50 k-row datasets need sub-5-second diffs without hitting the database per row.

DatasetVC solves this with content-addressed, immutable snapshots and a primary-key-aware diff engine.

---

## Architecture

```
┌─────────────────────┐      ┌──────────────────┐
│   SvelteKit UI      │─────▶│  Go REST API      │
│  (port 5173)        │      │  (port 8080)       │
└─────────────────────┘      └────────┬───────────┘
                                       │
                        ┌──────────────┴──────────────┐
                        │                             │
                  ┌─────▼──────┐             ┌────────▼──────┐
                  │ PostgreSQL │             │     MinIO      │
                  │  metadata  │             │  CSV objects   │
                  │ (port 5432)│             │  (port 9000)   │
                  └────────────┘             └───────────────┘
```

### Data model

```
datasets          (id, name, primary_key_col)
  └── snapshots   (id, dataset_id, parent_id, snapshot_hash, file_hash, row_count)
        └── snapshot_columns       (snapshot_id, column_name, column_type, column_index)
        └── snapshot_column_stats  (snapshot_id, column_name, min, max, mean, nulls, uniques)
  └── branches    (id, dataset_id, name, head_snapshot_id)
  └── merges      (id, source_branch_id, target_branch_id, snapshot_id)

diff_cache        (snapshot_a_hash, snapshot_b_hash, diff_json)   ← content-addressed cache
```

### Snapshot immutability

Every snapshot is identified by a **deterministic hash**:

```
snapshot_hash = sha256(file_hash | sorted_schema_json | parent_snapshot_id_or_ROOT)
```

- `file_hash` — SHA-256 of the raw CSV bytes (deduplicates unchanged files).
- `sorted_schema_json` — sorted column names + types (catches schema-only changes).
- `parent_snapshot_id` — embeds lineage so the same file from a different parent produces a different hash.

The `snapshot_hash` column has a `UNIQUE` constraint, so inserting a duplicate is a DB-level no-op. No `UPDATE` queries ever touch the `snapshots` table.

---

## API reference

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/datasets` | Create dataset (`name`, `primary_key_col`) |
| `GET`  | `/datasets` | List datasets with snapshot count |
| `GET`  | `/datasets/:id` | Get single dataset |
| `POST` | `/datasets/:id/snapshots` | Commit a CSV (`file`, `message?`, `branch_id?`) |
| `GET`  | `/datasets/:id/snapshots` | List snapshots (newest first) |
| `POST` | `/datasets/:id/branches` | Create branch (`name`, `head_snapshot_id?`) |
| `GET`  | `/datasets/:id/branches` | List branches |
| `PATCH`| `/branches/:id` | Fast-forward head (`head_snapshot_id` must be descendant) |
| `POST` | `/branches/:id/merge` | Merge source branch (fast-forward only) |
| `GET`  | `/snapshots/:a/diff/:b` | Row-level diff (`page`, `page_size`) |
| `GET`  | `/datasets/:id/columns/:col/history` | Column stats over time (`branch_id?`) |

### Diff response

```json
{
  "summary": { "added": 2, "deleted": 1, "modified": 3 },
  "added":    [ { "id": "11", "name": "Dave", "revenue": "4000" } ],
  "deleted":  [ { "id": "5",  "name": "Eve",  "revenue": "900"  } ],
  "modified": [
    { "key": "1", "changes": [{ "column": "revenue", "old": "1000", "new": "1500" }] }
  ],
  "schema_match": true,
  "cached": false,
  "computed_ms": 42,
  "page": 1,
  "page_size": 100,
  "total_rows": 6
}
```

---

## Getting started

### Prerequisites

- Docker & Docker Compose
- Go 1.22+
- Node.js 18+

### 1. Start infrastructure

```bash
cd docker-compose-pg
docker compose -f docker-compose-pg.yaml up -d
```

This starts Postgres (port 5432), MinIO (port 9000/9001), and runs `schema.sql` automatically.

### 2. Start the backend

```bash
cd backend
go run ./cmd/server
```

The server reads `.env` for `DATABASE_URL`, `MINIO_*` — defaults work with the Docker setup above.

### 3. Start the frontend

```bash
cd frontend
npm run dev
```

Open [http://localhost:5173](http://localhost:5173).

### 4. Load demo data

```bash
bash seed/seed.sh
```

This creates a `regional_sales` dataset and commits three quarterly CSV snapshots (Q1 → Q2 → Q3), each with additions, deletions, and modifications visible in the diff view.

### 5. Run integration tests

```bash
cd backend
DATABASE_URL=postgresql://root:secret@localhost:5432/dvc \
MINIO_ENDPOINT=localhost:9000 MINIO_ACCESS_KEY=minioadmin MINIO_SECRET_KEY=minioadmin \
go test ./internal/handler/ -v -run Integration -count=1
```

---

## Key design decisions

| Decision | Rationale |
|----------|-----------|
| Append-only `snapshots` table | Immutability enforced at DB level via `UNIQUE(snapshot_hash)` — no UPDATE ever |
| MinIO for CSV blobs | Keeps Postgres lean; S3-compatible for cloud migration |
| Fast-forward-only merges | Avoids 3-way merge complexity; simple ancestry check via recursive CTE |
| DB diff cache | Avoids re-loading 50 k-row files on every request; keyed by `(hash_a, hash_b)` |
| Column stats at commit time | Stats are cheap at write time; makes the history chart instant at read time |
