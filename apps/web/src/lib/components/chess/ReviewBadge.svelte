<script lang="ts">
	import type { MoveClassification } from '$lib/chess/review';

	export let classification: MoveClassification;
	export let size: 'sm' | 'md' | 'lg' = 'md';
	export let showLabel = false;

	const config: Record<
		MoveClassification,
		{ label: string; bg: string; text: string; symbol: string; border: string }
	> = {
		brilliant: {
			label: 'Brilliant',
			bg: 'bg-foreground',
			text: 'text-background',
			symbol: '!!',
			border: 'border-foreground'
		},
		great: {
			label: 'Great Move',
			bg: 'bg-foreground/90',
			text: 'text-background',
			symbol: '!',
			border: 'border-foreground/80'
		},
		best: {
			label: 'Best Move',
			bg: 'bg-foreground text-background',
			text: 'text-background',
			symbol: '★',
			border: 'border-foreground'
		},
		excellent: {
			label: 'Excellent',
			bg: 'bg-secondary',
			text: 'text-foreground',
			symbol: '✓',
			border: 'border-border'
		},
		good: {
			label: 'Good',
			bg: 'bg-secondary',
			text: 'text-foreground',
			symbol: '✓',
			border: 'border-border'
		},
		book: {
			label: 'Book Move',
			bg: 'bg-secondary',
			text: 'text-foreground',
			symbol: 'book',
			border: 'border-border'
		},
		inaccuracy: {
			label: 'Inaccuracy',
			bg: 'bg-amber-500/15',
			text: 'text-amber-500',
			symbol: '?!',
			border: 'border-amber-500/30'
		},
		mistake: {
			label: 'Mistake',
			bg: 'bg-orange-500/15',
			text: 'text-orange-500',
			symbol: '?',
			border: 'border-orange-500/30'
		},
		miss: {
			label: 'Miss',
			bg: 'bg-rose-500/15',
			text: 'text-rose-500',
			symbol: '⨉',
			border: 'border-rose-500/30'
		},
		blunder: {
			label: 'Blunder',
			bg: 'bg-destructive',
			text: 'text-destructive-foreground',
			symbol: '??',
			border: 'border-destructive'
		},
		forced: {
			label: 'Forced',
			bg: 'bg-secondary',
			text: 'text-muted-foreground',
			symbol: '□',
			border: 'border-border'
		}
	};

	$: current = config[classification] || config.good;

	$: dimClasses =
		size === 'sm'
			? 'w-4 h-4 text-[9px]'
			: size === 'lg'
				? 'w-8 h-8 text-sm'
				: 'w-6 h-6 text-xs';
</script>

<div class="inline-flex items-center gap-1.5 select-none" title={current.label}>
	<div
		class="{dimClasses} {current.bg} {current.text} border {current.border} rounded-full shadow-xs flex items-center justify-center font-black font-sans shrink-0 leading-none"
	>
		{#if current.symbol === 'book'}
			<svg
				class={size === 'sm' ? 'w-2.5 h-2.5 fill-current' : size === 'lg' ? 'w-4 h-4 fill-current' : 'w-3 h-3 fill-current'}
				viewBox="0 0 24 24"
			>
				<path d="M19 2H6c-1.2 0-2 .8-2 2v16c0 1.2.8 2 2 2h13a1 1 0 0 0 1-1V3a1 1 0 0 0-1-1zm-1 18H6.4c-.2 0-.4-.2-.4-.4V4.4c0-.2.2-.4.4-.4H18v16z"/>
				<path d="M8 6h8v2H8zm0 4h8v2H8zm0 4h5v2H8z"/>
			</svg>
		{:else}
			<span class="tracking-tighter">{current.symbol}</span>
		{/if}
	</div>

	{#if showLabel}
		<span class="text-xs font-bold text-foreground">
			{current.label}
		</span>
	{/if}
</div>
