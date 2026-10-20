<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Modal from '$lib/components/ui/Modal.svelte';
	import ChallengeModal from './ChallengeModal.svelte';

	export let isOpen = false;

	const dispatch = createEventDispatcher<{ close: void }>();

	type Tab = 'friends' | 'add' | 'requests' | 'challenges';
	let activeTab: Tab = 'friends';

	let friends: any[] = [];
	let incomingRequests: any[] = [];
	let outgoingRequests: any[] = [];
	let incomingChallenges: any[] = [];
	let outgoingChallenges: any[] = [];
	let loading = false;
	let error = '';
	let successMsg = '';

	// Add Friend form & search
	let addFriendUsername = '';
	let addFriendLoading = false;
	let searchResults: any[] = [];
	let isSearching = false;
	let searchTimer: any = null;

	function onSearchInput() {
		if (searchTimer) clearTimeout(searchTimer);
		const q = addFriendUsername.trim();
		if (q.length < 1) {
			searchResults = [];
			isSearching = false;
			return;
		}
		isSearching = true;
		searchTimer = setTimeout(async () => {
			try {
				searchResults = await api.users.search(q);
			} catch (err) {
				console.warn('Failed to search users:', err);
				searchResults = [];
			} finally {
				isSearching = false;
			}
		}, 250);
	}

	async function handleSendRequestTo(targetName: string) {
		addFriendLoading = true;
		error = '';
		successMsg = '';
		try {
			await api.friends.sendRequest(targetName);
			successMsg = `Friend request sent to ${targetName}!`;
			loadAll();
		} catch (e: any) {
			error = e.message || 'Failed to send friend request';
		} finally {
			addFriendLoading = false;
		}
	}

	// Target for challenging
	let challengeTarget: { id: string; username: string; rating: number; avatar: string } | null = null;
	let showChallengeModal = false;

	async function loadAll() {
		loading = true;
		error = '';
		try {
			const [fData, cData] = await Promise.all([
				api.friends.list().catch(() => ({ friends: [], incoming: [], outgoing: [] })),
				api.challenges.list().catch(() => ({ incoming: [], outgoing: [] }))
			]);
			friends = fData.friends || [];
			incomingRequests = fData.incoming || [];
			outgoingRequests = fData.outgoing || [];
			incomingChallenges = cData.incoming || [];
			outgoingChallenges = cData.outgoing || [];
		} catch (e: any) {
			error = e.message || 'Failed to load friends';
		} finally {
			loading = false;
		}
	}

	$: if (isOpen) {
		loadAll();
	}

	async function handleSendRequest() {
		if (!addFriendUsername.trim()) return;
		addFriendLoading = true;
		error = '';
		successMsg = '';
		try {
			await api.friends.sendRequest(addFriendUsername.trim());
			successMsg = `Friend request sent to ${addFriendUsername.trim()}!`;
			addFriendUsername = '';
			loadAll();
		} catch (e: any) {
			error = e.message || 'Failed to send friend request';
		} finally {
			addFriendLoading = false;
		}
	}

	async function handleRespondRequest(requestId: string, accept: boolean) {
		try {
			await api.friends.respondRequest(requestId, accept);
			loadAll();
		} catch (e: any) {
			error = e.message || 'Failed to respond to request';
		}
	}

	async function handleRemoveFriend(friendId: string) {
		if (!confirm('Are you sure you want to remove this friend?')) return;
		try {
			await api.friends.remove(friendId);
			loadAll();
		} catch (e: any) {
			error = e.message || 'Failed to remove friend';
		}
	}

	async function handleRespondChallenge(challengeId: string, accept: boolean) {
		try {
			const res = await api.challenges.respond(challengeId, accept);
			if (accept && res.gameId) {
				isOpen = false;
				goto(`/game/${res.gameId}`);
				return;
			}
			loadAll();
		} catch (e: any) {
			error = e.message || 'Failed to respond to challenge';
		}
	}

	async function handleCancelChallenge(challengeId: string) {
		try {
			await api.challenges.cancel(challengeId);
			loadAll();
		} catch (e: any) {
			error = e.message || 'Failed to cancel challenge';
		}
	}

	function openChallengeFor(f: any) {
		challengeTarget = {
			id: f.id,
			username: f.username,
			rating: f.rating || 400,
			avatar: f.avatar || ''
		};
		showChallengeModal = true;
	}
