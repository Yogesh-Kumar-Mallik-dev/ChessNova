<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import MoveIndicator from './MoveIndicator.svelte';
	import Icon from '$lib/components/icons/Icon.svelte';
	import ReviewBadge from './ReviewBadge.svelte';
	import type { MoveClassification } from '$lib/chess/review';

	export let file: number;
	export let rank: number;
	export let squareName: string;
	export let isLight: boolean;
	export let isSelected = false;
	export let isLastMove = false;
	export let isLastMoveFrom = false;
	export let isLastMoveTo = false;
	export let isLegal = false;
	export let isCapture = false;
	export let isHintSquare = false;
	export let legalCategory: 'normal' | 'capture' | 'castle' | 'en_passant' = 'normal';
	export let isCheck = false;
	export let isWinningKing = false;
	export let isLosingKing = false;
	export let losingKingReason: 'checkmate' | 'resignation' | 'timeout' = 'checkmate';
	export let isDrawKing = false;
	export let reviewClassification: MoveClassification | null = null;
	export let showRankCoord = false;
	export let showFileCoord = false;
	export let theme: 'monochrome' | 'emerald' | 'wood' | 'ocean' | 'slate' | 'charcoal' | 'coral' | 'cyber' = 'monochrome';

	const dispatch = createEventDispatcher<{
		click: string;
		drop: DragEvent;
		dragover: DragEvent;
	}>();

	function handleClick() {
		dispatch('click', squareName);
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		dispatch('drop', e);
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		if (e.dataTransfer) {
			e.dataTransfer.dropEffect = 'move';
		}
		dispatch('dragover', e);
	}

	$: squareColorClass = (() => {
		switch (theme) {
			case 'monochrome':
				return isLight ? 'bg-[#f4f4f6]' : 'bg-[#30323d]';
			case 'charcoal':
				return isLight ? 'bg-[#e4e4e7]' : 'bg-[#313238]';
			case 'slate':
				return isLight ? 'bg-[#dcdde1]' : 'bg-[#404552]';
			case 'wood':
				return isLight ? 'bg-[#eedbc5]' : 'bg-[#9b7653]';
			case 'ocean':
				return isLight ? 'bg-[#dbeafe]' : 'bg-[#467499]';
			case 'coral':
				return isLight ? 'bg-[#f5e6e8]' : 'bg-[#b36675]';
			case 'cyber':
				return isLight ? 'bg-[#cdd6e0]' : 'bg-[#2b3648]';
			case 'emerald':
			default:
				return isLight ? 'bg-[#eeeed2]' : 'bg-[#69884a]';
		}
	})();

	$: coordColorClass = (() => {
		switch (theme) {
			case 'monochrome':
				return isLight ? 'text-[#30323d]' : 'text-[#f4f4f6]';
			case 'charcoal':
				return isLight ? 'text-[#313238]' : 'text-[#e4e4e7]';
			case 'slate':
				return isLight ? 'text-[#404552]' : 'text-[#dcdde1]';
			case 'wood':
				return isLight ? 'text-[#9b7653]' : 'text-[#eedbc5]';
			case 'ocean':
				return isLight ? 'text-[#467499]' : 'text-[#dbeafe]';
			case 'coral':
				return isLight ? 'text-[#b36675]' : 'text-[#f5e6e8]';
			case 'cyber':
				return isLight ? 'text-slate-600' : 'text-slate-300';
			case 'emerald':
			default:
				return isLight ? 'text-[#69884a]' : 'text-[#eeeed2]';
		}
	})();
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div
	class="relative w-full h-full flex items-center justify-center {squareColorClass} transition-colors duration-150 select-none overflow-hidden"
	on:click={handleClick}
	on:dragover={handleDragOver}
	on:drop={handleDrop}
