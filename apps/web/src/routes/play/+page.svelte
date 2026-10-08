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
	import { saveLocalGame, stageGameForReview } from '$lib/stores/localGames';

	function getGamePgn(resultStr = '*'): string {
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
			'Black', 'Black',
			'Result', resultStr
		);
		return c.pgn();
	}

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

			// Check if game reached terminal state to archive game locally
			const chk = new Chess(nextFen);
			if (chk.isGameOver()) {
				let resultStr = '1/2-1/2';
				let outcomeStr = 'draw';
				let winnerStr = 'draw';

				if (chk.isCheckmate()) {
					outcomeStr = 'checkmate';
					if (chk.turn() === 'b') {
						resultStr = '1-0';
						winnerStr = 'white';
					} else {
						resultStr = '0-1';
						winnerStr = 'black';
					}
				} else if (chk.isStalemate()) {
					outcomeStr = 'stalemate';
				} else if (chk.isInsufficientMaterial()) {
					outcomeStr = 'insufficient_material';
				} else if (chk.isThreefoldRepetition()) {
					outcomeStr = 'threefold_repetition';
				}

				const fullPgn = getGamePgn(resultStr);
				saveLocalGame({
					date: new Date().toISOString(),
					event: 'Local Match',
					white: 'White',
					black: 'Black',
					result: resultStr,
					outcome: outcomeStr,
					winner: winnerStr,
					timeControl: 'Pass & Play',
					movesCount: moves.length,
					pgn: fullPgn,
					startFen: INITIAL_FEN
				});
			}
		}
	}

	function handleAnalyze() {
		if (typeof window !== 'undefined' && moves.length > 0) {
			const pgn = getGamePgn('*');
			stageGameForReview(pgn, 'White', 'Black');
			sessionStorage.setItem('review_moves', JSON.stringify(moves));
			saveLocalGame({
				date: new Date().toISOString(),
				event: 'Local Match',
				white: 'White',
				black: 'Black',
				result: '*',
				outcome: 'ongoing',
				timeControl: 'Pass & Play',
				movesCount: moves.length,
				pgn,
				startFen: INITIAL_FEN
			});
		}
		goto('/analysis?review=1');
	}

	function resetLocalGame() {
		// Archive previous game if it had moves before resetting
		if (moves.length >= 2) {
			const pgn = getGamePgn('*');
			saveLocalGame({
				date: new Date().toISOString(),
				event: 'Local Match',
				white: 'White',
				black: 'Black',
				result: '*',
				outcome: 'reset',
				timeControl: 'Pass & Play',
				movesCount: moves.length,
				pgn,
				startFen: INITIAL_FEN
			});
		}

		fen = INITIAL_FEN;
		turn = 'white';
		moves = [];
		lastMove = null;
		soundEffects.playGameStart();
	}

</script>

<div class="flex-1 max-w-6xl mx-auto w-full p-2 sm:px-4 sm:py-2.5 flex flex-col justify-center">
	<!-- Header -->
	<div class="mb-2 sm:mb-3 flex items-center justify-between">
		<div class="flex items-center gap-3">
			<div class="w-8 h-8 rounded-lg bg-sky-500/15 border border-sky-500/30 flex items-center justify-center text-sky-400">
				<Icon name="swords" size={16} />
			</div>
			<div>
				<div class="flex items-center gap-2">
					<h1 class="text-lg sm:text-xl font-extrabold text-white tracking-tight">Pass & Play</h1>
					<span class="hidden sm:inline-block text-[10px] font-mono px-2 py-0.5 rounded-full bg-sky-500/10 text-sky-400 border border-sky-500/20 font-bold">Local Match</span>
				</div>
				<p class="text-[11px] sm:text-xs text-slate-400">Standard FIDE rules with move reticles & check detection</p>
			</div>
		</div>

		<div class="flex items-center gap-2">
			<button
				class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-slate-900/80 hover:bg-slate-800 border border-slate-700/60 rounded-xl text-xs font-medium text-slate-200 transition shadow-sm"
				on:click={() => (orientation = orientation === 'white' ? 'black' : 'white')}
			>
				<Icon name="rotate-cw" size={13} className="text-slate-400" />
				<span>Flip</span>
			</button>
			<button
				class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-slate-900/80 hover:bg-slate-800 border border-slate-700/60 rounded-xl text-xs font-medium text-slate-200 transition shadow-sm"
				on:click={resetLocalGame}
			>
				<Icon name="rotate-ccw" size={13} className="text-slate-400" />
				<span>Reset</span>
			</button>
		</div>
	</div>

	<div class="grid grid-cols-1 lg:grid-cols-12 gap-4 lg:gap-6 items-center justify-center my-auto">
		<div class="lg:col-span-7 xl:col-span-8 flex justify-center">
			<div class="w-full max-w-[min(90vw,calc(100dvh-150px),580px)] flex justify-center">
				<ChessBoard
					{fen}
					{orientation}
					{turn}
					{lastMove}
					boardSizeClass="w-full"
					on:move={handleMove}
					on:newGame={resetLocalGame}
					on:rematch={resetLocalGame}
					on:analyze={handleAnalyze}
				/>
			</div>
		</div>

		<div class="lg:col-span-5 xl:col-span-4 h-[380px] lg:h-[min(calc(100dvh-150px),580px)] bg-slate-900/80 border border-slate-800/80 backdrop-blur-xl rounded-2xl flex flex-col overflow-hidden shadow-2xl">
			<div class="p-3 border-b border-slate-800/80 bg-slate-950/60 text-xs font-bold text-white flex justify-between items-center">
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
