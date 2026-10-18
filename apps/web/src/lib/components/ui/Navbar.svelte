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

<header class="h-14 sm:h-16 bg-card/95 border-b border-border px-3 sm:px-6 flex items-center justify-between sticky top-0 z-40 select-none shadow-2xs backdrop-blur-xs">
	<div class="flex items-center gap-6 lg:gap-8">
		<!-- Brand Logo -->
		<a href="/" class="flex items-center gap-2.5 text-foreground font-bold text-base sm:text-lg tracking-tight group">
			<div class="w-8 h-8 rounded-lg bg-foreground text-background flex items-center justify-center p-1.5 shadow-xs group-hover:opacity-90 transition-opacity overflow-hidden">
				<img src="/logo.png" alt="ChessNova" class="w-full h-full object-contain invert dark:invert-0" />
			</div>
			<span class="font-extrabold tracking-tight">
				ChessNova
			</span>
		</a>

		<!-- Desktop Navigation -->
		<nav class="hidden md:flex items-center gap-1">
			{#each navItems as item}
				{@const active = currentPath === item.href || (item.href !== '/' && currentPath.startsWith(item.href) && item.href !== '/play')}
				<a
					href={item.href}
					class="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors {active
						? 'bg-secondary text-foreground font-semibold shadow-2xs'
						: 'text-muted-foreground hover:text-foreground hover:bg-secondary/60'}"
				>
					<Icon name={item.icon} size={14} className={active ? 'text-foreground' : 'text-muted-foreground'} />
					<span>{item.label}</span>
				</a>
			{/each}
		</nav>
	</div>

	<!-- Actions & Auth Status -->
	<div class="flex items-center gap-2 sm:gap-2.5">
		<QuickSettings />

		{#if $authStore.loading}
			<div class="w-8 h-8 rounded-lg bg-secondary animate-pulse"></div>
		{:else if $authStore.user}
			<div class="flex items-center gap-2 sm:gap-2.5">
				<UserBadge
					username={$authStore.user.username}
					avatar={$authStore.user.avatar}
					rating={$authStore.ratings?.blitz || 400}
					ratingCategory="blitz"
					size="sm"
					href="/profile/{$authStore.user.username}"
				/>

				<Button
					variant="ghost"
					size="icon"
					className="text-muted-foreground hover:text-foreground hover:bg-secondary h-8 w-8"
					on:click={handleLogout}
					title="Sign Out"
				>
					<Icon name="log-out" size={14} />
				</Button>
			</div>
		{:else}
			<div class="hidden sm:flex items-center gap-2">
				<a href="/login">
					<Button variant="ghost" size="sm" className="text-muted-foreground hover:text-foreground">
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
			class="md:hidden p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-secondary transition"
			on:click={() => (mobileMenuOpen = !mobileMenuOpen)}
			aria-label="Toggle navigation menu"
		>
			<Icon name={mobileMenuOpen ? 'x' : 'menu'} size={18} />
		</button>
	</div>
</header>

<!-- Mobile Navigation Drawer -->
{#if mobileMenuOpen}
	<div class="md:hidden fixed inset-x-0 top-14 sm:top-16 z-30 bg-card border-b border-border p-3 shadow-xl flex flex-col gap-1 animate-in fade-in duration-150">
		{#each navItems as item}
			{@const active = currentPath === item.href}
			<a
				href={item.href}
				class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors {active
					? 'bg-secondary text-foreground font-semibold'
					: 'text-muted-foreground hover:bg-secondary hover:text-foreground'}"
				on:click={() => (mobileMenuOpen = false)}
			>
				<Icon name={item.icon} size={16} className={active ? 'text-foreground' : 'text-muted-foreground'} />
				<span>{item.label}</span>
			</a>
		{/each}

		{#if $authStore.user}
			<div class="pt-2 mt-1 border-t border-border flex items-center justify-between gap-2">
				<div class="flex-1" on:click={() => (mobileMenuOpen = false)} role="presentation">
					<UserBadge
						username={$authStore.user.username}
						avatar={$authStore.user.avatar}
						rating={$authStore.ratings?.blitz || 400}
						ratingCategory="blitz"
						size="sm"
						href="/profile/{$authStore.user.username}"
						className="w-full justify-between"
					/>
				</div>
				<Button
					variant="ghost"
					size="sm"
					className="text-muted-foreground hover:text-foreground hover:bg-secondary shrink-0 h-8"
					on:click={() => {
						mobileMenuOpen = false;
						handleLogout();
					}}
					title="Sign Out"
				>
					<Icon name="log-out" size={14} />
				</Button>
			</div>
		{:else}
			<div class="grid grid-cols-2 gap-2 pt-2 mt-1 border-t border-border">
				<a href="/login" class="w-full" on:click={() => (mobileMenuOpen = false)}>
					<Button variant="outline" size="sm" className="w-full">Log In</Button>
				</a>
				<a href="/register" class="w-full" on:click={() => (mobileMenuOpen = false)}>
					<Button variant="primary" size="sm" className="w-full">Play Free</Button>
				</a>
			</div>
		{/if}
	</div>
{/if}
