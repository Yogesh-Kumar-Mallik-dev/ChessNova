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
			bg: 'bg-[#1baca6]',
			text: 'text-white',
			symbol: '!!',
			border: 'border-[#179590]'
		},
		great: {
			label: 'Great Move',
			bg: 'bg-[#5c8bb0]',
			text: 'text-white',
			symbol: '!',
			border: 'border-[#4b779a]'
		},
		best: {
			label: 'Best Move',
			bg: 'bg-[#81b64c]',
			text: 'text-white',
			symbol: '★',
			border: 'border-[#6c9c3e]'
		},
		excellent: {
			label: 'Excellent',
			bg: 'bg-[#96be76]',
			text: 'text-white',
			symbol: '✓',
			border: 'border-[#7ea560]'
		},
		good: {
			label: 'Good',
			bg: 'bg-[#96a578]',
			text: 'text-white',
			symbol: '✓',
			border: 'border-[#7e8d62]'
		},
		book: {
			label: 'Book Move',
			bg: 'bg-[#bca380]',
			text: 'text-white',
			symbol: 'book',
			border: 'border-[#a68e6e]'
		},
		inaccuracy: {
			label: 'Inaccuracy',
			bg: 'bg-[#f0c15c]',
			text: 'text-white',
			symbol: '?!',
			border: 'border-[#d6aa4d]'
		},
		mistake: {
			label: 'Mistake',
			bg: 'bg-[#e67a2e]',
			text: 'text-white',
			symbol: '?',
			border: 'border-[#c76621]'
		},
		miss: {
			label: 'Miss',
			bg: 'bg-[#e05353]',
			text: 'text-white',
			symbol: '⨉',
			border: 'border-[#c44343]'
		},
		blunder: {
			label: 'Blunder',
			bg: 'bg-[#ca3431]',
			text: 'text-white',
			symbol: '??',
			border: 'border-[#af2a27]'
		},
		forced: {
			label: 'Forced',
			bg: 'bg-[#8c8a86]',
			text: 'text-white',
			symbol: '□',
			border: 'border-[#74726f]'
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
		class="{dimClasses} {current.bg} {current.text} border {current.border} rounded-full shadow-[0_2px_5px_rgba(0,0,0,0.5)] flex items-center justify-center font-black font-sans shrink-0 leading-none"
	>
		{#if current.symbol === 'book'}
			<!-- Crisp Book SVG icon -->
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
		<span class="text-xs font-bold text-slate-200">
			{current.label}
		</span>
	{/if}
</div>
