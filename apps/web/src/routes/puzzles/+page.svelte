<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth';
	import ChessBoard from '$lib/components/chess/ChessBoard.svelte';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import { Chess } from 'chess.js';
	import { soundEffects } from '$lib/audio/sounds';
	import { executeMoveClient, type Color, type PieceType } from '$lib/chess/engine';

	let currentPuzzle: any = null;
	let currentFen = '';
	let solutionIndex = 0;
	let finishedWithMate = false;
	let statusMessage = 'Find the best move in this position.';
	let statusType: 'info' | 'success' | 'error' = 'info';
	let loading = true;
	let orientation: Color = 'white';
	let turn: Color = 'white';
	let lastMove: { from: string; to: string; san?: string; isCapture?: boolean; isPromotion?: boolean } | null = null;
	let userRating = 400;
	let ratingDiff: number | null = null;
	let solvedCount = 0;

	// Puzzle Modes: 'practice' (category & tiered filters), 'daily' (today's puzzle), 'weekly' (7 weekly challenges)
	let puzzleMode: 'practice' | 'daily' | 'weekly' = 'practice';
	let selectedCategory: string = 'all'; // all, mate, fork, pin, skewer, capture, position
	let selectedTier: number = 0; // 0 = all, 1 = Novice, 2 = Intermediate, 3 = Advanced, 4 = Expert, 5 = Master
	let weeklyList: any[] = [];
	let currentWeeklyIndex = 0;

	// Practice & Assistance state
	let isRetrying = false;
	let hasAssistance = false;
	let hintSquare: string | null = null;
	let hintArrow: { from: string; to: string } | null = null;
	let hintCount = 0;
	let isSolutionRevealed = false;
	let solutionMoves: string[] = [];

	async function loadPuzzle(overrideMode?: 'practice' | 'daily' | 'weekly') {
		const mode = overrideMode || puzzleMode;
		puzzleMode = mode;
		loading = true;
		statusMessage = mode === 'daily'
			? "Today's Daily Challenge. Find the master combination!"
			: mode === 'weekly'
				? `Weekly Challenge ${currentWeeklyIndex + 1} of ${weeklyList.length || 7}. Calculate precisely!`
				: 'Find the best tactical sequence in this position.';
		statusType = 'info';
		solutionIndex = 0;
		finishedWithMate = false;
		lastMove = null;
		ratingDiff = null;
		isRetrying = false;
		hasAssistance = false;
		hintSquare = null;
		hintArrow = null;
		hintCount = 0;
		isSolutionRevealed = false;
		solutionMoves = [];

		try {
			let p: any = null;
			if (mode === 'daily') {
				p = await api.puzzles.getDaily();
			} else if (mode === 'weekly') {
				if (weeklyList.length === 0) {
					weeklyList = await api.puzzles.getWeekly();
				}
				if (weeklyList && weeklyList.length > 0) {
					p = weeklyList[currentWeeklyIndex % weeklyList.length];
				} else {
					p = await api.puzzles.getRandom();
				}
			} else {
				p = await api.puzzles.getRandom({
					category: selectedCategory !== 'all' ? selectedCategory : undefined,
					tier: selectedTier > 0 ? selectedTier : undefined
				});
			}

			currentPuzzle = p;
			currentFen = p.fen;

			// Determine whose turn it is in FEN
			const parts = p.fen.split(' ');
			turn = parts[1] === 'w' ? 'white' : 'black';
			orientation = turn;
		} catch (e: any) {
			statusMessage = 'Failed to load tactical puzzle.';
			statusType = 'error';
		} finally {
			loading = false;
		}
	}

	async function loadRandomPuzzle() {
		if (puzzleMode === 'weekly' && weeklyList.length > 0) {
			currentWeeklyIndex = (currentWeeklyIndex + 1) % weeklyList.length;
		}
		return loadPuzzle();
	}

	async function requestHint() {
		if (!currentPuzzle || statusType === 'success' || isSolutionRevealed) return;
		try {
			hasAssistance = true;
			const hintData = await api.puzzles.getHint(currentPuzzle.id, solutionIndex);
			if (hintCount === 0) {
				// Level 1 hint: Piece starting square
				hintSquare = hintData.from;
				statusMessage = `Tactical Hint: Focus on the piece on ${hintData.from.toUpperCase()}!`;
				statusType = 'info';
				hintCount = 1;
			} else {
				// Level 2 hint: Piece target vector arrow
				hintArrow = { from: hintData.from, to: hintData.to };
				statusMessage = `Direct Hint: Move piece to ${hintData.to.toUpperCase()}!`;
				statusType = 'info';
			}
		} catch (err: any) {
			statusMessage = 'Could not retrieve hint for this position.';
		}
	}

	async function revealSolution() {
		if (!currentPuzzle) return;
		try {
			hasAssistance = true;
			isSolutionRevealed = true;
			statusType = 'info';
			statusMessage = 'Solution revealed. Playing through the winning moves...';

			const solRes = await api.puzzles.getSolution(currentPuzzle.id);
			solutionMoves = solRes.solution;

			// Step through each remaining move with an animated delay
			playSolutionStep(solutionIndex);
		} catch (err: any) {
			statusMessage = 'Unable to fetch solution.';
		}
	}

	function playSolutionStep(idx: number) {
		if (!solutionMoves || idx >= solutionMoves.length) {
			statusMessage = 'Solution playback complete.';
			return;
		}
		const moveUci = solutionMoves[idx];
		const from = moveUci.slice(0, 2);
		const to = moveUci.slice(2, 4);
		const promo = (moveUci.slice(4) as PieceType) || undefined;

		const res = executeMoveClient(currentFen, from, to, promo);
		if (res) {
			currentFen = res.newFen;
			lastMove = { from, to, san: res.san, isCapture: res.isCapture, isPromotion: res.isPromotion };
			turn = res.turn;
		}

		if (idx + 1 < solutionMoves.length) {
			setTimeout(() => {
				playSolutionStep(idx + 1);
			}, 600);
		} else {
			const chk = new Chess(currentFen);
			if (chk.isCheckmate()) {
				statusMessage = 'Solution complete: Decisive checkmate delivered!';
				finishedWithMate = true;
			} else {
				statusMessage = 'Solution complete! Review the tactical pattern.';
			}
		}
	}

	function resetToInitial() {
		if (!currentPuzzle) return;
		currentFen = currentPuzzle.fen;
		solutionIndex = 0;
		finishedWithMate = false;
		isRetrying = true;
		hintSquare = null;
		hintArrow = null;
		lastMove = null;
		isSolutionRevealed = false;
		const parts = currentPuzzle.fen.split(' ');
		turn = parts[1] === 'w' ? 'white' : 'black';
		statusMessage = 'Position reset. Find the optimal tactical combination!';
		statusType = 'info';
	}

	function handleKeydown(e: KeyboardEvent) {
		if (statusType === 'success' && (e.key === ' ' || e.key === 'Enter')) {
			e.preventDefault();
			loadRandomPuzzle();
		}
	}

	async function handleMove(event: CustomEvent<{ from: string; to: string; promotion?: PieceType }>) {
		if (!currentPuzzle || statusType === 'success' || isSolutionRevealed) return;
		const { from, to, promotion } = event.detail;
		const uci = `${from}${to}${promotion ? promotion.toLowerCase() : ''}`;

		const res = executeMoveClient(currentFen, from, to, promotion);
		if (!res) return;

		currentFen = res.newFen;
		lastMove = { from, to, san: res.san, isCapture: res.isCapture, isPromotion: res.isPromotion };
		turn = res.turn;
		hintSquare = null;
		hintArrow = null;

		const chk = new Chess(res.newFen);
		const isMate = chk.isCheckmate();

		try {
			const check = await api.puzzles.solve(currentPuzzle.id, {
				move: uci,
				plyIndex: solutionIndex,
				isRetry: isRetrying,
				hasAssistance: hasAssistance
			});

			if (check.correct) {
				if (check.finished) {
					finishedWithMate = isMate;
					if (isMate) {
						if (hasAssistance || isRetrying) {
							statusMessage = 'Checkmate! Puzzle solved with practice/assistance.';
						} else {
							statusMessage = 'Checkmate! You discovered the winning mating sequence.';
							userRating = check.newRating;
							ratingDiff = check.diff;
							solvedCount++;
							if ($authStore.user) {
								authStore.updateRating('puzzles', check.newRating);
							}
						}
					} else {
						if (hasAssistance || isRetrying) {
							statusMessage = 'Puzzle solved with practice/assistance! (Rating unaffected)';
						} else {
							statusMessage = 'Brilliant! Tactical combination completed successfully.';
							userRating = check.newRating;
							ratingDiff = check.diff;
							solvedCount++;
							if ($authStore.user) {
								authStore.updateRating('puzzles', check.newRating);
							}
						}
					}
					statusType = 'success';
					soundEffects.playGameStart();
				} else {
					statusMessage = 'Best move found! Continue the tactical sequence...';
					statusType = 'info';
					solutionIndex += 2;
					if (check.replyMove) {
						const replyUci = check.replyMove;
						setTimeout(() => {
							const replyFrom = replyUci.slice(0, 2);
							const replyTo = replyUci.slice(2, 4);
							const replyPromo = replyUci.slice(4) || undefined;
							const replyRes = executeMoveClient(currentFen, replyFrom, replyTo, replyPromo);
							if (replyRes) {
								currentFen = replyRes.newFen;
								lastMove = { from: replyFrom, to: replyTo, san: replyRes.san, isCapture: replyRes.isCapture, isPromotion: replyRes.isPromotion };
								turn = replyRes.turn;
							}
						}, 350);
					}
				}
			} else {
				statusMessage = 'Not the optimal move. Retry or request a hint!';
				statusType = 'error';
				if (!isRetrying && !hasAssistance) {
					userRating = check.newRating;
					ratingDiff = check.diff;
					if ($authStore.user) {
						authStore.updateRating('puzzles', check.newRating);
					}
				}
				isRetrying = true;
			}
		} catch (err: any) {
			statusMessage = 'Verification error: ' + (err.message || 'unknown');
			statusType = 'error';
		}
	}

	onMount(() => {
		const pr = $authStore.ratings?.puzzles ?? $authStore.ratings?.puzzle;
		if (pr) {
			userRating = Math.max(100, pr);
		}
		loadRandomPuzzle();
	});
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="flex-1 max-w-7xl mx-auto w-full px-2 py-1 sm:px-4 sm:py-1.5 lg:px-6 flex flex-col justify-center min-h-0 lg:h-[calc(100dvh-4rem)] lg:max-h-[calc(100dvh-4rem)] overflow-hidden">
	<!-- Slim Unified Header & Filter Toolbar -->
	<div class="mb-1.5 flex flex-wrap items-center justify-between gap-2 bg-slate-900/80 border border-slate-800/80 rounded-xl px-2.5 py-1.5 backdrop-blur-md shrink-0 shadow-md">
		<!-- Left: Mode Switcher -->
		<div class="flex items-center gap-1">
			<button
				class="px-2.5 py-1 rounded-lg text-xs font-semibold transition flex items-center gap-1.5 {puzzleMode === 'practice'
					? 'bg-slate-800 text-white font-bold border border-slate-700/80 shadow-xs'
					: 'text-slate-400 hover:text-slate-200'}"
				on:click={() => loadPuzzle('practice')}
			>
				<Icon name="puzzle" size={13} className={puzzleMode === 'practice' ? 'text-emerald-400' : ''} />
				<span>Practice</span>
			</button>

			<button
				class="px-2.5 py-1 rounded-lg text-xs font-semibold transition flex items-center gap-1.5 {puzzleMode === 'daily'
					? 'bg-slate-800 text-amber-300 font-bold border border-slate-700/80 shadow-xs'
					: 'text-slate-400 hover:text-slate-200'}"
				on:click={() => loadPuzzle('daily')}
			>
				<Icon name="flame" size={13} className="text-amber-400" />
				<span>Daily</span>
			</button>

			<button
				class="px-2.5 py-1 rounded-lg text-xs font-semibold transition flex items-center gap-1.5 {puzzleMode === 'weekly'
					? 'bg-slate-800 text-indigo-300 font-bold border border-slate-700/80 shadow-xs'
					: 'text-slate-400 hover:text-slate-200'}"
				on:click={() => loadPuzzle('weekly')}
			>
				<Icon name="trophy" size={13} className="text-indigo-400" />
				<span class="hidden sm:inline">Weekly</span>
				<span class="text-[10px] font-mono opacity-80">({weeklyList.length || 7})</span>
			</button>
		</div>

		<!-- Middle: Filters (When in practice mode) -->
		{#if puzzleMode === 'practice'}
			<div class="hidden sm:flex items-center gap-2">
				<!-- Category Filter -->
				<div class="flex items-center gap-0.5 bg-slate-950/70 p-0.5 rounded-lg border border-slate-800">
					{#each ['all', 'mate', 'fork', 'pin', 'capture'] as cat}
						<button
							class="px-2 py-0.5 rounded text-[11px] font-medium transition capitalize {selectedCategory === cat
								? 'bg-sky-500/20 text-sky-300 border border-sky-500/40 font-bold'
								: 'text-slate-400 hover:text-slate-200'}"
							on:click={() => {
								selectedCategory = cat;
								loadPuzzle('practice');
							}}
						>
							{cat}
						</button>
					{/each}
				</div>

				<!-- Tier Filter -->
				<div class="flex items-center gap-0.5 bg-slate-950/70 p-0.5 rounded-lg border border-slate-800">
					<button
						class="px-1.5 py-0.5 rounded text-[11px] font-medium transition {selectedTier === 0
							? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 font-bold'
							: 'text-slate-400 hover:text-slate-200'}"
						on:click={() => {
							selectedTier = 0;
							loadPuzzle('practice');
						}}
					>
						All
					</button>
					{#each [1, 2, 3, 4, 5] as tierLvl}
						<button
							class="px-1.5 py-0.5 rounded text-[11px] font-medium transition {selectedTier === tierLvl
								? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 font-bold'
								: 'text-slate-400 hover:text-slate-200'}"
							title={`Tier ${tierLvl}`}
							on:click={() => {
								selectedTier = tierLvl;
								loadPuzzle('practice');
							}}
						>
							T{tierLvl}
						</button>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Right: Rating Badge & Quick Actions -->
		<div class="flex items-center gap-2">
			<div class="flex items-center gap-1.5 px-2 py-0.5 rounded-lg bg-slate-950/80 border border-slate-800 text-xs font-mono">
				<span class="text-slate-400 text-[10px] uppercase font-bold">Elo:</span>
				<span class="font-extrabold text-white">{userRating}</span>
				{#if ratingDiff !== null}
					<span class="text-[10px] font-bold {ratingDiff > 0 ? 'text-emerald-400' : 'text-rose-400'}">
						{ratingDiff > 0 ? `+${ratingDiff}` : ratingDiff}
					</span>
				{/if}
			</div>

			<button
				class="p-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg transition border border-slate-700/60 flex items-center justify-center"
				on:click={() => (orientation = orientation === 'white' ? 'black' : 'white')}
				title="Flip Board"
			>
				<Icon name="rotate-cw" size={13} />
			</button>
		</div>
	</div>

	<!-- Main Board & Side Panel Grid -->
	<div class="grid grid-cols-1 lg:grid-cols-12 gap-3 lg:gap-6 items-center justify-center my-auto min-h-0">
		<!-- Left: Hero Puzzle Board Column -->
		<div class="lg:col-span-7 xl:col-span-8 flex flex-col items-center justify-center min-h-0 w-full">
			<div class="w-full max-w-[min(96vw,calc(100dvh-170px),640px)] flex flex-col items-center">
				
				<!-- Turn Banner -->
				<div class="w-full flex items-center justify-between py-1 px-1 text-xs font-semibold select-none mb-1">
					<div class="flex items-center gap-2">
						<span class="w-2.5 h-2.5 rounded-full {turn === 'white' ? 'bg-white border border-slate-400' : 'bg-slate-900 border border-slate-600'}"></span>
						<span class="text-slate-300 text-xs font-bold capitalize">{turn} to play</span>
					</div>
					{#if currentPuzzle}
						<div class="flex items-center gap-1.5 text-[11px] font-mono text-slate-400">
							<span>Rating {currentPuzzle.rating}</span>
							{#if currentPuzzle.tier}
								<span class="px-1.5 py-0.2 rounded bg-cyan-500/15 text-cyan-300 border border-cyan-500/30 font-bold">
									T{currentPuzzle.tier}
								</span>
							{/if}
						</div>
					{/if}
				</div>

				<!-- Chess Board -->
				{#if loading}
					<div class="w-full aspect-square bg-slate-900/60 border border-slate-800/80 rounded-2xl flex flex-col items-center justify-center text-sm text-slate-400 gap-3 shadow-2xl backdrop-blur-xl animate-pulse">
						<div class="w-8 h-8 border-2 border-sky-500/30 border-t-sky-400 rounded-full animate-spin"></div>
						<span class="text-xs font-mono text-slate-400">Loading tactical position...</span>
					</div>
				{:else}
					<ChessBoard
						fen={currentFen}
						{orientation}
						{turn}
						{lastMove}
						{hintSquare}
						{hintArrow}
						mode="puzzle"
						interactive={!isSolutionRevealed && statusType !== 'success'}
						boardSizeClass="w-full"
						on:move={handleMove}
					/>
				{/if}

				<!-- Tactical Status & Immediate Action Bar (DIRECTLY ATTACHED UNDER BOARD) -->
				<div class="w-full mt-2 flex flex-col gap-1.5">
					<!-- Status Banner -->
					<div
						class="px-3 py-2 rounded-xl border flex items-center justify-between gap-2 transition-all duration-200 {statusType === 'success'
							? 'bg-emerald-950/50 border-emerald-500/50 text-emerald-200 shadow-md shadow-emerald-500/10'
							: statusType === 'error'
								? 'bg-rose-950/50 border-rose-500/50 text-rose-200'
								: 'bg-slate-900/80 border-slate-800 text-slate-300'}"
					>
						<div class="flex items-center gap-2 min-w-0">
							<div class="shrink-0">
								{#if statusType === 'success'}
									<div class="w-5 h-5 rounded-md bg-emerald-500/20 text-emerald-400 flex items-center justify-center border border-emerald-500/30">
										<Icon name={finishedWithMate ? 'crown' : 'check'} size={12} />
									</div>
								{:else if statusType === 'error'}
									<div class="w-5 h-5 rounded-md bg-rose-500/20 text-rose-400 flex items-center justify-center border border-rose-500/30">
										<Icon name="x" size={12} />
									</div>
								{:else}
									<div class="w-5 h-5 rounded-md bg-sky-500/20 text-sky-400 flex items-center justify-center border border-sky-500/30">
										<Icon name="sparkles" size={12} />
									</div>
								{/if}
							</div>
							<span class="text-xs font-semibold truncate leading-tight">
								{statusMessage}
							</span>
						</div>

						{#if statusType === 'success' && ratingDiff && ratingDiff > 0}
							<span class="px-2 py-0.5 rounded-md bg-emerald-500/20 border border-emerald-500/40 text-emerald-300 font-mono font-bold text-xs shrink-0 animate-bounce">
								+{ratingDiff}
							</span>
						{/if}
					</div>

					<!-- Tactical Buttons Bar -->
					<div class="flex items-center gap-2 w-full">
						{#if statusType === 'success'}
							<Button
								variant="primary"
								size="sm"
								className="flex-1 shadow-md shadow-sky-500/20 font-bold justify-center gap-1.5 py-2"
								on:click={loadRandomPuzzle}
							>
								<span>Next Puzzle</span>
								<Icon name="arrow-right" size={13} />
								<span class="text-[10px] text-white/70 font-mono font-normal">(Enter)</span>
							</Button>

							<Button
								variant="outline"
								size="sm"
								on:click={resetToInitial}
								className="text-slate-300 py-2"
								title="Retry position"
							>
								<Icon name="rotate-ccw" size={13} />
								<span class="hidden sm:inline">Retry</span>
							</Button>

							<a
								href="/analysis?fen={encodeURIComponent(currentFen)}"
								target="_blank"
								rel="noopener noreferrer"
								class="shrink-0"
							>
								<Button
									variant="outline"
									size="sm"
									className="text-slate-300 py-2"
									title="Analyze in Game Review"
								>
									<Icon name="search" size={13} />
									<span class="hidden sm:inline">Analyze</span>
								</Button>
							</a>
						{:else}
							<Button
								variant="outline"
								size="sm"
								disabled={isSolutionRevealed}
								on:click={requestHint}
								className="flex-1 text-slate-200 py-2"
								title="Get tactical hint without penalty"
							>
								<Icon name="sparkles" size={13} className="text-amber-400" />
								<span>{hintCount === 0 ? 'Hint' : 'Target'}</span>
							</Button>

							<Button
								variant="outline"
								size="sm"
								disabled={isSolutionRevealed}
								on:click={revealSolution}
								className="flex-1 text-slate-200 py-2"
								title="Reveal the winning move sequence"
							>
								<Icon name="eye" size={13} className="text-cyan-400" />
								<span>Solution</span>
							</Button>

							<Button
								variant="outline"
								size="sm"
								on:click={resetToInitial}
								className="text-slate-300 py-2"
								title="Reset position"
							>
								<Icon name="rotate-ccw" size={13} />
							</Button>

							<Button
								variant="outline"
								size="sm"
								on:click={loadRandomPuzzle}
								className="text-slate-300 py-2"
								title="Skip to next puzzle"
							>
								<Icon name="chevrons-right" size={13} />
							</Button>
						{/if}
					</div>
				</div>
			</div>
		</div>

		<!-- Right: Puzzle Details Card (Desktop) -->
		<div class="hidden lg:flex lg:col-span-5 xl:col-span-4 h-[min(calc(100dvh-110px),640px)] bg-slate-900/80 border border-slate-800/80 backdrop-blur-xl rounded-2xl p-5 shadow-2xl flex-col justify-between overflow-y-auto">
			<div class="flex items-center justify-between pb-4 border-b border-slate-800/80">
				<div>
					<span class="text-[10px] uppercase font-bold text-sky-400 tracking-wider">Puzzle Elo</span>
					<div class="flex items-center gap-2 mt-1">
						<span class="text-xl font-mono font-black text-white">{userRating}</span>
						{#if ratingDiff !== null}
							<span
								class="px-2 py-0.5 rounded-md text-xs font-mono font-bold {ratingDiff > 0
									? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
									: 'bg-rose-500/20 text-rose-400 border border-rose-500/30'} animate-bounce"
							>
								{ratingDiff > 0 ? `+${ratingDiff}` : ratingDiff}
							</span>
						{/if}
					</div>
					<div class="text-[10px] text-slate-500 mt-0.5">
						{solvedCount} solved today
					</div>
				</div>

				{#if currentPuzzle}
					<div class="text-right">
						<span class="text-[10px] uppercase font-bold text-slate-400 tracking-wider">Puzzle Level</span>
						<div class="text-lg font-mono font-extrabold text-cyan-400 flex items-center justify-end gap-1.5">
							<span>{currentPuzzle.rating}</span>
							{#if currentPuzzle.tier}
								<span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-cyan-500/20 text-cyan-300 border border-cyan-500/30">
									T{currentPuzzle.tier}
								</span>
							{/if}
						</div>
						<div class="text-[10px] text-slate-500 capitalize flex items-center justify-end gap-1.5 mt-0.5">
							{#if currentPuzzle.category}
								<span class="px-1.5 py-0.2 rounded bg-indigo-500/20 text-indigo-300 font-semibold uppercase text-[9px]">
									{currentPuzzle.category}
								</span>
							{/if}
							<span>{turn} to move</span>
						</div>
					</div>
				{/if}
			</div>

			{#if currentPuzzle && currentPuzzle.themes?.length}
				<div>
					<div class="flex items-center justify-between mb-2">
						<span class="text-[10px] uppercase font-bold text-slate-500 tracking-wider">Tactical Themes</span>
						<span class="text-[10px] text-slate-400 font-mono">
							#{currentPuzzle.id ? currentPuzzle.id.slice(0, 8) : 'tactics'}
						</span>
					</div>
					<div class="flex flex-wrap gap-1.5">
						{#each currentPuzzle.themes as theme}
							<span class="px-2.5 py-1 rounded-lg bg-slate-800/80 border border-slate-700/50 text-[11px] font-medium text-slate-300">
								#{theme}
							</span>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Practice Tier Quick Info -->
			<div class="p-3 bg-slate-950/60 rounded-xl border border-slate-800 text-xs text-slate-400 flex items-center justify-between">
				<div class="flex items-center gap-2">
					<Icon name="shield" size={15} className="text-cyan-400" />
					<span>Calculation Trainer</span>
				</div>
				<span class="font-mono text-[11px] text-slate-500">525+ Curated Puzzles</span>
			</div>
		</div>
	</div>
</div>
