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

<div class="flex-1 max-w-7xl mx-auto w-full px-2 py-1.5 sm:px-4 sm:py-2 lg:px-6 flex flex-col justify-center min-h-0 lg:h-[calc(100dvh-4rem)] lg:max-h-[calc(100dvh-4rem)] overflow-x-hidden overflow-y-auto lg:overflow-hidden">
	<!-- Slim Unified Header & Filter Toolbar -->
	<div class="mb-1.5 flex flex-wrap items-center justify-between gap-2 bg-card border border-border rounded-xl px-2.5 py-1.5 shrink-0 shadow-xs">
		<!-- Left: Mode Switcher -->
		<div class="flex items-center gap-1 bg-secondary/80 p-0.5 rounded-lg border border-border">
			<button
				class="px-2.5 py-1 rounded-md text-xs font-semibold transition flex items-center gap-1.5 {puzzleMode === 'practice'
					? 'bg-foreground text-background font-bold shadow-xs'
					: 'text-muted-foreground hover:text-foreground'}"
				on:click={() => loadPuzzle('practice')}
			>
				<Icon name="puzzle" size={13} />
				<span>Practice</span>
			</button>

			<button
				class="px-2.5 py-1 rounded-md text-xs font-semibold transition flex items-center gap-1.5 {puzzleMode === 'daily'
					? 'bg-foreground text-background font-bold shadow-xs'
					: 'text-muted-foreground hover:text-foreground'}"
				on:click={() => loadPuzzle('daily')}
			>
				<Icon name="flame" size={13} />
				<span>Daily</span>
			</button>

			<button
				class="px-2.5 py-1 rounded-md text-xs font-semibold transition flex items-center gap-1.5 {puzzleMode === 'weekly'
					? 'bg-foreground text-background font-bold shadow-xs'
					: 'text-muted-foreground hover:text-foreground'}"
				on:click={() => loadPuzzle('weekly')}
			>
				<Icon name="trophy" size={13} />
				<span class="hidden sm:inline">Weekly</span>
				<span class="text-[10px] font-mono opacity-80 tabular-nums">({weeklyList.length || 7})</span>
			</button>
		</div>

		<!-- Middle: Filters (When in practice mode) -->
		{#if puzzleMode === 'practice'}
			<!-- Mobile Category Select Dropdown -->
			<div class="sm:hidden flex items-center">
				<select
					class="bg-secondary text-foreground text-xs font-semibold py-1 px-2 rounded-lg border border-border focus:outline-none focus:ring-1 focus:ring-ring capitalize"
					value={selectedCategory}
					on:change={(e) => {
						selectedCategory = e.currentTarget.value;
						loadPuzzle('practice');
					}}
				>
					{#each ['all', 'mate', 'fork', 'pin', 'capture'] as cat}
						<option value={cat}>{cat}</option>
					{/each}
				</select>
			</div>

			<div class="hidden sm:flex items-center gap-2">
				<!-- Category Filter -->
				<div class="flex items-center gap-0.5 bg-secondary/60 p-0.5 rounded-lg border border-border">
					{#each ['all', 'mate', 'fork', 'pin', 'capture'] as cat}
						<button
							class="px-2 py-0.5 rounded text-[11px] font-medium transition capitalize {selectedCategory === cat
								? 'bg-foreground text-background font-bold shadow-xs'
								: 'text-muted-foreground hover:text-foreground'}"
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
				<div class="flex items-center gap-0.5 bg-secondary/60 p-0.5 rounded-lg border border-border">
					<button
						class="px-1.5 py-0.5 rounded text-[11px] font-medium transition {selectedTier === 0
							? 'bg-foreground text-background font-bold shadow-xs'
							: 'text-muted-foreground hover:text-foreground'}"
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
								? 'bg-foreground text-background font-bold shadow-xs'
								: 'text-muted-foreground hover:text-foreground'}"
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
			<div class="flex items-center gap-1.5 px-2 py-0.5 rounded-lg bg-secondary border border-border text-xs font-mono">
				<span class="text-muted-foreground text-[10px] uppercase font-bold">Elo:</span>
				<span class="font-bold text-foreground tabular-nums">{userRating}</span>
				{#if ratingDiff !== null}
					<span class="text-[10px] font-bold {ratingDiff > 0 ? 'text-emerald-500' : 'text-destructive'} tabular-nums">
						{ratingDiff > 0 ? `+${ratingDiff}` : ratingDiff}
					</span>
				{/if}
			</div>

			<button
				class="p-1.5 bg-secondary hover:bg-muted text-muted-foreground hover:text-foreground rounded-lg transition border border-border flex items-center justify-center"
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
			<div class="w-full max-w-[min(100%,calc(100dvh-200px),580px)] flex flex-col items-center">
				
				<!-- Turn Banner -->
				<div class="w-full flex items-center justify-between py-1 px-1 text-xs font-semibold select-none mb-1">
					<div class="flex items-center gap-2">
						<span class="w-2.5 h-2.5 rounded-full {turn === 'white' ? 'bg-foreground border border-border' : 'bg-background border border-border'}"></span>
						<span class="text-foreground text-xs font-bold capitalize">{turn} to play</span>
					</div>
					{#if currentPuzzle}
						<div class="flex items-center gap-1.5 text-[11px] font-mono text-muted-foreground">
							<span>Rating {currentPuzzle.rating}</span>
							{#if currentPuzzle.tier}
								<span class="px-1.5 py-0.2 rounded bg-secondary text-foreground border border-border font-bold">
									T{currentPuzzle.tier}
								</span>
							{/if}
						</div>
					{/if}
				</div>

				<!-- Chess Board -->
				{#if loading}
					<div class="w-full aspect-square bg-card border border-border rounded-xl flex flex-col items-center justify-center text-sm text-muted-foreground gap-3 shadow-md animate-pulse">
						<div class="w-8 h-8 border-2 border-foreground border-t-transparent rounded-full animate-spin"></div>
						<span class="text-xs font-mono text-muted-foreground">Loading tactical position...</span>
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

				<!-- Tactical Status & Immediate Action Bar -->
				<div class="w-full mt-2 flex flex-col gap-1.5">
					<!-- Status Banner -->
					<div
						class="px-3 py-2 rounded-xl border flex items-center justify-between gap-2 transition-all duration-200 {statusType === 'success'
							? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400'
							: statusType === 'error'
								? 'bg-destructive/10 border-destructive/30 text-destructive'
								: 'bg-card border-border text-foreground'}"
					>
						<div class="flex items-center gap-2 min-w-0">
							<div class="shrink-0">
								{#if statusType === 'success'}
									<div class="w-5 h-5 rounded-md bg-emerald-500/20 text-emerald-400 flex items-center justify-center border border-emerald-500/30">
										<Icon name={finishedWithMate ? 'crown' : 'check'} size={12} />
									</div>
								{:else if statusType === 'error'}
									<div class="w-5 h-5 rounded-md bg-destructive/20 text-destructive flex items-center justify-center border border-destructive/30">
										<Icon name="x" size={12} />
									</div>
								{:else}
									<div class="w-5 h-5 rounded-md bg-secondary text-foreground flex items-center justify-center border border-border">
										<Icon name="sparkles" size={12} />
									</div>
								{/if}
							</div>
							<span class="text-xs font-semibold truncate leading-tight">
								{statusMessage}
							</span>
						</div>

						{#if statusType === 'success' && ratingDiff && ratingDiff > 0}
							<span class="px-2 py-0.5 rounded-md bg-emerald-500/20 border border-emerald-500/40 text-emerald-400 font-mono font-bold text-xs shrink-0 tabular-nums">
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
								className="flex-1 font-bold justify-center gap-1.5 py-2"
								on:click={loadRandomPuzzle}
							>
								<span>Next Puzzle</span>
								<Icon name="arrow-right" size={13} />
								<span class="text-[10px] opacity-75 font-mono font-normal">(Enter)</span>
							</Button>

							<Button
								variant="outline"
								size="sm"
								on:click={resetToInitial}
								className="py-2"
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
									className="py-2"
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
								className="flex-1 py-2"
								title="Get tactical hint without penalty"
							>
								<Icon name="sparkles" size={13} className="text-muted-foreground" />
								<span>{hintCount === 0 ? 'Hint' : 'Target'}</span>
							</Button>

							<Button
								variant="outline"
								size="sm"
								disabled={isSolutionRevealed}
								on:click={revealSolution}
								className="flex-1 py-2"
								title="Reveal the winning move sequence"
							>
								<Icon name="eye" size={13} className="text-muted-foreground" />
								<span>Solution</span>
							</Button>

							<Button
								variant="outline"
								size="sm"
								on:click={resetToInitial}
								className="py-2"
								title="Reset position"
							>
								<Icon name="rotate-ccw" size={13} />
							</Button>

							<Button
								variant="outline"
								size="sm"
								on:click={loadRandomPuzzle}
								className="py-2"
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
		<div class="hidden lg:flex lg:col-span-5 xl:col-span-4 h-[min(calc(100dvh-110px),640px)] bg-card border border-border rounded-xl p-5 shadow-lg flex-col justify-between overflow-y-auto">
			<div class="flex items-center justify-between pb-4 border-b border-border">
				<div>
					<span class="text-[10px] uppercase font-bold text-muted-foreground tracking-wider">Puzzle Elo</span>
					<div class="flex items-center gap-2 mt-1">
						<span class="text-xl font-mono font-black text-foreground tabular-nums">{userRating}</span>
						{#if ratingDiff !== null}
							<span
								class="px-2 py-0.5 rounded-md text-xs font-mono font-bold {ratingDiff > 0
									? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
									: 'bg-destructive/20 text-destructive border border-destructive/30'} tabular-nums"
							>
								{ratingDiff > 0 ? `+${ratingDiff}` : ratingDiff}
							</span>
						{/if}
					</div>
					<div class="text-[10px] text-muted-foreground mt-0.5 tabular-nums">
						{solvedCount} solved today
					</div>
				</div>

				{#if currentPuzzle}
					<div class="text-right">
						<span class="text-[10px] uppercase font-bold text-muted-foreground tracking-wider">Puzzle Level</span>
						<div class="text-lg font-mono font-extrabold text-foreground flex items-center justify-end gap-1.5 tabular-nums">
							<span>{currentPuzzle.rating}</span>
							{#if currentPuzzle.tier}
								<span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-secondary text-foreground border border-border">
									T{currentPuzzle.tier}
								</span>
							{/if}
						</div>
						<div class="text-[10px] text-muted-foreground capitalize flex items-center justify-end gap-1.5 mt-0.5">
							{#if currentPuzzle.category}
								<span class="px-1.5 py-0.2 rounded bg-secondary text-muted-foreground border border-border font-semibold uppercase text-[9px]">
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
						<span class="text-[10px] uppercase font-bold text-muted-foreground tracking-wider">Tactical Themes</span>
						<span class="text-[10px] text-muted-foreground font-mono">
							#{currentPuzzle.id ? currentPuzzle.id.slice(0, 8) : 'tactics'}
						</span>
					</div>
					<div class="flex flex-wrap gap-1.5">
						{#each currentPuzzle.themes as theme}
							<span class="px-2.5 py-1 rounded-md bg-secondary border border-border text-[11px] font-medium text-foreground">
								#{theme}
							</span>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Practice Tier Quick Info -->
			<div class="p-3 bg-secondary/40 rounded-xl border border-border text-xs text-muted-foreground flex items-center justify-between">
				<div class="flex items-center gap-2">
					<Icon name="shield" size={15} />
					<span>Calculation Trainer</span>
				</div>
				<span class="font-mono text-[11px] text-muted-foreground">525+ Curated Puzzles</span>
			</div>
		</div>
	</div>
</div>
