<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { api } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import BorderBeam from '$lib/components/magicui/BorderBeam.svelte';

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
	let activeOngoingGame: any = null;

	onMount(async () => {
		if ($authStore.user) {
			try {
				const ag = await api.games.getActive();
				if (ag && ag.status === 'playing') {
					activeOngoingGame = ag;
				}
			} catch (_) {}
		}
	});

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

		if (activeOngoingGame) {
			try {
				await api.games.resign(activeOngoingGame.id);
			} catch (_) {}
			activeOngoingGame = null;
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

			pollTimer = setInterval(async () => {
				try {
					const status = await api.matchmaking.status();
					if (status && status.status === 'matched') {
						stopTimers();
						isSearching = false;
						const colorQuery = status.color ? `?color=${status.color}` : '';
						goto(`/game/${status.gameId}${colorQuery}`);
					}
				} catch (e) {
					// keep polling
				}
			}, 1500);
		} catch (err: any) {
			error = err.message || 'Failed to queue for match';
			isSearching = false;
			stopTimers();
		}
	}

	async function cancelMatchmaking() {
		stopTimers();
		isSearching = false;
		try {
			await api.matchmaking.leave();
		} catch (err) {
			console.error('Failed to leave matchmaking queue:', err);
		}
	}

	async function createDirectGame() {
		if (!$authStore.user) {
			goto('/login');
			return;
		}
		if (activeOngoingGame) {
			try {
				await api.games.resign(activeOngoingGame.id);
			} catch (_) {}
			activeOngoingGame = null;
		}
		try {
			const ag = await api.games.create({
				initial: selectedPreset.initial,
				increment: selectedPreset.increment
			});
			goto(`/game/${ag.id}?color=white`);
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
		<Badge variant="default" className="mb-3">
			<Icon name="zap" size={13} />
			<span>Online Matchmaking</span>
		</Badge>
		<h1 class="text-3xl sm:text-4xl font-black text-white tracking-tight">Play Online</h1>
		<p class="text-xs sm:text-sm text-slate-400 mt-1 max-w-md mx-auto">
			Select your preferred time control to find a rated match.
		</p>
	</div>

	{#if activeOngoingGame}
		<div class="mb-6 p-4 bg-amber-500/10 border border-amber-500/40 rounded-2xl flex flex-col sm:flex-row items-center justify-between gap-4 max-w-xl mx-auto shadow-xl animate-in fade-in duration-200">
			<div class="flex items-center gap-3">
				<div class="w-10 h-10 rounded-xl bg-amber-500/20 text-amber-400 flex items-center justify-center font-bold">
					<Icon name="swords" size={20} />
				</div>
				<div>
					<div class="text-sm font-bold text-white">Active Match in Progress</div>
					<div class="text-xs text-amber-200/80">
						You have an unfinished game in progress. Return to avoid abandonment forfeit!
					</div>
				</div>
			</div>
			<div class="flex items-center gap-2 w-full sm:w-auto">
				<Button
					variant="outline"
					size="sm"
					className="flex-1 sm:flex-initial text-rose-400 hover:text-rose-300 border-rose-500/40 hover:bg-rose-500/10"
					on:click={async () => {
						if (activeOngoingGame) {
							try {
								await api.games.resign(activeOngoingGame.id);
							} catch (_) {}
							activeOngoingGame = null;
						}
					}}
				>
					Forfeit
				</Button>
				<Button
					variant="primary"
					size="sm"
					className="flex-1 sm:flex-initial"
					on:click={() => goto(`/game/${activeOngoingGame.id}`)}
				>
					Resume
				</Button>
			</div>
		</div>
	{/if}

	{#if error}
		<div class="mb-6 p-4 bg-rose-500/10 border border-rose-500/30 rounded-xl text-sm text-rose-300 text-center max-w-md mx-auto shadow-md">
			{error}
		</div>
	{/if}

	{#if isSearching}
		<!-- Matchmaking Search Radar Modal with BorderBeam -->
		<div class="relative max-w-md mx-auto w-full p-8 sm:p-10 text-center flex flex-col items-center animate-fade-in bg-slate-900/80 backdrop-blur-xl border border-sky-500/40 rounded-2xl shadow-2xl overflow-hidden">
			<BorderBeam size={180} duration={6} colorFrom="#38bdf8" colorTo="#818cf8" />

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
			<div class="w-full grid grid-cols-3 gap-2 bg-slate-950/80 border border-slate-800 rounded-xl p-3.5 mb-6 text-center">
				<div>
					<span class="block text-white font-mono text-base font-extrabold">{waitSeconds}s</span>
					<span class="text-[10px] text-slate-400 uppercase tracking-wider font-semibold">Wait Time</span>
				</div>
				<div>
					<span class="block text-sky-400 font-mono text-base font-extrabold">±{tolerance}</span>
					<span class="text-[10px] text-slate-400 uppercase tracking-wider font-semibold">Tolerance</span>
				</div>
				<div>
					<span class="block text-indigo-400 font-mono text-base font-extrabold">{userRating}</span>
					<span class="text-[10px] text-slate-400 uppercase tracking-wider font-semibold">Your Rating</span>
				</div>
			</div>

			<Button
				variant="outline"
				size="lg"
				className="w-full text-slate-300 hover:text-white"
				on:click={cancelMatchmaking}
			>
				Cancel Search
			</Button>
		</div>
	{:else}
		<!-- Time Control Selector -->
		<Card className="p-6 sm:p-8 max-w-xl mx-auto w-full shadow-2xl border-slate-800">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-xs font-mono font-bold uppercase tracking-widest text-slate-400">Select Time Control</h2>
				<span class="text-[11px] font-mono text-sky-400 font-semibold">
					Your Rating: {userRating}
				</span>
			</div>

			<!-- Presets Grid -->
			<div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
				{#each presets as preset}
					{@const isSelected = selectedPreset.label === preset.label}
					<button
						class="flex flex-col items-center justify-center p-3 rounded-2xl border transition-all duration-150 text-center group cursor-pointer active:translate-y-px
						{isSelected
							? 'bg-slate-800 border-emerald-500 text-white ring-1 ring-emerald-500/40 shadow-xs'
							: 'bg-slate-950/60 border-slate-800 text-slate-400 hover:text-white hover:border-slate-750 hover:bg-slate-900'}"
						on:click={() => (selectedPreset = preset)}
					>
						<div class="w-8 h-8 rounded-lg flex items-center justify-center mb-1 {isSelected ? 'text-emerald-400' : 'text-slate-500 group-hover:text-slate-300'}">
							<Icon name={preset.icon} size={18} />
						</div>
						<span class="text-sm font-extrabold text-white leading-tight">{preset.label}</span>
						<span class="text-[10px] uppercase font-mono font-semibold text-slate-500 mt-0.5">{preset.type}</span>
					</button>
				{/each}
			</div>

			<!-- Launch Actions -->
			<div class="flex flex-col gap-3">
				<Button
					variant="primary"
					size="lg"
					className="w-full text-sm font-bold shadow-xs"
					on:click={startMatchmaking}
				>
					<Icon name="zap" size={18} />
					<span>Play {selectedPreset.label} {selectedPreset.type}</span>
				</Button>

				<Button
					variant="secondary"
					size="default"
					className="w-full"
					on:click={createDirectGame}
				>
					<Icon name="swords" size={14} className="text-slate-400" />
					<span>Direct Match vs Guest</span>
				</Button>
			</div>
		</Card>
	{/if}
</div>
