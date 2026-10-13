<script lang="ts">
	import { api } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import BorderBeam from '$lib/components/magicui/BorderBeam.svelte';

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
	<Card className="relative max-w-sm w-full p-6 sm:p-8 shadow-2xl border-slate-800 overflow-hidden">
		<BorderBeam size={160} duration={10} colorFrom="#38bdf8" colorTo="#a855f7" />
		<div class="flex flex-col items-center text-center mb-6">
			<div class="w-12 h-12 rounded-2xl bg-gradient-to-tr from-sky-500 to-indigo-600 flex items-center justify-center text-white shadow-lg shadow-sky-500/25 mb-3">
				<Icon name="crown" size={22} />
			</div>
			<h1 class="text-xl sm:text-2xl font-black text-white tracking-tight">Create Account</h1>
			<p class="text-xs text-slate-400 mt-1">Start playing and track your rating</p>
		</div>

		{#if error}
			<div class="mb-4 p-3 bg-rose-500/10 border border-rose-500/30 rounded-xl text-xs text-rose-300 flex items-center gap-2">
				<Icon name="x" size={14} className="shrink-0 text-rose-400" />
				<span>{error}</span>
			</div>
		{/if}

		<form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-4">
			<div>
				<label for="reg-username" class="block text-xs font-semibold text-slate-400 mb-1.5 uppercase tracking-wider">
					Username
				</label>
				<Input
					id="reg-username"
					type="text"
					bind:value={username}
					required
					minlength="3"
					placeholder="Choose a username"
				/>
			</div>

			<div>
				<label for="reg-email" class="block text-xs font-semibold text-slate-400 mb-1.5 uppercase tracking-wider">
					Email Address
				</label>
				<Input
					id="reg-email"
					type="email"
					bind:value={email}
					required
					placeholder="you@example.com"
				/>
			</div>

			<div>
				<label for="reg-password" class="block text-xs font-semibold text-slate-400 mb-1.5 uppercase tracking-wider">
					Password
				</label>
				<Input
					id="reg-password"
					type="password"
					bind:value={password}
					required
					minlength="6"
					placeholder="At least 6 characters"
				/>
			</div>

			<Button variant="glow" size="lg" type="submit" disabled={loading} className="w-full mt-2">
				{#if loading}
					<div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
					<span>Creating Account...</span>
				{:else}
					<span>Create Free Account</span>
				{/if}
			</Button>
		</form>

		<div class="mt-6 text-center text-xs text-slate-400">
			Already have an account?
			<a href="/login" class="text-sky-400 font-semibold hover:underline ml-1">
				Sign in
			</a>
		</div>
	</Card>
</div>
