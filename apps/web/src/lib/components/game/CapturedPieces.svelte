<script lang="ts">
	import { fenToBoard, type Color, type PieceType } from '$lib/chess/engine';
	import { PIECE_SVGS } from '$lib/chess/pieces';

	export let fen: string;
	export let forColor: Color; // The player color we are calculating captured pieces for
	export let showDeficit: boolean = true; // Show -N when behind

	const INITIAL_COUNTS: Record<PieceType, number> = {
		p: 8,
		n: 2,
		b: 2,
		r: 2,
		q: 1,
		k: 1
	};

	const PIECE_VALUES: Record<PieceType, number> = {
		p: 1,
		n: 3,
		b: 3,
		r: 5,
		q: 9,
		k: 0
	};

	$: opponentColor = (forColor === 'white' ? 'black' : 'white') as Color;

	$: currentBoard = fenToBoard(fen);

	// Count remaining pieces on board
	$: remaining = (() => {
		const counts: Record<Color, Record<PieceType, number>> = {
			white: { p: 0, n: 0, b: 0, r: 0, q: 0, k: 0 },
			black: { p: 0, n: 0, b: 0, r: 0, q: 0, k: 0 }
		};
		for (let r = 0; r < 8; r++) {
			for (let f = 0; f < 8; f++) {
				const p = currentBoard[r][f];
				if (p) {
					counts[p.color][p.type]++;
				}
			}
		}
		return counts;
	})();

	// Group captured opponent pieces by piece type: Q, R, B, N, P
	$: capturedGroups = (() => {
		const groups: Array<{ type: PieceType; count: number }> = [];
		const oppRemaining = remaining[opponentColor];
		for (const pt of ['q', 'r', 'b', 'n', 'p'] as PieceType[]) {
			const diff = Math.max(0, INITIAL_COUNTS[pt] - oppRemaining[pt]);
			if (diff > 0) {
				groups.push({ type: pt, count: diff });
			}
		}
		return groups;
	})();

	$: totalCapturedCount = capturedGroups.reduce((acc, g) => acc + g.count, 0);

	// Calculate material advantage
	$: materialAdvantage = (() => {
		let myTotal = 0;
		let oppTotal = 0;
		for (const pt of ['p', 'n', 'b', 'r', 'q'] as PieceType[]) {
			myTotal += remaining[forColor][pt] * PIECE_VALUES[pt];
			oppTotal += remaining[opponentColor][pt] * PIECE_VALUES[pt];
		}
		return myTotal - oppTotal;
	})();
</script>

{#if totalCapturedCount > 0 || (materialAdvantage !== 0 && (materialAdvantage > 0 || showDeficit))}
	<div class="inline-flex items-center gap-1 sm:gap-1.5 h-6 select-none shrink-0">
		{#if totalCapturedCount > 0}
			<div class="flex items-center gap-1 bg-secondary/40 border border-border/40 px-1.5 py-0.5 rounded-md shadow-xs">
				{#each capturedGroups as group}
					<div class="flex items-center -space-x-1.5 sm:-space-x-2">
						{#each Array(group.count) as _, i}
							{@const svgKey = opponentColor === 'white' ? group.type.toUpperCase() : group.type.toLowerCase()}
							<div class="w-3.5 h-3.5 sm:w-4 sm:h-4 piece-item piece-{opponentColor} shrink-0">
								{@html PIECE_SVGS[svgKey]}
							</div>
						{/each}
					</div>
				{/each}
			</div>
		{/if}

		{#if materialAdvantage > 0}
			<span class="text-[10px] sm:text-[11px] font-mono font-bold text-foreground bg-secondary border border-border px-1.5 py-0.5 rounded shadow-xs tabular-nums shrink-0">
				+{materialAdvantage}
			</span>
		{:else if materialAdvantage < 0 && showDeficit}
			<span class="text-[10px] sm:text-[11px] font-mono font-semibold text-muted-foreground bg-secondary/60 border border-border/50 px-1.5 py-0.5 rounded tabular-nums shrink-0">
				{materialAdvantage}
			</span>
		{/if}
	</div>
{/if}

<style>
	.piece-item :global(svg) {
		width: 100%;
		height: 100%;
		display: block;
		pointer-events: none;
	}

	/* White captured piece: high contrast shadow for dark and light surfaces */
	.piece-white :global(svg) {
		filter: drop-shadow(0 1px 1.5px rgba(0, 0, 0, 0.6));
	}

	/* Black captured piece: luminous white contour rim so it is razor sharp on dark/pitch backgrounds */
	.piece-black :global(svg) {
		filter: drop-shadow(0 0 1px rgba(255, 255, 255, 0.95)) drop-shadow(0 0 2px rgba(255, 255, 255, 0.35)) drop-shadow(0 1px 2px rgba(0, 0, 0, 0.7));
	}
</style>
