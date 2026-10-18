<script lang="ts">
	import { PIECE_SVGS } from '$lib/chess/pieces';
	import type { Color, PieceType } from '$lib/chess/engine';
	import { createEventDispatcher } from 'svelte';

	export let color: Color = 'white';
	const dispatch = createEventDispatcher<{ select: PieceType }>();

	const promoOptions: { type: PieceType; label: string }[] = [
		{ type: 'q', label: 'Queen' },
		{ type: 'r', label: 'Rook' },
		{ type: 'b', label: 'Bishop' },
		{ type: 'n', label: 'Knight' }
	];

	function getPieceSvg(type: PieceType): string {
		const key = color === 'white' ? type.toUpperCase() : type.toLowerCase();
		return PIECE_SVGS[key] || '';
	}
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-xs p-3 animate-in fade-in duration-200">
	<div class="bg-card border border-border p-4 sm:p-5 rounded-2xl shadow-2xl flex flex-col items-center max-w-[94vw] sm:max-w-md w-full">
		<h3 class="text-xs font-mono font-bold uppercase tracking-wider text-foreground mb-3 sm:mb-4 flex items-center gap-2">
			<span class="w-1.5 h-1.5 rounded-full bg-foreground animate-pulse"></span>
			Promote Pawn
		</h3>
		
		<div class="grid grid-cols-4 gap-2 sm:gap-3 w-full max-w-xs">
			{#each promoOptions as opt}
				<button
					class="aspect-square bg-secondary hover:bg-muted border border-border hover:border-foreground/40 rounded-xl p-2 sm:p-2.5 transition-all duration-150 flex flex-col items-center justify-center hover:scale-105 active:scale-95 shadow-xs group"
					on:click={() => dispatch('select', opt.type)}
					title={opt.label}
				>
					<div class="w-full h-full flex items-center justify-center group-hover:scale-110 transition-transform piece-{color}">
						{@html getPieceSvg(opt.type)}
					</div>
				</button>
			{/each}
		</div>
	</div>
</div>

<style>
	.piece-white :global(svg) {
		filter: drop-shadow(0 1px 1.5px rgba(0, 0, 0, 0.45)) drop-shadow(0 2px 4px rgba(0, 0, 0, 0.25));
	}
	.piece-black :global(svg) {
		filter: drop-shadow(0 0 1px rgba(255, 255, 255, 0.85)) drop-shadow(0 0 2px rgba(255, 255, 255, 0.35)) drop-shadow(0 2px 4px rgba(0, 0, 0, 0.7));
	}
</style>
