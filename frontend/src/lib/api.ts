import type {
	Branch,
	ColumnStatHistory,
	Dataset,
	DatasetSummary,
	DiffResult,
	Snapshot
} from './types';

const BASE = 'http://localhost:8080';

export const token = {
	get: (): string | null => (typeof localStorage !== 'undefined' ? localStorage.getItem('jwt') : null),
	set: (t: string) => localStorage.setItem('jwt', t),
	clear: () => localStorage.removeItem('jwt'),
	email: (): string | null => (typeof localStorage !== 'undefined' ? localStorage.getItem('user_email') : null),
	setEmail: (e: string) => localStorage.setItem('user_email', e)
};

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
	const t = token.get();
	const headers = new Headers(init.headers);
	if (t) headers.set('Authorization', `Bearer ${t}`);

	const res = await fetch(`${BASE}${path}`, { ...init, headers });

	if (res.status === 401) {
		token.clear();
		window.location.href = '/login';
		throw new Error('Unauthorized');
	}

	if (!res.ok) {
		const text = await res.text().catch(() => res.statusText);
		throw new Error(`${res.status}: ${text}`);
	}
	return res.json() as Promise<T>;
}

// ─── Auth ──────────────────────────────────────────────────────────────────────

export const auth = {
	register: async (email: string, password: string): Promise<string> => {
		const res = await fetch(`${BASE}/auth/register`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ email, password })
		});
		if (!res.ok) {
			const text = await res.text().catch(() => res.statusText);
			throw new Error(`${res.status}: ${text}`);
		}
		const data: { token: string } = await res.json();
		token.set(data.token);
		token.setEmail(email);
		return data.token;
	},

	login: async (email: string, password: string): Promise<string> => {
		const res = await fetch(`${BASE}/auth/login`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ email, password })
		});
		if (!res.ok) {
			const text = await res.text().catch(() => res.statusText);
			throw new Error(`${res.status}: ${text}`);
		}
		const data: { token: string } = await res.json();
		token.set(data.token);
		token.setEmail(email);
		return data.token;
	},

	logout: () => {
		token.clear();
		window.location.href = '/login';
	}
};

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
