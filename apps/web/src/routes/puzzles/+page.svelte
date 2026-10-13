<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth';
	import ChessBoard from '$lib/components/chess/ChessBoard.svelte';
	import Icon from '$lib/components/icons/Icon.svelte';
	import { executeMoveClient, type Color, type PieceType } from '$lib/chess/engine';

	let currentPuzzle: any = null;
	let currentFen = '';
	let solutionIndex = 0;
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
			statusMessage = 'Solution complete! Review the tactical pattern.';
		}
	}

	function resetToInitial() {
		if (!currentPuzzle) return;
		currentFen = currentPuzzle.fen;
		solutionIndex = 0;
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

		try {
			const check = await api.puzzles.solve(currentPuzzle.id, {
				move: uci,
				plyIndex: solutionIndex,
				isRetry: isRetrying,
				hasAssistance: hasAssistance
			});

			if (check.correct) {
				if (check.finished) {
					if (hasAssistance || isRetrying) {
						statusMessage = 'Puzzle solved with practice/assistance! (Rating unaffected)';
					} else {
						statusMessage = 'Brilliant! Puzzle completed successfully.';
						userRating = check.newRating;
						ratingDiff = check.diff;
						solvedCount++;
					}
					statusType = 'success';
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
				}
				isRetrying = true;
			}
		} catch (err: any) {
			statusMessage = 'Verification error: ' + (err.message || 'unknown');
			statusType = 'error';
		}
	}

	onMount(() => {
		if ($authStore.ratings?.puzzles) {
			userRating = Math.max(100, $authStore.ratings.puzzles);
		}
		loadRandomPuzzle();
	});
</script>

