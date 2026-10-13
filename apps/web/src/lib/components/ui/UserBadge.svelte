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
			return 'bg-purple-500/15 border-purple-500/30 text-purple-300';
		}
		if (upper === 'GM' || upper === 'WGM') {
			return 'bg-amber-500/20 border-amber-500/40 text-amber-300 font-black';
		}
		if (upper === 'IM' || upper === 'FM' || upper === 'CM') {
			return 'bg-sky-500/15 border-sky-500/30 text-sky-300';
		}
		return 'bg-amber-500/15 border-amber-500/30 text-amber-400';
	}
</script>

{#if interactive && username}
	<a
		href={targetHref}
		target={target || undefined}
		rel={computedRel}
		class={cn(
			'group inline-flex items-center rounded-xl bg-slate-900/80 hover:bg-slate-850/90 border border-slate-800 hover:border-sky-500/40 shadow-xs hover:shadow-[0_0_14px_rgba(14,165,233,0.12)] transition-all duration-150 active:scale-[0.98] select-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-400/60',
			s.container,
			className
		)}
		title={`View ${username}'s profile`}
	>
		<!-- Avatar / Status Indicator -->
		{#if showAvatar}
			<div class="relative shrink-0">
				<div class={cn(
					'overflow-hidden flex items-center justify-center font-bold shadow-inner ring-1 ring-slate-700/60 group-hover:ring-sky-400/50 bg-gradient-to-br from-slate-800 to-slate-950 transition-all',
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
						<span class="text-sky-400 group-hover:text-white transition-colors">
							{username ? username[0].toUpperCase() : '?'}
						</span>
					{/if}
				</div>

				{#if showStatus && isOnline}
					<span
						class="absolute -bottom-0.5 -right-0.5 w-2 h-2 rounded-full bg-emerald-400 ring-2 ring-slate-950 shadow-[0_0_6px_rgba(52,211,153,0.85)]"
						title="Online"
					></span>
				{/if}
			</div>
		{/if}

		<!-- Optional Title Badge (e.g. GM, IM, BOT) -->
		{#if title}
			<span class={cn(
				'px-1.5 py-0.5 rounded border font-extrabold text-[10px] tracking-wide shrink-0 transition-transform group-hover:scale-105',
				getTitleBadgeClass(title)
			)}>
				{title}
			</span>
		{/if}

		<!-- Username -->
		<span class="font-bold text-slate-200 group-hover:text-white transition-colors truncate max-w-[120px] sm:max-w-[160px]">
			{username}
		</span>

		<!-- Rating Pill -->
		{#if showRating && rating !== null && rating !== undefined}
			<div class={cn(
				'shrink-0 flex items-center gap-1 rounded-md bg-slate-950/80 border border-slate-850 text-sky-400 font-mono font-extrabold group-hover:border-slate-700 transition-colors',
				s.ratingPill
			)}>
				<Icon name={categoryIcons[ratingCategory] || 'flame'} size={s.icon} className="text-slate-500 group-hover:text-sky-400 transition-colors" />
				<span>{rating}</span>
			</div>
		{/if}
	</a>
{:else}
	<div
		class={cn(
			'inline-flex items-center rounded-xl bg-slate-900/80 border border-slate-800 shadow-xs select-none',
			s.container,
			className
		)}
	>
		<!-- Avatar / Status Indicator -->
		{#if showAvatar}
			<div class="relative shrink-0">
				<div class={cn(
					'overflow-hidden flex items-center justify-center font-bold shadow-inner ring-1 ring-slate-700/60 bg-gradient-to-br from-slate-800 to-slate-950',
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
						<span class="text-sky-400">
							{username ? username[0].toUpperCase() : '?'}
						</span>
					{/if}
				</div>

				{#if showStatus && isOnline}
					<span class="absolute -bottom-0.5 -right-0.5 w-2 h-2 rounded-full bg-emerald-400 ring-2 ring-slate-950 shadow-[0_0_6px_rgba(52,211,153,0.85)]"></span>
				{/if}
			</div>
		{/if}

		<!-- Title Badge -->
		{#if title}
			<span class={cn(
				'px-1.5 py-0.5 rounded border font-extrabold text-[10px] tracking-wide shrink-0',
				getTitleBadgeClass(title)
			)}>
				{title}
			</span>
		{/if}

		<!-- Username -->
		<span class="font-bold text-slate-200 truncate max-w-[120px] sm:max-w-[160px]">
			{username}
		</span>

		<!-- Rating Pill -->
		{#if showRating && rating !== null && rating !== undefined}
			<div class={cn(
				'shrink-0 flex items-center gap-1 rounded-md bg-slate-950/80 border border-slate-850 text-sky-400 font-mono font-extrabold',
				s.ratingPill
			)}>
				<Icon name={categoryIcons[ratingCategory] || 'flame'} size={s.icon} className="text-slate-500" />
				<span>{rating}</span>
			</div>
		{/if}
	</div>
{/if}
