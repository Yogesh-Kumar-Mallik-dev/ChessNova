<script lang="ts">
	import { onDestroy } from 'svelte';
	import { api } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import BorderBeam from '$lib/components/magicui/BorderBeam.svelte';

	let step: 'details' | 'otp' = 'details';

	let username = '';
	let email = '';
	let password = '';
	let otp = '';
	let devOtp = '';
	let error = '';
	let successMsg = '';
	let loading = false;
	let resendCooldown = 0;
	let cooldownTimer: any = null;

	function startCooldown(seconds = 60) {
		resendCooldown = seconds;
		if (cooldownTimer) clearInterval(cooldownTimer);
		cooldownTimer = setInterval(() => {
			if (resendCooldown > 0) {
				resendCooldown--;
			} else {
				clearInterval(cooldownTimer);
				cooldownTimer = null;
			}
		}, 1000);
	}

	onDestroy(() => {
		if (cooldownTimer) clearInterval(cooldownTimer);
	});

	async function handleSendOTP() {
		error = '';
		successMsg = '';
		loading = true;
		try {
			const res = await api.auth.sendRegisterOtp({ username, email, password });
			devOtp = res.devOtp || '';
			successMsg = `Verification code sent to ${email}`;
			step = 'otp';
			startCooldown(60);
		} catch (err: any) {
			error = err.message || 'Failed to send verification code';
		} finally {
			loading = false;
		}
	}

	async function handleVerifyOTP() {
		if (otp.trim().length !== 6) {
			error = 'Please enter a valid 6-digit verification code';
			return;
		}

		error = '';
		loading = true;
		try {
			const res = await api.auth.verifyRegisterOtp({ email, otp: otp.trim() });
			authStore.loginSuccess(res.user, res.tokens);
			goto('/play/online');
		} catch (err: any) {
			error = err.message || 'Verification failed. Please check the code and try again.';
		} finally {
			loading = false;
		}
	}

	async function handleResendOTP() {
		if (resendCooldown > 0 || loading) return;
		error = '';
		loading = true;
		try {
			const res = await api.auth.resendRegisterOtp({ email });
			if (res.devOtp) devOtp = res.devOtp;
			successMsg = 'A fresh verification code has been dispatched to your email.';
			startCooldown(60);
		} catch (err: any) {
			error = err.message || 'Failed to resend code';
		} finally {
			loading = false;
		}
	}

	function handleBackToDetails() {
		step = 'details';
		error = '';
		successMsg = '';
		otp = '';
	}
</script>

<div class="flex-1 flex items-center justify-center p-4">
	<Card className="relative max-w-sm w-full p-6 sm:p-8 shadow-2xl border-slate-800 overflow-hidden">
		<BorderBeam size={160} duration={10} colorFrom="#38bdf8" colorTo="#a855f7" />
		
		<div class="flex flex-col items-center text-center mb-6">
			<div class="w-14 h-14 rounded-2xl bg-gradient-to-tr from-sky-500/20 to-indigo-600/30 border border-sky-500/30 flex items-center justify-center p-1.5 text-white shadow-lg shadow-sky-500/20 mb-3 overflow-hidden">
				<img src="/logo.png" alt="ChessNova" class="w-full h-full object-contain drop-shadow" />
			</div>

			{#if step === 'details'}
				<h1 class="text-xl sm:text-2xl font-black text-white tracking-tight">Create Account</h1>
				<p class="text-xs text-slate-400 mt-1">Start playing and track your rating</p>
			{:else}
				<h1 class="text-xl sm:text-2xl font-black text-white tracking-tight">Verify Your Email</h1>
				<p class="text-xs text-slate-400 mt-1">
					Enter the 6-digit code sent to <span class="text-sky-300 font-semibold">{email}</span>
				</p>
			{/if}
		</div>

		{#if error}
			<div class="mb-4 p-3 bg-rose-500/10 border border-rose-500/30 rounded-xl text-xs text-rose-300 flex items-center gap-2 animate-in fade-in duration-150">
				<Icon name="x" size={14} className="shrink-0 text-rose-400" />
				<span>{error}</span>
			</div>
		{/if}

		{#if successMsg && !error}
			<div class="mb-4 p-3 bg-emerald-500/10 border border-emerald-500/30 rounded-xl text-xs text-emerald-300 flex items-center gap-2 animate-in fade-in duration-150">
				<Icon name="check" size={14} className="shrink-0 text-emerald-400" />
				<span>{successMsg}</span>
			</div>
		{/if}

		{#if devOtp}
			<div class="mb-4 p-3 bg-sky-950/60 border border-sky-500/40 rounded-xl text-xs text-sky-200 flex items-center justify-between">
				<span>Verification Code: <strong class="font-mono tracking-widest text-white text-sm">{devOtp}</strong></span>
				<button
					type="button"
					class="px-2 py-0.5 bg-sky-600 hover:bg-sky-500 text-white font-bold rounded text-[11px]"
					on:click={() => (otp = devOtp)}
				>
					Fill
				</button>
			</div>
		{/if}

		{#if step === 'details'}
			<form on:submit|preventDefault={handleSendOTP} class="flex flex-col gap-4">
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

				<Button variant="primary" size="lg" type="submit" disabled={loading} className="w-full mt-2">
					{#if loading}
						<div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
						<span>Sending Verification Code...</span>
					{:else}
						<span>Continue with Verification</span>
					{/if}
				</Button>
			</form>

			<div class="mt-6 text-center text-xs text-slate-400">
				Already have an account?
				<a href="/login" class="text-sky-400 font-semibold hover:underline ml-1">
					Sign in
				</a>
			</div>
		{:else}
			<form on:submit|preventDefault={handleVerifyOTP} class="flex flex-col gap-4">
				<div>
					<label for="reg-otp" class="block text-xs font-semibold text-slate-400 mb-1.5 uppercase tracking-wider text-center">
						6-Digit Security Code
					</label>
					<Input
						id="reg-otp"
						type="text"
						inputmode="numeric"
						pattern="[0-9]*"
						maxlength="6"
						bind:value={otp}
						required
						placeholder="••••••"
						className="text-center font-mono text-xl tracking-[0.4em] font-bold h-12"
					/>
				</div>

				<Button variant="primary" size="lg" type="submit" disabled={loading || otp.trim().length !== 6} className="w-full mt-1">
					{#if loading}
						<div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
						<span>Activating Account...</span>
					{:else}
						<span>Verify & Activate Account</span>
					{/if}
				</Button>

				<div class="flex items-center justify-between text-xs pt-1">
					<button
						type="button"
						class="text-slate-400 hover:text-slate-200 transition font-medium"
						on:click={handleBackToDetails}
					>
						&larr; Change Email
					</button>

					<button
						type="button"
						class="text-sky-400 hover:text-sky-300 font-semibold disabled:opacity-40 disabled:hover:text-sky-400 disabled:cursor-not-allowed transition"
						disabled={resendCooldown > 0 || loading}
						on:click={handleResendOTP}
					>
						{resendCooldown > 0 ? `Resend code in ${resendCooldown}s` : 'Resend Code'}
					</button>
				</div>
			</form>
		{/if}
	</Card>
</div>
