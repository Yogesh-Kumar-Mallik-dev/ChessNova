<script lang="ts">
	import ChessBoard from '$lib/components/chess/ChessBoard.svelte';
	import MoveList from '$lib/components/game/MoveList.svelte';
	import Icon from '$lib/components/icons/Icon.svelte';
	import { api } from '$lib/api/client';
	import { INITIAL_FEN, executeMoveClient, type Color, type PieceType } from '$lib/chess/engine';
	import { soundEffects } from '$lib/audio/sounds';
	import { goto } from '$app/navigation';
	import { Chess } from 'chess.js';
	import { saveLocalGame, stageGameForReview } from '$lib/stores/localGames';

	// Game Modes
	type GameMode = 'pass_and_play' | 'vs_computer';
	let gameMode: GameMode = 'vs_computer';

	// Bot Config
	let botLevel = 2; // 1: Novice (400), 2: Casual (800), 3: Intermediate (1200), 4: Advanced (1600), 5: Master (2000)
	let playerColor: Color = 'white';
	let isBotThinking = false;

	const botLevels = [
		{ level: 1, label: 'Novice', elo: 400, desc: 'Learning basic moves' },
		{ level: 2, label: 'Casual', elo: 800, desc: 'Casual club player' },
		{ level: 3, label: 'Intermediate', elo: 1200, desc: 'Solid tactical play' },
		{ level: 4, label: 'Advanced', elo: 1600, desc: 'Strong positional play' },
		{ level: 5, label: 'Master', elo: 2000, desc: 'High depth Stockfish' }
	];

	let fen = INITIAL_FEN;
	let turn: Color = 'white';
	let orientation: Color = 'white';
	let moves: Array<{ from: string; to: string; san: string; fen: string }> = [];
	let lastMove: { from: string; to: string; san?: string; isCapture?: boolean; isPromotion?: boolean } | null = null;

	$: orientation = gameMode === 'vs_computer' ? playerColor : orientation;

	function getGamePgn(resultStr = '*'): string {
		const c = new Chess();
		for (const m of moves) {
			try {
				c.move(m.san || { from: m.from, to: m.to });
			} catch (_) {}
		}
		const whiteName = gameMode === 'vs_computer' ? (playerColor === 'white' ? 'Player' : `Stockfish Lvl ${botLevel}`) : 'White';
		const blackName = gameMode === 'vs_computer' ? (playerColor === 'black' ? 'Player' : `Stockfish Lvl ${botLevel}`) : 'Black';
		c.header(
			'Event', gameMode === 'vs_computer' ? `VS Stockfish (Lvl ${botLevel})` : 'Local Match',
			'Site', 'ChessNova',
			'Date', new Date().toISOString().slice(0, 10).replace(/-/g, '.'),
			'White', whiteName,
			'Black', blackName,
			'Result', resultStr
		);
		return c.pgn();
	}

	async function triggerBotMoveIfNeeded(currentFen: string, currentTurn: Color) {
		if (gameMode !== 'vs_computer' || currentTurn === playerColor || isBotThinking) return;

		const chk = new Chess(currentFen);
		if (chk.isGameOver()) return;

		isBotThinking = true;
		try {
			const res = await api.chess.botMove(currentFen, botLevel);
			if (res && res.valid) {
				const cap = res.isCapture || res.san?.includes('x') || false;
				const promo = res.isPromotion || res.san?.includes('=') || false;
				lastMove = { from: res.from, to: res.to, san: res.san, isCapture: cap, isPromotion: promo };
				moves = [...moves, { from: res.from, to: res.to, san: res.san, fen: res.newFen }];
				fen = res.newFen;
				turn = res.turn as Color;

				checkGameCompletion(res.newFen);
			}
		} catch (err) {
			console.error('Failed to trigger bot move:', err);
		} finally {
			isBotThinking = false;
		}
	}

	function checkGameCompletion(nextFen: string) {
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
			const whiteName = gameMode === 'vs_computer' ? (playerColor === 'white' ? 'Player' : `Stockfish Lvl ${botLevel}`) : 'White';
			const blackName = gameMode === 'vs_computer' ? (playerColor === 'black' ? 'Player' : `Stockfish Lvl ${botLevel}`) : 'Black';

			saveLocalGame({
				date: new Date().toISOString(),
				event: gameMode === 'vs_computer' ? `VS Stockfish (Lvl ${botLevel})` : 'Local Match',
				white: whiteName,
				black: blackName,
				result: resultStr,
				outcome: outcomeStr,
				winner: winnerStr,
				timeControl: gameMode === 'vs_computer' ? 'Computer Bot' : 'Pass & Play',
				movesCount: moves.length,
				pgn: fullPgn,
				startFen: INITIAL_FEN
			});
		}
	}

	async function handleMove(event: CustomEvent<{ from: string; to: string; promotion?: PieceType; isCapture?: boolean; isPromotion?: boolean }>) {
		if (gameMode === 'vs_computer' && (turn !== playerColor || isBotThinking)) {
			return;
		}

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

			checkGameCompletion(nextFen);

			if (gameMode === 'vs_computer' && nextTurn !== playerColor) {
				setTimeout(() => triggerBotMoveIfNeeded(nextFen, nextTurn), 200);
			}
		}
	}

	function handleAnalyze() {
		if (typeof window !== 'undefined' && moves.length > 0) {
			const pgn = getGamePgn('*');
			const whiteName = gameMode === 'vs_computer' ? (playerColor === 'white' ? 'Player' : `Stockfish Lvl ${botLevel}`) : 'White';
			const blackName = gameMode === 'vs_computer' ? (playerColor === 'black' ? 'Player' : `Stockfish Lvl ${botLevel}`) : 'Black';

			stageGameForReview(pgn, whiteName, blackName);
			sessionStorage.setItem('review_moves', JSON.stringify(moves));
			saveLocalGame({
				date: new Date().toISOString(),
				event: gameMode === 'vs_computer' ? `VS Stockfish (Lvl ${botLevel})` : 'Local Match',
				white: whiteName,
				black: blackName,
				result: '*',
				outcome: 'ongoing',
				timeControl: gameMode === 'vs_computer' ? 'Computer Bot' : 'Pass & Play',
				movesCount: moves.length,
				pgn,
				startFen: INITIAL_FEN
			});
		}
		goto('/analysis?review=1');
	}

	function resetLocalGame() {
		if (moves.length >= 2) {
			const pgn = getGamePgn('*');
			const whiteName = gameMode === 'vs_computer' ? (playerColor === 'white' ? 'Player' : `Stockfish Lvl ${botLevel}`) : 'White';
			const blackName = gameMode === 'vs_computer' ? (playerColor === 'black' ? 'Player' : `Stockfish Lvl ${botLevel}`) : 'Black';

			saveLocalGame({
				date: new Date().toISOString(),
				event: gameMode === 'vs_computer' ? `VS Stockfish (Lvl ${botLevel})` : 'Local Match',
				white: whiteName,
				black: blackName,
				result: '*',
				outcome: 'reset',
				timeControl: gameMode === 'vs_computer' ? 'Computer Bot' : 'Pass & Play',
				movesCount: moves.length,
				pgn,
				startFen: INITIAL_FEN
			});
		}

		fen = INITIAL_FEN;
		turn = 'white';
		moves = [];
		lastMove = null;
		isBotThinking = false;
		soundEffects.playGameStart();

		if (gameMode === 'vs_computer' && playerColor === 'black') {
			setTimeout(() => triggerBotMoveIfNeeded(INITIAL_FEN, 'white'), 400);
		}
	}

	function setGameMode(mode: GameMode) {
		gameMode = mode;
		resetLocalGame();
	}

	function setPlayerColor(color: Color) {
		playerColor = color;
		resetLocalGame();
	}

	function setBotLevel(lvl: number) {
		botLevel = lvl;
	}
