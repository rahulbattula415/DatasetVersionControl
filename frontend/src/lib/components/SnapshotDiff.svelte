<script lang="ts">
	import type { DiffResult } from '$lib/types';

	let { diff, primaryKey }: { diff: DiffResult; primaryKey: string } = $props();

	type Filter = 'all' | 'added' | 'deleted' | 'modified';
	let filter = $state<Filter>('all');

	const showAdded = $derived(filter === 'all' || filter === 'added');
	const showDeleted = $derived(filter === 'all' || filter === 'deleted');
	const showModified = $derived(filter === 'all' || filter === 'modified');

	// All columns across both schemas (union)
	const allColumns = $derived([...new Set([...diff.schema_a, ...diff.schema_b])]);
</script>

<div class="space-y-4">
	<!-- Summary bar -->
	<div class="flex flex-wrap items-center gap-3 rounded-xl border border-gray-800 bg-gray-900 p-4">
		<div class="flex gap-4 text-sm">
			<button
				onclick={() => filter = 'all'}
				class="font-medium transition-colors"
				class:text-white={filter === 'all'}
				class:text-gray-400={filter !== 'all'}
			>
				All <span class="text-gray-500">{diff.summary.added + diff.summary.deleted + diff.summary.modified}</span>
			</button>
			<button
				onclick={() => filter = 'added'}
				class="font-medium transition-colors"
				class:text-emerald-400={filter === 'added'}
				class:text-gray-400={filter !== 'added'}
			>
				+ Added <span>{diff.summary.added}</span>
			</button>
			<button
				onclick={() => filter = 'deleted'}
				class="font-medium transition-colors"
				class:text-red-400={filter === 'deleted'}
				class:text-gray-400={filter !== 'deleted'}
			>
				− Deleted <span>{diff.summary.deleted}</span>
			</button>
			<button
				onclick={() => filter = 'modified'}
				class="font-medium transition-colors"
				class:text-amber-400={filter === 'modified'}
				class:text-gray-400={filter !== 'modified'}
			>
				~ Modified <span>{diff.summary.modified}</span>
			</button>
		</div>

		<div class="ml-auto flex items-center gap-2 text-xs text-gray-500">
			{#if diff.cached}
			<span class="rounded-full bg-gray-800 px-2 py-0.5">cached</span>
			{:else}
			<span class="rounded-full bg-gray-800 px-2 py-0.5">{diff.computed_ms}ms</span>
			{/if}
			{#if !diff.schema_match}
			<span class="rounded-full bg-amber-900/50 px-2 py-0.5 text-amber-400">schema mismatch</span>
			{/if}
		</div>
	</div>

	{#if !diff.schema_match && diff.schema_diff}
	<div class="rounded-lg border border-amber-800 bg-amber-900/20 px-4 py-3 text-sm">
		<p class="font-medium text-amber-300">Schema changed between snapshots</p>
		{#if diff.schema_diff.added_columns.length}
		<p class="mt-1 text-amber-200/70">Added columns: {diff.schema_diff.added_columns.join(', ')}</p>
		{/if}
		{#if diff.schema_diff.removed_columns.length}
		<p class="mt-1 text-amber-200/70">Removed columns: {diff.schema_diff.removed_columns.join(', ')}</p>
		{/if}
	</div>
	{/if}

	{#if diff.summary.added + diff.summary.deleted + diff.summary.modified === 0}
	<div class="flex items-center justify-center rounded-xl border border-gray-800 bg-gray-900 py-12 text-gray-500">
		Snapshots are identical.
	</div>
	{:else}

	<!-- Diff table -->
	<div class="overflow-x-auto rounded-xl border border-gray-800">
		<table class="w-full text-sm">
			<thead>
				<tr class="border-b border-gray-800 bg-gray-900">
					<th class="w-6 px-3 py-2"></th>
					{#each allColumns as col}
					<th class="px-3 py-2 text-left font-medium text-gray-400 whitespace-nowrap">{col}</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#if showAdded}
				{#each diff.added as row}
				<tr class="border-b border-gray-800/50 bg-emerald-950/40 hover:bg-emerald-950/70">
					<td class="px-3 py-2 text-center text-emerald-400 font-bold">+</td>
					{#each allColumns as col}
					<td class="px-3 py-2 font-mono text-xs text-emerald-200">{row[col] ?? ''}</td>
					{/each}
				</tr>
				{/each}
				{/if}

				{#if showDeleted}
				{#each diff.deleted as row}
				<tr class="border-b border-gray-800/50 bg-red-950/40 hover:bg-red-950/70">
					<td class="px-3 py-2 text-center text-red-400 font-bold">−</td>
					{#each allColumns as col}
					<td class="px-3 py-2 font-mono text-xs text-red-200 line-through decoration-red-700/50">{row[col] ?? ''}</td>
					{/each}
				</tr>
				{/each}
				{/if}

				{#if showModified}
				{#each diff.modified as modRow}
				{@const changedCols = new Set(modRow.changes.map(c => c.column))}
				<tr class="border-b border-gray-800/50 bg-amber-950/30 hover:bg-amber-950/60">
					<td class="px-3 py-2 text-center text-amber-400 font-bold">~</td>
					{#each allColumns as col}
					{@const change = modRow.changes.find(c => c.column === col)}
					<td class="px-3 py-2 font-mono text-xs {changedCols.has(col) ? 'bg-amber-900/30' : ''}">
						{#if change}
						<span class="block text-red-300 line-through decoration-red-700/50">{change.old}</span>
						<span class="block text-emerald-300">{change.new}</span>
						{:else}
						<span class="text-gray-400">{modRow.changes.find(c => c.column === primaryKey)?.old ?? modRow.key}</span>
						{/if}
					</td>
					{/each}
				</tr>
				{/each}
				{/if}
			</tbody>
		</table>

		{#if diff.total_rows > diff.page_size}
		<div class="flex justify-center border-t border-gray-800 bg-gray-900 py-3 text-sm text-gray-500">
			Showing page {diff.page} of {Math.ceil(diff.total_rows / diff.page_size)} · {diff.total_rows} total changes
		</div>
		{/if}
	</div>
	{/if}
</div>
