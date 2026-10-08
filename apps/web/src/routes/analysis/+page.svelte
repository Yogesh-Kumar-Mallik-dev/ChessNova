<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { Chess } from 'chess.js';
	import ChessBoard from '$lib/components/chess/ChessBoard.svelte';
	import EvalBar from '$lib/components/chess/EvalBar.svelte';
	import MoveList from '$lib/components/game/MoveList.svelte';
	import GameReview from '$lib/components/chess/GameReview.svelte';
	import Icon from '$lib/components/icons/Icon.svelte';
	import { api } from '$lib/api/client';
	import { INITIAL_FEN, executeMoveClient, type Color, type PieceType } from '$lib/chess/engine';
	import type { GameReviewResult } from '$lib/chess/review';
	import { soundEffects } from '$lib/audio/sounds';
	import {
		getLocalGames,
		saveLocalGame,
		deleteLocalGame,
		clearAllLocalGames,
		type LocalGameRecord
	} from '$lib/stores/localGames';

	let startFen = $page.url.searchParams.get('fen') || INITIAL_FEN;
	let fen = startFen;
	let customFenInput = fen;
	let customPgnInput = '';
	let orientation: Color = 'white';
	let turn: Color = 'white';
	let moves: Array<{ from: string; to: string; san: string; fen: string }> = [];
	let currentPly = 0;
	let lastMove: { from: string; to: string; san?: string; isCapture?: boolean; isPromotion?: boolean } | null = null;

	let whitePlayerName = 'White';
	let blackPlayerName = 'Black';
	let rightPanelTab: 'review' | 'tools' | 'history' = 'review';
	let reviewResult: GameReviewResult | null = null;
	let showPgnModal = false;
	let copiedPgn = false;
	let currentPgn = '';
	let localGamesList: LocalGameRecord[] = [];


	function loadPgnContent(pgnText: string) {
		if (!pgnText || !pgnText.trim()) return;
		try {
			const c = new Chess();
			c.loadPgn(pgnText.trim());
			const history = c.history({ verbose: true });
			const headers = c.header();

			whitePlayerName = headers['White'] || 'White';
			blackPlayerName = headers['Black'] || 'Black';

			const replayChess = new Chess();
			if (headers['FEN']) {
				try {
					replayChess.load(headers['FEN']);
					startFen = headers['FEN'];
				} catch (_) {
					startFen = INITIAL_FEN;
				}
			} else {
				startFen = INITIAL_FEN;
			}

			const parsedMoves: Array<{ from: string; to: string; san: string; fen: string }> = [];
			for (const h of history) {
				replayChess.move({ from: h.from, to: h.to, promotion: h.promotion });
				parsedMoves.push({
					from: h.from,
					to: h.to,
					san: h.san,
					fen: replayChess.fen()
				});
			}

			moves = parsedMoves;
			currentPly = parsedMoves.length;
			fen = replayChess.fen();
			lastMove =
				parsedMoves.length > 0
					? {
							from: parsedMoves[parsedMoves.length - 1].from,
							to: parsedMoves[parsedMoves.length - 1].to,
							san: parsedMoves[parsedMoves.length - 1].san,
							isCapture: parsedMoves[parsedMoves.length - 1].san?.includes('x')
						}
					: null;
			turn = replayChess.turn() === 'w' ? 'white' : 'black';
			currentPgn = pgnText.trim();
			customPgnInput = currentPgn;
			reviewResult = null;
			rightPanelTab = 'review';
			showPgnModal = false;
		} catch (err) {
			console.error('Failed to parse PGN:', err);
		}
	}

	onMount(async () => {
		if (typeof window !== 'undefined') {
			const pgnParam = $page.url.searchParams.get('pgn');
			const gameIdParam = $page.url.searchParams.get('gameId');
			const savedPgn = sessionStorage.getItem('review_pgn');

			if (pgnParam) {
				loadPgnContent(decodeURIComponent(pgnParam));
			} else if (savedPgn) {
				loadPgnContent(savedPgn);
				sessionStorage.removeItem('review_pgn');
			} else if (gameIdParam) {
				try {
					const fetchedPgn = await api.games.getPgn(gameIdParam);
					if (fetchedPgn) {
						loadPgnContent(fetchedPgn);
					}
				} catch (e) {
					console.error('Failed to load game PGN:', e);
				}
			} else {
				const savedMoves = sessionStorage.getItem('review_moves');
				const savedPlayers = sessionStorage.getItem('review_players');

				if (savedMoves) {
					try {
						moves = JSON.parse(savedMoves);
						currentPly = moves.length;
						if (moves.length > 0) {
							fen = moves[moves.length - 1].fen;
							lastMove = { from: moves[moves.length - 1].from, to: moves[moves.length - 1].to, san: moves[moves.length - 1].san, isCapture: moves[moves.length - 1].san?.includes('x') };
							const parts = fen.split(' ');
							turn = parts.length > 1 && parts[1] === 'b' ? 'black' : 'white';
						}
					} catch (e) {
						console.error('Failed to parse saved review moves:', e);
					}
					sessionStorage.removeItem('review_moves');
				}

				if (savedPlayers) {
					try {
						const p = JSON.parse(savedPlayers);
						whitePlayerName = p.white || 'White';
						blackPlayerName = p.black || 'Black';
					} catch (e) {
						console.error('Failed to parse saved players:', e);
					}
					sessionStorage.removeItem('review_players');
				}
			}

			refreshLocalGames();

			// If URL has review param, default to review tab
			if ($page.url.searchParams.get('review') === '1' || moves.length > 0 || currentPgn) {
				rightPanelTab = 'review';
			} else {
				rightPanelTab = 'tools';
			}
		}
	});


	$: if (!currentPgn && moves.length > 0) {
		const c = new Chess();
		for (const m of moves) {
			try {
				c.move(m.san || { from: m.from, to: m.to });
			} catch (_) {}
		}
		c.header(
			'White', whitePlayerName,
			'Black', blackPlayerName,
			'Event', 'Game Analysis',
			'Site', 'ChessNova',
			'Date', new Date().toISOString().slice(0, 10).replace(/-/g, '.')
		);
		currentPgn = c.pgn();
	}

	$: currentScoreCp = (() => {
		if (reviewResult && reviewResult.moves.length > 0) {
			if (currentPly === 0) {
				return reviewResult.evalGraph?.[0]?.scoreCp || 0;
			}
			if (currentPly <= reviewResult.moves.length) {
				return reviewResult.moves[currentPly - 1].evalAfter;
			}
		}
		return 0;
	})();

	$: activeReviewMove =
		reviewResult && currentPly > 0 && currentPly <= reviewResult.moves.length
			? {
					from: reviewResult.moves[currentPly - 1].from,
					to: reviewResult.moves[currentPly - 1].to,
					classification: reviewResult.moves[currentPly - 1].classification,
					bestMoveUci: reviewResult.moves[currentPly - 1].bestMoveUci
				}
			: null;

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
			console.warn('Server move validation failed in analysis, using fallback:', e);
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
			moves = [...moves.slice(0, currentPly), { from, to, san: moveSan, fen: nextFen }];
			currentPly = moves.length;
			fen = nextFen;
			customFenInput = nextFen;
			turn = nextTurn;
			reviewResult = null;
		}
	}

	function goToPly(ply: number) {
		if (ply < 0) ply = 0;
		if (ply > moves.length) ply = moves.length;
		currentPly = ply;

		if (ply === 0) {
			fen = startFen;
			lastMove = null;
			const parts = startFen.split(' ');
			turn = parts.length > 1 && parts[1] === 'b' ? 'black' : 'white';
		} else {
			const m = moves[ply - 1];
			fen = m.fen;
			lastMove = { from: m.from, to: m.to };
			const parts = m.fen.split(' ');
			turn = parts.length > 1 && parts[1] === 'b' ? 'black' : 'white';
		}
		customFenInput = fen;
	}

	function loadCustomFen() {
		if (customFenInput.trim()) {
			startFen = customFenInput.trim();
			fen = startFen;
			moves = [];
			currentPly = 0;
			lastMove = null;
			reviewResult = null;
			const parts = fen.split(' ');
			if (parts.length > 1) {
				turn = parts[1] === 'w' ? 'white' : 'black';
			}
		}
	}

	function loadPgn() {
		loadPgnContent(customPgnInput);
	}

	function copyPgnText() {
		if (currentPgn) {
			navigator.clipboard.writeText(currentPgn);
			copiedPgn = true;
			setTimeout(() => (copiedPgn = false), 2000);
		}
	}

	function downloadPgnFile() {
		if (!currentPgn) return;
		const blob = new Blob([currentPgn], { type: 'application/x-chess-pgn;charset=utf-8' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `${whitePlayerName}_vs_${blackPlayerName}_analysis.pgn`;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		URL.revokeObjectURL(url);
	}

	function refreshLocalGames() {
		localGamesList = getLocalGames();
	}

	function handleLoadSavedGame(rec: LocalGameRecord) {
		loadPgnContent(rec.pgn);
		rightPanelTab = 'review';
	}

	function handleDeleteSavedGame(id: string) {
		deleteLocalGame(id);
		refreshLocalGames();
	}

	function handleClearAllSavedGames() {
		if (confirm('Clear all locally saved match history?')) {
			clearAllLocalGames();
			refreshLocalGames();
		}
	}

	function resetToStart() {
		startFen = INITIAL_FEN;
		fen = INITIAL_FEN;
		customFenInput = INITIAL_FEN;
		moves = [];
		currentPly = 0;
		turn = 'white';
		lastMove = null;
		reviewResult = null;
		currentPgn = '';
		customPgnInput = '';
		soundEffects.playGameStart();
	}


	function handleKeydown(e: KeyboardEvent) {
		if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
			return;
		}
		if (e.key === 'ArrowLeft') {
			e.preventDefault();
			goToPly(currentPly - 1);
		} else if (e.key === 'ArrowRight') {
			e.preventDefault();
			goToPly(currentPly + 1);
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="flex-1 max-w-7xl mx-auto w-full p-2 sm:px-4 sm:py-2.5 flex flex-col justify-center select-none">
	<!-- Header -->
	<div class="mb-2 sm:mb-3 flex items-center justify-between gap-3">
		<div class="flex items-center gap-3">
			<div class="w-8 h-8 rounded-lg bg-emerald-500/15 border border-emerald-500/30 flex items-center justify-center text-[#81b64c]">
				<Icon name="search" size={16} />
			</div>
			<div>
				<div class="flex items-center gap-2">
					<h1 class="text-base sm:text-lg font-extrabold text-white tracking-tight">Stockfish Review Laboratory</h1>
					<span class="hidden md:inline-block text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/10 text-[#81b64c] border border-emerald-500/20 font-bold">Chess.com Review & CAPS</span>
				</div>
				<p class="text-[11px] text-stone-400 hidden sm:block">Coach evaluation, move classifications, eval bar, and local archives</p>
			</div>
		</div>

		<div class="flex items-center gap-1.5 sm:gap-2 flex-wrap justify-end">
			{#if currentPgn || moves.length > 0}
				<button
					class="inline-flex items-center gap-1 px-2.5 py-1.5 bg-stone-900/90 hover:bg-stone-800 border border-stone-700/60 rounded-xl text-xs font-bold text-stone-200 transition shadow-sm"
					on:click={copyPgnText}
					title="Copy Game PGN"
				>
					<Icon name={copiedPgn ? 'check' : 'download'} size={13} className={copiedPgn ? 'text-emerald-400' : 'text-stone-400'} />
					<span>{copiedPgn ? 'Copied' : 'Copy'}</span>
				</button>
				<button
					class="inline-flex items-center gap-1 px-2.5 py-1.5 bg-stone-900/90 hover:bg-stone-800 border border-stone-700/60 rounded-xl text-xs font-bold text-stone-200 transition shadow-sm"
					on:click={downloadPgnFile}
					title="Download .pgn file"
				>
					<Icon name="download" size={13} className="text-stone-400" />
					<span>PGN</span>
				</button>
			{/if}

			<button
				class="inline-flex items-center gap-1.5 px-2.5 py-1.5 bg-stone-900/90 hover:bg-stone-800 border border-stone-700/60 rounded-xl text-xs font-bold text-stone-200 transition shadow-sm"
				on:click={() => (showPgnModal = true)}
			>
				<Icon name="download" size={13} className="text-[#81b64c]" />
				<span>Import</span>
			</button>
			<button
				class="inline-flex items-center gap-1.5 px-2.5 py-1.5 bg-stone-900/90 hover:bg-stone-800 border border-stone-700/60 rounded-xl text-xs font-bold text-stone-200 transition shadow-sm"
				on:click={() => (orientation = orientation === 'white' ? 'black' : 'white')}
			>
				<Icon name="rotate-cw" size={13} className="text-stone-400" />
				<span>Flip</span>
			</button>
			<button
				class="inline-flex items-center gap-1.5 px-2.5 py-1.5 bg-stone-900/90 hover:bg-stone-800 border border-stone-700/60 rounded-xl text-xs font-bold text-stone-200 transition shadow-sm"
				on:click={resetToStart}
			>
				<Icon name="rotate-ccw" size={13} className="text-stone-400" />
				<span>Reset</span>
			</button>
		</div>
	</div>

	<div class="grid grid-cols-1 lg:grid-cols-12 gap-3 lg:gap-5 items-center justify-center my-auto">
		<!-- Left: Board with Eval Bar and Best Move Arrow -->
		<div class="lg:col-span-7 xl:col-span-8 flex justify-center items-center gap-2 sm:gap-3">
			<div class="hidden sm:flex h-[min(88vw,calc(100dvh-150px),580px)]">
				<EvalBar scoreCp={currentScoreCp} {orientation} />
			</div>
			<div class="shrink-0 flex justify-center w-full max-w-[min(88vw,calc(100dvh-150px),580px)]">
				<ChessBoard
					{fen}
					{orientation}
					{turn}
					{lastMove}
					boardSizeClass="w-full"
					reviewMove={activeReviewMove}
					on:move={handleMove}
				/>
			</div>
		</div>

		<!-- Right: Game Review / Tools Panel -->
		<div class="lg:col-span-5 xl:col-span-4 h-[420px] lg:h-[min(calc(100dvh-150px),620px)] flex flex-col">
			<!-- Panel Tab Selector -->
			<div class="flex items-center p-1 bg-[#1e1d1b] border border-[#3d3b37] rounded-xl mb-2.5 shrink-0">
				<button
					class="flex-1 py-1.5 px-2.5 rounded-lg text-xs font-extrabold transition flex items-center justify-center gap-1.5 {rightPanelTab === 'review'
						? 'bg-[#81b64c] text-white shadow'
						: 'text-stone-400 hover:text-white'}"
					on:click={() => (rightPanelTab = 'review')}
				>
					<Icon name="search" size={14} />
					<span>Review</span>
				</button>

				<button
					class="flex-1 py-1.5 px-2.5 rounded-lg text-xs font-extrabold transition flex items-center justify-center gap-1.5 {rightPanelTab === 'history'
						? 'bg-[#383633] text-white shadow border border-[#484643]'
						: 'text-stone-400 hover:text-white'}"
					on:click={() => {
						refreshLocalGames();
						rightPanelTab = 'history';
					}}
				>
					<Icon name="clock" size={14} />
					<span>Past Games ({localGamesList.length})</span>
				</button>

				<button
					class="flex-1 py-1.5 px-2.5 rounded-lg text-xs font-extrabold transition flex items-center justify-center gap-1.5 {rightPanelTab === 'tools'
						? 'bg-[#383633] text-white shadow border border-[#484643]'
						: 'text-stone-400 hover:text-white'}"
					on:click={() => (rightPanelTab = 'tools')}
				>
					<Icon name="settings" size={14} />
					<span>Scorecard</span>
				</button>
			</div>

			<!-- Main Panel Content -->
			<div class="flex-1 overflow-hidden">
				{#if rightPanelTab === 'review'}
					<!-- Chess.com Game Review Panel -->
					<GameReview
						bind:reviewResult
						pgn={currentPgn}
						{moves}
						{whitePlayerName}
						{blackPlayerName}
						{currentPly}
						on:selectPly={(e) => goToPly(e.detail)}
					/>
				{:else if rightPanelTab === 'history'}
					<!-- Local Past Matches Archive for Guests and Users -->
					<div class="bg-[#262522] border border-[#3d3b37] rounded-2xl p-4 shadow-2xl flex flex-col h-full overflow-hidden">
						<div class="flex items-center justify-between pb-3 border-b border-[#3d3b37] shrink-0">
							<div>
								<h3 class="text-xs font-bold text-white uppercase tracking-wider">Local Match Archive</h3>
								<p class="text-[11px] text-stone-400">Offline & guest match history saved in browser</p>
							</div>
							{#if localGamesList.length > 0}
								<button
									class="px-2 py-1 bg-stone-800 hover:bg-rose-900/60 hover:text-rose-200 text-stone-400 text-[11px] font-bold rounded-lg transition border border-stone-700"
									on:click={handleClearAllSavedGames}
								>
									Clear All
								</button>
							{/if}
						</div>

						<div class="flex-1 overflow-y-auto py-2 flex flex-col gap-2">
							{#if localGamesList.length === 0}
								<div class="flex flex-col items-center justify-center h-full text-center py-12 px-4 gap-2">
									<div class="w-10 h-10 rounded-xl bg-stone-800 flex items-center justify-center text-stone-500">
										<Icon name="clock" size={20} />
									</div>
									<span class="text-xs font-bold text-stone-300">No Saved Games Yet</span>
									<p class="text-[11px] text-stone-500 max-w-[200px]">
										Play online or Pass & Play matches to build your local review history.
									</p>
								</div>
							{:else}
								{#each localGamesList as rec}
									{@const won = rec.result === '1-0'}
									{@const lost = rec.result === '0-1'}
									{@const draw = rec.result === '1/2-1/2'}
									{@const resClass = draw
										? 'text-amber-400 bg-amber-500/10 border-amber-500/20'
										: won
											? 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20'
											: lost
												? 'text-rose-400 bg-rose-500/10 border-rose-500/20'
												: 'text-stone-300 bg-stone-800 border-stone-700'}

									<div class="p-3 bg-[#1e1d1b] border border-[#3d3b37] hover:border-stone-500 rounded-xl transition flex items-center justify-between gap-3 group">
										<div class="min-w-0 flex-1">
											<div class="flex items-center gap-2 mb-1">
												<span class="px-1.5 py-0.5 rounded text-[10px] font-mono font-bold border {resClass}">
													{rec.result}
												</span>
												<span class="text-[11px] font-bold text-white truncate">
													{rec.white} vs {rec.black}
												</span>
											</div>
											<div class="flex items-center gap-2 text-[10px] text-stone-400">
												<span>{rec.event}</span>
												<span>•</span>
												<span>{rec.movesCount} plies</span>
												<span>•</span>
												<span>{new Date(rec.timestamp).toLocaleDateString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}</span>
											</div>
										</div>

										<div class="flex items-center gap-1.5 shrink-0">
											<button
												class="px-2.5 py-1.5 bg-[#81b64c] hover:bg-[#91c65d] active:bg-[#73a443] text-white font-extrabold text-[11px] uppercase tracking-wider rounded-lg shadow transition flex items-center gap-1"
												on:click={() => handleLoadSavedGame(rec)}
												title="Review this game"
											>
												<Icon name="search" size={12} />
												<span>Review</span>
											</button>

											<button
												class="p-1.5 text-stone-500 hover:text-rose-400 hover:bg-stone-800 rounded-lg transition"
												on:click={() => handleDeleteSavedGame(rec.id)}
												title="Delete game"
											>
												<Icon name="x" size={13} />
											</button>

										</div>
									</div>
								{/each}
							{/if}
						</div>
					</div>
				{:else}
					<!-- Tools, FEN & Scorecard Panel -->
					<div class="bg-[#262522] border border-[#3d3b37] rounded-2xl p-4 shadow-2xl flex flex-col gap-3.5 h-full">
						<!-- FEN Input -->
						<div>
							<label for="fen-input" class="block text-[10px] font-bold uppercase tracking-wider text-stone-400 mb-1">
								FEN Position
							</label>
							<div class="flex gap-2">
								<input
									id="fen-input"
									type="text"
									bind:value={customFenInput}
									class="flex-1 px-3 py-1.5 bg-[#1b1a18] border border-[#3d3b37] rounded-xl text-xs text-white font-mono focus:outline-none focus:border-[#81b64c] transition placeholder:text-stone-600"
									placeholder="Paste FEN..."
								/>
								<button
									class="px-3 py-1.5 bg-[#383633] hover:bg-[#484643] text-stone-200 font-bold text-xs rounded-xl border border-[#484643] transition"
									on:click={loadCustomFen}
								>
									Load
								</button>
							</div>
						</div>

						<!-- Move History List -->
						<div class="flex-1 bg-[#1b1a18] border border-[#3d3b37] rounded-xl overflow-hidden flex flex-col">
							<div class="p-2.5 border-b border-[#3d3b37] text-xs font-bold text-stone-400 flex justify-between items-center bg-[#21201d]">
								<span class="tracking-wide uppercase text-[10px]">Scorecard (FIDE SAN)</span>
								<button class="text-xs text-stone-500 hover:text-stone-300 transition" on:click={resetToStart}>
									Clear
								</button>
							</div>
							<div class="flex-1 overflow-hidden p-2">
								<MoveList {moves} activePly={currentPly} on:selectPly={(e) => goToPly(e.detail)} />
							</div>
						</div>

						<!-- Navigation Buttons -->
						<div class="grid grid-cols-4 gap-2 pt-2 border-t border-[#3d3b37]">
							<button
								class="py-2 flex items-center justify-center bg-[#383633] hover:bg-[#484643] text-stone-300 rounded-xl transition"
								on:click={() => goToPly(0)}
								title="Initial Position"
							>
								<Icon name="chevrons-left" size={16} />
							</button>
							<button
								class="py-2 flex items-center justify-center bg-[#383633] hover:bg-[#484643] text-stone-300 rounded-xl transition"
								on:click={() => goToPly(currentPly - 1)}
								title="Previous Move"
							>
								<Icon name="chevron-left" size={16} />
							</button>
							<button
								class="py-2 flex items-center justify-center bg-[#383633] hover:bg-[#484643] text-stone-300 rounded-xl transition"
								on:click={() => goToPly(currentPly + 1)}
								title="Next Move"
							>
								<Icon name="chevron-right" size={16} />
							</button>
							<button
								class="py-2 flex items-center justify-center bg-[#383633] hover:bg-[#484643] text-stone-300 rounded-xl transition"
								on:click={() => goToPly(moves.length)}
								title="Latest Move"
							>
								<Icon name="chevrons-right" size={16} />
							</button>
						</div>
					</div>
				{/if}
			</div>

		</div>
	</div>
</div>

<!-- Import PGN Modal -->
{#if showPgnModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4 animate-in fade-in">
		<div class="bg-[#262522] border border-[#3d3b37] rounded-2xl p-5 max-w-md w-full shadow-2xl flex flex-col gap-3">
			<div class="flex items-center justify-between border-b border-[#3d3b37] pb-2">
				<div class="flex items-center gap-2">
					<Icon name="download" size={16} className="text-[#81b64c]" />
					<h3 class="text-sm font-bold text-white">Import Game PGN</h3>
				</div>
				<button class="text-stone-400 hover:text-white" on:click={() => (showPgnModal = false)}>
					<Icon name="x" size={16} />
				</button>
			</div>

			<p class="text-xs text-stone-400">
				Paste any chess game PGN below to run full Stockfish Game Review with accuracy scores and coaching.
			</p>

			<textarea
				bind:value={customPgnInput}
				class="w-full h-36 bg-[#1b1a18] border border-[#3d3b37] rounded-xl p-3 text-xs text-white font-mono focus:outline-none focus:border-[#81b64c] transition"
				placeholder="1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. c3 Nf6..."
			></textarea>

			<div class="flex items-center justify-end gap-2 pt-2">
				<button
					class="px-4 py-2 bg-[#383633] hover:bg-[#484643] text-stone-300 font-bold text-xs rounded-xl transition"
					on:click={() => (showPgnModal = false)}
				>
					Cancel
				</button>
				<button
					class="px-4 py-2 bg-[#81b64c] hover:bg-[#91c65d] active:bg-[#73a443] text-white font-extrabold text-xs uppercase tracking-wide rounded-xl border-b-4 border-[#5d8336] active:border-b-0 active:translate-y-1 transition shadow-md"
					on:click={loadPgn}
				>
					Analyze Game
				</button>
			</div>
		</div>
	</div>
{/if}