</script>

<Modal {isOpen} title="Friends & Direct Matches" on:close={() => dispatch('close')}>
	<div class="space-y-4 text-foreground">
		<!-- Tabs -->
		<div class="flex items-center gap-1 p-1 rounded-xl bg-secondary border border-border text-xs font-semibold">
			<button
				class="flex-1 py-1.5 px-2 rounded-lg transition flex items-center justify-center gap-1.5 cursor-pointer {activeTab === 'friends'
					? 'bg-foreground text-background font-bold shadow-2xs'
					: 'text-muted-foreground hover:text-foreground'}"
				on:click={() => (activeTab = 'friends')}
			>
				<Icon name="users" size={13} />
				<span>Friends</span>
				{#if friends.length > 0}
					<span class="px-1.5 py-0.2 rounded-full text-[10px] {activeTab === 'friends' ? 'bg-background text-foreground' : 'bg-card text-muted-foreground'}">
						{friends.length}
					</span>
				{/if}
			</button>

			<button
				class="flex-1 py-1.5 px-2 rounded-lg transition flex items-center justify-center gap-1.5 cursor-pointer {activeTab === 'challenges'
					? 'bg-foreground text-background font-bold shadow-2xs'
					: 'text-muted-foreground hover:text-foreground'}"
				on:click={() => (activeTab = 'challenges')}
			>
				<Icon name="swords" size={13} />
				<span>Matches</span>
				{#if incomingChallenges.length > 0}
					<span class="px-1.5 py-0.2 rounded-full bg-destructive text-destructive-foreground text-[10px] font-bold animate-pulse">
						{incomingChallenges.length}
					</span>
				{/if}
			</button>

			<button
				class="flex-1 py-1.5 px-2 rounded-lg transition flex items-center justify-center gap-1.5 cursor-pointer {activeTab === 'requests'
					? 'bg-foreground text-background font-bold shadow-2xs'
					: 'text-muted-foreground hover:text-foreground'}"
				on:click={() => (activeTab = 'requests')}
			>
				<Icon name="user-check" size={13} />
				<span>Requests</span>
				{#if incomingRequests.length > 0}
					<span class="px-1.5 py-0.2 rounded-full bg-destructive text-destructive-foreground text-[10px] font-bold">
						{incomingRequests.length}
					</span>
				{/if}
			</button>

			<button
				class="flex-1 py-1.5 px-2 rounded-lg transition flex items-center justify-center gap-1.5 cursor-pointer {activeTab === 'add'
					? 'bg-foreground text-background font-bold shadow-2xs'
					: 'text-muted-foreground hover:text-foreground'}"
				on:click={() => (activeTab = 'add')}
			>
				<Icon name="user-plus" size={13} />
				<span>Add</span>
			</button>
		</div>

		<!-- Notifications / Errors -->
		{#if error}
			<div class="p-2.5 rounded-lg bg-destructive/10 border border-destructive/30 text-destructive text-xs font-medium">
				{error}
			</div>
		{/if}
		{#if successMsg}
			<div class="p-2.5 rounded-lg bg-emerald-500/10 border border-emerald-500/30 text-emerald-500 text-xs font-medium">
				{successMsg}
			</div>
		{/if}

		<!-- Tab 1: Friends List -->
		{#if activeTab === 'friends'}
			<div class="space-y-2 max-h-72 overflow-y-auto pr-1">
				{#if loading && friends.length === 0}
					<div class="py-8 text-center text-xs text-muted-foreground animate-pulse">Loading friends...</div>
				{:else if friends.length === 0}
					<div class="py-10 text-center text-muted-foreground text-xs space-y-2">
						<div class="w-10 h-10 rounded-xl bg-secondary border border-border flex items-center justify-center mx-auto text-muted-foreground">
							<Icon name="users" size={18} />
						</div>
						<p class="font-semibold text-foreground">No friends added yet.</p>
						<p class="text-[11px] text-muted-foreground">Add players by username to challenge them to rated or casual matches anytime.</p>
						<Button variant="secondary" size="sm" className="mt-2 text-xs" on:click={() => (activeTab = 'add')}>
							<Icon name="user-plus" size={12} />
							<span>Add a Friend</span>
						</Button>
					</div>
				{:else}
					{#each friends as f}
						<div class="flex items-center justify-between p-2.5 rounded-xl bg-card border border-border hover:border-foreground/20 transition gap-2">
							<div class="flex items-center gap-2.5 min-w-0">
								<div class="w-8 h-8 rounded-lg bg-secondary border border-border flex items-center justify-center font-bold text-xs text-foreground overflow-hidden shrink-0">
									{#if f.avatar}
										<img src={f.avatar} alt={f.username} class="w-full h-full object-cover" />
									{:else}
										<Icon name="user" size={14} />
									{/if}
								</div>
								<div class="min-w-0">
									<a href="/profile/{f.username}" class="font-bold text-xs hover:underline truncate block text-foreground">
										{f.username}
									</a>
									<span class="text-[10px] font-mono text-muted-foreground">Blitz {f.rating || 400}</span>
								</div>
							</div>

							<div class="flex items-center gap-1.5 shrink-0">
								<Button variant="primary" size="sm" className="text-xs py-1 px-2.5" on:click={() => openChallengeFor(f)}>
									<Icon name="swords" size={12} />
									<span>Challenge</span>
								</Button>
								<button
									class="p-1.5 rounded-lg text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition cursor-pointer"
									on:click={() => handleRemoveFriend(f.id)}
									title="Remove friend"
								>
									<Icon name="user-x" size={13} />
								</button>
							</div>
						</div>
					{/each}
				{/if}
			</div>

		<!-- Tab 2: Challenges (Incoming & Outgoing) -->
		{:else if activeTab === 'challenges'}
			<div class="space-y-3 max-h-72 overflow-y-auto pr-1">
				<div>
					<h4 class="text-xs font-bold text-foreground mb-2 flex items-center justify-between">
						<span>Incoming Match Challenges</span>
						{#if incomingChallenges.length > 0}
							<span class="text-[10px] font-mono text-destructive">{incomingChallenges.length} pending</span>
						{/if}
					</h4>
					{#if incomingChallenges.length === 0}
						<p class="text-xs text-muted-foreground italic py-2">No incoming match challenges right now.</p>
					{:else}
						<div class="space-y-1.5">
							{#each incomingChallenges as ch}
								<div class="p-2.5 rounded-xl bg-card border-2 border-emerald-500/30 flex items-center justify-between gap-2 shadow-xs">
									<div class="min-w-0">
										<div class="flex items-center gap-1.5 text-xs font-bold text-foreground truncate">
											<Icon name="swords" size={13} className="text-emerald-500" />
											<span class="truncate">{ch.challengerUsername}</span>
											<span class="text-[10px] font-mono text-muted-foreground">({ch.challengerRating})</span>
										</div>
										<div class="text-[11px] text-muted-foreground mt-0.5 flex items-center gap-1.5">
											<span class="font-bold text-foreground">{ch.initial / 60}+{ch.increment}</span>
											<span>•</span>
											<span class="px-1.5 py-0.2 rounded text-[10px] font-semibold {ch.rated ? 'bg-primary/20 text-primary' : 'bg-secondary text-muted-foreground'}">
												{ch.rated ? 'Rated' : 'Casual'}
											</span>
											<span>•</span>
											<span class="capitalize">{ch.colorPref}</span>
										</div>
									</div>

									<div class="flex items-center gap-1.5 shrink-0">
										<Button variant="primary" size="sm" className="text-xs py-1 px-2.5 font-bold" on:click={() => handleRespondChallenge(ch.id, true)}>
											<Icon name="check" size={12} />
											<span>Accept</span>
										</Button>
										<Button variant="secondary" size="sm" className="text-xs py-1 px-2" on:click={() => handleRespondChallenge(ch.id, false)}>
											<Icon name="x" size={12} />
										</Button>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>

				<div class="pt-2 border-t border-border">
					<h4 class="text-xs font-bold text-foreground mb-2">Outgoing Sent Challenges</h4>
					{#if outgoingChallenges.length === 0}
						<p class="text-xs text-muted-foreground italic py-1">No outgoing challenges.</p>
					{:else}
						<div class="space-y-1.5">
							{#each outgoingChallenges as ch}
								<div class="p-2 rounded-xl bg-secondary/50 border border-border flex items-center justify-between text-xs">
									<div class="min-w-0 truncate">
										<span class="font-semibold text-foreground">vs {ch.challengedUsername}</span>
										<span class="text-[10px] text-muted-foreground ml-1.5">({ch.initial / 60}+{ch.increment} • {ch.rated ? 'Rated' : 'Casual'})</span>
									</div>
									<Button variant="outline" size="sm" className="text-[11px] py-0.5 px-2" on:click={() => handleCancelChallenge(ch.id)}>
										Cancel
									</Button>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</div>

		<!-- Tab 3: Requests -->
		{:else if activeTab === 'requests'}
			<div class="space-y-3 max-h-72 overflow-y-auto pr-1">
				<div>
					<h4 class="text-xs font-bold text-foreground mb-2">Incoming Friend Requests</h4>
					{#if incomingRequests.length === 0}
						<p class="text-xs text-muted-foreground italic py-2">No incoming friend requests.</p>
					{:else}
						<div class="space-y-1.5">
							{#each incomingRequests as req}
								<div class="p-2.5 rounded-xl bg-card border border-border flex items-center justify-between gap-2">
									<div class="flex items-center gap-2 min-w-0">
										<div class="w-7 h-7 rounded-lg bg-secondary border border-border flex items-center justify-center font-bold text-xs shrink-0">
											<Icon name="user" size={12} />
										</div>
										<span class="font-bold text-xs text-foreground truncate">{req.requesterUsername}</span>
									</div>
									<div class="flex items-center gap-1.5 shrink-0">
										<Button variant="primary" size="sm" className="text-xs py-1 px-2.5 font-bold" on:click={() => handleRespondRequest(req.id, true)}>
											<Icon name="user-check" size={12} />
											<span>Accept</span>
										</Button>
										<Button variant="secondary" size="sm" className="text-xs py-1 px-2" on:click={() => handleRespondRequest(req.id, false)}>
											<Icon name="x" size={12} />
										</Button>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>

				<div class="pt-2 border-t border-border">
					<h4 class="text-xs font-bold text-foreground mb-2">Outgoing Sent Requests</h4>
					{#if outgoingRequests.length === 0}
						<p class="text-xs text-muted-foreground italic py-1">No pending outgoing requests.</p>
					{:else}
						<div class="space-y-1.5">
							{#each outgoingRequests as req}
								<div class="p-2 rounded-xl bg-secondary/50 border border-border flex items-center justify-between text-xs">
									<span class="text-foreground font-semibold truncate">{req.addresseeUsername}</span>
									<span class="text-[10px] text-muted-foreground italic">Pending</span>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</div>

		<!-- Tab 4: Add Friend & User Search -->
		{:else if activeTab === 'add'}
			<div class="space-y-3 py-2">
				<div>
					<label for="friend-username-input" class="block text-xs font-bold text-foreground mb-1.5">
						Search Player by Username
					</label>
					<div class="flex gap-2">
						<div class="relative flex-1">
							<input
								id="friend-username-input"
								type="text"
								bind:value={addFriendUsername}
								on:input={onSearchInput}
								placeholder="e.g. magnuscarlsen or yogesh"
								class="w-full pl-8 pr-3 py-2 bg-secondary border border-border rounded-xl text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-foreground"
								on:keydown={(e) => e.key === 'Enter' && handleSendRequest()}
							/>
							<div class="absolute left-2.5 top-2.5 text-muted-foreground pointer-events-none">
								{#if isSearching}
									<div class="w-3.5 h-3.5 border-2 border-foreground border-t-transparent rounded-full animate-spin"></div>
								{:else}
									<Icon name="search" size={13} />
								{/if}
							</div>
						</div>

						<Button variant="primary" size="sm" className="text-xs px-4" on:click={handleSendRequest} disabled={addFriendLoading || !addFriendUsername.trim()}>
							{#if addFriendLoading}
								<span>Sending...</span>
							{:else}
								<Icon name="user-plus" size={13} />
								<span>Add</span>
							{/if}
						</Button>
					</div>
					<p class="text-[11px] text-muted-foreground mt-1.5">
						Search for any player or type their exact username to send a direct friend request.
					</p>
				</div>

				<!-- Live Search Results -->
				{#if searchResults.length > 0}
					<div class="mt-3 space-y-1.5 max-h-56 overflow-y-auto pr-1">
						<span class="text-[10px] uppercase font-bold tracking-wider text-muted-foreground block mb-1">Search Results</span>
						{#each searchResults as u}
							{@const isAlreadyFriend = friends.some((f) => f.username === u.username)}
							{@const isPendingOutgoing = outgoingRequests.some((r) => r.addresseeUsername === u.username)}
							{@const isPendingIncoming = incomingRequests.some((r) => r.requesterUsername === u.username)}

							<div class="flex items-center justify-between p-2 rounded-xl bg-card border border-border hover:border-foreground/20 transition gap-2">
								<div class="flex items-center gap-2 min-w-0">
									<div class="w-7 h-7 rounded-lg bg-secondary border border-border flex items-center justify-center font-bold text-xs text-foreground overflow-hidden shrink-0">
										{#if u.avatar}
											<img src={u.avatar} alt={u.username} class="w-full h-full object-cover" />
										{:else}
											<Icon name="user" size={13} />
										{/if}
									</div>
									<div class="min-w-0">
										<a href="/profile/{u.username}" class="font-bold text-xs hover:underline truncate block text-foreground">
											{u.username}
										</a>
										<span class="text-[10px] font-mono text-muted-foreground">Blitz {u.rating || 400}</span>
									</div>
								</div>

								<div class="flex items-center gap-1.5 shrink-0">
									{#if isAlreadyFriend}
										<span class="px-2 py-0.5 rounded text-[10px] font-semibold text-emerald-500 bg-emerald-500/10 border border-emerald-500/20">
											Friend
										</span>
									{:else if isPendingOutgoing}
										<span class="px-2 py-0.5 rounded text-[10px] font-semibold text-muted-foreground bg-secondary">
											Pending
										</span>
									{:else if isPendingIncoming}
										<Button variant="primary" size="sm" className="text-xs py-1 px-2 text-[11px]" on:click={() => handleRespondRequest(incomingRequests.find(r => r.requesterUsername === u.username).id, true)}>
											Accept
										</Button>
									{:else}
										<Button
											variant="secondary"
											size="sm"
											className="text-xs py-1 px-2.5"
											on:click={() => handleSendRequestTo(u.username)}
											disabled={addFriendLoading}
										>
											<Icon name="user-plus" size={11} />
											<span>Add</span>
										</Button>
									{/if}

									<Button
										variant="outline"
										size="sm"
										className="text-xs py-1 px-2"
										on:click={() => openChallengeFor({ id: u.id, username: u.username, rating: u.rating, avatar: u.avatar })}
										title="Direct Challenge"
									>
										<Icon name="swords" size={12} />
									</Button>
								</div>
							</div>
						{/each}
					</div>
				{:else if addFriendUsername.trim().length > 0 && !isSearching}
					<div class="py-4 text-center text-xs text-muted-foreground">
						No players found matching "{addFriendUsername.trim()}". You can still click "Add" to send a request if the username is exact.
					</div>
				{/if}
			</div>
		{/if}
	</div>
</Modal>

{#if challengeTarget}
	<ChallengeModal
		isOpen={showChallengeModal}
		opponentId={challengeTarget.id}
		opponentUsername={challengeTarget.username}
		opponentRating={challengeTarget.rating}
		opponentAvatar={challengeTarget.avatar}
		on:close={() => {
			showChallengeModal = false;
			challengeTarget = null;
			loadAll();
		}}
		on:created={() => {
			loadAll();
		}}
	/>
{/if}
