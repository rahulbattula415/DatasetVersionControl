export interface Dataset {
	id: string;
	name: string;
	primary_key_col: string;
	created_by: string;
	created_at: string;
}

export interface DatasetSummary extends Dataset {
	snapshot_count: number;
	last_updated: string | null;
}

export interface SnapshotColumn {
	column_name: string;
	column_type: 'numeric' | 'text';
	column_index: number;
}

export interface Snapshot {
	id: string;
	dataset_id: string;
	parent_id: string | null;
	snapshot_hash: string;
	file_hash: string;
	row_count: number;
	message: string | null;
	created_by: string;
	created_at: string;
	columns?: SnapshotColumn[];
}

export interface Branch {
	id: string;
	dataset_id: string;
	name: string;
	head_snapshot_id: string | null;
	created_at: string;
}

export interface ColumnChange {
	column: string;
	old: string;
	new: string;
}

export interface ModifiedRow {
	key: string;
	changes: ColumnChange[];
}

export interface DiffSummary {
	added: number;
	deleted: number;
	modified: number;
}

export interface DiffResult {
	summary: DiffSummary;
	added: Record<string, string>[];
	deleted: Record<string, string>[];
	modified: ModifiedRow[];
	schema_a: string[];
	schema_b: string[];
	schema_match: boolean;
	schema_diff?: { added_columns: string[]; removed_columns: string[] };
	computed_ms: number;
	cached: boolean;
	page: number;
	page_size: number;
	total_rows: number;
}

export interface ColumnStatHistory {
	snapshot_id: string;
	created_at: string;
	message: string | null;
	min_value: string | null;
	max_value: string | null;
	mean_value: number | null;
	null_count: number;
	unique_count: number;
}
