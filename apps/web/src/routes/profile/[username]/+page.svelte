<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth';
	import { stageGameForReview } from '$lib/stores/localGames';
	import Icon from '$lib/components/icons/Icon.svelte';

	let username = $page.params.username || '';
	let profileData: any = null;
	let externalStats: any = null;
	let externalGames: any[] = [];
	let loading = true;
	let loadingExternal = false;
	let error = '';

	// Active navigation tab
	let activeTab: 'nova-games' | 'external-games' | 'stats' = 'nova-games';
	let externalPlatformFilter: 'all' | 'chesscom' | 'lichess' = 'all';

	// Modals state
	let showEditModal = false;
	let showLinkModal = false;
	let editBio = '';
	let editCountry = '';
	let editChessComUser = '';
	let editLichessUser = '';
	let savingProfile = false;
	let linkingAccounts = false;
	let modalError = '';
	let modalSuccess = '';

	// Avatar upload state
	let avatarInput: HTMLInputElement;
	let uploadingAvatar = false;
	let avatarError = '';

	$: isOwnProfile = $authStore.user?.username?.toLowerCase() === username.toLowerCase();

	const ratingCategories: Array<{ id: 'bullet' | 'blitz' | 'rapid' | 'classical'; label: string; icon: 'zap' | 'flame' | 'timer' | 'hourglass' }> = [
		{ id: 'bullet', label: 'Bullet', icon: 'zap' },
		{ id: 'blitz', label: 'Blitz', icon: 'flame' },
		{ id: 'rapid', label: 'Rapid', icon: 'timer' },
		{ id: 'classical', label: 'Classical', icon: 'hourglass' }
	];

	async function loadProfile() {
		loading = true;
		error = '';
		try {
			profileData = await api.users.getUser(username);
			editBio = profileData.user?.bio || '';
			editCountry = profileData.user?.country || '';
			editChessComUser = profileData.user?.chessComUsername || '';
			editLichessUser = profileData.user?.lichessUsername || '';

			// Load external platform stats and games if linked
			if (profileData.user?.chessComUsername || profileData.user?.lichessUsername) {
				loadExternalData();
			}
		} catch (err: any) {
			error = err.message || 'Failed to load profile';
		} finally {
			loading = false;
		}
	}

	async function loadExternalData() {
		loadingExternal = true;
		try {
			const [stats, games] = await Promise.all([
				api.users.getExternalStats(username).catch(() => null),
				api.users.getExternalGames(username, externalPlatformFilter, 15).catch(() => [])
			]);
			externalStats = stats;
			externalGames = games || [];
		} catch (err) {
			console.warn('Failed to load external stats/games:', err);
		} finally {
			loadingExternal = false;
		}
	}

	async function handleAvatarSelect(e: Event) {
		const target = e.target as HTMLInputElement;
		const file = target.files?.[0];
		if (!file) return;

		if (file.size > 5 * 1024 * 1024) {
			avatarError = 'Avatar file size must be less than 5MB';
			return;
		}

		uploadingAvatar = true;
		avatarError = '';
		try {
			const res = await api.users.uploadAvatar(file);
			if (profileData?.user) {
				profileData.user.avatar = res.avatar;
			}
			authStore.updateUser({ avatar: res.avatar });
		} catch (err: any) {
			avatarError = err.message || 'Failed to upload avatar';
		} finally {
			uploadingAvatar = false;
			target.value = '';
		}
	}

	async function handleSaveProfile() {
		savingProfile = true;
		modalError = '';
		modalSuccess = '';
		try {
			await api.users.updateProfile({
				bio: editBio,
				country: editCountry
			});
			if (profileData?.user) {
				profileData.user.bio = editBio;
				profileData.user.country = editCountry;
			}
			authStore.updateUser({ bio: editBio, country: editCountry });
			modalSuccess = 'Profile updated successfully!';
			setTimeout(() => {
				showEditModal = false;
				modalSuccess = '';
			}, 1000);
		} catch (err: any) {
			modalError = err.message || 'Failed to update profile';
		} finally {
			savingProfile = false;
		}
	}

	async function handleLinkAccounts() {
		linkingAccounts = true;
		modalError = '';
		modalSuccess = '';
		try {
			const res = await api.users.linkAccounts({
				chessComUsername: editChessComUser,
				lichessUsername: editLichessUser
			});
			if (profileData?.user) {
				profileData.user.chessComUsername = res.chessComUsername;
				profileData.user.lichessUsername = res.lichessUsername;
			}
			authStore.updateUser({
				chessComUsername: res.chessComUsername,
				lichessUsername: res.lichessUsername
			});
			modalSuccess = 'Accounts linked and verified!';
			await loadExternalData();
			setTimeout(() => {
				showLinkModal = false;
				modalSuccess = '';
			}, 1200);
		} catch (err: any) {
			modalError = err.message || 'Failed to link accounts';
		} finally {
			linkingAccounts = false;
		}
	}

	function handleAnalyzeExternalGame(g: any) {
		if (!g.pgn) return;
		stageGameForReview(g.pgn, g.white, g.black);
		goto('/analysis?review=1');
	}

	onMount(() => {
		loadProfile();
	});
