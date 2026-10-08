<script lang="ts">
	import { onDestroy } from 'svelte';
	import { api } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import Icon from '$lib/components/icons/Icon.svelte';

	interface Preset {
		label: string;
		initial: number;
		increment: number;
		type: string;
		icon: 'zap' | 'flame' | 'timer' | 'hourglass';
		description: string;
	}

	const presets: Preset[] = [
		{ label: '1 min', initial: 60, increment: 0, type: 'Bullet', icon: 'zap', description: 'Lightning reflex' },
		{ label: '2 | 1', initial: 120, increment: 1, type: 'Bullet', icon: 'zap', description: 'Bullet with increment' },
		{ label: '3 min', initial: 180, increment: 0, type: 'Blitz', icon: 'flame', description: 'Standard blitz' },
		{ label: '3 | 2', initial: 180, increment: 2, type: 'Blitz', icon: 'flame', description: 'Competitive blitz' },
		{ label: '5 min', initial: 300, increment: 0, type: 'Blitz', icon: 'flame', description: 'Full blitz' },
		{ label: '10 min', initial: 600, increment: 0, type: 'Rapid', icon: 'timer', description: 'Deep tactical' },
		{ label: '15 | 10', initial: 900, increment: 10, type: 'Rapid', icon: 'timer', description: 'Tournament rapid' },
		{ label: '30 min', initial: 1800, increment: 0, type: 'Classical', icon: 'hourglass', description: 'Pure classical' }
	];

	let selectedPreset = presets[2]; // Default 3 min Blitz
	let isSearching = false;
	let waitSeconds = 0;
	let waitTimer: any = null;
	let pollTimer: any = null;
	let error = '';

	$: tolerance = (() => {
		if (waitSeconds < 10) return 50;
		if (waitSeconds < 20) return 100;
		if (waitSeconds < 30) return 200;
		return 300;
	})();

	$: userRating = (() => {
		const cat = selectedPreset.type.toLowerCase();
		return $authStore.ratings?.[cat] || 400;
	})();

	async function startMatchmaking() {
		if (!$authStore.user) {
			goto('/login');
			return;
		}

		error = '';
		isSearching = true;
		waitSeconds = 0;

		try {
			await api.matchmaking.join({
				initial: selectedPreset.initial,
				increment: selectedPreset.increment
			});

			waitTimer = setInterval(() => {
				waitSeconds++;
			}, 1000);

			pollTimer = setInterval(checkStatus, 1500);
		} catch (err: any) {
			error = err.message || 'Failed to join matchmaking queue';
			isSearching = false;
		}
	}

	async function cancelMatchmaking() {
		stopTimers();
		isSearching = false;
		try {
			await api.matchmaking.leave();
		} catch {
			// ignore
		}
	}

	async function checkStatus() {
		try {
			const res = await api.matchmaking.status();
			if (res.status === 'matched') {
				stopTimers();
				goto(`/game/${res.gameId}`);
			}
		} catch {
			// ignore
		}
	}

	async function createDirectGame() {
		if (!$authStore.user) {
			goto('/login');
			return;
		}
		try {
			const ag = await api.games.create({
				initial: selectedPreset.initial,
				increment: selectedPreset.increment
			});
			goto(`/game/${ag.id}`);
		} catch (err: any) {
			error = err.message;
		}
	}

	function stopTimers() {
		if (waitTimer) clearInterval(waitTimer);
		if (pollTimer) clearInterval(pollTimer);
		waitTimer = null;
		pollTimer = null;
	}

	onDestroy(() => {
		stopTimers();
	});
</script>

