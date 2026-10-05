<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import Icon from '$lib/components/icons/Icon.svelte';

	let activeCategory = 'blitz';
	let leaderboard: any[] = [];
	let loading = true;
	let error = '';

	const categories: Array<{ id: string; label: string; icon: 'zap' | 'flame' | 'timer' | 'hourglass' }> = [
		{ id: 'bullet', label: 'Bullet', icon: 'zap' },
		{ id: 'blitz', label: 'Blitz', icon: 'flame' },
		{ id: 'rapid', label: 'Rapid', icon: 'timer' },
		{ id: 'classical', label: 'Classical', icon: 'hourglass' }
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
	<div class="mb-6">
		<div class="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-amber-500/10 border border-amber-500/20 text-amber-400 text-xs font-semibold mb-2">
			<Icon name="trophy" size={14} />
			<span>Global Standings</span>
		</div>
		<h1 class="text-2xl sm:text-3xl font-extrabold text-white tracking-tight">Leaderboard</h1>
		<p class="text-xs sm:text-sm text-slate-400 mt-1">Top rated grandmasters and players across all time controls</p>
	</div>

	<!-- Time Control Category Tabs -->
	<div class="flex gap-2 pb-4 mb-6 overflow-x-auto">
		{#each categories as cat}
			<button
				class="flex items-center gap-2 px-4 py-2.5 rounded-xl font-semibold text-xs transition-all duration-150 {activeCategory === cat.id
					? 'bg-gradient-to-r from-sky-500 to-indigo-600 text-white shadow-lg shadow-sky-500/20'
					: 'bg-slate-900/80 hover:bg-slate-800 text-slate-400 hover:text-white border border-slate-800/80'}"
				on:click={() => loadLeaderboard(cat.id)}
			>
				<Icon name={cat.icon} size={15} />
				<span>{cat.label}</span>
			</button>
		{/each}
	</div>

	<!-- Table Container -->
	<div class="bg-slate-900/80 border border-slate-800/80 backdrop-blur-xl rounded-2xl overflow-hidden shadow-2xl">
		{#if loading}
			<div class="p-16 flex flex-col items-center justify-center gap-3 text-slate-400">
				<div class="w-8 h-8 border-2 border-sky-500/30 border-t-sky-400 rounded-full animate-spin"></div>
				<span class="text-xs font-mono">Fetching leaderboards...</span>
			</div>
		{:else if error}
			<div class="p-8 text-center text-rose-300 text-sm">
				{error}
			</div>
		{:else if leaderboard.length === 0}
			<div class="p-16 text-center text-slate-500 text-sm">
				No players ranked in this time control yet. Play a game to establish the first rating!
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-left text-sm">
					<thead class="bg-slate-950/60 border-b border-slate-800/80 text-[11px] text-slate-400 uppercase font-bold tracking-wider">
						<tr>
							<th class="py-3.5 px-4 w-16 text-center">Rank</th>
							<th class="py-3.5 px-4">Player</th>
							<th class="py-3.5 px-4 text-right">Rating</th>
							<th class="py-3.5 px-4 text-center">Games</th>
							<th class="py-3.5 px-4 text-right">W / L / D</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-slate-800/60">
						{#each leaderboard as entry, i}
							<tr class="hover:bg-slate-800/40 transition">
								<td class="py-3.5 px-4 text-center">
									{#if i === 0}
										<div class="inline-flex items-center justify-center w-7 h-7 rounded-lg bg-amber-500/20 text-amber-400 font-bold text-xs ring-1 ring-amber-500/40">
											<Icon name="trophy" size={14} />
										</div>
									{:else if i === 1}
										<div class="inline-flex items-center justify-center w-7 h-7 rounded-lg bg-slate-300/20 text-slate-200 font-bold text-xs ring-1 ring-slate-400/40">
											<Icon name="award" size={14} />
										</div>
									{:else if i === 2}
										<div class="inline-flex items-center justify-center w-7 h-7 rounded-lg bg-amber-700/20 text-amber-500 font-bold text-xs ring-1 ring-amber-700/40">
											<Icon name="award" size={14} />
										</div>
									{:else}
										<span class="font-mono text-xs text-slate-500 font-semibold">#{i + 1}</span>
									{/if}
								</td>
								<td class="py-3.5 px-4">
									<a
										href="/profile/{entry.username}"
										class="font-bold text-white hover:text-sky-400 transition"
									>
										{entry.username}
									</a>
								</td>
								<td class="py-3.5 px-4 text-right font-mono font-extrabold text-cyan-400">
									{entry.rating}
								</td>
								<td class="py-3.5 px-4 text-center font-mono text-slate-400 text-xs">
									{entry.games}
								</td>
								<td class="py-3.5 px-4 text-right font-mono text-xs">
									<span class="text-emerald-400 font-medium">{entry.wins}</span>
									<span class="text-slate-600 mx-1">/</span>
									<span class="text-rose-400 font-medium">{entry.losses}</span>
									<span class="text-slate-600 mx-1">/</span>
									<span class="text-amber-400 font-medium">{entry.draws}</span>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</div>
