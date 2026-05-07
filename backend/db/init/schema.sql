CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users: email + bcrypt password hash
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Datasets: each represents a named versioned table with a designated primary key column
CREATE TABLE IF NOT EXISTS datasets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    primary_key_col TEXT NOT NULL,
    created_by      TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Snapshots: immutable content-addressed versions of a dataset's CSV
-- snapshot_hash = sha256(file_hash | schema_json | parent_hash)
-- Once inserted, rows are never updated or deleted (append-only)
CREATE TABLE IF NOT EXISTS snapshots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_id      UUID NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    parent_id       UUID REFERENCES snapshots(id),
    snapshot_hash   TEXT NOT NULL UNIQUE,  -- uniqueness enforces immutability
    file_hash       TEXT NOT NULL,
    row_count       INT NOT NULL CHECK (row_count >= 0),
    message         TEXT,
    created_by      TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_snapshots_dataset_id ON snapshots(dataset_id);
CREATE INDEX IF NOT EXISTS idx_snapshots_parent_id  ON snapshots(parent_id);
CREATE INDEX IF NOT EXISTS idx_snapshots_hash       ON snapshots(snapshot_hash);

-- Per-snapshot column schema (column names, types, and order)
CREATE TABLE IF NOT EXISTS snapshot_columns (
    snapshot_id     UUID NOT NULL REFERENCES snapshots(id) ON DELETE CASCADE,
    column_name     TEXT NOT NULL,
    column_type     TEXT NOT NULL CHECK (column_type IN ('numeric', 'text')),
    column_index    INT  NOT NULL CHECK (column_index >= 0),
    PRIMARY KEY (snapshot_id, column_name)
);

-- Per-snapshot, per-column statistics computed at commit time
CREATE TABLE IF NOT EXISTS snapshot_column_stats (
    snapshot_id     UUID NOT NULL REFERENCES snapshots(id) ON DELETE CASCADE,
    column_name     TEXT NOT NULL,
    min_value       TEXT,
    max_value       TEXT,
    mean_value      DOUBLE PRECISION,
    null_count      INT NOT NULL DEFAULT 0 CHECK (null_count  >= 0),
    unique_count    INT NOT NULL DEFAULT 0 CHECK (unique_count >= 0),
    PRIMARY KEY (snapshot_id, column_name)
);

-- Branches: named pointers to a snapshot (mutable head, append-only history)
CREATE TABLE IF NOT EXISTS branches (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_id       UUID NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    head_snapshot_id UUID REFERENCES snapshots(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (dataset_id, name)
);

CREATE INDEX IF NOT EXISTS idx_branches_dataset_id ON branches(dataset_id);

-- Merge events: record when a source branch was merged into a target branch
CREATE TABLE IF NOT EXISTS merges (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_branch_id UUID NOT NULL REFERENCES branches(id),
    target_branch_id UUID NOT NULL REFERENCES branches(id),
    snapshot_id      UUID NOT NULL REFERENCES snapshots(id),
    merged_by        TEXT NOT NULL,
    merged_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Diff cache: stores pre-computed diffs keyed by the two snapshot hashes
CREATE TABLE IF NOT EXISTS diff_cache (
    snapshot_a_hash TEXT NOT NULL,
    snapshot_b_hash TEXT NOT NULL,
    diff_json       JSONB NOT NULL,
    computed_ms     INT  NOT NULL DEFAULT 0,
    computed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (snapshot_a_hash, snapshot_b_hash)
);
