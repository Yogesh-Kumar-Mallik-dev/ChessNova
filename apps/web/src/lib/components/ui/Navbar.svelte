<script lang="ts">
	import { authStore } from '$lib/stores/auth';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import Icon from '$lib/components/icons/Icon.svelte';
	import QuickSettings from '$lib/components/ui/QuickSettings.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import UserBadge from '$lib/components/ui/UserBadge.svelte';

	let mobileMenuOpen = false;

	function handleLogout() {
		authStore.logout();
		goto('/login');
	}

	$: currentPath = $page.url.pathname;

	const navItems: Array<{ href: string; label: string; icon: 'swords' | 'zap' | 'puzzle' | 'search' | 'trophy' }> = [
		{ href: '/play', label: 'Play', icon: 'swords' },
		{ href: '/play/online', label: 'Online', icon: 'zap' },
		{ href: '/puzzles', label: 'Puzzles', icon: 'puzzle' },
		{ href: '/analysis', label: 'Analysis', icon: 'search' },
		{ href: '/leaderboard', label: 'Leaderboard', icon: 'trophy' }
	];
</script>

<header class="h-14 sm:h-16 bg-slate-950/80 backdrop-blur-xl border-b border-slate-800/80 px-3 sm:px-6 flex items-center justify-between sticky top-0 z-40 select-none">
	<div class="flex items-center gap-6 lg:gap-8">
		<!-- Brand Logo -->
		<a href="/" class="flex items-center gap-2.5 text-white font-bold text-lg sm:text-xl tracking-tight group">
			<div class="w-9 h-9 rounded-xl bg-gradient-to-tr from-sky-500/20 to-indigo-600/30 border border-sky-500/30 flex items-center justify-center p-1 shadow-md shadow-sky-500/20 group-hover:scale-105 transition-transform overflow-hidden">
				<img src="/logo.png" alt="ChessNova" class="w-full h-full object-contain drop-shadow" />
			</div>
			<span class="font-extrabold text-white">
				Chess<span class="text-sky-400">Nova</span>
			</span>
		</a>

		<!-- Desktop Navigation -->
		<nav class="hidden md:flex items-center gap-1">
			{#each navItems as item}
				{@const active = currentPath === item.href || (item.href !== '/' && currentPath.startsWith(item.href) && item.href !== '/play')}
				<a
					href={item.href}
					class="flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold transition-all duration-150 {active
						? 'bg-slate-800 text-white shadow-sm'
						: 'text-slate-400 hover:text-slate-200 hover:bg-slate-850/60'}"
				>
					<Icon name={item.icon} size={14} className={active ? 'text-sky-400' : 'text-slate-500'} />
					<span>{item.label}</span>
				</a>
			{/each}
		</nav>
	</div>

	<!-- Actions & Auth Status -->
	<div class="flex items-center gap-2.5 sm:gap-3">
		<QuickSettings />

		{#if $authStore.loading}
			<div class="w-8 h-8 rounded-full bg-slate-800/60 animate-pulse"></div>
		{:else if $authStore.user}
			<div class="flex items-center gap-2 sm:gap-3">
				<UserBadge
					username={$authStore.user.username}
					avatar={$authStore.user.avatar}
					rating={$authStore.ratings?.blitz || 400}
					ratingCategory="blitz"
					size="md"
					href="/profile/{$authStore.user.username}"
				/>

				<Button
					variant="ghost"
					size="icon"
					className="text-slate-400 hover:text-rose-400 hover:bg-slate-850"
					on:click={handleLogout}
					title="Sign Out"
				>
					<Icon name="log-out" size={15} />
				</Button>
			</div>
		{:else}
			<div class="hidden sm:flex items-center gap-2">
				<a href="/login">
					<Button variant="ghost" size="sm" className="text-slate-300 hover:text-white">
						Log In
					</Button>
				</a>
				<a href="/register">
					<Button variant="primary" size="sm">
						Play Free
					</Button>
				</a>
			</div>
		{/if}

		<!-- Mobile Menu Toggle Button -->
		<button
			class="md:hidden p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-850 transition"
			on:click={() => (mobileMenuOpen = !mobileMenuOpen)}
			aria-label="Toggle navigation menu"
		>
			<Icon name={mobileMenuOpen ? 'x' : 'menu'} size={18} />
		</button>
	</div>
</header>

<!-- Mobile Navigation Drawer -->
{#if mobileMenuOpen}
	<div class="md:hidden fixed inset-x-0 top-16 z-30 bg-slate-950/95 border-b border-slate-800/80 backdrop-blur-2xl p-4 shadow-2xl animate-fade-in flex flex-col gap-2">
		{#each navItems as item}
			{@const active = currentPath === item.href}
			<a
				href={item.href}
				class="flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-semibold transition {active
					? 'bg-slate-800 text-white border border-slate-700/80'
					: 'text-slate-300 hover:bg-slate-900'}"
				on:click={() => (mobileMenuOpen = false)}
			>
				<Icon name={item.icon} size={18} className={active ? 'text-emerald-400' : 'text-slate-400'} />
				<span>{item.label}</span>
			</a>
		{/each}

		{#if $authStore.user}
			<div class="pt-3 mt-2 border-t border-slate-800 flex items-center justify-between gap-3">
				<div class="flex-1" on:click={() => (mobileMenuOpen = false)} role="presentation">
					<UserBadge
						username={$authStore.user.username}
						avatar={$authStore.user.avatar}
						rating={$authStore.ratings?.blitz || 400}
						ratingCategory="blitz"
						size="md"
						href="/profile/{$authStore.user.username}"
						className="w-full justify-between"
					/>
				</div>
				<Button
					variant="ghost"
					size="sm"
					className="text-slate-400 hover:text-rose-400 hover:bg-slate-900 shrink-0"
					on:click={() => {
						mobileMenuOpen = false;
						handleLogout();
					}}
					title="Sign Out"
				>
					<Icon name="log-out" size={15} />
				</Button>
			</div>
		{:else}
			<div class="grid grid-cols-2 gap-2 pt-3 mt-2 border-t border-slate-800">
				<a href="/login" class="w-full" on:click={() => (mobileMenuOpen = false)}>
					<Button variant="outline" className="w-full">Log In</Button>
				</a>
				<a href="/register" class="w-full" on:click={() => (mobileMenuOpen = false)}>
					<Button variant="primary" className="w-full">Play Free</Button>
				</a>
			</div>
		{/if}
	</div>
{/if}
