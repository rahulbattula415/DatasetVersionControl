import type {
	Branch,
	ColumnStatHistory,
	Dataset,
	DatasetSummary,
	DiffResult,
	Snapshot
} from './types';

const BASE = 'http://localhost:8080';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(`${BASE}${path}`, init);
	if (!res.ok) {
		const text = await res.text().catch(() => res.statusText);
		throw new Error(`${res.status}: ${text}`);
	}
	return res.json() as Promise<T>;
}

// ─── Datasets ──────────────────────────────────────────────────────────────────

export const api = {
	datasets: {
		list: () => request<DatasetSummary[]>('/datasets'),
		get: (id: string) => request<Dataset>(`/datasets/${id}`),
		create: (name: string, primary_key_col: string) =>
			request<Dataset>('/datasets', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ name, primary_key_col })
			})
	},

	snapshots: {
		list: (datasetId: string) => request<Snapshot[]>(`/datasets/${datasetId}/snapshots`),
		upload: (datasetId: string, file: File, message: string, branchId?: string) => {
			const form = new FormData();
			form.append('file', file);
			if (message) form.append('message', message);
			if (branchId) form.append('branch_id', branchId);
			return request<Snapshot>(`/datasets/${datasetId}/snapshots`, { method: 'POST', body: form });
		}
	},

	branches: {
		list: (datasetId: string) => request<Branch[]>(`/datasets/${datasetId}/branches`),
		create: (datasetId: string, name: string, headSnapshotId?: string) =>
			request<Branch>(`/datasets/${datasetId}/branches`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ name, head_snapshot_id: headSnapshotId ?? null })
			}),
		advance: (branchId: string, headSnapshotId: string) =>
			request<Branch>(`/branches/${branchId}`, {
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ head_snapshot_id: headSnapshotId })
			}),
		merge: (targetBranchId: string, sourceBranchId: string) =>
			request<{ merged: boolean; target_branch: Branch }>(`/branches/${targetBranchId}/merge`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ source_branch_id: sourceBranchId })
			})
	},

	diff: {
		get: (snapAId: string, snapBId: string, page = 1, pageSize = 100) =>
			request<DiffResult>(
				`/snapshots/${snapAId}/diff/${snapBId}?page=${page}&page_size=${pageSize}`
			)
	},

	columns: {
		history: (datasetId: string, col: string, branchId?: string) => {
			const qs = branchId ? `?branch_id=${branchId}` : '';
			return request<ColumnStatHistory[]>(`/datasets/${datasetId}/columns/${col}/history${qs}`);
		}
	}
};
