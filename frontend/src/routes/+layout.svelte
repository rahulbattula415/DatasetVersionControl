<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	let { children } = $props();
</script>

<div class="min-h-screen bg-gray-950 text-gray-100">
	<header class="border-b border-gray-800 bg-gray-900">
		<div class="mx-auto flex max-w-7xl items-center gap-4 px-6 py-3">
			<a href="/" class="flex items-center gap-2 font-bold text-indigo-400 hover:text-indigo-300">
				<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round"
						d="M4 7v10c0 2 1 3 3 3h10c2 0 3-1 3-3V7M4 7c0-2 1-3 3-3h10c2 0 3 1 3 3M4 7h16M9 3v4M15 3v4" />
				</svg>
				DatasetVC
			</a>
			<span class="text-gray-600">/</span>
			<nav class="flex items-center gap-4 text-sm text-gray-400">
				<a href="/" class:text-white={$page.url.pathname === '/'} class="hover:text-gray-200">
					Datasets
				</a>
			</nav>
		</div>
	</header>

	<main class="mx-auto max-w-7xl px-6 py-8">
		{@render children()}
	</main>
</div>
