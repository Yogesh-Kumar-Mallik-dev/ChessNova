<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { api } from '$lib/api/client';
	import Icon from '$lib/components/icons/Icon.svelte';

	let username = $page.params.username || '';
	let profileData: any = null;
	let loading = true;
	let error = '';

	const ratingCategories: Array<{ id: 'bullet' | 'blitz' | 'rapid' | 'classical'; label: string; icon: 'zap' | 'flame' | 'timer' | 'hourglass' }> = [
		{ id: 'bullet', label: 'Bullet', icon: 'zap' },
		{ id: 'blitz', label: 'Blitz', icon: 'flame' },
		{ id: 'rapid', label: 'Rapid', icon: 'timer' },
		{ id: 'classical', label: 'Classical', icon: 'hourglass' }
	];

	onMount(async () => {
		try {
			profileData = await api.users.getUser(username);
		} catch (err: any) {
			error = err.message || 'Failed to load profile';
		} finally {
			loading = false;
		}
	});
</script>

<div class="flex-1 max-w-4xl mx-auto w-full p-4 sm:p-6">
	{#if loading}
		<div class="p-16 flex flex-col items-center justify-center gap-3 text-slate-400">
			<div class="w-8 h-8 border-2 border-sky-500/30 border-t-sky-400 rounded-full animate-spin"></div>
			<span class="text-xs font-mono">Loading profile data...</span>
		</div>
	{:else if error}
		<div class="p-8 text-center text-rose-300 text-sm bg-rose-950/20 border border-rose-800/40 rounded-2xl">
			{error}
		</div>
	{:else if profileData}
		<!-- Header Card -->
		<div class="bg-slate-900/80 border border-slate-800/80 backdrop-blur-xl rounded-2xl p-6 sm:p-8 shadow-2xl mb-6 flex flex-col sm:flex-row items-center sm:items-start gap-6 text-center sm:text-left">
			<div class="w-20 h-20 rounded-2xl bg-gradient-to-br from-sky-500/20 to-indigo-600/30 border border-sky-500/30 flex items-center justify-center text-3xl font-extrabold text-sky-400 shadow-inner">
				{profileData.user.username[0].toUpperCase()}
			</div>

			<div class="flex-1">
				<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
					<div>
						<h1 class="text-2xl sm:text-3xl font-extrabold text-white tracking-tight">
							{profileData.user.username}
						</h1>
						<p class="text-xs text-slate-400 mt-1">
							Member since {new Date(profileData.user.createdAt).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })}
						</p>
					</div>

					<div class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-slate-800/80 border border-slate-700/60 text-slate-300 text-xs self-center sm:self-start">
						<Icon name="shield" size={13} className="text-sky-400" />
						<span>Verified Player</span>
					</div>
				</div>

				<!-- Ratings Grid -->
				<div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-6">
					{#each ratingCategories as cat}
						{@const r = profileData.ratings?.[cat.id]}
						<div class="bg-slate-950/60 border border-slate-800/80 p-3.5 rounded-xl hover:border-slate-700 transition">
							<div class="flex items-center justify-between text-slate-400 mb-1">
								<span class="text-[10px] uppercase font-bold tracking-wider">{cat.label}</span>
								<Icon name={cat.icon} size={13} className="text-slate-500" />
							</div>
							<div class="text-xl font-mono font-extrabold text-white">
								{r?.rating || 400}
							</div>
							<div class="text-[10px] text-slate-500 mt-0.5">
								{r?.games || 0} games played
							</div>
						</div>
					{/each}
				</div>
			</div>
		</div>

		<!-- Recent Games -->
		<div class="bg-slate-900/80 border border-slate-800/80 backdrop-blur-xl rounded-2xl p-6 shadow-2xl">
			<div class="flex items-center justify-between mb-5">
				<div>
					<h2 class="text-lg font-bold text-white tracking-tight">Match History</h2>
					<p class="text-xs text-slate-400">Archived rated games and evaluations</p>
				</div>
			</div>

			{#if !profileData.recentGames || profileData.recentGames.length === 0}
				<div class="p-12 text-center text-slate-500 text-xs italic">
					No completed games found on record yet.
				</div>
			{:else}
				<div class="divide-y divide-slate-800/60">
					{#each profileData.recentGames as g}
						{@const isWhite = g.white.username === profileData.user.username}
						{@const opp = isWhite ? g.black : g.white}
						{@const won = (g.result === '1-0' && isWhite) || (g.result === '0-1' && !isWhite)}
						{@const isDraw = g.result === '1/2-1/2'}
						{@const resultClass = isDraw
							? 'text-amber-400 bg-amber-500/10 border-amber-500/20'
							: won
								? 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20'
								: 'text-rose-400 bg-rose-500/10 border-rose-500/20'}

						<div class="py-3.5 flex items-center justify-between gap-4 hover:bg-slate-800/30 px-3 rounded-xl transition">
							<div class="flex items-center gap-3">
								<span
									class="w-3 h-3 rounded-full shrink-0 {isWhite
										? 'bg-slate-100 ring-1 ring-white/50 shadow-sm'
										: 'bg-slate-950 border border-slate-700'}"
									title={isWhite ? 'Played as White' : 'Played as Black'}
								></span>
								<div>
									<div class="text-sm font-semibold text-white flex items-center gap-2">
										<span class="text-xs text-slate-400">vs</span>
										<a href="/profile/{opp.username}" class="hover:text-sky-400 transition font-bold">
											{opp.username}
										</a>
										<span class="text-xs font-mono text-slate-500">({opp.rating})</span>
									</div>
									<div class="text-[11px] text-slate-400 mt-0.5">
										{g.timeControl ? `${g.timeControl.initial / 60}+${g.timeControl.increment}` : '5+0'} • {new Date(g.startedAt).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
									</div>
								</div>
							</div>

							<div class="flex items-center gap-2">
								<span class="px-2.5 py-1 rounded-lg text-xs font-mono font-bold border {resultClass}">
									{g.result}
								</span>

								<a
									href="/analysis?gameId={g.id}&review=1"
									class="px-3 py-1.5 bg-[#81b64c] hover:bg-[#91c65d] text-white font-bold text-xs rounded-lg shadow transition flex items-center gap-1.5"
								>
									<Icon name="search" size={13} />
									<span>Review</span>
								</a>


								<a
									href="/api/v1/games/{g.id}/pgn"
									download
									class="p-1.5 bg-slate-950 hover:bg-slate-800 text-slate-400 hover:text-white border border-slate-800 rounded-lg transition"
									title="Download PGN"
								>
									<Icon name="download" size={14} />
								</a>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>