</script>

<div class="flex-1 max-w-6xl mx-auto w-full p-2 sm:px-4 sm:py-2.5 flex flex-col justify-center">
	<!-- Header & Mode Switcher -->
	<div class="mb-2 sm:mb-3 flex flex-wrap items-center justify-between gap-2">
		<div class="flex items-center gap-3">
			<div class="w-8 h-8 rounded-lg bg-sky-500/15 border border-sky-500/30 flex items-center justify-center text-sky-400">
				<Icon name="swords" size={16} />
			</div>
			<div>
				<div class="flex items-center gap-2">
					<h1 class="text-lg sm:text-xl font-extrabold text-white tracking-tight">
						{gameMode === 'vs_computer' ? `Stockfish Bot (Level ${botLevel})` : 'Pass & Play'}
					</h1>
					<span class="text-[10px] font-mono px-2 py-0.5 rounded-full {gameMode === 'vs_computer' ? 'bg-indigo-500/15 text-indigo-400 border border-indigo-500/30' : 'bg-sky-500/10 text-sky-400 border border-sky-500/20'} font-bold">
						{gameMode === 'vs_computer' ? `~${botLevels[botLevel - 1].elo} ELO` : 'Local Match'}
					</span>
				</div>
				<p class="text-[11px] sm:text-xs text-slate-400">
					{gameMode === 'vs_computer' ? botLevels[botLevel - 1].desc : 'Standard FIDE rules with move reticles & check detection'}
				</p>
			</div>
		</div>

		<!-- Mode & Control Buttons -->
		<div class="flex items-center gap-2">
			<div class="inline-flex p-0.5 rounded-xl bg-slate-900 border border-slate-800 text-xs">
				<button
					class="px-2.5 py-1 rounded-lg font-medium transition {gameMode === 'vs_computer' ? 'bg-sky-500 text-slate-950 font-bold shadow-sm' : 'text-slate-400 hover:text-white'}"
					on:click={() => setGameMode('vs_computer')}
				>
					Play Bot
				</button>
				<button
					class="px-2.5 py-1 rounded-lg font-medium transition {gameMode === 'pass_and_play' ? 'bg-sky-500 text-slate-950 font-bold shadow-sm' : 'text-slate-400 hover:text-white'}"
					on:click={() => setGameMode('pass_and_play')}
				>
					Pass & Play
				</button>
			</div>

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

	<!-- Bot Level Selector Sub-bar (if vs_computer) -->
	{#if gameMode === 'vs_computer'}
		<div class="mb-3 px-3 py-2 rounded-xl bg-slate-900/60 border border-slate-800/80 flex flex-wrap items-center justify-between gap-2 text-xs">
			<div class="flex items-center gap-2">
				<span class="text-slate-400 font-medium text-[11px] uppercase tracking-wider">Difficulty:</span>
				<div class="flex items-center gap-1">
					{#each botLevels as bl}
						<button
							class="px-2 py-0.5 rounded-lg text-xs font-mono font-medium transition {botLevel === bl.level ? 'bg-indigo-600 text-white font-bold ring-1 ring-indigo-400/50' : 'bg-slate-800/80 text-slate-300 hover:bg-slate-700'}"
							on:click={() => setBotLevel(bl.level)}
						>
							Lvl {bl.level} ({bl.elo})
						</button>
					{/each}
				</div>
			</div>

			<div class="flex items-center gap-2">
				<span class="text-slate-400 font-medium text-[11px] uppercase tracking-wider">You Play:</span>
				<div class="inline-flex p-0.5 rounded-lg bg-slate-800 border border-slate-700 text-xs">
					<button
						class="px-2 py-0.5 rounded font-medium transition {playerColor === 'white' ? 'bg-white text-slate-950 font-bold' : 'text-slate-400 hover:text-white'}"
						on:click={() => setPlayerColor('white')}
					>
						White
					</button>
					<button
						class="px-2 py-0.5 rounded font-medium transition {playerColor === 'black' ? 'bg-slate-950 text-white font-bold' : 'text-slate-400 hover:text-white'}"
						on:click={() => setPlayerColor('black')}
					>
						Black
					</button>
				</div>
			</div>
		</div>
	{/if}

	<div class="grid grid-cols-1 lg:grid-cols-12 gap-4 lg:gap-6 items-center justify-center my-auto">
		<!-- Chess Board -->
		<div class="lg:col-span-7 xl:col-span-8 flex justify-center">
			<div class="w-full max-w-[min(90vw,calc(100dvh-150px),580px)] flex justify-center">
				<ChessBoard
					{fen}
					{orientation}
					{turn}
					{lastMove}
					interactive={!(gameMode === 'vs_computer' && (turn !== playerColor || isBotThinking))}
					boardSizeClass="w-full"
					on:move={handleMove}
					on:newGame={resetLocalGame}
					on:rematch={resetLocalGame}
					on:analyze={handleAnalyze}
				/>
			</div>
		</div>

		<!-- Sidebar Move List & Bot Status -->
		<div class="lg:col-span-5 xl:col-span-4 h-[380px] lg:h-[min(calc(100dvh-150px),580px)] bg-slate-900/80 border border-slate-800/80 backdrop-blur-xl rounded-2xl flex flex-col overflow-hidden shadow-2xl">
			<div class="p-3 border-b border-slate-800/80 bg-slate-950/60 text-xs font-bold text-white flex justify-between items-center">
				<div class="flex items-center gap-2">
					<span class="tracking-wide uppercase text-[10px] text-slate-400">Scorecard</span>
					{#if isBotThinking}
						<span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-indigo-500/10 text-indigo-400 text-[10px] font-mono animate-pulse">
							<span class="w-1.5 h-1.5 rounded-full bg-indigo-400"></span>
							Thinking...
						</span>
					{/if}
				</div>
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