<div class="flex-1 max-w-4xl mx-auto w-full px-4 py-8 flex flex-col justify-center">
	<div class="text-center mb-8">
		<div class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-sky-500/10 border border-sky-500/30 text-xs font-mono text-sky-400 font-bold mb-3">
			<Icon name="zap" size={13} />
			<span>SERVER-MATCHED ARENA</span>
		</div>
		<h1 class="text-3xl sm:text-4xl font-black text-white tracking-tight">Play Online</h1>
		<p class="text-xs sm:text-sm text-slate-400 mt-1 max-w-md mx-auto">
			Choose your time control to enter the matchmaking queue.
		</p>
	</div>

	{#if error}
		<div class="mb-6 p-4 bg-rose-950/40 border border-rose-700/50 rounded-xl text-sm text-rose-200 text-center max-w-md mx-auto shadow-md">
			{error}
		</div>
	{/if}

	{#if isSearching}
		<!-- Cosmic Matchmaking Radar Modal -->
		<div class="max-w-md mx-auto w-full bg-gradient-to-b from-[#141c2c] to-[#0c1017] border border-sky-500/40 rounded-3xl p-8 sm:p-10 shadow-[0_0_50px_rgba(56,189,248,0.2)] text-center flex flex-col items-center animate-in fade-in zoom-in-95 duration-200">
			<!-- Radar Scanner Graphic -->
			<div class="relative w-28 h-28 rounded-full border border-sky-500/30 flex items-center justify-center mb-6">
				<div class="absolute inset-2 rounded-full border border-sky-500/20"></div>
				<div class="absolute inset-6 rounded-full border border-sky-500/15"></div>
				<div class="absolute inset-0 rounded-full border-2 border-sky-400 border-t-transparent animate-spin"></div>
				<div class="w-3 h-3 rounded-full bg-sky-400 shadow-[0_0_12px_rgba(56,189,248,0.9)] animate-ping"></div>
			</div>

			<h2 class="text-xl font-bold text-white mb-1">Searching for Opponent...</h2>
			<div class="text-xs font-mono text-sky-400 font-bold mb-6 flex items-center gap-1.5">
				<Icon name={selectedPreset.icon} size={14} />
				<span>{selectedPreset.label} • {selectedPreset.type}</span>
			</div>

			<!-- Dynamic Stats Grid -->
			<div class="w-full grid grid-cols-3 gap-2 bg-[#090d15] border border-slate-800 rounded-xl p-3.5 mb-6 text-center">
				<div>
					<span class="block text-white font-mono text-base font-extrabold">{waitSeconds}s</span>
					<span class="text-[10px] text-slate-500 uppercase tracking-wider font-semibold">Wait Time</span>
				</div>
				<div>
					<span class="block text-sky-400 font-mono text-base font-extrabold">±{tolerance}</span>
					<span class="text-[10px] text-slate-500 uppercase tracking-wider font-semibold">Tolerance</span>
				</div>
				<div>
					<span class="block text-indigo-400 font-mono text-base font-extrabold">{userRating}</span>
					<span class="text-[10px] text-slate-500 uppercase tracking-wider font-semibold">Your Rating</span>
				</div>
			</div>

			<button
				class="w-full py-3 bg-slate-800/80 hover:bg-slate-750 text-slate-300 font-bold text-xs uppercase tracking-wider rounded-xl transition border border-slate-700/80 active:scale-95"
				on:click={cancelMatchmaking}
			>
				Cancel Search
			</button>
		</div>
	{:else}
		<!-- Time Control Selector -->
		<div class="bg-gradient-to-b from-[#121824] to-[#0c1017] border border-slate-800 rounded-3xl p-6 sm:p-8 max-w-xl mx-auto w-full shadow-2xl">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-xs font-mono font-bold uppercase tracking-widest text-slate-400">Select Discipline</h2>
				<span class="text-[11px] font-mono text-sky-400 font-semibold">
					Your Rating: {userRating}
				</span>
			</div>

			<!-- Presets Grid -->
			<div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
				{#each presets as preset}
					{@const isSelected = selectedPreset.label === preset.label}
					<button
						class="flex flex-col items-center justify-center p-3 rounded-2xl border transition-all duration-150 text-center group
						{isSelected
							? 'bg-sky-500/15 border-sky-400 text-white shadow-[0_0_15px_rgba(56,189,248,0.25)] scale-[1.02]'
							: 'bg-[#090d15] border-slate-800 text-slate-400 hover:text-white hover:border-slate-700 hover:bg-slate-850'}"
						on:click={() => (selectedPreset = preset)}
					>
						<div class="w-8 h-8 rounded-lg flex items-center justify-center mb-1 {isSelected ? 'text-sky-400' : 'text-slate-500 group-hover:text-slate-300'}">
							<Icon name={preset.icon} size={18} />
						</div>
						<span class="text-sm font-extrabold text-white leading-tight">{preset.label}</span>
						<span class="text-[10px] uppercase font-mono font-semibold text-slate-500 mt-0.5">{preset.type}</span>
					</button>
				{/each}
			</div>

			<!-- Launch Actions -->
			<div class="flex flex-col gap-3">
				<button
					class="w-full py-4 bg-gradient-to-r from-sky-500 to-indigo-600 hover:from-sky-400 hover:to-indigo-500 text-white font-black text-base rounded-xl transition-all shadow-[0_4px_25px_rgba(56,189,248,0.35)] hover:shadow-[0_6px_35px_rgba(56,189,248,0.5)] active:scale-[0.98] flex items-center justify-center gap-2"
					on:click={startMatchmaking}
				>
					<Icon name="zap" size={18} />
					<span>Queue for {selectedPreset.label} {selectedPreset.type}</span>
				</button>

				<button
					class="w-full py-3 bg-slate-900/80 hover:bg-slate-800 border border-slate-800 text-slate-300 font-semibold text-xs rounded-xl transition flex items-center justify-center gap-2"
					on:click={createDirectGame}
				>
					<Icon name="swords" size={14} className="text-slate-500" />
					<span>Instant Match vs Guest</span>
				</button>
			</div>
		</div>
	{/if}
</div>
