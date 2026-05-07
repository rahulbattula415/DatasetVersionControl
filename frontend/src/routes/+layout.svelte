<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { token, auth } from '$lib/api';

	let { children } = $props();

	const publicPaths = ['/login', '/register'];
	let userEmail = $state('');
	let ready = $state(false);

	onMount(() => {
		userEmail = token.email() ?? '';
		if (!token.get() && !publicPaths.includes($page.url.pathname)) {
			goto('/login');
		} else {
			ready = true;
		}
	});

	function logout() {
		auth.logout();
	}
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

			{#if !publicPaths.includes($page.url.pathname)}
				<span class="text-gray-600">/</span>
				<nav class="flex items-center gap-4 text-sm text-gray-400">
					<a href="/" class:text-white={$page.url.pathname === '/'} class="hover:text-gray-200">
						Datasets
					</a>
				</nav>
			{/if}

			<div class="ml-auto flex items-center gap-3">
				{#if userEmail}
					<span class="text-sm text-gray-500">{userEmail}</span>
					<button
						onclick={logout}
						class="rounded-lg border border-gray-700 px-3 py-1 text-sm text-gray-400 hover:border-gray-500 hover:text-gray-200"
					>
						Sign out
					</button>
				{/if}
			</div>
		</div>
	</header>

	<main class="mx-auto max-w-7xl px-6 py-8">
		{#if ready || publicPaths.includes($page.url.pathname)}
			{@render children()}
		{/if}
	</main>
</div>
