<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { DatasetSummary } from '$lib/types';

	let datasets: DatasetSummary[] = $state([]);
	let loading = $state(true);
	let error = $state('');
	let showCreate = $state(false);
	let creating = $state(false);
	let newName = $state('');
	let newPKCol = $state('');

	onMount(async () => {
		try {
			datasets = await api.datasets.list();
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	});

	async function createDataset() {
		if (!newName.trim() || !newPKCol.trim()) return;
		creating = true;
		try {
			const d = await api.datasets.create(newName.trim(), newPKCol.trim());
			datasets = [{ ...d, snapshot_count: 0, last_updated: null }, ...datasets];
			newName = '';
			newPKCol = '';
			showCreate = false;
		} catch (e: any) {
			error = e.message;
		} finally {
			creating = false;
		}
	}

	function relativeTime(iso: string | null) {
		if (!iso) return '—';
		const diff = Date.now() - new Date(iso).getTime();
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return 'just now';
		if (mins < 60) return `${mins}m ago`;
		const hrs = Math.floor(mins / 60);
		if (hrs < 24) return `${hrs}h ago`;
		return `${Math.floor(hrs / 24)}d ago`;
	}
</script>

<svelte:head><title>DatasetVC — Datasets</title></svelte:head>

<div class="mb-6 flex items-center justify-between">
	<div>
		<h1 class="text-2xl font-bold">Datasets</h1>
		<p class="mt-1 text-sm text-gray-400">Version-controlled CSV datasets with full diff history.</p>
	</div>
	<button
		onclick={() => (showCreate = !showCreate)}
		class="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium hover:bg-indigo-500 transition-colors"
	>
		+ New Dataset
	</button>
</div>

{#if showCreate}
<div class="mb-6 rounded-xl border border-gray-700 bg-gray-900 p-5">
	<h2 class="mb-4 font-semibold">Create Dataset</h2>
	<div class="flex gap-3">
		<input
			bind:value={newName}
			placeholder="Dataset name"
			class="flex-1 rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-sm placeholder-gray-500 focus:border-indigo-500 focus:outline-none"
		/>
		<input
			bind:value={newPKCol}
			placeholder="Primary key column (e.g. id)"
			class="flex-1 rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-sm placeholder-gray-500 focus:border-indigo-500 focus:outline-none"
		/>
		<button
			onclick={createDataset}
			disabled={creating}
			class="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium hover:bg-indigo-500 disabled:opacity-50 transition-colors"
		>
			{creating ? 'Creating…' : 'Create'}
		</button>
	</div>
</div>
{/if}

{#if error}
<div class="mb-4 rounded-lg border border-red-700 bg-red-900/30 px-4 py-3 text-sm text-red-300">{error}</div>
{/if}

{#if loading}
<div class="flex items-center justify-center py-20 text-gray-500">
	<svg class="mr-2 h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24">
		<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
		<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z"/>
	</svg>
	Loading datasets…
</div>
{:else if datasets.length === 0}
<div class="flex flex-col items-center justify-center py-20 text-gray-500">
	<p class="text-lg">No datasets yet.</p>
	<p class="mt-1 text-sm">Click <span class="text-indigo-400">+ New Dataset</span> to get started.</p>
</div>
{:else}
<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
	{#each datasets as d}
	<a
		href="/datasets/{d.id}"
		class="group rounded-xl border border-gray-800 bg-gray-900 p-5 transition-all hover:border-indigo-700 hover:bg-gray-800"
	>
		<div class="flex items-start justify-between">
			<div class="min-w-0">
				<h2 class="truncate font-semibold group-hover:text-indigo-300">{d.name}</h2>
				<p class="mt-0.5 text-xs text-gray-500">pk: <code class="text-gray-400">{d.primary_key_col}</code></p>
			</div>
			<span class="ml-2 shrink-0 rounded-full bg-gray-800 px-2 py-0.5 text-xs font-medium text-gray-400 group-hover:bg-gray-700">
				{d.snapshot_count} {d.snapshot_count === 1 ? 'snapshot' : 'snapshots'}
			</span>
		</div>
		<p class="mt-3 text-xs text-gray-600">Updated {relativeTime(d.last_updated)}</p>
	</a>
	{/each}
</div>
{/if}
