<script lang="ts">
	import { authStore } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import Icon from '$lib/components/icons/Icon.svelte';

	function handleLogout() {
		authStore.logout();
		goto('/login');
	}
</script>

<header class="h-16 bg-[#0c1017]/90 backdrop-blur-xl border-b border-slate-800/80 px-4 sm:px-6 flex items-center justify-between sticky top-0 z-40 select-none">
	<div class="flex items-center gap-8">
		<!-- Brand Logo -->
		<a href="/" class="flex items-center gap-2.5 text-white font-black text-xl tracking-tight group">
			<div class="w-9 h-9 rounded-xl bg-gradient-to-tr from-indigo-600 to-sky-400 flex items-center justify-center shadow-[0_0_15px_rgba(56,189,248,0.4)] group-hover:shadow-[0_0_20px_rgba(56,189,248,0.6)] transition-all">
				<Icon name="crown" size={20} className="text-white drop-shadow" />
			</div>
			<span class="bg-gradient-to-r from-white via-slate-200 to-sky-300 bg-clip-text text-transparent font-extrabold">
				Chess<span class="text-sky-400">Nova</span>
			</span>
		</a>

		<!-- Navigation links -->
		<nav class="hidden md:flex items-center gap-1 text-xs font-bold uppercase tracking-wider text-slate-400">
			<a href="/play" class="flex items-center gap-1.5 px-3 py-2 rounded-lg hover:text-white hover:bg-slate-800/60 transition">
				<Icon name="swords" size={14} className="text-slate-500" />
				<span>Local</span>
			</a>
			<a href="/play/online" class="flex items-center gap-1.5 px-3 py-2 rounded-lg hover:text-white hover:bg-slate-800/60 transition">
				<Icon name="zap" size={14} className="text-sky-400" />
				<span>Online</span>
			</a>
			<a href="/puzzles" class="flex items-center gap-1.5 px-3 py-2 rounded-lg hover:text-white hover:bg-slate-800/60 transition">
				<Icon name="puzzle" size={14} className="text-emerald-400" />
				<span>Puzzles</span>
			</a>
			<a href="/analysis" class="flex items-center gap-1.5 px-3 py-2 rounded-lg hover:text-white hover:bg-slate-800/60 transition">
				<Icon name="search" size={14} className="text-indigo-400" />
				<span>Analysis</span>
			</a>
			<a href="/leaderboard" class="flex items-center gap-1.5 px-3 py-2 rounded-lg hover:text-white hover:bg-slate-800/60 transition">
				<Icon name="trophy" size={14} className="text-amber-400" />
				<span>Rankings</span>
			</a>
		</nav>
	</div>

	<!-- Auth Status -->
	<div class="flex items-center gap-3">
		{#if $authStore.loading}
			<div class="w-8 h-8 rounded-full bg-slate-800/60 animate-pulse"></div>
		{:else if $authStore.user}
			<div class="flex items-center gap-3">
				<a
					href="/profile/{$authStore.user.username}"
					class="flex items-center gap-2.5 py-1.5 px-3 rounded-xl bg-slate-800/80 hover:bg-slate-750 border border-slate-700/80 text-xs font-semibold text-slate-200 transition shadow-sm"
				>
					<span class="w-2 h-2 rounded-full bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]"></span>
					<span>{$authStore.user.username}</span>
					<span class="bg-slate-900 px-2 py-0.5 rounded-md text-[11px] font-mono text-sky-400 font-bold border border-slate-800">
						{$authStore.ratings?.blitz || 1200}
					</span>
				</a>

				<button
					class="p-2 rounded-lg text-slate-400 hover:text-rose-400 hover:bg-slate-800/60 transition"
					on:click={handleLogout}
					title="Sign Out"
				>
					<Icon name="log-out" size={16} />
				</button>
			</div>
		{:else}
			<div class="flex items-center gap-2.5 text-xs font-bold">
				<a href="/login" class="px-3.5 py-2 rounded-xl text-slate-300 hover:text-white hover:bg-slate-800/60 transition">
					Log In
				</a>
				<a
					href="/register"
					class="px-4 py-2 rounded-xl bg-gradient-to-r from-sky-500 to-indigo-600 hover:from-sky-400 hover:to-indigo-500 text-white transition shadow-[0_0_15px_rgba(56,189,248,0.3)] font-extrabold"
				>
					Join Free
				</a>
			</div>
		{/if}
	</div>
</header>
