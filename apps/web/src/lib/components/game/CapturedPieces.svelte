<script lang="ts">
	import { fenToBoard, type Color, type PieceType } from '$lib/chess/engine';
	import { PIECE_SVGS } from '$lib/chess/pieces';

	export let fen: string;
	export let forColor: Color; // The player color we are calculating captured pieces for

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

	// Pieces of opponent that we captured
	$: capturedByUs = (() => {
		const result: PieceType[] = [];
		const oppRemaining = remaining[opponentColor];
		for (const pt of ['q', 'r', 'b', 'n', 'p'] as PieceType[]) {
			const diff = INITIAL_COUNTS[pt] - oppRemaining[pt];
			for (let i = 0; i < diff; i++) {
				result.push(pt);
			}
		}
		return result;
	})();

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

<div class="flex items-center gap-1.5 h-6 overflow-hidden">
	<div class="flex items-center -space-x-1.5">
		{#each capturedByUs as pt}
			{@const svgKey = opponentColor === 'white' ? pt.toUpperCase() : pt.toLowerCase()}
			<div class="w-4 h-4 opacity-75 inline-block">
				{@html PIECE_SVGS[svgKey]}
			</div>
		{/each}
	</div>
	{#if materialAdvantage > 0}
		<span class="text-[11px] font-mono font-extrabold text-emerald-400 bg-emerald-500/10 border border-emerald-500/30 px-1.5 py-0.2 rounded ml-1.5 shadow-sm">
			+{materialAdvantage}
		</span>
	{/if}
</div>
