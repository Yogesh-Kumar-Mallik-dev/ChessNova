<script lang="ts">
	import { cn } from '$lib/utils';
	import Icon from '$lib/components/icons/Icon.svelte';

	export let username: string = '';
	export let rating: number | string | null | undefined = null;
	export let avatar: string | null | undefined = null;
	export let ratingCategory: 'blitz' | 'rapid' | 'bullet' | 'classical' | 'puzzle' = 'blitz';
	export let size: 'sm' | 'md' | 'lg' = 'md';
	export let title: string | null | undefined = null;
	export let isOnline: boolean = true;
	export let interactive: boolean = true;
	export let href: string = '';
	export let target: string = '';
	export let rel: string = '';
	export let className: string = '';
	export let showRating: boolean = true;
	export let showAvatar: boolean = true;
	export let showStatus: boolean = true;

	let imgError = false;

	$: targetHref = href || (username ? `/profile/${encodeURIComponent(username)}` : '#');
	$: computedRel = rel || (target === '_blank' ? 'noopener noreferrer' : undefined);

	const categoryIcons: Record<string, 'flame' | 'timer' | 'zap' | 'hourglass' | 'puzzle'> = {
		blitz: 'flame',
		rapid: 'timer',
		bullet: 'zap',
		classical: 'hourglass',
		puzzle: 'puzzle'
	};

	const sizeStyles = {
		sm: {
			container: 'py-1 px-2 gap-1.5 text-xs',
			avatar: 'w-5 h-5 text-[10px] rounded-md',
			icon: 11,
			ratingPill: 'px-1.5 py-0.5 text-[10px]'
		},
		md: {
			container: 'py-1.5 px-2.5 gap-2 text-xs',
			avatar: 'w-6 h-6 text-xs rounded-lg',
			icon: 12,
			ratingPill: 'px-2 py-0.5 text-[11px]'
		},
		lg: {
			container: 'py-2 px-3 gap-2.5 text-sm',
			avatar: 'w-8 h-8 text-sm rounded-xl',
			icon: 14,
			ratingPill: 'px-2.5 py-0.5 text-xs'
		}
	};

	$: s = sizeStyles[size] || sizeStyles.md;

	function getTitleBadgeClass(t: string): string {
		const upper = t.toUpperCase();
		if (upper === 'BOT' || upper === 'AI') {
			return 'bg-secondary text-muted-foreground border-border';
		}
		if (upper === 'GM' || upper === 'WGM') {
			return 'bg-foreground text-background font-black border-transparent shadow-xs';
		}
		if (upper === 'IM' || upper === 'FM' || upper === 'CM') {
			return 'bg-secondary text-foreground font-bold border-border';
		}
		return 'bg-secondary text-foreground font-semibold border-border';
	}
</script>

{#if interactive && username}
	<a
		href={targetHref}
		target={target || undefined}
		rel={computedRel}
		class={cn(
			'group inline-flex items-center rounded-lg bg-secondary/60 hover:bg-secondary border border-border hover:border-foreground/20 shadow-xs transition-colors select-none focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring',
			s.container,
			className
		)}
		title={`View ${username}'s profile`}
	>
		<!-- Avatar / Status Indicator -->
		{#if showAvatar}
			<div class="relative shrink-0">
				<div class={cn(
					'overflow-hidden flex items-center justify-center font-bold border border-border bg-card transition-colors',
					s.avatar
				)}>
					{#if avatar && !imgError}
						<img
							src={avatar}
							alt={username}
							class="w-full h-full object-cover"
							on:error={() => (imgError = true)}
						/>
					{:else}
						<span class="text-foreground/80 group-hover:text-foreground transition-colors font-mono">
							{username ? username[0].toUpperCase() : '?'}
						</span>
					{/if}
				</div>

				{#if showStatus && isOnline}
					<span
						class="absolute -bottom-0.5 -right-0.5 w-2 h-2 rounded-full bg-emerald-500 ring-2 ring-card"
						title="Online"
					></span>
				{/if}
			</div>
		{/if}

		<!-- Optional Title Badge (e.g. GM, IM, BOT) -->
		{#if title}
			<span class={cn(
				'px-1.5 py-0.5 rounded border text-[10px] tracking-wide shrink-0',
				getTitleBadgeClass(title)
			)}>
				{title}
			</span>
		{/if}

		<!-- Username -->
		<span class="font-semibold text-foreground truncate max-w-[120px] sm:max-w-[160px]">
			{username}
		</span>

		<!-- Rating Pill -->
		{#if showRating && rating !== null && rating !== undefined}
			<div class={cn(
				'shrink-0 flex items-center gap-1 rounded border border-border bg-card text-foreground/80 font-mono font-medium',
				s.ratingPill
			)}>
				<Icon name={categoryIcons[ratingCategory] || 'flame'} size={s.icon} className="text-muted-foreground group-hover:text-foreground transition-colors" />
				<span>{rating}</span>
			</div>
		{/if}
	</a>
{:else}
	<div
		class={cn(
			'inline-flex items-center rounded-lg bg-secondary/50 border border-border shadow-xs select-none',
			s.container,
			className
		)}
	>
		<!-- Avatar / Status Indicator -->
		{#if showAvatar}
			<div class="relative shrink-0">
				<div class={cn(
					'overflow-hidden flex items-center justify-center font-bold border border-border bg-card',
					s.avatar
				)}>
					{#if avatar && !imgError}
						<img
							src={avatar}
							alt={username}
							class="w-full h-full object-cover"
							on:error={() => (imgError = true)}
						/>
					{:else}
						<span class="text-foreground/80 font-mono">
							{username ? username[0].toUpperCase() : '?'}
						</span>
					{/if}
				</div>

				{#if showStatus && isOnline}
					<span class="absolute -bottom-0.5 -right-0.5 w-2 h-2 rounded-full bg-emerald-500 ring-2 ring-card"></span>
				{/if}
			</div>
		{/if}

		<!-- Title Badge -->
		{#if title}
			<span class={cn(
				'px-1.5 py-0.5 rounded border text-[10px] tracking-wide shrink-0',
				getTitleBadgeClass(title)
			)}>
				{title}
			</span>
		{/if}

		<!-- Username -->
		<span class="font-semibold text-foreground truncate max-w-[120px] sm:max-w-[160px]">
			{username}
		</span>

		<!-- Rating Pill -->
		{#if showRating && rating !== null && rating !== undefined}
			<div class={cn(
				'shrink-0 flex items-center gap-1 rounded border border-border bg-card text-foreground/80 font-mono font-medium',
				s.ratingPill
			)}>
				<Icon name={categoryIcons[ratingCategory] || 'flame'} size={s.icon} className="text-muted-foreground" />
				<span>{rating}</span>
			</div>
		{/if}
	</div>
{/if}
