<script lang="ts">
	import { onMount } from 'svelte';
	import { token } from '$lib/api';

	let loggedIn = $state(false);
	onMount(() => {
		loggedIn = !!token.get();
	});

	// Static mock data — the landing renders the product's own artifacts
	// (a commit timeline and a primary-key diff) rather than marketing chrome.
	const commits = [
		{ hash: '7f3a9c2e10', message: 'Q3 regional sales', rows: 142, when: 'today', initial: false },
		{ hash: '2b8e4f1a90', message: 'Q2 regional sales', rows: 138, when: '14d ago', initial: false },
		{ hash: 'a0c5d7e221', message: 'Q1 regional sales — initial import', rows: 130, when: '92d ago', initial: true }
	];
	const cols = ['id', 'region', 'revenue', 'units'];
</script>

<svelte:head>
	<title>DatasetVC — Version control for CSV datasets</title>
	<meta
		name="description"
		content="Git-like version control for CSV datasets: content-addressed immutable snapshots, primary-key-aware diffs, branching, fast-forward merges, and per-column statistics over time."
	/>
</svelte:head>

<!-- ─── Hero ──────────────────────────────────────────────────────────────── -->
<section class="mx-auto max-w-3xl py-12 lg:py-20">
	<h1 class="text-2xl font-bold leading-tight tracking-tight">
		Version control for CSV datasets.
	</h1>
	<p class="mt-4 max-w-2xl text-sm leading-relaxed text-ink-soft">
		Every upload becomes a content-addressed, immutable snapshot. Diff any two versions
		row-by-row by primary key, branch and fast-forward-merge like Git, and watch per-column
		statistics move across a dataset's history. The engine is exact; the interface makes that
		legible.
	</p>

	<div class="mt-8 flex flex-wrap items-center gap-3">
		{#if loggedIn}
			<a
				href="/datasets"
				class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-hover"
			>
				Open app
			</a>
		{:else}
			<a
				href="/register"
				class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-hover"
			>
				Get started
			</a>
			<a
				href="/login"
				class="rounded-lg border border-edge-strong px-4 py-2 text-sm font-medium text-ink-soft transition-colors hover:border-gray-500 hover:text-gray-200"
			>
				Sign in
			</a>
		{/if}
	</div>

	<p class="mt-6 font-mono text-xs text-ink-soft">
		<span class="text-primary-data">snapshot_hash</span> = sha256(file_hash · schema · parent)
	</p>
</section>

<!-- ─── The Commit Log (signature artifacts) ──────────────────────────────── -->
<section class="border-t border-edge py-12 lg:py-16">
	<h2 class="text-sm font-semibold text-gray-300">The commit log</h2>
	<p class="mt-1 max-w-2xl text-sm text-ink-soft">
		A dataset is an append-only chain of immutable commits. Read its lineage on the left;
		read exactly what changed on the right.
	</p>

	<div class="mt-6 grid gap-6 lg:grid-cols-12">
		<!-- Snapshot timeline -->
		<div class="space-y-2 lg:col-span-5">
			{#each commits as snap, i}
			<div class="flex items-start gap-4 rounded-xl border border-edge bg-surface p-4">
				<div class="flex flex-col items-center self-stretch">
					<div class="h-3 w-3 shrink-0 rounded-full bg-primary-hover"></div>
					{#if i < commits.length - 1}
					<div class="mt-1 w-0.5 flex-1 bg-raised" style="min-height:2rem"></div>
					{/if}
				</div>
				<div class="min-w-0 flex-1">
					<div class="flex flex-wrap items-center gap-2">
						<code class="rounded bg-raised px-2 py-0.5 font-mono text-xs text-primary-data">{snap.hash}</code>
						<span class="min-w-0 font-medium">{snap.message}</span>
						{#if snap.initial}
						<span class="rounded-full bg-emerald-900/50 px-2 py-0.5 text-xs text-emerald-400">initial</span>
						{/if}
					</div>
					<p class="mt-1 text-xs text-ink-soft">{snap.rows} rows · {snap.when}</p>
				</div>
			</div>
			{/each}
		</div>

		<!-- Diff table -->
		<div class="space-y-4 lg:col-span-7">
			<div class="flex flex-wrap items-center gap-3 rounded-xl border border-edge bg-surface p-4">
				<div class="flex gap-4 text-sm">
					<span class="font-medium text-white">All <span class="text-ink-soft">3</span></span>
					<span class="font-medium text-emerald-400">+ Added <span>1</span></span>
					<span class="font-medium text-red-400">− Deleted <span>1</span></span>
					<span class="font-medium text-amber-400">~ Modified <span>1</span></span>
				</div>
				<div class="ml-auto flex items-center gap-2 text-xs text-ink-soft">
					<span class="rounded-full bg-raised px-2 py-0.5">0.4ms</span>
				</div>
			</div>

			<div class="overflow-x-auto rounded-xl border border-edge">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-edge bg-surface">
							<th class="w-6 px-3 py-2"></th>
							{#each cols as col}
							<th class="px-3 py-2 text-left font-medium text-ink-soft whitespace-nowrap">{col}</th>
							{/each}
						</tr>
					</thead>
					<tbody>
						<!-- Added -->
						<tr class="border-b border-edge/50 bg-emerald-950/40">
							<td class="px-3 py-2 text-center font-bold text-emerald-400">+</td>
							<td class="px-3 py-2 font-mono text-xs text-emerald-200">214</td>
							<td class="px-3 py-2 font-mono text-xs text-emerald-200">APAC</td>
							<td class="px-3 py-2 font-mono text-xs text-emerald-200">48200</td>
							<td class="px-3 py-2 font-mono text-xs text-emerald-200">512</td>
						</tr>
						<!-- Deleted -->
						<tr class="border-b border-edge/50 bg-red-950/40">
							<td class="px-3 py-2 text-center font-bold text-red-400">−</td>
							<td class="px-3 py-2 font-mono text-xs text-red-200 line-through decoration-red-700/50">087</td>
							<td class="px-3 py-2 font-mono text-xs text-red-200 line-through decoration-red-700/50">LATAM</td>
							<td class="px-3 py-2 font-mono text-xs text-red-200 line-through decoration-red-700/50">19050</td>
							<td class="px-3 py-2 font-mono text-xs text-red-200 line-through decoration-red-700/50">203</td>
						</tr>
						<!-- Modified -->
						<tr class="border-b border-edge/50 bg-amber-950/30">
							<td class="px-3 py-2 text-center font-bold text-amber-400">~</td>
							<td class="px-3 py-2 font-mono text-xs text-ink-soft">142</td>
							<td class="px-3 py-2 font-mono text-xs text-ink-soft">EMEA</td>
							<td class="px-3 py-2 font-mono text-xs bg-amber-900/30">
								<span class="block text-red-300 line-through decoration-red-700/50">61400</span>
								<span class="block text-emerald-300">67850</span>
							</td>
							<td class="px-3 py-2 font-mono text-xs text-ink-soft">640</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>
	</div>
</section>

<!-- ─── Capabilities (spec sheet, not a feature-card grid) ─────────────────── -->
<section class="border-t border-edge py-12 lg:py-16">
	<h2 class="text-sm font-semibold text-gray-300">What the engine guarantees</h2>

	<dl class="mt-6 divide-y divide-edge border-y border-edge">
		<div class="grid gap-2 py-5 sm:grid-cols-12 sm:gap-6">
			<dt class="font-mono text-xs text-primary-data sm:col-span-3">snapshot</dt>
			<dd class="text-sm leading-relaxed text-ink-soft sm:col-span-9">
				Content-addressed and immutable. A commit is keyed by
				<code class="rounded bg-raised px-1 py-0.5 font-mono text-xs text-gray-300">sha256(file · schema · parent)</code>,
				so re-uploading the same bytes reuses the existing snapshot — history is never mutated.
			</dd>
		</div>
		<div class="grid gap-2 py-5 sm:grid-cols-12 sm:gap-6">
			<dt class="font-mono text-xs text-primary-data sm:col-span-3">diff</dt>
			<dd class="text-sm leading-relaxed text-ink-soft sm:col-span-9">
				Primary-key-aware. Rows are matched by key and classified as
				<span class="text-emerald-400">added</span>,
				<span class="text-red-400">deleted</span>, or
				<span class="text-amber-400">modified</span> — never a naive line diff. Each result is cached
				per snapshot pair.
			</dd>
		</div>
		<div class="grid gap-2 py-5 sm:grid-cols-12 sm:gap-6">
			<dt class="font-mono text-xs text-primary-data sm:col-span-3">branch · merge</dt>
			<dd class="text-sm leading-relaxed text-ink-soft sm:col-span-9">
				Branch a dataset from any snapshot. Merges are fast-forward only, with ancestry enforced by
				the engine — a non-descendant head is rejected, not silently squashed.
			</dd>
		</div>
		<div class="grid gap-2 py-5 sm:grid-cols-12 sm:gap-6">
			<dt class="font-mono text-xs text-primary-data sm:col-span-3">stats</dt>
			<dd class="text-sm leading-relaxed text-ink-soft sm:col-span-9">
				min / max / mean / nulls / uniques captured per column at commit time, charted across the
				dataset's full history so a drift is a line you can see.
			</dd>
		</div>
	</dl>
</section>

<!-- ─── Closing ───────────────────────────────────────────────────────────── -->
<section class="border-t border-edge py-12 lg:py-16">
	<h2 class="text-2xl font-bold tracking-tight">Start versioning your data.</h2>
	<p class="mt-2 max-w-2xl text-sm text-ink-soft">
		Create a dataset, upload a CSV, and read your first diff — no configuration.
	</p>
	<div class="mt-6">
		<a
			href={loggedIn ? '/datasets' : '/register'}
			class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-hover"
		>
			{loggedIn ? 'Open app' : 'Get started'}
		</a>
	</div>
</section>

<footer class="border-t border-edge py-6">
	<p class="text-xs text-ink-soft">
		<span class="font-medium text-primary-link">DatasetVC</span> — content-addressed, immutable, primary-key-aware.
	</p>
</footer>
