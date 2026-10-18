<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import UserBadge from '$lib/components/ui/UserBadge.svelte';

	let activeCategory = 'blitz';
	let leaderboard: any[] = [];
	let loading = true;
	let error = '';

	const categories: Array<{ id: string; label: string; icon: 'zap' | 'flame' | 'timer' | 'hourglass' | 'puzzle' }> = [
		{ id: 'bullet', label: 'Bullet', icon: 'zap' },
		{ id: 'blitz', label: 'Blitz', icon: 'flame' },
		{ id: 'rapid', label: 'Rapid', icon: 'timer' },
		{ id: 'classical', label: 'Classical', icon: 'hourglass' },
		{ id: 'puzzles', label: 'Puzzles', icon: 'puzzle' }
	];

	async function loadLeaderboard(cat: string) {
		activeCategory = cat;
		loading = true;
		error = '';
		try {
			const res = await api.leaderboard.get(cat);
			leaderboard = Array.isArray(res) ? res : [];
		} catch (err: any) {
			error = err.message || 'Failed to load leaderboard';
			leaderboard = [];
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadLeaderboard('blitz');
	});
</script>

<div class="flex-1 max-w-4xl mx-auto w-full p-4 sm:p-6">
	<!-- Header -->
	<div class="mb-6 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
		<div>
			<Badge variant="outline" className="mb-2 font-mono text-xs uppercase tracking-wider">
				<Icon name="trophy" size={12} className="mr-1" />
				<span>Global Standings</span>
			</Badge>
			<h1 class="text-2xl sm:text-3xl font-black text-foreground tracking-tight">Leaderboard</h1>
			<p class="text-xs sm:text-sm text-muted-foreground mt-0.5">Top ranked players across all rating pools</p>
		</div>
	</div>

	<!-- Time Control Category Tabs -->
	<div class="flex gap-2 pb-3 mb-5 overflow-x-auto select-none">
		{#each categories as cat}
			{@const active = activeCategory === cat.id}
			<button
				class="flex items-center gap-2 px-3.5 py-2 rounded-xl font-semibold text-xs transition-all duration-150 whitespace-nowrap {active
					? 'bg-foreground text-background font-bold shadow-xs'
					: 'bg-card hover:bg-muted text-muted-foreground hover:text-foreground border border-border'}"
				on:click={() => loadLeaderboard(cat.id)}
			>
				<Icon name={cat.icon} size={14} />
				<span>{cat.label}</span>
			</button>
		{/each}
	</div>

	<!-- Table Container -->
	<Card className="overflow-hidden shadow-lg border-border">
		{#if loading}
			<div class="p-16 flex flex-col items-center justify-center gap-3 text-muted-foreground">
				<div class="w-8 h-8 border-2 border-foreground border-t-transparent rounded-full animate-spin"></div>
				<span class="text-xs font-mono">Loading rankings...</span>
			</div>
		{:else if error}
			<div class="p-8 text-center text-destructive text-sm">
				{error}
			</div>
		{:else if leaderboard.length === 0}
			<div class="p-16 text-center text-muted-foreground text-sm">
				No players ranked in this category yet. Play a game to establish the first rating!
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-left text-sm">
					<thead class="bg-secondary/40 border-b border-border text-[11px] text-muted-foreground uppercase font-bold tracking-wider">
						<tr>
							<th class="py-3 px-4 w-16 text-center">Rank</th>
							<th class="py-3 px-4">Player</th>
							<th class="py-3 px-4 text-right">Rating</th>
							<th class="py-3 px-4 text-center">Games</th>
							<th class="py-3 px-4 text-right">Record</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border/60">
						{#each leaderboard as entry, i}
							<tr class="hover:bg-muted/40 transition">
								<td class="py-3.5 px-4 text-center">
									{#if i === 0}
										<div class="inline-flex items-center justify-center w-7 h-7 rounded-lg bg-amber-500/15 text-amber-500 font-bold text-xs border border-amber-500/30">
											<Icon name="trophy" size={14} />
										</div>
									{:else if i === 1}
										<div class="inline-flex items-center justify-center w-7 h-7 rounded-lg bg-secondary text-foreground font-bold text-xs border border-border">
											<Icon name="award" size={14} />
										</div>
									{:else if i === 2}
										<div class="inline-flex items-center justify-center w-7 h-7 rounded-lg bg-amber-700/15 text-amber-600 font-bold text-xs border border-amber-700/30">
											<Icon name="award" size={14} />
										</div>
									{:else}
										<span class="font-mono text-xs text-muted-foreground font-semibold tabular-nums">#{i + 1}</span>
									{/if}
								</td>
								<td class="py-2 px-4">
									<UserBadge
										username={entry.username}
										avatar={entry.avatar}
										showRating={false}
										size="sm"
										className="bg-transparent border-transparent hover:bg-secondary/60"
									/>
								</td>
								<td class="py-3.5 px-4 text-right font-mono font-extrabold text-foreground tabular-nums">
									{entry.rating}
								</td>
								<td class="py-3.5 px-4 text-center font-mono text-muted-foreground text-xs tabular-nums">
									{entry.games}
								</td>
								<td class="py-3.5 px-4 text-right font-mono text-xs tabular-nums">
									<span class="text-emerald-500 font-semibold">{entry.wins}W</span>
									<span class="text-muted-foreground/60 mx-1">•</span>
									<span class="text-destructive font-semibold">{entry.losses}L</span>
									{#if entry.draws}
										<span class="text-muted-foreground/60 mx-1">•</span>
										<span class="text-amber-500 font-semibold">{entry.draws}D</span>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</Card>
</div>
