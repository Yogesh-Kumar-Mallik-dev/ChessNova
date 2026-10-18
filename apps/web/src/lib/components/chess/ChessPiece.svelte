<script lang="ts">
	import { PIECE_SVGS } from '$lib/chess/pieces';
	import type { Piece } from '$lib/chess/engine';

	export let piece: Piece;
	export let isDragging = false;
	export let draggable = true;

	$: pieceKey = piece.color === 'white' ? piece.type.toUpperCase() : piece.type.toLowerCase();
	$: svgContent = PIECE_SVGS[pieceKey] || '';
</script>

<div
	class="chess-piece-wrapper piece-{piece.color} w-full h-full flex items-center justify-center select-none transition-transform duration-75 {draggable ? 'cursor-grab active:cursor-grabbing' : 'cursor-default'} {isDragging ? 'opacity-30 scale-105' : 'hover:scale-[1.03]'}"
	{draggable}
	role="img"
	aria-label="{piece.color} {piece.type}"
	on:dragstart
	on:dragend
>
	{@html svgContent}
</div>

<style>
	.chess-piece-wrapper :global(svg) {
		width: 88%;
		height: 88%;
		display: block;
		pointer-events: none;
	}

	/* White pieces: crisp dark drop shadow for maximum depth on both light and dark squares */
	.piece-white :global(svg) {
		filter: drop-shadow(0 1px 1.5px rgba(0, 0, 0, 0.45)) drop-shadow(0 2px 4px rgba(0, 0, 0, 0.25));
	}

	/* Black pieces: crisp luminous contour rim + deep ground shadow for unmistakable silhouette clarity on dark squares */
	.piece-black :global(svg) {
		filter: drop-shadow(0 0 1px rgba(255, 255, 255, 0.85)) drop-shadow(0 0 2px rgba(255, 255, 255, 0.35)) drop-shadow(0 2px 4px rgba(0, 0, 0, 0.7));
	}
</style>
