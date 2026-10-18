<script lang="ts">
	import { api } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Card from '$lib/components/ui/Card.svelte';

	let username = '';
	let password = '';
	let error = '';
	let loading = false;

	async function handleSubmit() {
		error = '';
		loading = true;
		try {
			const res = await api.auth.login({ username, password });
			authStore.loginSuccess(res.user, res.tokens);
			goto('/play/online');
		} catch (err: any) {
			error = err.message || 'Invalid username or password';
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex-1 flex items-center justify-center p-4">
	<Card className="max-w-sm w-full p-6 sm:p-8 shadow-xl border-border">
		<div class="flex flex-col items-center text-center mb-6">
			<div class="w-12 h-12 rounded-xl bg-secondary border border-border flex items-center justify-center p-2 mb-3 shadow-xs">
				<img src="/logo.png" alt="ChessNova" class="w-full h-full object-contain" />
			</div>
			<h1 class="text-xl sm:text-2xl font-black text-foreground tracking-tight">Welcome Back</h1>
			<p class="text-xs text-muted-foreground mt-1">Sign in to resume your games</p>
		</div>

		{#if error}
			<div class="mb-4 p-3 bg-destructive/10 border border-destructive/30 rounded-xl text-xs text-destructive flex items-center gap-2">
				<Icon name="x" size={14} className="shrink-0" />
				<span>{error}</span>
			</div>
		{/if}

		<form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-4">
			<div>
				<label for="username" class="block text-xs font-semibold text-muted-foreground mb-1.5 uppercase tracking-wider">
					Username
				</label>
				<Input
					id="username"
					type="text"
					bind:value={username}
					required
					placeholder="Enter your username"
				/>
			</div>

			<div>
				<div class="flex items-center justify-between mb-1.5">
					<label for="password" class="block text-xs font-semibold text-muted-foreground uppercase tracking-wider">
						Password
					</label>
				</div>
				<Input
					id="password"
					type="password"
					bind:value={password}
					required
					placeholder="••••••••"
				/>
			</div>

			<Button variant="primary" size="lg" type="submit" disabled={loading} className="w-full mt-2">
				{#if loading}
					<div class="w-4 h-4 border-2 border-background/30 border-t-background rounded-full animate-spin"></div>
					<span>Signing In...</span>
				{:else}
					<span>Log In</span>
				{/if}
			</Button>
		</form>

		<div class="mt-6 text-center text-xs text-muted-foreground">
			Don't have an account?
			<a href="/register" class="text-foreground font-semibold hover:underline ml-1">
				Create one free
			</a>
		</div>
	</Card>
</div>
