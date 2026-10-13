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

	// Tabs: 'review', 'moves', 'history', 'advanced'
	type PanelTab = 'review' | 'moves' | 'history' | 'advanced';
	let rightPanelTab: PanelTab = 'review';
	let showAdvanced = false;

	let reviewResult: GameReviewResult | null = null;
	let showPgnModal = false;
	let copiedPgn = false;
	let copiedFen = false;
	let currentPgn = '';
	let localGamesList: LocalGameRecord[] = [];

	function toggleAdvanced() {
		showAdvanced = !showAdvanced;
		if (typeof window !== 'undefined') {
			localStorage.setItem('chessnova_show_advanced', showAdvanced ? '1' : '0');
		}
		if (showAdvanced) {
			rightPanelTab = 'advanced';
		} else if (rightPanelTab === 'advanced') {
			rightPanelTab = 'review';
		}
	}

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
			const advPref = localStorage.getItem('chessnova_show_advanced');
			if (advPref === '1') {
				showAdvanced = true;
			}

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
		}
	});

	$: currentReviewedMove = (() => {
		if (!reviewResult || currentPly <= 0 || currentPly > reviewResult.moves.length) {
			return null;
		}
		return reviewResult.moves[currentPly - 1];
	})();

	$: activeReviewMove = currentReviewedMove
		? {
				from: currentReviewedMove.from,
				to: currentReviewedMove.to,
				classification: currentReviewedMove.classification,
				bestMoveUci: currentReviewedMove.bestMoveUci
			}
		: null;

	$: currentScoreCp = (() => {
		if (reviewResult && currentPly > 0 && currentPly <= reviewResult.evalGraph.length) {
			return reviewResult.evalGraph[currentPly - 1].scoreCp;
		}
		return 0;
	})();

	function handleMove(event: CustomEvent<{ from: string; to: string; promotion?: PieceType }>) {
		const { from, to, promotion } = event.detail;
		const res = executeMoveClient(fen, from, to, promotion);
		if (!res || !res.valid) return;

		if (res.isCheckmate) {
			soundEffects.playCheckmate();
		} else if (res.isCheck) {
			soundEffects.playCheck();
		} else if (res.isPromotion) {
			soundEffects.playPromotion();
		} else if (res.isCapture) {
			soundEffects.playCapture();
		} else {
			soundEffects.playNormalMove();
		}

		// If branching off during review, truncate subsequent moves
		const newMoves = moves.slice(0, currentPly);
		newMoves.push({
			from,
			to,
			san: res.san,
			fen: res.newFen
		});

		moves = newMoves;
		currentPly = newMoves.length;
		fen = res.newFen;
		customFenInput = fen;
		turn = res.turn;
		lastMove = { from, to, san: res.san, isCapture: res.isCapture, isPromotion: res.isPromotion };

		// Rebuild PGN
		const c = new Chess(startFen === INITIAL_FEN ? undefined : startFen);
		for (const m of moves) {
			try {
				c.move(m.san || { from: m.from, to: m.to });
			} catch (_) {}
		}
		currentPgn = c.pgn();
		customPgnInput = currentPgn;
		reviewResult = null;
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
			lastMove = { from: m.from, to: m.to, san: m.san, isCapture: m.san?.includes('x') };
			const parts = m.fen.split(' ');
			turn = parts.length > 1 && parts[1] === 'b' ? 'black' : 'white';
		}
		customFenInput = fen;
	}

	function loadCustomFen() {
		if (customFenInput.trim()) {
			try {
				const test = new Chess(customFenInput.trim());
				startFen = customFenInput.trim();
				fen = startFen;
				moves = [];
				currentPly = 0;
				lastMove = null;
				reviewResult = null;
				turn = test.turn() === 'w' ? 'white' : 'black';
			} catch (e) {
				alert('Invalid FEN position string');
			}
		}
	}

	function copyFenText() {
		if (fen) {
			navigator.clipboard.writeText(fen);
			copiedFen = true;
			setTimeout(() => (copiedFen = false), 2000);
		}
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
		if (rec.white) whitePlayerName = rec.white;
		if (rec.black) blackPlayerName = rec.black;
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

<div class="flex-1 max-w-7xl mx-auto w-full p-2 sm:px-4 sm:py-3 flex flex-col justify-center select-none">
	<!-- Top Bar -->
	<div class="mb-3 flex items-center justify-between gap-3 flex-wrap">
		<div class="flex items-center gap-3">
			<div class="w-9 h-9 rounded-xl bg-sky-500/15 border border-sky-500/30 flex items-center justify-center text-sky-400 shadow-md">
				<Icon name="search" size={17} />
			</div>
			<div>
				<h1 class="text-base sm:text-lg font-black text-white tracking-tight">Game Analysis & Review</h1>
				<p class="text-[11px] text-slate-400 hidden sm:block">Move-by-move evaluations, advantage curve, and key moments</p>
			</div>
		</div>

		<!-- Action Buttons & Advanced Toggle -->
		<div class="flex items-center gap-2">
			<!-- Advanced Mode Toggle -->
			<button
				class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold transition border cursor-pointer active:translate-y-px {showAdvanced
					? 'bg-slate-800 border-slate-600 text-slate-100 shadow-xs'
					: 'bg-slate-900/80 border-slate-800 text-slate-400 hover:text-white hover:bg-slate-850'}"
				on:click={toggleAdvanced}
				title="Toggle advanced FEN, PGN, and position tools"
			>
				<Icon name="settings" size={13} className={showAdvanced ? 'text-emerald-400' : 'text-slate-400'} />
				<span>Advanced Tools</span>
				<span class="w-1.5 h-1.5 rounded-full {showAdvanced ? 'bg-emerald-400' : 'bg-slate-600'}"></span>
			</button>

			<button
				class="inline-flex items-center gap-1.5 px-2.5 py-1.5 bg-slate-900/90 hover:bg-slate-800 border border-slate-700/60 rounded-xl text-xs font-semibold text-slate-200 transition shadow-xs active:translate-y-px"
				on:click={() => (orientation = orientation === 'white' ? 'black' : 'white')}
				title="Flip Board"
			>
				<Icon name="rotate-cw" size={13} className="text-slate-400" />
				<span class="hidden sm:inline">Flip</span>
			</button>

			<button
				class="inline-flex items-center gap-1.5 px-2.5 py-1.5 bg-slate-900/90 hover:bg-slate-800 border border-slate-700/60 rounded-xl text-xs font-semibold text-slate-200 transition shadow-xs active:translate-y-px"
				on:click={resetToStart}
				title="Reset to Starting Position"
			>
				<Icon name="rotate-ccw" size={13} className="text-slate-400" />
				<span class="hidden sm:inline">Reset</span>
			</button>
		</div>
	</div>

	<!-- Main Board & Side Panel Grid -->
	<div class="grid grid-cols-1 lg:grid-cols-12 gap-3 lg:gap-5 items-center justify-center my-auto">
		<!-- Left: Board with Eval Bar -->
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

		<!-- Right: Analysis Panel -->
		<div class="lg:col-span-5 xl:col-span-4 h-[440px] lg:h-[min(calc(100dvh-150px),620px)] flex flex-col">
			<!-- Panel Tab Navigation -->
			<div class="flex items-center p-1 bg-slate-950/80 border border-slate-800 rounded-xl mb-2.5 shrink-0 overflow-x-auto">
				<button
					class="flex-1 py-1.5 px-2.5 rounded-lg text-xs font-semibold transition flex items-center justify-center gap-1.5 whitespace-nowrap {rightPanelTab === 'review'
						? 'bg-slate-800 text-white font-bold border border-slate-700/80 shadow-xs'
						: 'text-slate-400 hover:text-slate-200'}"
					on:click={() => (rightPanelTab = 'review')}
				>
					<Icon name="search" size={13} className={rightPanelTab === 'review' ? 'text-emerald-400' : ''} />
					<span>Review</span>
				</button>

				<button
					class="flex-1 py-1.5 px-2.5 rounded-lg text-xs font-semibold transition flex items-center justify-center gap-1.5 whitespace-nowrap {rightPanelTab === 'moves'
						? 'bg-slate-800 text-white font-bold border border-slate-700/80 shadow-xs'
						: 'text-slate-400 hover:text-slate-200'}"
					on:click={() => (rightPanelTab = 'moves')}
				>
					<Icon name="play" size={13} className={rightPanelTab === 'moves' ? 'text-emerald-400' : ''} />
					<span>Moves ({moves.length})</span>
				</button>

				<button
					class="flex-1 py-1.5 px-2.5 rounded-lg text-xs font-semibold transition flex items-center justify-center gap-1.5 whitespace-nowrap {rightPanelTab === 'history'
						? 'bg-slate-800 text-white font-bold border border-slate-700/80 shadow-xs'
						: 'text-slate-400 hover:text-slate-200'}"
					on:click={() => {
						refreshLocalGames();
						rightPanelTab = 'history';
					}}
				>
					<Icon name="clock" size={13} className={rightPanelTab === 'history' ? 'text-emerald-400' : ''} />
					<span>Saved</span>
				</button>

				{#if showAdvanced}
					<button
						class="flex-1 py-1.5 px-2.5 rounded-lg text-xs font-semibold transition flex items-center justify-center gap-1.5 whitespace-nowrap {rightPanelTab === 'advanced'
							? 'bg-slate-800 text-white font-bold border border-slate-700/80 shadow-xs'
							: 'text-slate-400 hover:text-slate-200'}"
						on:click={() => (rightPanelTab = 'advanced')}
					>
						<Icon name="settings" size={13} className={rightPanelTab === 'advanced' ? 'text-emerald-400' : ''} />
						<span>Advanced</span>
					</button>
				{/if}
			</div>

			<!-- Main Panel Content -->
			<div class="flex-1 overflow-hidden">
				{#if rightPanelTab === 'review'}
					<!-- 1. Review Tab (Clean, consumer-friendly review) -->
					<GameReview
						bind:reviewResult
						pgn={currentPgn}
						{moves}
						{whitePlayerName}
						{blackPlayerName}
						{currentPly}
						on:selectPly={(e) => goToPly(e.detail)}
					/>
				{:else if rightPanelTab === 'moves'}
					<!-- 2. Moves Tab (Clean move notation list & stepper for normal players) -->
					<div class="bg-slate-900/90 border border-slate-800 rounded-2xl p-4 shadow-2xl flex flex-col h-full overflow-hidden backdrop-blur-xl">
						<div class="flex items-center justify-between pb-3 border-b border-slate-800 shrink-0">
							<div>
								<h3 class="text-xs font-bold text-white uppercase tracking-wider">Move Ledger</h3>
								<p class="text-[11px] text-slate-400">Step through moves using click or keyboard arrows</p>
							</div>
							{#if moves.length > 0}
								<button
									class="px-2 py-1 bg-slate-800 hover:bg-slate-700 text-slate-300 text-[11px] font-bold rounded-lg transition border border-slate-700/60"
									on:click={resetToStart}
								>
									Clear
								</button>
							{/if}
						</div>

						<div class="flex-1 overflow-hidden py-2">
							<MoveList
								{moves}
								activePly={currentPly}
								whiteName={whitePlayerName}
								blackName={blackPlayerName}
								on:selectPly={(e) => goToPly(e.detail)}
							/>
						</div>

						<!-- Step Controls -->
						<div class="grid grid-cols-4 gap-2 pt-2 border-t border-slate-800 shrink-0">
							<button
								class="py-2 flex items-center justify-center bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl transition"
								on:click={() => goToPly(0)}
								disabled={currentPly <= 0}
								title="Start of Game"
							>
								<Icon name="chevrons-left" size={16} />
							</button>
							<button
								class="py-2 flex items-center justify-center bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl transition"
								on:click={() => goToPly(currentPly - 1)}
								disabled={currentPly <= 0}
								title="Previous Move"
							>
								<Icon name="chevron-left" size={16} />
							</button>
							<button
								class="py-2 flex items-center justify-center bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl transition"
								on:click={() => goToPly(currentPly + 1)}
								disabled={currentPly >= moves.length}
								title="Next Move"
							>
								<Icon name="chevron-right" size={16} />
							</button>
							<button
								class="py-2 flex items-center justify-center bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl transition"
								on:click={() => goToPly(moves.length)}
								disabled={currentPly >= moves.length}
								title="Latest Move"
							>
								<Icon name="chevrons-right" size={16} />
							</button>
						</div>
					</div>
				{:else if rightPanelTab === 'history'}
					<!-- 3. Saved Past Games Tab -->
					<div class="bg-slate-900/90 border border-slate-800 rounded-2xl p-4 shadow-2xl flex flex-col h-full overflow-hidden backdrop-blur-xl">
						<div class="flex items-center justify-between pb-3 border-b border-slate-800 shrink-0">
							<div>
								<h3 class="text-xs font-bold text-white uppercase tracking-wider">Saved Matches</h3>
								<p class="text-[11px] text-slate-400">Match history automatically preserved for review</p>
							</div>
							{#if localGamesList.length > 0}
								<button
									class="px-2 py-1 bg-slate-800 hover:bg-rose-900/60 hover:text-rose-200 text-slate-400 text-[11px] font-bold rounded-lg transition border border-slate-700"
									on:click={handleClearAllSavedGames}
								>
									Clear All
								</button>
							{/if}
						</div>

						<div class="flex-1 overflow-y-auto py-2 flex flex-col gap-2">
							{#if localGamesList.length === 0}
								<div class="flex flex-col items-center justify-center h-full text-center py-12 px-4 gap-2">
									<div class="w-10 h-10 rounded-xl bg-slate-800 flex items-center justify-center text-slate-500">
										<Icon name="clock" size={20} />
									</div>
									<span class="text-xs font-bold text-slate-300">No Saved Games</span>
									<p class="text-[11px] text-slate-500 max-w-[200px]">
										Play online or against bots to build your review archive.
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
												: 'text-slate-300 bg-slate-800 border-slate-700'}

									<div class="p-3 bg-slate-950/80 border border-slate-800 hover:border-slate-700 rounded-xl transition flex items-center justify-between gap-3 group">
										<div class="min-w-0 flex-1">
											<div class="flex items-center gap-2 mb-1">
												<span class="px-1.5 py-0.5 rounded text-[10px] font-mono font-bold border {resClass}">
													{rec.result}
												</span>
												<span class="text-[11px] font-bold text-white truncate">
													{rec.white} vs {rec.black}
												</span>
											</div>
											<div class="flex items-center gap-2 text-[10px] text-slate-400">
												<span>{rec.event}</span>
												<span>•</span>
												<span>{rec.movesCount} plies</span>
											</div>
										</div>

										<div class="flex items-center gap-1.5 shrink-0">
											<button
												class="px-3 py-1.5 bg-[#81b64c] hover:bg-[#72a441] active:bg-[#639037] text-white font-bold text-xs rounded-lg shadow-xs transition flex items-center gap-1.5 active:translate-y-px"
												on:click={() => handleLoadSavedGame(rec)}
												title="Review this game"
											>
												<Icon name="search" size={12} />
												<span>Review</span>
											</button>

											<button
												class="p-1.5 text-slate-500 hover:text-rose-400 hover:bg-slate-800 rounded-lg transition"
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
				{:else if showAdvanced}
					<!-- 4. Advanced Tab (Hidden behind Advanced Toggle for power users) -->
					<div class="bg-slate-900/90 border border-slate-800 rounded-2xl p-4 shadow-2xl flex flex-col gap-4 h-full backdrop-blur-xl overflow-y-auto">
						<div>
							<div class="flex items-center justify-between mb-1">
								<h3 class="text-xs font-bold text-white uppercase tracking-wider">Advanced Workbench</h3>
								<span class="text-[10px] font-mono font-bold text-indigo-400 bg-indigo-500/10 px-2 py-0.5 rounded-full border border-indigo-500/20">Power Mode</span>
							</div>
							<p class="text-[11px] text-slate-400">Inspect FEN strings, import custom PGNs, and configure positions.</p>
						</div>

						<!-- Position FEN Inspector & Loader -->
						<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-3 flex flex-col gap-2">
							<div class="flex items-center justify-between">
								<label for="fen-input" class="text-[10px] font-bold uppercase tracking-wider text-slate-400">
									Current Position FEN
								</label>
								<button
									class="text-[10px] text-sky-400 hover:text-sky-300 font-mono font-semibold flex items-center gap-1"
									on:click={copyFenText}
								>
									<Icon name={copiedFen ? 'check' : 'download'} size={11} className={copiedFen ? 'text-emerald-400' : ''} />
									<span>{copiedFen ? 'Copied' : 'Copy FEN'}</span>
								</button>
							</div>

							<input
								id="fen-input"
								type="text"
								bind:value={customFenInput}
								class="w-full px-3 py-1.5 bg-slate-900 border border-slate-800 rounded-lg text-xs text-slate-200 font-mono focus:outline-none focus:border-sky-500 transition placeholder:text-slate-600"
								placeholder="Paste FEN..."
							/>

							<div class="flex items-center justify-between gap-2 pt-1">
								<span class="text-[10px] text-slate-500">Edit string to jump to custom position</span>
								<button
									class="px-3 py-1 bg-indigo-600 hover:bg-indigo-500 text-white font-bold text-xs rounded-lg transition"
									on:click={loadCustomFen}
								>
									Load FEN
								</button>
							</div>
						</div>

						<!-- PGN Tools -->
						<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-3 flex flex-col gap-2">
							<div class="flex items-center justify-between">
								<span class="text-[10px] font-bold uppercase tracking-wider text-slate-400">Game PGN Record</span>
								<div class="flex items-center gap-2">
									{#if currentPgn}
										<button
											class="text-[10px] text-slate-300 hover:text-white font-mono flex items-center gap-1"
											on:click={copyPgnText}
										>
											<Icon name={copiedPgn ? 'check' : 'download'} size={11} className={copiedPgn ? 'text-emerald-400' : ''} />
											<span>{copiedPgn ? 'Copied' : 'Copy PGN'}</span>
										</button>
										<button
											class="text-[10px] text-slate-300 hover:text-white font-mono flex items-center gap-1"
											on:click={downloadPgnFile}
										>
											<Icon name="download" size={11} />
											<span>Download</span>
										</button>
									{/if}
								</div>
							</div>

							<div class="flex gap-2">
								<button
									class="flex-1 py-2 px-3 bg-slate-800 hover:bg-slate-700 text-white font-bold text-xs rounded-lg transition flex items-center justify-center gap-1.5 border border-slate-700/60"
									on:click={() => (showPgnModal = true)}
								>
									<Icon name="download" size={13} className="text-sky-400" />
									<span>Import Raw PGN</span>
								</button>
							</div>
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
		<div class="bg-slate-900 border border-slate-800 rounded-2xl p-5 max-w-md w-full shadow-2xl flex flex-col gap-3 backdrop-blur-xl">
			<div class="flex items-center justify-between border-b border-slate-800 pb-2">
				<div class="flex items-center gap-2">
					<Icon name="download" size={16} className="text-sky-400" />
					<h3 class="text-sm font-bold text-white">Import Game PGN</h3>
				</div>
				<button class="text-slate-400 hover:text-white" on:click={() => (showPgnModal = false)}>
					<Icon name="x" size={16} />
				</button>
			</div>

			<p class="text-xs text-slate-400">
				Paste any standard chess game PGN below to generate a complete Game Review with move classifications and accuracy analysis.
			</p>

			<textarea
				bind:value={customPgnInput}
				class="w-full h-36 bg-slate-950/90 border border-slate-800 rounded-xl p-3 text-xs text-white font-mono focus:outline-none focus:border-sky-500 transition"
				placeholder="1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. c3 Nf6..."
			></textarea>

			<div class="flex items-center justify-end gap-2 pt-2">
				<button
					class="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 font-bold text-xs rounded-xl transition"
					on:click={() => (showPgnModal = false)}
				>
					Cancel
				</button>
				<button
					class="px-4 py-2 bg-sky-500 hover:bg-sky-400 active:bg-sky-600 text-white font-extrabold text-xs uppercase tracking-wide rounded-xl shadow-md transition"
					on:click={() => loadPgnContent(customPgnInput)}
				>
					Analyze Game
				</button>
			</div>
		</div>
	</div>
{/if}
