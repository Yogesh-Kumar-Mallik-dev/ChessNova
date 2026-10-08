<script lang="ts">
	import ChessBoard from '$lib/components/chess/ChessBoard.svelte';
	import MoveList from '$lib/components/game/MoveList.svelte';
	import Icon from '$lib/components/icons/Icon.svelte';
	import { api } from '$lib/api/client';
	import { INITIAL_FEN, executeMoveClient, type Color, type PieceType } from '$lib/chess/engine';
	import { soundEffects } from '$lib/audio/sounds';

	let fen = INITIAL_FEN;
	let turn: Color = 'white';
	let orientation: Color = 'white';
	let moves: Array<{ from: string; to: string; san: string; fen: string }> = [];
	let lastMove: { from: string; to: string; san?: string; isCapture?: boolean; isPromotion?: boolean } | null = null;

	import { goto } from '$app/navigation';
	import { Chess } from 'chess.js';

	async function handleMove(event: CustomEvent<{ from: string; to: string; promotion?: PieceType; isCapture?: boolean; isPromotion?: boolean }>) {
		const { from, to, promotion, isCapture, isPromotion } = event.detail;
		let nextFen = '';
		let moveSan = '';
		let nextTurn: Color = 'white';
		let cap = isCapture || false;
		let promo = isPromotion || !!promotion;

		try {
			const result = await api.chess.move(fen, from, to, promotion);
			if (result && result.valid) {
				nextFen = result.newFen;
				moveSan = result.san;
				nextTurn = result.turn as Color;
				if (result.san?.includes('x') || result.isCapture) cap = true;
				if (result.san?.includes('=') || result.isPromotion) promo = true;
			}
		} catch (e) {
			console.warn('Server move validation failed, using fallback:', e);
		}

		if (!nextFen) {
			const fallback = executeMoveClient(fen, from, to, promotion);
			if (fallback && fallback.valid) {
				nextFen = fallback.newFen;
				moveSan = fallback.san;
				nextTurn = fallback.turn;
				if (fallback.san?.includes('x') || fallback.isCapture) cap = true;
				if (fallback.san?.includes('=') || fallback.isPromotion) promo = true;
			}
		}

		if (nextFen) {
			lastMove = { from, to, san: moveSan, isCapture: cap, isPromotion: promo };
			moves = [...moves, { from, to, san: moveSan, fen: nextFen }];
			fen = nextFen;
			turn = nextTurn;
		}
	}

	function handleAnalyze() {
		if (typeof window !== 'undefined' && moves.length > 0) {
			const c = new Chess();
			for (const m of moves) {
				try {
					c.move(m.san || { from: m.from, to: m.to });
				} catch (_) {}
			}
			c.header(
				'Event', 'Local Match',
				'Site', 'ChessNova',
				'Date', new Date().toISOString().slice(0, 10).replace(/-/g, '.'),
				'White', 'White',
				'Black', 'Black'
			);
			const pgn = c.pgn();
			sessionStorage.setItem('review_pgn', pgn);
			sessionStorage.setItem('review_moves', JSON.stringify(moves));
			sessionStorage.setItem(
				'review_players',
				JSON.stringify({
					white: 'White',
					black: 'Black'
				})
			);
		}
		goto('/analysis?review=1');
	}

	function resetLocalGame() {
		fen = INITIAL_FEN;
		turn = 'white';
		moves = [];
		lastMove = null;
		soundEffects.playGameStart();
	}
</script>

<div class="flex-1 max-w-6xl mx-auto w-full p-4 sm:p-6 flex flex-col justify-center">
	<!-- Header -->
	<div class="mb-6 flex items-center justify-between">
		<div>
			<div class="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-sky-500/10 border border-sky-500/20 text-sky-400 text-xs font-semibold mb-2">
				<Icon name="swords" size={14} />
				<span>Local Match</span>
			</div>
			<h1 class="text-2xl sm:text-3xl font-extrabold text-white tracking-tight">Pass & Play</h1>
			<p class="text-xs sm:text-sm text-slate-400 mt-1">Standard FIDE rules, legal move reticles, and live check detection</p>
		</div>

		<div class="flex items-center gap-2">
			<button
				class="inline-flex items-center gap-2 px-3.5 py-2 bg-slate-900/80 hover:bg-slate-800 border border-slate-700/60 rounded-xl text-xs font-medium text-slate-200 transition shadow-sm"
				on:click={() => (orientation = orientation === 'white' ? 'black' : 'white')}
			>
				<Icon name="rotate-cw" size={14} className="text-slate-400" />
				<span>Flip</span>
			</button>
			<button
				class="inline-flex items-center gap-2 px-3.5 py-2 bg-slate-900/80 hover:bg-slate-800 border border-slate-700/60 rounded-xl text-xs font-medium text-slate-200 transition shadow-sm"
				on:click={resetLocalGame}
			>
				<Icon name="rotate-ccw" size={14} className="text-slate-400" />
				<span>Reset</span>
			</button>
		</div>
	</div>

	<div class="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start justify-center">
		<div class="lg:col-span-8 flex justify-center">
			<ChessBoard
				{fen}
				{orientation}
				{turn}
				{lastMove}
				on:move={handleMove}
				on:newGame={resetLocalGame}
				on:rematch={resetLocalGame}
				on:analyze={handleAnalyze}
			/>
		</div>

		<div class="lg:col-span-4 h-[400px] lg:h-[580px] bg-slate-900/80 border border-slate-800/80 backdrop-blur-xl rounded-2xl flex flex-col overflow-hidden shadow-2xl">
			<div class="p-3.5 border-b border-slate-800/80 bg-slate-950/60 text-xs font-bold text-white flex justify-between items-center">
				<span class="tracking-wide uppercase text-[10px] text-slate-400">Scorecard (FIDE SAN)</span>
				<div class="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-slate-800/80 border border-slate-700/60">
					<span class="w-2 h-2 rounded-full {turn === 'white' ? 'bg-white' : 'bg-slate-900 border border-slate-600'}"></span>
					<span class="text-xs font-mono font-bold text-slate-200 uppercase">{turn} to move</span>
				</div>
			</div>
			<div class="flex-1 overflow-hidden p-2">
				<MoveList {moves} />
			</div>
		</div>
	</div>
</div>