>
	<!-- Selected square highlight: clean amber aura -->
	{#if isSelected}
		<div class="absolute inset-0 bg-amber-400/35 border-2 border-amber-300/80 z-0"></div>
	{/if}

	<!-- Tactical Hint Square highlight -->
	{#if isHintSquare}
		<div class="absolute inset-0 bg-amber-400/30 border-2 border-amber-400/80 z-0 pointer-events-none"></div>
	{/if}

	<!-- Last move highlights: non-glaring amber tones -->
	{#if isLastMoveFrom}
		<div class="absolute inset-0 bg-amber-400/20 z-0"></div>
	{:else if isLastMoveTo}
		<div class="absolute inset-0 bg-amber-400/30 border border-amber-400/40 z-0"></div>
	{:else if isLastMove}
		<div class="absolute inset-0 bg-amber-400/25 z-0"></div>
	{/if}

	<!-- CHECK / CHECKMATE SQUARE HIGHLIGHT: Circular radial glow -->
	{#if isCheck || isLosingKing}
		<div class="absolute inset-0 check-glow z-0 pointer-events-none"></div>
	{/if}

	<!-- WINNING KING BADGE: Top-right corner circular gold badge with crown -->
	{#if isWinningKing}
		<div class="absolute top-1 right-1 sm:top-1.5 sm:right-1.5 z-30 pointer-events-none">
			<div
				class="w-5 h-5 sm:w-6 sm:h-6 rounded-full bg-gradient-to-b from-[#ffcf44] to-[#f59e0b] border border-[#fde68a] shadow-[0_2px_6px_rgba(0,0,0,0.6)] flex items-center justify-center badge-pop"
				title="Winner"
			>
				<Icon name="crown" size={13} className="text-[#3a2500]" />
			</div>
		</div>
	{/if}

	<!-- LOSING KING BADGE: Top-right corner circular red badge with '#' / flag / clock -->
	{#if isLosingKing}
		<div class="absolute top-1 right-1 sm:top-1.5 sm:right-1.5 z-30 pointer-events-none">
			<div
				class="w-5 h-5 sm:w-6 sm:h-6 rounded-full bg-[#cc1a1a] border border-red-300/40 shadow-[0_2px_6px_rgba(0,0,0,0.6)] flex items-center justify-center badge-pop"
				title={losingKingReason === 'checkmate' ? 'Checkmated' : losingKingReason === 'resignation' ? 'Resigned' : 'Timed out'}
			>
				{#if losingKingReason === 'resignation'}
					<Icon name="flag" size={12} className="text-white fill-white" />
				{:else if losingKingReason === 'timeout'}
					<Icon name="clock" size={12} className="text-white" />
				{:else}
					<span class="text-white text-[11px] sm:text-[13px] font-black font-sans leading-none select-none">#</span>
				{/if}
			</div>
		</div>
	{/if}

	<!-- DRAW KING BADGE: Top-right corner circular slate badge with '½' -->
	{#if isDrawKing}
		<div class="absolute top-1 right-1 sm:top-1.5 sm:right-1.5 z-30 pointer-events-none">
			<div
				class="w-5 h-5 sm:w-6 sm:h-6 rounded-full bg-slate-600 border border-slate-400/40 shadow-[0_2px_6px_rgba(0,0,0,0.6)] flex items-center justify-center badge-pop"
				title="Draw"
			>
				<span class="text-white text-[10px] sm:text-[11px] font-bold font-sans leading-none select-none">½</span>
			</div>
		</div>
	{/if}

	<!-- REVIEW CLASSIFICATION BADGE -->
	{#if reviewClassification && !isWinningKing && !isLosingKing}
		<div class="absolute top-1 right-1 sm:top-1.5 sm:right-1.5 z-30 pointer-events-none badge-pop">
			<ReviewBadge classification={reviewClassification} size="md" />
		</div>
	{/if}

	<!-- Rank label -->
	{#if showRankCoord}
		<span class="absolute top-0.5 left-1 text-[10px] sm:text-xs font-mono font-bold pointer-events-none {coordColorClass} z-10 leading-none">
			{rank + 1}
		</span>
	{/if}

	<!-- File label -->
	{#if showFileCoord}
		<span class="absolute bottom-0.5 right-1 text-[10px] sm:text-xs font-mono font-bold pointer-events-none {coordColorClass} z-10 leading-none">
			{String.fromCharCode(97 + file)}
		</span>
	{/if}

	<!-- Piece slot -->
	<div class="relative w-full h-full z-10 flex items-center justify-center pointer-events-auto">
		<slot />
	</div>

	<!-- Legal Move reticle -->
	{#if isLegal}
		<MoveIndicator category={legalCategory} {isCapture} {isLight} />
	{/if}
</div>

<style>
	/* Radial red check glow */
	.check-glow {
		background: radial-gradient(ellipse at center, #ff0000 0%, #e70000 25%, rgba(169, 0, 0, 0) 89%, rgba(169, 0, 0, 0) 100%);
	}

	/* Spring badge pop-in */
	.badge-pop {
		animation: badgePop 280ms cubic-bezier(0.175, 0.885, 0.32, 1.275) forwards;
	}

	@keyframes badgePop {
		0% {
			transform: scale(0);
			opacity: 0;
		}
		65% {
			transform: scale(1.22);
			opacity: 1;
		}
		100% {
			transform: scale(1);
			opacity: 1;
		}
	}
</style>
