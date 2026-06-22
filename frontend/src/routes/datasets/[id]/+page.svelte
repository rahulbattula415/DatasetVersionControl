<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { api } from '$lib/api';
	import type { Branch, Dataset, DiffResult, Snapshot } from '$lib/types';
	import SnapshotDiff from '$lib/components/SnapshotDiff.svelte';
	import ColumnExplorer from '$lib/components/ColumnExplorer.svelte';

	const datasetId = $derived($page.params.id ?? '');

	let dataset: Dataset | null = $state(null);
	let snapshots: Snapshot[] = $state([]);
	let branches: Branch[] = $state([]);
	let loading = $state(true);
	let error = $state('');

	let tab = $state<'history' | 'branches' | 'upload' | 'explore'>('history');
	let activeBranch: Branch | null = $state(null);

	// Upload state
	let uploadFile: File | null = $state(null);
	let uploadMessage = $state('');
	let uploading = $state(false);
	let uploadError = $state('');
	let uploadSuccess = $state('');
	let dragOver = $state(false);

	// Diff state
	let diffSnapA = $state('');
	let diffSnapB = $state('');
	let diffResult: DiffResult | null = $state(null);
	let diffLoading = $state(false);
	let diffError = $state('');

	// Branch create state
	let newBranchName = $state('');
	let newBranchFrom = $state('');
	let creatingBranch = $state(false);

	onMount(async () => {
		await reload();
	});

	async function reload() {
		loading = true;
		error = '';
		try {
			[dataset, snapshots, branches] = await Promise.all([
				api.datasets.get(datasetId),
				api.snapshots.list(datasetId),
				api.branches.list(datasetId)
			]);
			activeBranch = branches.find((b) => b.name === 'main') ?? branches[0] ?? null;
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function uploadSnapshot() {
		if (!uploadFile) return;
		uploading = true;
		uploadError = '';
		uploadSuccess = '';
		try {
			const snap = await api.snapshots.upload(
				datasetId,
				uploadFile,
				uploadMessage,
				activeBranch?.id
			);
			uploadSuccess = `Snapshot committed: ${snap.snapshot_hash.slice(0, 12)}… (${snap.row_count} rows)`;
			uploadFile = null;
			uploadMessage = '';
			snapshots = await api.snapshots.list(datasetId);
			branches = await api.branches.list(datasetId);
			activeBranch = branches.find((b) => b.id === activeBranch?.id) ?? activeBranch;
		} catch (e: any) {
			uploadError = e.message;
		} finally {
			uploading = false;
		}
	}

	async function runDiff() {
		if (!diffSnapA || !diffSnapB) return;
		diffLoading = true;
		diffError = '';
		diffResult = null;
		try {
			diffResult = await api.diff.get(diffSnapA, diffSnapB);
		} catch (e: any) {
			diffError = e.message;
		} finally {
			diffLoading = false;
		}
	}

	async function createBranch() {
		if (!newBranchName.trim()) return;
		creatingBranch = true;
		try {
			await api.branches.create(datasetId, newBranchName.trim(), newBranchFrom || undefined);
			newBranchName = '';
			newBranchFrom = '';
			branches = await api.branches.list(datasetId);
		} catch (e: any) {
			error = e.message;
		} finally {
			creatingBranch = false;
		}
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		dragOver = false;
		const f = e.dataTransfer?.files[0];
		if (f && f.name.endsWith('.csv')) uploadFile = f;
	}

	function fmt(iso: string) {
		return new Date(iso).toLocaleString();
	}

	function shortHash(h: string) {
		return h.slice(0, 10);
	}

	const branchSnapshots = $derived(
		activeBranch
			? snapshots.filter((s) => isInLineage(s, activeBranch!, snapshots))
			: snapshots
	);

	function isInLineage(snap: Snapshot, branch: Branch, all: Snapshot[]): boolean {
		// Simple lineage: traverse from branch head following parent_id
		if (!branch.head_snapshot_id) return false;
		const byId = Object.fromEntries(all.map((s) => [s.id, s]));
		let cur: Snapshot | undefined = byId[branch.head_snapshot_id];
		while (cur) {
			if (cur.id === snap.id) return true;
			cur = cur.parent_id ? byId[cur.parent_id] : undefined;
		}
		return false;
	}
</script>

<svelte:head><title>{dataset?.name ?? 'Dataset'} — DatasetVC</title></svelte:head>

{#if loading}
<div role="status" class="flex items-center justify-center py-20 text-ink-soft">
	<svg class="mr-2 h-5 w-5 animate-spin" aria-hidden="true" fill="none" viewBox="0 0 24 24">
		<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
		<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z"/>
	</svg>
	Loading…
</div>
{:else if error}
<div role="alert" class="rounded-lg border border-red-700 bg-red-900/30 px-4 py-3 text-sm text-red-300">{error}</div>
{:else if dataset}

<!-- Header -->
<div class="mb-6 flex flex-wrap items-start justify-between gap-4">
	<div>
		<div class="flex items-center gap-2 text-sm text-ink-soft">
			<a href="/datasets" class="hover:text-gray-300">Datasets</a>
			<span>/</span>
			<span class="text-gray-200">{dataset.name}</span>
		</div>
		<h1 class="mt-1 text-2xl font-bold wrap-break-word">{dataset.name}</h1>
		<p class="mt-1 text-sm text-ink-soft">
			Primary key: <code class="rounded bg-raised px-1 py-0.5 text-primary-data">{dataset.primary_key_col}</code>
			· {snapshots.length} snapshots
		</p>
	</div>

	<!-- Branch switcher -->
	<div class="flex items-center gap-2">
		<label for="branch-switcher" class="text-sm text-ink-soft">Branch:</label>
		<select
			id="branch-switcher"
			class="rounded-lg border border-edge-strong bg-raised px-3 py-1.5 text-sm focus:border-primary-hover focus:outline-none"
			value={activeBranch?.id ?? ''}
			onchange={(e) => {
				activeBranch = branches.find((b) => b.id === (e.target as HTMLSelectElement).value) ?? null;
			}}
		>
			{#each branches as b}
				<option value={b.id}>{b.name}</option>
			{/each}
		</select>
	</div>
</div>

<!-- Tabs -->
<div class="mb-6 flex gap-1 overflow-x-auto border-b border-edge">
	{#each (['history', 'branches', 'upload', 'explore'] as const) as t}
	<button
		onclick={() => (tab = t)}
		class="shrink-0 whitespace-nowrap px-4 py-2 text-sm font-medium capitalize transition-colors"
		class:border-b-2={tab === t}
		class:border-primary-hover={tab === t}
		class:text-white={tab === t}
		class:text-ink-soft={tab !== t}
		class:hover:text-gray-200={tab !== t}
	>
		{t}
	</button>
	{/each}
</div>

<!-- ─── History tab ──────────────────────────────────────────────────────── -->
{#if tab === 'history'}
<div class="space-y-4">
	<!-- Diff comparison -->
	<div class="rounded-xl border border-edge bg-surface p-4">
		<h2 class="mb-3 font-semibold text-sm text-gray-300">Compare Snapshots</h2>
		<div class="flex flex-wrap items-center gap-3">
			<select
				bind:value={diffSnapA}
				class="flex-1 min-w-40 rounded-lg border border-edge-strong bg-raised px-3 py-2 text-sm focus:border-primary-hover focus:outline-none"
			>
				<option value="">Snapshot A (base)</option>
				{#each snapshots as s}
					<option value={s.id}>{shortHash(s.snapshot_hash)} — {s.message ?? 'no message'} ({s.row_count} rows)</option>
				{/each}
			</select>
			<span class="text-ink-soft" aria-hidden="true">→</span>
			<select
				bind:value={diffSnapB}
				class="flex-1 min-w-40 rounded-lg border border-edge-strong bg-raised px-3 py-2 text-sm focus:border-primary-hover focus:outline-none"
			>
				<option value="">Snapshot B (head)</option>
				{#each snapshots as s}
					<option value={s.id}>{shortHash(s.snapshot_hash)} — {s.message ?? 'no message'} ({s.row_count} rows)</option>
				{/each}
			</select>
			<button
				onclick={runDiff}
				disabled={!diffSnapA || !diffSnapB || diffLoading}
				class="rounded-lg bg-primary px-4 py-2 text-sm font-medium hover:bg-primary-hover disabled:opacity-40 transition-colors"
			>
				{diffLoading ? 'Computing…' : 'Compare'}
			</button>
		</div>
	</div>

	{#if diffError}
	<div role="alert" class="rounded-lg border border-red-700 bg-red-900/30 px-4 py-3 text-sm text-red-300">{diffError}</div>
	{/if}

	{#if diffResult}
	<SnapshotDiff diff={diffResult} primaryKey={dataset.primary_key_col} />
	{/if}

	<!-- Snapshot timeline -->
	<div class="space-y-2">
		<h2 class="font-semibold text-sm text-gray-300">Snapshot History</h2>
		{#each branchSnapshots as snap, i}
		<div class="flex items-start gap-4 rounded-xl border border-edge bg-surface p-4">
			<div class="flex flex-col items-center">
				<div class="h-3 w-3 rounded-full bg-primary-hover"></div>
				{#if i < branchSnapshots.length - 1}
				<div class="mt-1 w-0.5 flex-1 bg-raised" style="min-height:2rem"></div>
				{/if}
			</div>
			<div class="min-w-0 flex-1">
				<div class="flex flex-wrap items-center gap-2">
					<code class="rounded bg-raised px-2 py-0.5 text-xs font-mono text-primary-data">
						{shortHash(snap.snapshot_hash)}
					</code>
					{#if snap.message}
					<span class="min-w-0 wrap-break-word font-medium">{snap.message}</span>
					{:else}
					<span class="text-ink-soft italic">no message</span>
					{/if}
					{#if snap.parent_id === null}
					<span class="rounded-full bg-emerald-900/50 px-2 py-0.5 text-xs text-emerald-400">initial</span>
					{/if}
				</div>
				<p class="mt-1 text-xs text-ink-soft">
					{snap.row_count} rows · {fmt(snap.created_at)}
				</p>
			</div>
			<button
				onclick={() => { diffSnapB = snap.id; }}
				class="shrink-0 rounded px-2 py-1 text-xs text-ink-soft hover:bg-raised hover:text-gray-300"
				title="Set as diff target"
			>
				→ diff
			</button>
		</div>
		{:else}
		<p class="text-sm text-ink-soft">No snapshots on this branch yet.</p>
		{/each}
	</div>
</div>

<!-- ─── Branches tab ─────────────────────────────────────────────────────── -->
{:else if tab === 'branches'}
<div class="space-y-4">
	<!-- Create branch -->
	<div class="rounded-xl border border-edge bg-surface p-4">
		<h2 class="mb-3 font-semibold text-sm text-gray-300">New Branch</h2>
		<div class="flex flex-wrap gap-3">
			<input
				bind:value={newBranchName}
				placeholder="Branch name"
				class="flex-1 min-w-40 rounded-lg border border-edge-strong bg-raised px-3 py-2 text-sm placeholder-ink-soft focus:border-primary-hover focus:outline-none"
			/>
			<select
				bind:value={newBranchFrom}
				class="flex-1 min-w-40 rounded-lg border border-edge-strong bg-raised px-3 py-2 text-sm focus:border-primary-hover focus:outline-none"
			>
				<option value="">From snapshot (optional)</option>
				{#each snapshots as s}
					<option value={s.id}>{shortHash(s.snapshot_hash)} — {s.message ?? 'no message'}</option>
				{/each}
			</select>
			<button
				onclick={createBranch}
				disabled={creatingBranch || !newBranchName.trim()}
				class="rounded-lg bg-primary px-4 py-2 text-sm font-medium hover:bg-primary-hover disabled:opacity-40 transition-colors"
			>
				{creatingBranch ? 'Creating…' : 'Create'}
			</button>
		</div>
	</div>

	<!-- Branch list -->
	<div class="space-y-2">
		{#each branches as b}
		<div class="flex items-center justify-between rounded-xl border border-edge bg-surface p-4">
			<div>
				<div class="flex items-center gap-2 min-w-0">
					<span class="min-w-0 wrap-break-word font-medium">{b.name}</span>
					{#if b.name === 'main'}
					<span class="rounded-full bg-indigo-900/50 px-2 py-0.5 text-xs text-primary-link">default</span>
					{/if}
				</div>
				{#if b.head_snapshot_id}
				{@const headSnap = snapshots.find(s => s.id === b.head_snapshot_id)}
				<p class="mt-0.5 text-xs text-ink-soft">
					HEAD: <code class="text-primary-data">{shortHash(b.head_snapshot_id)}</code>
					{#if headSnap} · {headSnap.message ?? 'no message'} · {headSnap.row_count} rows{/if}
				</p>
				{:else}
				<p class="mt-0.5 text-xs text-ink-soft">No commits yet</p>
				{/if}
			</div>
			<button
				onclick={() => { activeBranch = b; tab = 'history'; }}
				class="rounded px-3 py-1 text-xs text-ink-soft hover:bg-raised hover:text-gray-200"
			>
				View
			</button>
		</div>
		{/each}
	</div>
</div>

<!-- ─── Upload tab ───────────────────────────────────────────────────────── -->
{:else if tab === 'upload'}
<div class="max-w-xl space-y-4">
	<div class="rounded-xl border border-edge bg-surface p-5">
		<h2 class="mb-4 font-semibold">Commit New Snapshot</h2>
		<p class="mb-4 text-sm text-ink-soft">
			Uploading to branch <span class="font-medium text-primary-data">{activeBranch?.name ?? '—'}</span>.
		</p>

		<!-- Drop zone -->
		<div
			class="mb-4 rounded-xl border-2 border-dashed p-8 text-center transition-colors cursor-pointer"
			class:border-primary-hover={dragOver}
			class:border-edge-strong={!dragOver}
			ondragover={(e) => { e.preventDefault(); dragOver = true; }}
			ondragleave={() => { dragOver = false; }}
			ondrop={handleDrop}
			onclick={() => document.getElementById('csv-input')?.click()}
			role="button"
			tabindex="0"
			aria-label="Upload a CSV file: drop a file here, or press Enter or Space to browse"
			onkeydown={(e) => {
				if (e.key === 'Enter' || e.key === ' ') {
					e.preventDefault();
					document.getElementById('csv-input')?.click();
				}
			}}
		>
			<input
				id="csv-input"
				type="file"
				accept=".csv"
				class="hidden"
				onchange={(e) => { uploadFile = (e.target as HTMLInputElement).files?.[0] ?? null; }}
			/>
			{#if uploadFile}
			<p class="text-primary-data font-medium">{uploadFile.name}</p>
			<p class="mt-1 text-xs text-ink-soft">{(uploadFile.size / 1024).toFixed(1)} KB</p>
			{:else}
			<p class="text-ink-soft">Drop a CSV here or click to browse</p>
			<p class="mt-1 text-xs text-ink-soft">Max 100 MB</p>
			{/if}
		</div>

		<input
			bind:value={uploadMessage}
			placeholder="Commit message (optional)"
			class="mb-4 w-full rounded-lg border border-edge-strong bg-raised px-3 py-2 text-sm placeholder-ink-soft focus:border-primary-hover focus:outline-none"
		/>

		<button
			onclick={uploadSnapshot}
			disabled={!uploadFile || uploading}
			class="w-full rounded-lg bg-primary py-2 text-sm font-medium hover:bg-primary-hover disabled:opacity-40 transition-colors"
		>
			{uploading ? 'Committing…' : 'Commit Snapshot'}
		</button>

		{#if uploadSuccess}
		<div role="status" class="mt-3 rounded-lg border border-emerald-700 bg-emerald-900/30 px-4 py-3 text-sm text-emerald-300">{uploadSuccess}</div>
		{/if}
		{#if uploadError}
		<div role="alert" class="mt-3 rounded-lg border border-red-700 bg-red-900/30 px-4 py-3 text-sm text-red-300">{uploadError}</div>
		{/if}
	</div>
</div>

<!-- ─── Explore tab ──────────────────────────────────────────────────────── -->
{:else if tab === 'explore'}
<ColumnExplorer
	datasetId={datasetId}
	columns={snapshots[0]?.columns ?? []}
	branchId={activeBranch?.id}
/>
{/if}

{/if}
