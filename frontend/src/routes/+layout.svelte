<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { token, auth } from '$lib/api';

	let { children } = $props();

	// `/` is the public landing page; auth pages are public too.
	const publicPaths = ['/', '/login', '/register'];
	const authPaths = ['/login', '/register'];
	let userEmail = $state('');

	onMount(() => {
		userEmail = token.email() ?? '';
		const path = window.location.pathname;
		if (!token.get() && !publicPaths.includes(path)) {
			goto('/login');
		}
	});

	function logout() {
		auth.logout();
	}
</script>

<div class="min-h-screen bg-canvas text-ink">
	<header class="border-b border-edge bg-surface">
		<div class="mx-auto flex max-w-7xl items-center gap-4 px-6 py-3">
			<a href="/" class="flex shrink-0 items-center gap-2 font-bold text-primary-link hover:text-primary-data">
				<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round"
						d="M4 7v10c0 2 1 3 3 3h10c2 0 3-1 3-3V7M4 7c0-2 1-3 3-3h10c2 0 3 1 3 3M4 7h16M9 3v4M15 3v4" />
				</svg>
				DatasetVC
			</a>

			{#if !publicPaths.includes($page.url.pathname)}
				<span class="text-gray-500" aria-hidden="true">/</span>
				<nav class="flex items-center gap-4 text-sm text-ink-soft">
					<a href="/datasets" class:text-white={$page.url.pathname === '/datasets'} class="hover:text-gray-200">
						Datasets
					</a>
				</nav>
			{/if}

			<div class="ml-auto flex min-w-0 items-center gap-3">
				{#if userEmail}
					{#if $page.url.pathname === '/'}
						<a href="/datasets" class="shrink-0 text-sm text-ink-soft hover:text-gray-200">Datasets</a>
					{/if}
					<span class="hidden max-w-[24ch] truncate text-sm text-ink-soft sm:inline-block">{userEmail}</span>
					<button
						onclick={logout}
						class="shrink-0 rounded-lg border border-edge-strong px-3 py-1 text-sm text-ink-soft hover:border-gray-500 hover:text-gray-200"
					>
						Sign out
					</button>
				{:else if !authPaths.includes($page.url.pathname)}
					<a href="/login" class="shrink-0 text-sm text-ink-soft hover:text-gray-200">Sign in</a>
					<a
						href="/register"
						class="shrink-0 rounded-lg bg-primary px-3 py-1 text-sm font-medium text-white hover:bg-primary-hover transition-colors"
					>
						Get started
					</a>
				{/if}
			</div>
		</div>
	</header>

	<main class="mx-auto max-w-7xl px-6 py-8">
		{@render children()}
	</main>
</div>