<div class="flex-1 max-w-6xl mx-auto w-full p-2 sm:px-4 sm:py-2.5 flex flex-col justify-center">
	<!-- Header -->
	<div class="mb-2 sm:mb-3 flex items-center justify-between">
		<div class="flex items-center gap-3">
			<div class="w-8 h-8 rounded-lg bg-cyan-500/15 border border-cyan-500/30 flex items-center justify-center text-cyan-400">
				<Icon name="puzzle" size={16} />
			</div>
			<div>
				<div class="flex items-center gap-2">
					<h1 class="text-lg sm:text-xl font-extrabold text-white tracking-tight">Tactical Puzzles</h1>
					<span class="text-[10px] font-mono px-2 py-0.5 rounded-full bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 font-bold">Lichess Database</span>
				</div>
				<p class="text-[11px] sm:text-xs text-slate-400">Train calculation and sharp pattern recognition</p>
			</div>
		</div>

		<button
			class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-slate-900/80 hover:bg-slate-800 border border-slate-700/60 rounded-xl text-xs font-medium text-slate-200 transition shadow-sm"
			on:click={() => (orientation = orientation === 'white' ? 'black' : 'white')}
		>
			<Icon name="rotate-cw" size={13} className="text-slate-400" />
			<span>Flip Board</span>
		</button>
	</div>

	<!-- Mode Switcher & Category / Tier Filter Bar -->
	<div class="mb-3 bg-slate-900/70 border border-slate-800/80 rounded-2xl p-2.5 backdrop-blur-xl flex flex-col md:flex-row items-center justify-between gap-3 shadow-lg">
		<!-- Left: Mode Switcher -->
		<div class="flex items-center gap-1 bg-slate-950/80 p-1 rounded-xl border border-slate-800/80 w-full md:w-auto">
			<button
				class="flex-1 md:flex-initial px-3 py-1.5 rounded-lg text-xs font-semibold transition flex items-center justify-center gap-1.5 {puzzleMode === 'practice'
					? 'bg-cyan-500 text-white shadow-md shadow-cyan-500/25'
					: 'text-slate-400 hover:text-slate-200'}"
				on:click={() => loadPuzzle('practice')}
			>
				<Icon name="puzzle" size={13} />
				<span>Practice Lab</span>
			</button>

			<button
				class="flex-1 md:flex-initial px-3 py-1.5 rounded-lg text-xs font-semibold transition flex items-center justify-center gap-1.5 {puzzleMode === 'daily'
					? 'bg-amber-500 text-slate-950 font-bold shadow-md shadow-amber-500/25'
					: 'text-slate-400 hover:text-slate-200'}"
				on:click={() => loadPuzzle('daily')}
			>
				<Icon name="flame" size={13} />
				<span>Daily Puzzle</span>
			</button>

			<button
				class="flex-1 md:flex-initial px-3 py-1.5 rounded-lg text-xs font-semibold transition flex items-center justify-center gap-1.5 {puzzleMode === 'weekly'
					? 'bg-indigo-500 text-white shadow-md shadow-indigo-500/25'
					: 'text-slate-400 hover:text-slate-200'}"
				on:click={() => loadPuzzle('weekly')}
			>
				<Icon name="trophy" size={13} />
				<span>Weekly ({weeklyList.length || 7})</span>
			</button>
		</div>

		{#if puzzleMode === 'practice'}
			<!-- Right: Category and Tier Selectors -->
			<div class="flex flex-wrap items-center gap-2 w-full md:w-auto justify-end">
				<!-- Category Selector -->
				<div class="flex items-center gap-1 bg-slate-950/60 p-1 rounded-xl border border-slate-800">
					<span class="text-[10px] font-bold text-slate-400 uppercase px-1.5">Theme:</span>
					{#each ['all', 'mate', 'capture', 'fork', 'pin', 'skewer', 'position'] as cat}
						<button
							class="px-2 py-1 rounded-lg text-[11px] font-medium transition capitalize {selectedCategory === cat
								? 'bg-sky-500/20 text-sky-300 border border-sky-500/40'
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

				<!-- Progressive Tier Selector -->
				<div class="flex items-center gap-1 bg-slate-950/60 p-1 rounded-xl border border-slate-800">
					<span class="text-[10px] font-bold text-slate-400 uppercase px-1.5">Tier:</span>
					<button
						class="px-2 py-1 rounded-lg text-[11px] font-medium transition {selectedTier === 0
							? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40'
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
							class="px-2 py-1 rounded-lg text-[11px] font-medium transition {selectedTier === tierLvl
								? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 font-bold'
								: 'text-slate-400 hover:text-slate-200'}"
							title={`Tier ${tierLvl}: ${tierLvl === 1 ? 'Novice (<900)' : tierLvl === 2 ? 'Intermediate (900-1300)' : tierLvl === 3 ? 'Advanced (1300-1650)' : tierLvl === 4 ? 'Expert (1650-2000)' : 'Master (2000+)'}`}
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
	</div>

	<div class="grid grid-cols-1 lg:grid-cols-12 gap-4 lg:gap-6 items-center justify-center my-auto">
		<!-- Left: Puzzle Board -->
		<div class="lg:col-span-7 xl:col-span-8 flex justify-center">
			<div class="w-full max-w-[min(90vw,calc(100dvh-150px),580px)] flex justify-center">
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
						interactive={!isSolutionRevealed && statusType !== 'success'}
						boardSizeClass="w-full"
						on:move={handleMove}
					/>
				{/if}
			</div>
		</div>

		<!-- Right: Puzzle Controls & Feedback -->
		<div class="lg:col-span-5 xl:col-span-4 h-[440px] lg:h-[min(calc(100dvh-150px),580px)] bg-slate-900/80 border border-slate-800/80 backdrop-blur-xl rounded-2xl p-5 shadow-2xl flex flex-col justify-between overflow-y-auto">
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
						{#if currentPuzzle.lichessId}
							<a
								href="https://lichess.org/training/{currentPuzzle.lichessId}"
								target="_blank"
								rel="noreferrer"
								class="text-[10px] text-cyan-400 hover:text-cyan-300 font-mono flex items-center gap-1"
							>
								<span>#{currentPuzzle.lichessId}</span>
								<Icon name="external-link" size={10} />
							</a>
						{/if}
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

			<!-- Status Feedback Banner -->
			<div
				class="p-4 rounded-xl border flex items-center gap-3 transition-all duration-200 {statusType === 'success'
					? 'bg-emerald-950/40 border-emerald-500/40 text-emerald-200'
					: statusType === 'error'
						? 'bg-rose-950/40 border-rose-500/40 text-rose-200'
						: 'bg-slate-950/60 border-slate-800 text-slate-300'}"
			>
				<div class="shrink-0">
					{#if statusType === 'success'}
						<div class="w-8 h-8 rounded-full bg-emerald-500/20 text-emerald-400 flex items-center justify-center">
							<Icon name="check" size={16} />
						</div>
					{:else if statusType === 'error'}
						<div class="w-8 h-8 rounded-full bg-rose-500/20 text-rose-400 flex items-center justify-center">
							<Icon name="x" size={16} />
						</div>
					{:else}
						<div class="w-8 h-8 rounded-full bg-sky-500/20 text-sky-400 flex items-center justify-center">
							<Icon name="sparkles" size={16} />
						</div>
					{/if}
				</div>
				<div class="text-xs font-semibold leading-relaxed">
					{statusMessage}
				</div>
			</div>

			<!-- Tactical Laboratory Actions -->
			<div class="flex flex-col gap-2 pt-2">
				<div class="grid grid-cols-2 gap-2">
					<button
						class="py-2.5 px-3 bg-slate-800/80 hover:bg-slate-700/80 border border-slate-700/60 text-slate-200 font-medium text-xs rounded-xl transition flex items-center justify-center gap-1.5 disabled:opacity-40"
						disabled={statusType === 'success' || isSolutionRevealed}
						on:click={requestHint}
						title="Get tactical hint without penalty"
					>
						<Icon name="sparkles" size={14} className="text-amber-400" />
						<span>{hintCount === 0 ? 'Get Hint' : 'Target Hint'}</span>
					</button>

					<button
						class="py-2.5 px-3 bg-slate-800/80 hover:bg-slate-700/80 border border-slate-700/60 text-slate-200 font-medium text-xs rounded-xl transition flex items-center justify-center gap-1.5 disabled:opacity-40"
						disabled={isSolutionRevealed}
						on:click={revealSolution}
						title="Reveal the winning move sequence"
					>
						<Icon name="eye" size={14} className="text-cyan-400" />
						<span>Show Solution</span>
					</button>
				</div>

				<div class="grid grid-cols-2 gap-2">
					<button
						class="py-2.5 px-3 bg-slate-800/80 hover:bg-slate-700/80 border border-slate-700/60 text-slate-200 font-medium text-xs rounded-xl transition flex items-center justify-center gap-1.5"
						on:click={resetToInitial}
						title="Retry the current position from the start"
					>
						<Icon name="rotate-ccw" size={14} />
						<span>Retry Puzzle</span>
					</button>

					<button
						class="py-2.5 px-3 bg-gradient-to-r from-sky-500 to-indigo-600 hover:from-sky-400 hover:to-indigo-500 text-white font-semibold text-xs rounded-xl transition duration-150 shadow-md shadow-sky-500/20 flex items-center justify-center gap-1.5"
						on:click={loadRandomPuzzle}
					>
						<span>Next Puzzle</span>
						<Icon name="arrow-right" size={14} />
					</button>
				</div>
			</div>
		</div>
	</div>
</div>
