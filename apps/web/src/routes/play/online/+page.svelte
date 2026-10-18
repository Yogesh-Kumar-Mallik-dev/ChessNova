<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { api } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Card from '$lib/components/ui/Card.svelte';

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
		if (waitSeconds < 10) return '50';
		if (waitSeconds < 20) return '100';
		if (waitSeconds < 30) return '200';
		if (waitSeconds < 45) return '500';
		return 'Any';
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

<div class="flex-1 max-w-4xl mx-auto w-full px-4 py-8 sm:py-12 flex flex-col justify-center">
	<div class="text-center mb-8">
		<Badge variant="outline" className="mb-3 font-mono text-xs uppercase tracking-wider">
			<Icon name="zap" size={12} className="mr-1" />
			<span>Matchmaking Queue</span>
		</Badge>
		<h1 class="text-3xl sm:text-4xl font-black text-foreground tracking-tight">Play Online</h1>
		<p class="text-xs sm:text-sm text-muted-foreground mt-1.5 max-w-md mx-auto">
			Select a standard time control to queue for a live rated pairing.
		</p>
	</div>

	{#if activeOngoingGame}
		<div class="mb-6 p-4 bg-amber-500/10 border border-amber-500/30 rounded-xl flex flex-col sm:flex-row items-center justify-between gap-4 max-w-xl mx-auto shadow-sm">
			<div class="flex items-center gap-3">
				<div class="w-9 h-9 rounded-lg bg-amber-500/20 text-amber-500 flex items-center justify-center font-bold">
					<Icon name="swords" size={18} />
				</div>
				<div>
					<div class="text-sm font-bold text-foreground">Active Match in Progress</div>
					<div class="text-xs text-muted-foreground">
						You have an unfinished game. Return to avoid clock forfeiture.
					</div>
				</div>
			</div>
			<div class="flex items-center gap-2 w-full sm:w-auto">
				<Button
					variant="outline"
					size="sm"
					className="flex-1 sm:flex-initial text-destructive border-destructive/30 hover:bg-destructive/10"
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
		<div class="mb-6 p-4 bg-destructive/10 border border-destructive/30 rounded-xl text-sm text-destructive text-center max-w-md mx-auto">
			{error}
		</div>
	{/if}

	{#if isSearching}
		<!-- Matchmaking Search Radar Status Card -->
		<div class="max-w-md mx-auto w-full p-8 sm:p-10 text-center flex flex-col items-center bg-card border border-border rounded-2xl shadow-xl">
			<!-- Sleek Radar Display -->
			<div class="relative w-28 h-28 rounded-full border border-border flex items-center justify-center mb-6">
				<div class="absolute inset-3 rounded-full border border-border/60"></div>
				<div class="absolute inset-7 rounded-full border border-border/40"></div>
				<!-- Radar Sweep -->
				<div class="absolute inset-0 rounded-full border-2 border-foreground border-t-transparent animate-spin"></div>
				<div class="w-3 h-3 rounded-full bg-foreground animate-ping"></div>
			</div>

			<h2 class="text-xl font-bold text-foreground mb-1">Searching for Opponent</h2>
			<div class="text-xs font-mono text-muted-foreground font-semibold mb-6 flex items-center gap-1.5">
				<Icon name={selectedPreset.icon} size={13} />
				<span>{selectedPreset.label} • {selectedPreset.type}</span>
			</div>

			<!-- Dynamic Queue Stats -->
			<div class="w-full grid grid-cols-3 gap-2 bg-secondary/50 border border-border rounded-xl p-3.5 mb-6 text-center">
				<div>
					<span class="block text-foreground font-mono text-base font-bold tabular-nums">{waitSeconds}s</span>
					<span class="text-[10px] text-muted-foreground uppercase tracking-wider font-semibold">Elapsed</span>
				</div>
				<div>
					<span class="block text-foreground font-mono text-base font-bold tabular-nums">{tolerance === 'Any' ? 'Any' : `±${tolerance}`}</span>
					<span class="text-[10px] text-muted-foreground uppercase tracking-wider font-semibold">Tolerance</span>
				</div>
				<div>
					<span class="block text-foreground font-mono text-base font-bold tabular-nums">{userRating}</span>
					<span class="text-[10px] text-muted-foreground uppercase tracking-wider font-semibold">Your Rating</span>
				</div>
			</div>

			<Button
				variant="outline"
				size="lg"
				className="w-full"
				on:click={cancelMatchmaking}
			>
				Cancel Search
			</Button>
		</div>
	{:else}
		<!-- Time Control Selector -->
		<Card className="p-6 sm:p-8 max-w-xl mx-auto w-full shadow-lg">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-xs font-mono font-bold uppercase tracking-wider text-muted-foreground">Select Time Control</h2>
				<span class="text-xs font-mono text-foreground font-semibold tabular-nums">
					Current Rating: {userRating}
				</span>
			</div>

			<!-- Presets Grid -->
			<div class="grid grid-cols-2 sm:grid-cols-4 gap-2.5 mb-6">
				{#each presets as preset}
					{@const isSelected = selectedPreset.label === preset.label}
					<button
						type="button"
						class="flex flex-col items-center justify-center p-3.5 rounded-xl border transition-all text-center cursor-pointer active:translate-y-px
						{isSelected
							? 'bg-foreground text-background border-foreground font-bold shadow-xs'
							: 'bg-card border-border text-foreground hover:bg-muted/70 hover:border-foreground/30'}"
						on:click={() => (selectedPreset = preset)}
					>
						<div class="w-7 h-7 rounded-lg flex items-center justify-center mb-1 {isSelected ? 'text-background' : 'text-muted-foreground'}">
							<Icon name={preset.icon} size={17} />
						</div>
						<span class="text-sm font-extrabold tracking-tight leading-tight">{preset.label}</span>
						<span class="text-[10px] uppercase font-mono font-semibold opacity-75 mt-0.5">{preset.type}</span>
					</button>
				{/each}
			</div>

			<!-- Launch Actions -->
			<div class="flex flex-col gap-2.5">
				<Button
					variant="primary"
					size="lg"
					className="w-full text-sm font-bold"
					on:click={startMatchmaking}
				>
					<Icon name="zap" size={17} />
					<span>Queue for {selectedPreset.label} {selectedPreset.type}</span>
				</Button>

				<Button
					variant="secondary"
					size="default"
					className="w-full text-xs font-medium"
					on:click={createDirectGame}
				>
					<Icon name="swords" size={14} className="text-muted-foreground" />
					<span>Direct Match vs Guest</span>
				</Button>
			</div>
		</Card>
	{/if}
</div>
