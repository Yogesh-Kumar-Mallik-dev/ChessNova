<script lang="ts">
	import { onDestroy } from 'svelte';
	import { authStore } from '$lib/stores/auth';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { isTauriEnvironment } from '$lib/config';
	import Icon from '$lib/components/icons/Icon.svelte';
	import QuickSettings from '$lib/components/ui/QuickSettings.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import UserBadge from '$lib/components/ui/UserBadge.svelte';
	import FriendsModal from '$lib/components/friends/FriendsModal.svelte';

	let mobileMenuOpen = false;
	let showFriendsModal = false;

	$: if (typeof document !== 'undefined') {
		if (mobileMenuOpen) {
			document.body.style.overflow = 'hidden';
		} else {
			document.body.style.overflow = '';
		}
	}

	onDestroy(() => {
		if (typeof document !== 'undefined') {
			document.body.style.overflow = '';
		}
	});

	$: if ($page.url.pathname) {
		mobileMenuOpen = false;
	}

	function handleLogout() {
		authStore.logout();
		goto('/login');
	}

	$: currentPath = $page.url.pathname;

	const allNavItems: Array<{ href: string; label: string; icon: 'swords' | 'zap' | 'puzzle' | 'search' | 'trophy' | 'download' }> = [
		{ href: '/play', label: 'Play', icon: 'swords' },
		{ href: '/play/online', label: 'Online', icon: 'zap' },
		{ href: '/puzzles', label: 'Puzzles', icon: 'puzzle' },
		{ href: '/analysis', label: 'Analysis', icon: 'search' },
		{ href: '/leaderboard', label: 'Leaderboard', icon: 'trophy' },
		{ href: '/download', label: 'Desktop', icon: 'download' }
	];

	$: isDesktop = isTauriEnvironment();
	$: navItems = isDesktop ? allNavItems.filter((item) => item.href !== '/download') : allNavItems;
</script>

<header class="h-14 sm:h-16 bg-card/95 border-b border-border px-3 sm:px-6 flex items-center justify-between sticky top-0 z-40 select-none shadow-2xs backdrop-blur-xs">
	<div class="flex items-center gap-6 lg:gap-8">
		<!-- Brand Logo -->
		<a href="/" class="flex items-center gap-2.5 text-foreground font-bold text-base sm:text-lg tracking-tight group">
			<div class="w-8 h-8 rounded-lg bg-foreground text-background flex items-center justify-center p-1.5 shadow-xs group-hover:opacity-90 transition-opacity overflow-hidden">
				<img src="/logo.png" alt="ChessPasant" class="w-full h-full object-contain invert dark:invert-0" />
			</div>
			<span class="font-extrabold tracking-tight">
				ChessPasant
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
			<div class="flex items-center gap-1.5 sm:gap-2">
				<button
					class="p-1.5 sm:p-2 rounded-lg bg-secondary/80 hover:bg-secondary text-foreground transition relative cursor-pointer border border-border"
					on:click={() => (showFriendsModal = true)}
					title="Friends & Match Challenges"
				>
					<Icon name="users" size={15} />
				</button>

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
	<!-- Backdrop overlay to prevent interaction with the background page and dim background -->
	<!-- svelte-ignore a11y-click-events-have-key-events -->
	<!-- svelte-ignore a11y-no-static-element-interactions -->
	<div
		class="md:hidden fixed inset-0 top-14 sm:top-16 z-30 bg-black/60 backdrop-blur-2xs animate-in fade-in duration-150"
		on:click={() => (mobileMenuOpen = false)}
	></div>

	<div
		data-mobile-menu="true"
		class="md:hidden fixed inset-x-0 top-14 sm:top-16 z-40 bg-card border-b border-border p-3 shadow-2xl flex flex-col gap-1 animate-in fade-in slide-in-from-top-2 duration-150 max-h-[calc(100dvh-4rem)] overflow-y-auto"
	>
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
				<button
					class="p-2 rounded-lg bg-secondary text-foreground shrink-0 border border-border cursor-pointer flex items-center gap-1.5 text-xs font-semibold"
					on:click={() => {
						mobileMenuOpen = false;
						showFriendsModal = true;
					}}
					title="Friends & Challenges"
				>
					<Icon name="users" size={14} />
					<span>Friends</span>
				</button>
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

<FriendsModal isOpen={showFriendsModal} on:close={() => (showFriendsModal = false)} />
