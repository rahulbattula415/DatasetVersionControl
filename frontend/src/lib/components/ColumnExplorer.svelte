<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { api } from '$lib/api';
	import type { ColumnStatHistory, SnapshotColumn } from '$lib/types';

	let {
		datasetId,
		columns,
		branchId
	}: { datasetId: string; columns: SnapshotColumn[]; branchId?: string } = $props();

	let selectedCol = $state(columns[0]?.column_name ?? '');
	let history: ColumnStatHistory[] = $state([]);
	let loading = $state(false);
	let error = $state('');

	let canvas: HTMLCanvasElement | undefined = $state();
	let chartInstance: any = null;

	const latest = $derived(history[history.length - 1] ?? null);
	const showChart = $derived(history.length > 1);

	async function loadHistory() {
		if (!selectedCol) return;
		loading = true;
		error = '';
		try {
			history = await api.columns.history(datasetId, selectedCol, branchId);
			if (history.length > 1) await drawChart();
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function drawChart() {
		if (!canvas) return;

		const { Chart, registerables } = await import('chart.js');
		Chart.register(...registerables);

		if (chartInstance) chartInstance.destroy();

		const labels = history.map((h) =>
			new Date(h.created_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
		);

		const datasets = [];
		const hasNumeric = history.some((h) => h.mean_value !== null);

		if (hasNumeric) {
			datasets.push({
				label: 'Mean',
				data: history.map((h) => h.mean_value),
				borderColor: '#6366f1',
				backgroundColor: 'rgba(99,102,241,0.1)',
				fill: true,
				tension: 0.3,
				yAxisID: 'y'
			});
		}

		datasets.push({
			label: 'Unique',
			data: history.map((h) => h.unique_count),
			borderColor: '#10b981',
			backgroundColor: 'transparent',
			tension: 0.3,
			yAxisID: hasNumeric ? 'y2' : 'y'
		});

		datasets.push({
			label: 'Nulls',
			data: history.map((h) => h.null_count),
			borderColor: '#f59e0b',
			backgroundColor: 'transparent',
			borderDash: [4, 4],
			tension: 0.3,
			yAxisID: hasNumeric ? 'y2' : 'y'
		});

		chartInstance = new Chart(canvas, {
			type: 'line',
			data: { labels, datasets },
			options: {
				responsive: true,
				interaction: { mode: 'index', intersect: false },
				plugins: {
					legend: { labels: { color: '#9ca3af' } },
					tooltip: {
						callbacks: {
							afterBody: (items) => {
								const idx = items[0]?.dataIndex;
								if (idx == null) return '';
								const h = history[idx];
								return [
									h.message ? `Commit: ${h.message}` : '',
									h.min_value != null ? `Min: ${h.min_value}` : '',
									h.max_value != null ? `Max: ${h.max_value}` : ''
								].filter(Boolean);
							}
						}
					}
				},
				scales: {
					x: { ticks: { color: '#6b7280' }, grid: { color: '#1f2937' } },
					y: {
						ticks: { color: '#6b7280' },
						grid: { color: '#1f2937' },
						title: { display: hasNumeric, text: 'Mean value', color: '#9ca3af' }
					},
					...(hasNumeric
						? {
								y2: {
									position: 'right',
									ticks: { color: '#6b7280' },
									grid: { drawOnChartArea: false },
									title: { display: true, text: 'Count', color: '#9ca3af' }
								}
							}
						: {})
				}
			}
		});
	}

	onMount(() => {
		if (selectedCol) loadHistory();
	});

	onDestroy(() => {
		if (chartInstance) chartInstance.destroy();
	});

	$effect(() => {
		if (selectedCol) loadHistory();
	});
</script>

<div class="space-y-4">
	<div class="flex items-center gap-3">
		<h2 class="font-semibold">Column Explorer</h2>
		<select
			bind:value={selectedCol}
			class="rounded-lg border border-gray-700 bg-gray-800 px-3 py-1.5 text-sm focus:border-indigo-500 focus:outline-none"
		>
			{#each columns as col}
			<option value={col.column_name}>{col.column_name} ({col.column_type})</option>
			{/each}
		</select>
		{#if loading}
		<svg class="h-4 w-4 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24">
			<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
			<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z"/>
		</svg>
		{/if}
	</div>

	{#if error}
	<div class="rounded-lg border border-red-700 bg-red-900/30 px-4 py-3 text-sm text-red-300">{error}</div>
	{/if}

	{#if history.length === 0 && !loading}
	<div class="flex items-center justify-center rounded-xl border border-gray-800 bg-gray-900 py-12 text-gray-500">
		No stats available for this column yet.
	</div>
	{:else if latest}

	<!-- Stats cards — always shown -->
	<div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
		{#each [
			{ label: 'Min', value: latest.min_value ?? '—', color: 'text-white' },
			{ label: 'Max', value: latest.max_value ?? '—', color: 'text-white' },
			{ label: 'Mean', value: latest.mean_value != null ? Number(latest.mean_value).toFixed(3) : '—', color: 'text-indigo-300' },
			{ label: 'Nulls', value: latest.null_count, color: latest.null_count > 0 ? 'text-amber-400' : 'text-emerald-400' },
			{ label: 'Unique', value: latest.unique_count, color: 'text-emerald-400' },
		] as stat}
		<div class="rounded-xl border border-gray-800 bg-gray-900 px-4 py-3">
			<p class="text-xs text-gray-500">{stat.label}</p>
			<p class="mt-1 text-lg font-mono font-semibold {stat.color}">{stat.value}</p>
		</div>
		{/each}
	</div>

	<!-- Trend chart — only shown when there are 2+ snapshots -->
	{#if showChart}
	<div class="rounded-xl border border-gray-800 bg-gray-900 p-4">
		<p class="mb-3 text-xs text-gray-500">Trend across {history.length} snapshots</p>
		<canvas bind:this={canvas}></canvas>
	</div>
	{:else}
	<p class="text-xs text-gray-600">Upload more snapshots to see trends over time.</p>
	{/if}

	<!-- History table — only shown when there are 2+ snapshots -->
	{#if showChart}
	<div class="overflow-x-auto rounded-xl border border-gray-800">
		<table class="w-full text-sm">
			<thead>
				<tr class="border-b border-gray-800 bg-gray-900">
					<th class="px-3 py-2 text-left font-medium text-gray-400">Snapshot</th>
					<th class="px-3 py-2 text-left font-medium text-gray-400">Commit</th>
					<th class="px-3 py-2 text-right font-medium text-gray-400">Min</th>
					<th class="px-3 py-2 text-right font-medium text-gray-400">Max</th>
					<th class="px-3 py-2 text-right font-medium text-gray-400">Mean</th>
					<th class="px-3 py-2 text-right font-medium text-gray-400">Nulls</th>
					<th class="px-3 py-2 text-right font-medium text-gray-400">Unique</th>
				</tr>
			</thead>
			<tbody>
				{#each history as h}
				<tr class="border-b border-gray-800/50 hover:bg-gray-900">
					<td class="px-3 py-2 font-mono text-xs text-indigo-300">{h.snapshot_id.slice(0, 8)}</td>
					<td class="px-3 py-2 text-gray-400 text-xs">{h.message ?? '—'}</td>
					<td class="px-3 py-2 text-right font-mono text-xs">{h.min_value ?? '—'}</td>
					<td class="px-3 py-2 text-right font-mono text-xs">{h.max_value ?? '—'}</td>
					<td class="px-3 py-2 text-right font-mono text-xs">{h.mean_value?.toFixed(2) ?? '—'}</td>
					<td class="px-3 py-2 text-right font-mono text-xs text-amber-400">{h.null_count}</td>
					<td class="px-3 py-2 text-right font-mono text-xs text-emerald-400">{h.unique_count}</td>
				</tr>
				{/each}
			</tbody>
		</table>
	</div>
	{/if}

	{/if}
</div>
