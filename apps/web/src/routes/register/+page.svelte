<script lang="ts">
	import { api } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import Icon from '$lib/components/icons/Icon.svelte';

	let username = '';
	let email = '';
	let password = '';
	let error = '';
	let loading = false;

	async function handleSubmit() {
		error = '';
		loading = true;
		try {
			const res = await api.auth.register({ username, email, password });
			authStore.loginSuccess(res.user, res.tokens);
			goto('/play/online');
		} catch (err: any) {
			error = err.message || 'Registration failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex-1 flex items-center justify-center p-4">
	<div class="bg-slate-900/80 border border-slate-800/80 backdrop-blur-xl rounded-2xl p-6 sm:p-8 max-w-sm w-full shadow-2xl shadow-indigo-950/20">
		<div class="flex flex-col items-center text-center mb-6">
			<div class="w-12 h-12 rounded-xl bg-gradient-to-br from-sky-500 to-indigo-600 flex items-center justify-center text-white shadow-lg shadow-sky-500/25 mb-3">
				<Icon name="crown" size={24} />
			</div>
			<h1 class="text-2xl font-extrabold text-white tracking-tight">Create Account</h1>
			<p class="text-xs text-slate-400 mt-1">Join ChessNova and compete on the global stage</p>
		</div>

		{#if error}
			<div class="mb-4 p-3 bg-rose-950/40 border border-rose-700/50 rounded-xl text-xs text-rose-300">
				{error}
			</div>
		{/if}

		<form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-4">
			<div>
				<label for="reg-username" class="block text-xs font-semibold text-slate-400 mb-1.5 uppercase tracking-wider">
					Username
				</label>
				<input
					id="reg-username"
					type="text"
					bind:value={username}
					required
					minlength="3"
					placeholder="magnus"
					class="w-full px-3.5 py-2.5 bg-slate-950/80 border border-slate-800 rounded-xl text-white text-sm focus:outline-none focus:border-sky-500 focus:ring-1 focus:ring-sky-500/40 transition placeholder:text-slate-600"
				/>
			</div>

			<div>
				<label for="reg-email" class="block text-xs font-semibold text-slate-400 mb-1.5 uppercase tracking-wider">
					Email Address
				</label>
				<input
					id="reg-email"
					type="email"
					bind:value={email}
					required
					placeholder="magnus@chessnova.io"
					class="w-full px-3.5 py-2.5 bg-slate-950/80 border border-slate-800 rounded-xl text-white text-sm focus:outline-none focus:border-sky-500 focus:ring-1 focus:ring-sky-500/40 transition placeholder:text-slate-600"
				/>
			</div>

			<div>
				<label for="reg-password" class="block text-xs font-semibold text-slate-400 mb-1.5 uppercase tracking-wider">
					Password
				</label>
				<input
					id="reg-password"
					type="password"
					bind:value={password}
					required
					minlength="6"
					placeholder="••••••••"
					class="w-full px-3.5 py-2.5 bg-slate-950/80 border border-slate-800 rounded-xl text-white text-sm focus:outline-none focus:border-sky-500 focus:ring-1 focus:ring-sky-500/40 transition placeholder:text-slate-600"
				/>
			</div>

			<button
				type="submit"
				disabled={loading}
				class="w-full mt-2 py-3 bg-gradient-to-r from-sky-500 to-indigo-600 hover:from-sky-400 hover:to-indigo-500 disabled:opacity-50 text-white font-bold rounded-xl transition shadow-lg shadow-sky-500/20"
			>
				{loading ? 'Creating Account...' : 'Sign Up'}
			</button>
		</form>

		<div class="mt-6 text-center text-xs text-slate-400">
			Already have an account?
			<a href="/login" class="text-sky-400 hover:text-sky-300 font-semibold ml-1">Log In</a>
		</div>
	</div>
</div>