</script>

<svelte:head>
	<title>{username} - Player Profile | ChessNova</title>
</svelte:head>

<div class="flex-1 max-w-6xl mx-auto w-full p-3 sm:p-6 select-none">
	{#if loading}
		<div class="p-20 flex flex-col items-center justify-center gap-3 text-slate-400">
			<div class="w-9 h-9 border-2 border-sky-500/30 border-t-sky-400 rounded-full animate-spin"></div>
			<span class="text-xs font-mono tracking-wider uppercase text-slate-500">Loading Player Dossier...</span>
		</div>
	{:else if error}
		<div class="p-8 text-center text-rose-300 text-sm bg-rose-950/20 border border-rose-800/40 rounded-2xl shadow-xl max-w-md mx-auto my-12">
			<Icon name="x" size={24} className="mx-auto text-rose-400 mb-2" />
			<p class="font-bold">{error}</p>
			<a href="/leaderboard" class="mt-4 inline-block text-xs font-bold text-sky-400 hover:underline">Return to Leaderboards</a>
		</div>
	{:else if profileData}
		<!-- 1. Hero Dossier Banner -->
		<div class="relative bg-slate-900/90 border border-slate-800 rounded-3xl p-6 sm:p-8 shadow-2xl backdrop-blur-xl mb-6 overflow-hidden">
			<!-- Subtle glow background element -->
			<div class="absolute -top-24 -right-24 w-72 h-72 bg-sky-500/10 rounded-full blur-3xl pointer-events-none"></div>
			<div class="absolute -bottom-24 -left-24 w-72 h-72 bg-indigo-500/10 rounded-full blur-3xl pointer-events-none"></div>

			<div class="relative z-10 flex flex-col md:flex-row items-center md:items-start gap-6 text-center md:text-left">
				<!-- Avatar Area with Upload Trigger -->
				<div class="relative group shrink-0">
					<div class="w-24 h-24 sm:w-28 sm:h-28 rounded-2xl bg-gradient-to-br from-slate-800 to-slate-950 border-2 border-slate-700/80 shadow-2xl overflow-hidden flex items-center justify-center">
						{#if profileData.user.avatar}
							<img
								src={profileData.user.avatar}
								alt={profileData.user.username}
								class="w-full h-full object-cover"
								on:error={() => (profileData.user.avatar = '')}
							/>
						{:else}
							<div class="w-full h-full bg-gradient-to-br from-sky-500/20 via-indigo-600/30 to-purple-600/20 flex items-center justify-center text-4xl font-black text-sky-400 shadow-inner">
								{profileData.user.username[0].toUpperCase()}
							</div>
						{/if}
					</div>

					{#if isOwnProfile}
						<input
							type="file"
							accept="image/jpeg,image/png,image/webp,image/gif"
							class="hidden"
							bind:this={avatarInput}
							on:change={handleAvatarSelect}
						/>
						<button
							class="absolute -bottom-2 -right-2 p-2 bg-sky-500 hover:bg-sky-400 text-slate-950 rounded-xl shadow-lg transition active:translate-y-px"
							on:click={() => avatarInput.click()}
							title="Change avatar picture"
							disabled={uploadingAvatar}
						>
							{#if uploadingAvatar}
								<div class="w-3.5 h-3.5 border-2 border-slate-950 border-t-transparent rounded-full animate-spin"></div>
							{:else}
								<Icon name="camera" size={14} />
							{/if}
						</button>
					{/if}
				</div>

				<!-- Identity Details -->
				<div class="flex-1 min-w-0">
					<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
						<div>
							<div class="flex items-center justify-center sm:justify-start gap-2.5 flex-wrap">
								<h1 class="text-2xl sm:text-3xl font-black text-white tracking-tight">
									{profileData.user.username}
								</h1>
								<span class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full bg-sky-500/10 border border-sky-500/20 text-sky-400 text-[11px] font-bold">
									<Icon name="shield" size={11} />
									<span>Verified Player</span>
								</span>
								{#if profileData.user.country}
									<span class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full bg-slate-800 border border-slate-700 text-slate-300 text-[11px] font-semibold">
										<Icon name="flag" size={11} className="text-slate-400" />
										<span>{profileData.user.country}</span>
									</span>
								{/if}
							</div>
							<p class="text-xs text-slate-400 mt-1">
								Member since {new Date(profileData.user.createdAt).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })}
							</p>
						</div>

						<!-- Profile Owner Controls -->
						{#if isOwnProfile}
							<div class="flex items-center justify-center sm:justify-end gap-2 shrink-0">
								<button
									class="px-3 py-1.5 rounded-xl bg-slate-800 hover:bg-slate-750 text-slate-200 text-xs font-bold border border-slate-700 transition flex items-center gap-1.5 active:translate-y-px"
									on:click={() => (showEditModal = true)}
								>
									<Icon name="edit" size={13} className="text-sky-400" />
									<span>Edit Profile</span>
								</button>
								<button
									class="px-3 py-1.5 rounded-xl bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white text-xs font-bold shadow-md transition flex items-center gap-1.5 active:translate-y-px"
									on:click={() => (showLinkModal = true)}
								>
									<Icon name="external-link" size={13} />
									<span>Link Accounts</span>
								</button>
							</div>
						{/if}
					</div>

					<!-- Bio -->
					<div class="mt-3.5 text-xs text-slate-300 leading-relaxed max-w-2xl bg-slate-950/40 p-3 rounded-xl border border-slate-800/60">
						{#if profileData.user.bio}
							<p>{profileData.user.bio}</p>
						{:else}
							<p class="italic text-slate-500">
								{isOwnProfile ? 'No bio added yet. Click "Edit Profile" to tell other players about your chess style!' : 'This player hasn\'t written a bio yet.'}
							</p>
						{/if}
					</div>

					<!-- Linked Accounts Badges Bar -->
					<div class="mt-4 flex items-center justify-center sm:justify-start gap-2.5 flex-wrap">
						<!-- Chess.com Badge -->
						{#if profileData.user.chessComUsername}
							<div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-xl bg-[#81b64c]/10 border border-[#81b64c]/30 text-xs font-bold text-white">
								<div class="w-2 h-2 rounded-full bg-[#81b64c] shadow-xs animate-pulse"></div>
								<span class="text-[#96cd5f]">Chess.com:</span>
								<span>{profileData.user.chessComUsername}</span>
							</div>
						{:else if isOwnProfile}
							<button
								class="inline-flex items-center gap-1.5 px-3 py-1 rounded-xl bg-slate-800/60 hover:bg-slate-800 border border-dashed border-slate-700 text-slate-400 text-xs font-semibold transition"
								on:click={() => (showLinkModal = true)}
							>
								<Icon name="external-link" size={12} />
								<span>Connect Chess.com</span>
							</button>
						{/if}

						<!-- Lichess Badge -->
						{#if profileData.user.lichessUsername}
							<div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-xl bg-slate-800 border border-slate-700 text-xs font-bold text-white">
								<div class="w-2 h-2 rounded-full bg-white shadow-xs animate-pulse"></div>
								<span class="text-slate-300">Lichess:</span>
								<span>{profileData.user.lichessUsername}</span>
							</div>
						{:else if isOwnProfile}
							<button
								class="inline-flex items-center gap-1.5 px-3 py-1 rounded-xl bg-slate-800/60 hover:bg-slate-800 border border-dashed border-slate-700 text-slate-400 text-xs font-semibold transition"
								on:click={() => (showLinkModal = true)}
							>
								<Icon name="external-link" size={12} />
								<span>Connect Lichess</span>
							</button>
						{/if}
					</div>

					{#if avatarError}
						<p class="text-[11px] text-rose-400 mt-2">{avatarError}</p>
					{/if}
				</div>
			</div>
		</div>

		<!-- 2. Tri-Platform Comparative Elo Matrix -->
		<div class="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 sm:p-6 shadow-2xl backdrop-blur-xl mb-6">
			<div class="flex items-center justify-between pb-4 border-b border-slate-800 flex-wrap gap-2">
				<div>
					<div class="flex items-center gap-2">
						<h2 class="text-sm sm:text-base font-black text-white uppercase tracking-wider">Multi-Platform Elo Matrix</h2>
						<span class="text-[10px] font-mono font-bold text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full border border-emerald-500/20">Synced</span>
					</div>
					<p class="text-[11px] text-slate-400 mt-0.5">Direct comparative rating breakdown across ChessNova, Chess.com, and Lichess</p>
				</div>

				{#if profileData.user.chessComUsername || profileData.user.lichessUsername}
					<button
						class="px-3 py-1.5 bg-slate-800 hover:bg-slate-750 text-slate-300 hover:text-white rounded-xl text-xs font-bold transition border border-slate-700 flex items-center gap-1.5"
						on:click={loadExternalData}
						disabled={loadingExternal}
					>
						<Icon name="rotate-cw" size={12} className={loadingExternal ? 'animate-spin text-sky-400' : ''} />
						<span>Sync Live Ratings</span>
					</button>
				{/if}
			</div>

			<!-- Comparison Cards Grid -->
			<div class="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4">
				<!-- Platform 1: ChessNova -->
				<div class="bg-slate-950/80 border border-sky-500/30 rounded-xl p-4 shadow-lg flex flex-col justify-between">
					<div>
						<div class="flex items-center justify-between mb-3">
							<div class="flex items-center gap-2">
								<div class="w-6 h-6 rounded-lg bg-sky-500/20 flex items-center justify-center text-sky-400 font-bold text-xs">
									✦
								</div>
								<span class="text-xs font-extrabold text-white">ChessNova</span>
							</div>
							<span class="text-[10px] uppercase font-bold text-sky-400 bg-sky-500/10 px-2 py-0.5 rounded-full border border-sky-500/20">Primary</span>
						</div>

						<div class="space-y-2.5">
							{#each ratingCategories as cat}
								{@const r = profileData.ratings?.[cat.id]?.rating || 400}
								<div class="flex items-center justify-between text-xs">
									<span class="text-slate-400 flex items-center gap-1.5">
										<Icon name={cat.icon} size={12} className="text-slate-500" />
										<span>{cat.label}</span>
									</span>
									<span class="font-mono font-bold text-white">{r}</span>
								</div>
							{/each}
						</div>
					</div>

					<div class="pt-3 mt-3 border-t border-slate-850 text-[10px] text-slate-500 flex items-center justify-between">
						<span>Authoritative Platform</span>
						<span>400-2850 Scale</span>
					</div>
				</div>

				<!-- Platform 2: Chess.com -->
				<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-4 shadow-lg flex flex-col justify-between">
					<div>
						<div class="flex items-center justify-between mb-3">
							<div class="flex items-center gap-2">
								<div class="w-6 h-6 rounded-lg bg-[#81b64c]/20 flex items-center justify-center text-[#96cd5f] font-bold text-xs">
									♟
								</div>
								<span class="text-xs font-extrabold text-white">Chess.com</span>
							</div>
							{#if profileData.user.chessComUsername}
								<span class="text-[10px] font-mono text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full border border-emerald-500/20 truncate max-w-[100px]">
									@{profileData.user.chessComUsername}
								</span>
							{:else}
								<span class="text-[10px] text-slate-500">Unlinked</span>
							{/if}
						</div>

						{#if profileData.user.chessComUsername && externalStats?.chesscom}
							<div class="space-y-2.5">
								<div class="flex items-center justify-between text-xs">
									<span class="text-slate-400 flex items-center gap-1.5">
										<Icon name="timer" size={12} className="text-slate-500" />
										<span>Rapid</span>
									</span>
									<span class="font-mono font-bold text-white">{externalStats.chesscom.rapid || '—'}</span>
								</div>
								<div class="flex items-center justify-between text-xs">
									<span class="text-slate-400 flex items-center gap-1.5">
										<Icon name="flame" size={12} className="text-slate-500" />
										<span>Blitz</span>
									</span>
									<span class="font-mono font-bold text-white">{externalStats.chesscom.blitz || '—'}</span>
								</div>
								<div class="flex items-center justify-between text-xs">
									<span class="text-slate-400 flex items-center gap-1.5">
										<Icon name="zap" size={12} className="text-slate-500" />
										<span>Bullet</span>
									</span>
									<span class="font-mono font-bold text-white">{externalStats.chesscom.bullet || '—'}</span>
								</div>
								<div class="flex items-center justify-between text-xs">
									<span class="text-slate-400 flex items-center gap-1.5">
										<Icon name="puzzle" size={12} className="text-slate-500" />
										<span>Tactics</span>
									</span>
									<span class="font-mono font-bold text-white">{externalStats.chesscom.puzzle || '—'}</span>
								</div>
							</div>
						{:else if profileData.user.chessComUsername}
							<div class="py-6 text-center text-xs text-slate-500">
								{#if loadingExternal}
									<span class="animate-pulse">Syncing Chess.com ratings...</span>
								{:else}
									<span>No rated games found on Chess.com</span>
								{/if}
							</div>
						{:else}
							<div class="py-6 text-center text-xs text-slate-500 flex flex-col items-center gap-2">
								<p>Connect your account to compare ratings</p>
								{#if isOwnProfile}
									<button
										class="px-2.5 py-1 bg-[#81b64c] hover:bg-[#72a441] text-white text-[11px] font-bold rounded-lg transition"
										on:click={() => (showLinkModal = true)}
									>
										Link Now
									</button>
								{/if}
							</div>
						{/if}
					</div>

					<div class="pt-3 mt-3 border-t border-slate-850 text-[10px] text-slate-500 flex items-center justify-between">
						<span>Public API Sync</span>
						<span>Glicko-1 Curve</span>
					</div>
				</div>

				<!-- Platform 3: Lichess -->
				<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-4 shadow-lg flex flex-col justify-between">
					<div>
						<div class="flex items-center justify-between mb-3">
							<div class="flex items-center gap-2">
								<div class="w-6 h-6 rounded-lg bg-white/10 flex items-center justify-center text-white font-bold text-xs">
									♞
								</div>
								<span class="text-xs font-extrabold text-white">Lichess</span>
							</div>
							{#if profileData.user.lichessUsername}
								<span class="text-[10px] font-mono text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full border border-emerald-500/20 truncate max-w-[100px]">
									@{profileData.user.lichessUsername}
								</span>
							{:else}
								<span class="text-[10px] text-slate-500">Unlinked</span>
							{/if}
						</div>

						{#if profileData.user.lichessUsername && externalStats?.lichess}
							<div class="space-y-2.5">
								<div class="flex items-center justify-between text-xs">
									<span class="text-slate-400 flex items-center gap-1.5">
										<Icon name="timer" size={12} className="text-slate-500" />
										<span>Rapid</span>
									</span>
									<span class="font-mono font-bold text-white">{externalStats.lichess.rapid || '—'}</span>
								</div>
								<div class="flex items-center justify-between text-xs">
									<span class="text-slate-400 flex items-center gap-1.5">
										<Icon name="flame" size={12} className="text-slate-500" />
										<span>Blitz</span>
									</span>
									<span class="font-mono font-bold text-white">{externalStats.lichess.blitz || '—'}</span>
								</div>
								<div class="flex items-center justify-between text-xs">
									<span class="text-slate-400 flex items-center gap-1.5">
										<Icon name="zap" size={12} className="text-slate-500" />
										<span>Bullet</span>
									</span>
									<span class="font-mono font-bold text-white">{externalStats.lichess.bullet || '—'}</span>
								</div>
								<div class="flex items-center justify-between text-xs">
									<span class="text-slate-400 flex items-center gap-1.5">
										<Icon name="puzzle" size={12} className="text-slate-500" />
										<span>Puzzles</span>
									</span>
									<span class="font-mono font-bold text-white">{externalStats.lichess.puzzle || '—'}</span>
								</div>
							</div>
						{:else if profileData.user.lichessUsername}
							<div class="py-6 text-center text-xs text-slate-500">
								{#if loadingExternal}
									<span class="animate-pulse">Syncing Lichess ratings...</span>
								{:else}
									<span>No rated games found on Lichess</span>
								{/if}
							</div>
						{:else}
							<div class="py-6 text-center text-xs text-slate-500 flex flex-col items-center gap-2">
								<p>Connect your account to compare ratings</p>
								{#if isOwnProfile}
									<button
										class="px-2.5 py-1 bg-slate-800 hover:bg-slate-700 text-white text-[11px] font-bold rounded-lg transition border border-slate-700"
										on:click={() => (showLinkModal = true)}
									>
										Link Now
									</button>
								{/if}
							</div>
						{/if}
					</div>

					<div class="pt-3 mt-3 border-t border-slate-850 text-[10px] text-slate-500 flex items-center justify-between">
						<span>Public API Sync</span>
						<span>Glicko-2 Curve</span>
					</div>
				</div>
			</div>
		</div>

		<!-- 3. Navigation Tabs: Match History vs External Games vs Performance Stats -->
		<div class="bg-slate-900/90 border border-slate-800 rounded-2xl p-5 sm:p-6 shadow-2xl backdrop-blur-xl">
			<!-- Tab Switcher -->
			<div class="flex items-center justify-between pb-4 border-b border-slate-800 flex-wrap gap-3">
				<div class="flex items-center gap-2 bg-slate-950 p-1 rounded-xl border border-slate-800">
					<button
						class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition flex items-center gap-1.5 {activeTab === 'nova-games'
							? 'bg-slate-800 text-white shadow-xs'
							: 'text-slate-400 hover:text-slate-200'}"
						on:click={() => (activeTab = 'nova-games')}
					>
						<Icon name="swords" size={13} className={activeTab === 'nova-games' ? 'text-sky-400' : ''} />
						<span>ChessNova Matches</span>
					</button>

					<button
						class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition flex items-center gap-1.5 {activeTab === 'external-games'
							? 'bg-slate-800 text-white shadow-xs'
							: 'text-slate-400 hover:text-slate-200'}"
						on:click={() => {
							activeTab = 'external-games';
							if (externalGames.length === 0 && (profileData.user.chessComUsername || profileData.user.lichessUsername)) {
								loadExternalData();
							}
						}}
					>
						<Icon name="external-link" size={13} className={activeTab === 'external-games' ? 'text-emerald-400' : ''} />
						<span>Imported External Games</span>
						{#if externalGames.length > 0}
							<span class="px-1.5 py-0.2 rounded-full bg-emerald-500/20 text-emerald-400 text-[10px]">
								{externalGames.length}
							</span>
						{/if}
					</button>

					<button
						class="px-3.5 py-1.5 rounded-lg text-xs font-bold transition flex items-center gap-1.5 {activeTab === 'stats'
							? 'bg-slate-800 text-white shadow-xs'
							: 'text-slate-400 hover:text-slate-200'}"
						on:click={() => (activeTab = 'stats')}
					>
						<Icon name="award" size={13} className={activeTab === 'stats' ? 'text-amber-400' : ''} />
						<span>Performance Stats</span>
					</button>
				</div>

				{#if activeTab === 'external-games'}
					<!-- Platform filter for external games -->
					<div class="flex items-center gap-1 text-xs">
						<span class="text-slate-500 text-[11px] mr-1">Source:</span>
						{#each ['all', 'chesscom', 'lichess'] as p}
							<button
								class="px-2.5 py-1 rounded-lg text-[11px] font-bold capitalize transition border {externalPlatformFilter === p
									? 'bg-slate-800 border-slate-700 text-white'
									: 'text-slate-400 border-transparent hover:text-slate-200'}"
								on:click={() => {
									externalPlatformFilter = p as any;
									loadExternalData();
								}}
							>
								{p === 'chesscom' ? 'Chess.com' : p === 'lichess' ? 'Lichess' : 'All Platforms'}
							</button>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Tab 1: ChessNova Matches -->
			{#if activeTab === 'nova-games'}
				<div class="pt-4">
					{#if !profileData.recentGames || profileData.recentGames.length === 0}
						<div class="py-16 text-center text-slate-500 text-xs flex flex-col items-center justify-center gap-2">
							<div class="w-10 h-10 rounded-xl bg-slate-800/80 flex items-center justify-center text-slate-500">
								<Icon name="clock" size={20} />
							</div>
							<span class="font-bold text-slate-400">No ChessNova rated matches played yet.</span>
							<p class="text-[11px] text-slate-500 max-w-xs">Play matchmaking online or invite friends to generate rated game records.</p>
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

								<div class="py-3 flex items-center justify-between gap-4 hover:bg-slate-800/30 px-3 rounded-xl transition">
									<div class="flex items-center gap-3">
										<span
											class="w-3.5 h-3.5 rounded-full shrink-0 {isWhite
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
												{g.timeControl ? `${g.timeControl.initial / 60}+${g.timeControl.increment}` : '5+0'} • {new Date(g.startedAt).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })}
											</div>
										</div>
									</div>

									<div class="flex items-center gap-2">
										<span class="px-2.5 py-1 rounded-lg text-xs font-mono font-bold border {resultClass}">
											{g.result}
										</span>

										<a
											href="/analysis?gameId={g.id}&review=1"
											class="px-3 py-1.5 bg-[#81b64c] hover:bg-[#72a441] active:bg-[#639037] text-white font-bold text-xs rounded-lg shadow-xs transition flex items-center gap-1.5 active:translate-y-px"
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
											<Icon name="download" size={13} />
										</a>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>

			<!-- Tab 2: External Games (Chess.com & Lichess) with 1-Click Game Review -->
			{:else if activeTab === 'external-games'}
				<div class="pt-4">
					{#if !profileData.user.chessComUsername && !profileData.user.lichessUsername}
						<div class="py-16 text-center text-slate-500 text-xs flex flex-col items-center justify-center gap-3">
							<div class="w-12 h-12 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
								<Icon name="external-link" size={24} />
							</div>
							<span class="font-bold text-white text-sm">No External Accounts Linked</span>
							<p class="text-[11px] text-slate-400 max-w-md">
								Connect your Chess.com and Lichess accounts to import your match archives and run deep Stockfish game reviews on all your games in one place.
							</p>
							{#if isOwnProfile}
								<button
									class="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs rounded-xl shadow-md transition mt-1"
									on:click={() => (showLinkModal = true)}
								>
									Link Chess.com & Lichess
								</button>
							{/if}
						</div>
					{:else if loadingExternal}
						<div class="py-16 flex flex-col items-center justify-center gap-3 text-slate-400">
							<div class="w-8 h-8 border-2 border-emerald-500/30 border-t-emerald-400 rounded-full animate-spin"></div>
							<span class="text-xs font-mono">Fetching latest games from Chess.com and Lichess...</span>
						</div>
					{:else if externalGames.length === 0}
						<div class="py-16 text-center text-slate-500 text-xs flex flex-col items-center justify-center gap-2">
							<span class="font-bold text-slate-400">No recent games found for linked accounts.</span>
							<button
								class="text-xs text-sky-400 hover:underline mt-1"
								on:click={loadExternalData}
							>
								Refresh External Archives
							</button>
						</div>
					{:else}
						<div class="divide-y divide-slate-800/60">
							{#each externalGames as eg}
								{@const isChessCom = eg.platform === 'chesscom'}
								{@const platformColor = isChessCom ? 'text-[#81b64c] bg-[#81b64c]/10 border-[#81b64c]/20' : 'text-slate-300 bg-slate-800 border-slate-700'}

								<div class="py-3 flex items-center justify-between gap-4 hover:bg-slate-800/30 px-3 rounded-xl transition">
									<div class="flex items-center gap-3 min-w-0">
										<!-- Platform Badge -->
										<span class="px-2 py-1 rounded-lg text-[10px] font-bold uppercase tracking-wider border shrink-0 {platformColor}">
											{isChessCom ? 'Chess.com' : 'Lichess'}
										</span>

										<div class="min-w-0">
											<div class="text-sm font-semibold text-white flex items-center gap-2 truncate">
												<span>{eg.white}</span>
												<span class="text-[11px] font-mono text-slate-500">({eg.whiteRating || '?'})</span>
												<span class="text-xs text-slate-500 font-normal">vs</span>
												<span>{eg.black}</span>
												<span class="text-[11px] font-mono text-slate-500">({eg.blackRating || '?'})</span>
											</div>
											<div class="text-[11px] text-slate-400 mt-0.5 flex items-center gap-2">
												<span class="capitalize">{eg.timeControl}</span>
												<span>•</span>
												<span>{new Date(eg.playedAt).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })}</span>
											</div>
										</div>
									</div>

									<div class="flex items-center gap-2 shrink-0">
										<span class="px-2.5 py-1 rounded-lg text-xs font-mono font-bold bg-slate-950 border border-slate-800 text-slate-200">
											{eg.result}
										</span>

										<!-- 1-Click Game Review Button -->
										<button
											class="px-3 py-1.5 bg-[#81b64c] hover:bg-[#72a441] active:bg-[#639037] text-white font-bold text-xs rounded-lg shadow-xs transition flex items-center gap-1.5 active:translate-y-px"
											on:click={() => handleAnalyzeExternalGame(eg)}
											title="Run Stockfish Game Review on this game"
										>
											<Icon name="search" size={13} />
											<span>Review in ChessNova</span>
										</button>

										<a
											href={eg.url}
											target="_blank"
											rel="noopener noreferrer"
											class="p-1.5 bg-slate-950 hover:bg-slate-800 text-slate-400 hover:text-white border border-slate-800 rounded-lg transition"
											title="View on original platform"
										>
											<Icon name="external-link" size={13} />
										</a>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>

			<!-- Tab 3: Performance Insights -->
			{:else if activeTab === 'stats'}
				<div class="pt-6 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
					{#each ratingCategories as cat}
						{@const r = profileData.ratings?.[cat.id]}
						<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-4 flex flex-col justify-between">
							<div>
								<div class="flex items-center justify-between text-slate-400 mb-2">
									<span class="text-xs font-bold uppercase tracking-wider">{cat.label}</span>
									<Icon name={cat.icon} size={15} className="text-slate-400" />
								</div>
								<div class="text-2xl font-mono font-black text-white">
									{r?.rating || 400}
								</div>
							</div>
							<div class="mt-4 pt-3 border-t border-slate-850 text-[11px] text-slate-500 flex items-center justify-between">
								<span>Matches Played</span>
								<span class="font-bold text-slate-300">{r?.games || 0}</span>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>

<!-- Edit Profile Modal -->
{#if showEditModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-md">
		<div class="bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-2xl max-w-md w-full">
			<div class="flex items-center justify-between pb-3 border-b border-slate-800 mb-4">
				<h3 class="text-base font-bold text-white">Edit Profile</h3>
				<button class="p-1 text-slate-400 hover:text-white" on:click={() => (showEditModal = false)}>
					<Icon name="x" size={16} />
				</button>
			</div>

			<div class="space-y-4">
				<div>
					<label for="profile-bio-input" class="block text-xs font-bold text-slate-300 mb-1">Biography</label>
					<textarea
						id="profile-bio-input"
						class="w-full h-24 bg-slate-950 border border-slate-800 rounded-xl p-3 text-xs text-white focus:outline-none focus:border-sky-500 resize-none"
						maxlength="500"
						placeholder="Tell other players about your chess background, favorite openings, or goals..."
						bind:value={editBio}
					></textarea>
					<span class="text-[10px] text-slate-500">{editBio.length} / 500 characters</span>
				</div>

				<div>
					<label for="profile-country-input" class="block text-xs font-bold text-slate-300 mb-1">Country / Region</label>
					<input
						id="profile-country-input"
						type="text"
						class="w-full bg-slate-950 border border-slate-800 rounded-xl p-2.5 text-xs text-white focus:outline-none focus:border-sky-500"
						maxlength="50"
						placeholder="e.g. United States, India, Norway, Germany"
						bind:value={editCountry}
					/>
				</div>

				{#if modalError}
					<div class="p-2.5 bg-rose-950/40 border border-rose-800/40 rounded-xl text-rose-300 text-xs font-semibold">
						{modalError}
					</div>
				{/if}

				{#if modalSuccess}
					<div class="p-2.5 bg-emerald-950/40 border border-emerald-800/40 rounded-xl text-emerald-300 text-xs font-semibold">
						{modalSuccess}
					</div>
				{/if}

				<div class="flex items-center justify-end gap-2 pt-2">
					<button
						class="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-bold rounded-xl transition"
						on:click={() => (showEditModal = false)}
						disabled={savingProfile}
					>
						Cancel
					</button>
					<button
						class="px-4 py-2 bg-sky-500 hover:bg-sky-400 text-slate-950 text-xs font-bold rounded-xl transition flex items-center gap-1.5"
						on:click={handleSaveProfile}
						disabled={savingProfile}
					>
						{#if savingProfile}
							<div class="w-3.5 h-3.5 border-2 border-slate-950 border-t-transparent rounded-full animate-spin"></div>
							<span>Saving...</span>
						{:else}
							<span>Save Profile</span>
						{/if}
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}

<!-- Link External Accounts Modal -->
{#if showLinkModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-md">
		<div class="bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-2xl max-w-md w-full">
			<div class="flex items-center justify-between pb-3 border-b border-slate-800 mb-4">
				<div>
					<h3 class="text-base font-bold text-white">Link External Accounts</h3>
					<p class="text-[11px] text-slate-400">Import games and compare ratings across platforms</p>
				</div>
				<button class="p-1 text-slate-400 hover:text-white" on:click={() => (showLinkModal = false)}>
					<Icon name="x" size={16} />
				</button>
			</div>

			<div class="space-y-4">
				<div>
					<label for="link-chesscom-input" class="block text-xs font-bold text-slate-300 mb-1 flex items-center gap-1.5">
						<span class="text-[#81b64c]">♟</span>
						<span>Chess.com Username</span>
					</label>
					<input
						id="link-chesscom-input"
						type="text"
						class="w-full bg-slate-950 border border-slate-800 rounded-xl p-2.5 text-xs text-white focus:outline-none focus:border-[#81b64c]"
						placeholder="e.g. magnuscarlsen (leave blank to unlink)"
						bind:value={editChessComUser}
					/>
				</div>

				<div>
					<label for="link-lichess-input" class="block text-xs font-bold text-slate-300 mb-1 flex items-center gap-1.5">
						<span>♞</span>
						<span>Lichess Username</span>
					</label>
					<input
						id="link-lichess-input"
						type="text"
						class="w-full bg-slate-950 border border-slate-800 rounded-xl p-2.5 text-xs text-white focus:outline-none focus:border-slate-500"
						placeholder="e.g. magnuscarlsen (leave blank to unlink)"
						bind:value={editLichessUser}
					/>
				</div>

				{#if modalError}
					<div class="p-2.5 bg-rose-950/40 border border-rose-800/40 rounded-xl text-rose-300 text-xs font-semibold">
						{modalError}
					</div>
				{/if}

				{#if modalSuccess}
					<div class="p-2.5 bg-emerald-950/40 border border-emerald-800/40 rounded-xl text-emerald-300 text-xs font-semibold">
						{modalSuccess}
					</div>
				{/if}

				<div class="flex items-center justify-end gap-2 pt-2">
					<button
						class="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-bold rounded-xl transition"
						on:click={() => (showLinkModal = false)}
						disabled={linkingAccounts}
					>
						Cancel
					</button>
					<button
						class="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold rounded-xl transition flex items-center gap-1.5 shadow-md"
						on:click={handleLinkAccounts}
						disabled={linkingAccounts}
					>
						{#if linkingAccounts}
							<div class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
							<span>Verifying...</span>
						{:else}
							<span>Verify & Save</span>
						{/if}
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}
