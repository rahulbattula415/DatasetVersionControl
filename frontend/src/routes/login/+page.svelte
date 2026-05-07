<script lang="ts">
	import { auth, token } from '$lib/api';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	onMount(() => {
		if (token.get()) goto('/');
	});

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;
		try {
			await auth.login(email, password);
			goto('/');
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Login failed';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head><title>Sign in — DatasetVC</title></svelte:head>

<div class="flex min-h-[70vh] items-center justify-center">
	<div class="w-full max-w-sm rounded-xl border border-gray-800 bg-gray-900 p-8 shadow-xl">
		<h1 class="mb-6 text-center text-2xl font-bold text-white">Sign in</h1>

		{#if error}
			<div class="mb-4 rounded-lg bg-red-900/40 px-4 py-3 text-sm text-red-300">{error}</div>
		{/if}

		<form onsubmit={submit} class="space-y-4">
			<div>
				<label class="mb-1 block text-sm text-gray-400" for="email">Email</label>
				<input
					id="email"
					type="email"
					bind:value={email}
					required
					autocomplete="email"
					class="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white placeholder-gray-500 focus:border-indigo-500 focus:outline-none"
					placeholder="you@example.com"
				/>
			</div>
			<div>
				<label class="mb-1 block text-sm text-gray-400" for="password">Password</label>
				<input
					id="password"
					type="password"
					bind:value={password}
					required
					autocomplete="current-password"
					class="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white placeholder-gray-500 focus:border-indigo-500 focus:outline-none"
					placeholder="••••••••"
				/>
			</div>
			<button
				type="submit"
				disabled={loading}
				class="w-full rounded-lg bg-indigo-600 px-4 py-2 font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
			>
				{loading ? 'Signing in…' : 'Sign in'}
			</button>
		</form>

		<p class="mt-6 text-center text-sm text-gray-500">
			No account?
			<a href="/register" class="text-indigo-400 hover:text-indigo-300">Register</a>
		</p>
	</div>
</div>
