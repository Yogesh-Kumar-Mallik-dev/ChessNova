<script lang="ts">
	import { afterUpdate, createEventDispatcher, onDestroy, onMount } from 'svelte';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import ReviewBadge from './ReviewBadge.svelte';
	import { api } from '$lib/api/client';
	import { reviewGameLocally } from '$lib/engine/localEngine';
	import {
		type GameReviewResult,
		type ReviewedMove,
		type MoveClassification
	} from '$lib/chess/review';

	export let moves: Array<{ from: string; to: string; san: string; fen: string }>;
	export let whitePlayerName = 'White';
	export let blackPlayerName = 'Black';
	export let currentPly = 0;
	export let pgn = '';

	const dispatch = createEventDispatcher<{
		selectPly: number;
		close: void;
	}>();

	export let reviewResult: GameReviewResult | null = null;
	let isAnalyzing = false;
	let progressPercent = 0;
	let progressText = 'Initializing analysis engine...';
	let progressInterval: any = null;
	let activeTab: 'review' | 'overview' = 'review';
	let isAutoPlaying = false;
	let autoPlayTimer: any = null;
	let copiedPgn = false;
	let moveContainer: HTMLDivElement;

	const classificationOrder: MoveClassification[] = [
		'brilliant',
		'great',
		'best',
		'excellent',
		'good',
		'book',
		'inaccuracy',
		'mistake',
		'miss',
		'blunder',
		'forced'
	];

	onMount(async () => {
		if ((moves && moves.length > 0) || pgn) {
			await startReview();
		}
	});

	onDestroy(() => {
		if (progressInterval) clearInterval(progressInterval);
		if (autoPlayTimer) clearInterval(autoPlayTimer);
	});

	let lastAnalyzedPgn = '';

	$: if (pgn && pgn !== lastAnalyzedPgn && !reviewResult && !isAnalyzing) {
		startReview();
	}

	export async function startReview() {
		if ((!moves || moves.length === 0) && !pgn) return;
		isAnalyzing = true;
		progressPercent = 12;
		progressText = 'Parsing game notation & moves...';
		lastAnalyzedPgn = pgn;

		if (progressInterval) clearInterval(progressInterval);

		// Multi-step progressive status simulation while Stockfish executes
		const steps = [
			{ at: 28, text: 'Identifying ECO opening theory...' },
			{ at: 48, text: 'Stockfish calculating position evaluations...' },
			{ at: 68, text: 'Classifying tactical turning points...' },
			{ at: 84, text: 'Analyzing Opening, Middlegame & Endgame phases...' },
			{ at: 93, text: 'Calibrating performance ratings to platform Elo...' }
		];
		let stepIdx = 0;
		progressInterval = setInterval(() => {
			if (stepIdx < steps.length) {
				progressPercent = steps[stepIdx].at;
				progressText = steps[stepIdx].text;
				stepIdx++;
			}
		}, 380);

		const isLocalStaged = typeof window !== 'undefined' && sessionStorage.getItem('review_is_local') === 'true';
		const hasGameId = typeof window !== 'undefined' && new URLSearchParams(window.location.search).has('gameId');

		// 1. Cost-effective and instant local Stockfish evaluation for Bot & Pass-and-Play games
		if (!hasGameId && isLocalStaged && moves && moves.length > 0) {
			if (progressInterval) clearInterval(progressInterval);
			try {
				progressText = 'Starting local Stockfish game review...';
				const localRes = await reviewGameLocally(
					moves,
					{ white: whitePlayerName, black: blackPlayerName, pgn },
					(pct, txt) => {
						progressPercent = pct;
						progressText = txt;
					}
				);
				if (localRes) {
					progressPercent = 100;
					progressText = 'Local review complete!';
					reviewResult = localRes;
					if (reviewResult?.white) whitePlayerName = reviewResult.white;
					if (reviewResult?.black) blackPlayerName = reviewResult.black;
					if (reviewResult?.pgn) pgn = reviewResult.pgn;
					return;
				}
			} catch (err) {
				console.warn('[Review] Local review error, falling back to server:', err);
			}
		}

		// 2. Authoritative server Stockfish for rated multiplayer games or fallback
		try {
			const payload = pgn ? { pgn, moves } : moves;
			const res = (await api.analysis.review(payload)) as GameReviewResult;
			if (progressInterval) clearInterval(progressInterval);
			progressPercent = 100;
			progressText = 'Analysis complete!';
			reviewResult = res;
			if (reviewResult?.white) whitePlayerName = reviewResult.white;
			if (reviewResult?.black) blackPlayerName = reviewResult.black;
			if (reviewResult?.pgn) pgn = reviewResult.pgn;
		} catch (e: any) {
			if (progressInterval) clearInterval(progressInterval);
			console.error('Game analysis failed:', e);
			progressText = `Review error: ${e?.message || 'Server error'}`;
		} finally {
			setTimeout(() => {
				isAnalyzing = false;
			}, 300);
		}
	}

	function copyPgn() {
		const textToCopy = reviewResult?.pgn || pgn;
		if (textToCopy) {
			navigator.clipboard.writeText(textToCopy);
			copiedPgn = true;
			setTimeout(() => (copiedPgn = false), 2000);
		}
	}

	function downloadPgn() {
		const text = reviewResult?.pgn || pgn;
		if (!text) return;
		const blob = new Blob([text], { type: 'application/x-chess-pgn;charset=utf-8' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `${whitePlayerName}_vs_${blackPlayerName}_${new Date().toISOString().slice(0, 10)}.pgn`;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		URL.revokeObjectURL(url);
	}

	function getEstimatedElo(color: 'white' | 'black', accuracy: number): number {
		if (reviewResult?.estimatedElo && reviewResult.estimatedElo[color]) {
			return reviewResult.estimatedElo[color];
		}
		// Calibrated fallback matching our platform 400-2850 Elo range
		const base = 400 + Math.pow(Math.max(0, accuracy - 30) / 70, 1.35) * 2200;
		return Math.min(2850, Math.max(400, Math.round(base / 10) * 10));
	}

	function jumpToNextKeyMove() {
		if (!reviewResult || reviewResult.moves.length === 0) return;
		for (let i = currentPly; i < reviewResult.moves.length; i++) {
			const c = reviewResult.moves[i].classification;
			if (['blunder', 'mistake', 'miss', 'inaccuracy', 'brilliant', 'great'].includes(c)) {
				jumpToPly(i + 1);
				return;
			}
		}
		for (let i = 0; i < currentPly; i++) {
			const c = reviewResult.moves[i].classification;
			if (['blunder', 'mistake', 'miss', 'inaccuracy', 'brilliant', 'great'].includes(c)) {
				jumpToPly(i + 1);
				return;
			}
		}
	}

	function jumpToPrevKeyMove() {
		if (!reviewResult || reviewResult.moves.length === 0) return;
		for (let i = currentPly - 2; i >= 0; i--) {
			const c = reviewResult.moves[i].classification;
			if (['blunder', 'mistake', 'miss', 'inaccuracy', 'brilliant', 'great'].includes(c)) {
				jumpToPly(i + 1);
				return;
			}
		}
	}

	$: coachQuote = (() => {
		if (!reviewResult) return '';
		const avg = (reviewResult.whiteAccuracy + reviewResult.blackAccuracy) / 2;
		if (avg >= 90) {
			return 'Masterclass game! Both players exhibited high-level positional accuracy and tactical discipline.';
		} else if (avg >= 80) {
			return 'Sharp and competitive! Solid strategic awareness with decisive turning points.';
		} else if (avg >= 70) {
			return 'An eventful contest! Notice the moments where tactical chances shifted the balance.';
		} else {
			return 'A wild, tactical clash! Stepping through the key inaccuracies will uncover big opportunities.';
		}
	})();

	$: currentReviewedMove =
		reviewResult && currentPly > 0 && currentPly <= reviewResult.moves.length
			? reviewResult.moves[currentPly - 1]
			: null;

	// Pair moves by turn number: Turn 1 (White, Black), Turn 2 (White, Black)...
	$: pairedReviewedMoves = (() => {
		if (!reviewResult || !reviewResult.moves) return [];
		const result: { turnNumber: number; white: ReviewedMove; black?: ReviewedMove }[] = [];
		for (let i = 0; i < reviewResult.moves.length; i += 2) {
			result.push({
				turnNumber: Math.floor(i / 2) + 1,
				white: reviewResult.moves[i],
				black: reviewResult.moves[i + 1]
			});
		}
		return result;
	})();

	function jumpToPly(ply: number) {
		if (ply < 0) ply = 0;
		if (ply > moves.length) ply = moves.length;
		dispatch('selectPly', ply);
	}

	function handlePrev() {
		if (currentPly > 0) jumpToPly(currentPly - 1);
	}

	function handleNext() {
		if (currentPly < moves.length) jumpToPly(currentPly + 1);
	}

	function toggleAutoPlay() {
		if (isAutoPlaying) {
			clearInterval(autoPlayTimer);
			isAutoPlaying = false;
		} else {
			isAutoPlaying = true;
			autoPlayTimer = setInterval(() => {
				if (currentPly < moves.length) {
					jumpToPly(currentPly + 1);
				} else {
					clearInterval(autoPlayTimer);
					isAutoPlaying = false;
				}
			}, 1300);
		}
	}

	function formatScore(cp: number): string {
		const pawns = (cp / 100).toFixed(1);
		return cp > 0 ? `+${pawns}` : `${pawns}`;
	}

	let prevActivePly: number | null = null;
	afterUpdate(() => {
		if (moveContainer && prevActivePly !== currentPly) {
			prevActivePly = currentPly;
			const activeBtn = moveContainer.querySelector<HTMLElement>('[data-active="true"]');
			if (activeBtn) {
				const containerTop = moveContainer.scrollTop;
				const containerHeight = moveContainer.clientHeight;
				const btnTop = activeBtn.offsetTop - moveContainer.offsetTop;
				const btnHeight = activeBtn.offsetHeight;

				if (btnTop < containerTop) {
					moveContainer.scrollTo({ top: btnTop - 8, behavior: 'smooth' });
				} else if (btnTop + btnHeight > containerTop + containerHeight) {
					moveContainer.scrollTo({ top: btnTop + btnHeight - containerHeight + 8, behavior: 'smooth' });
				}
			}
		}
	});
</script>

<div class="flex flex-col h-full bg-card border border-border rounded-xl shadow-lg overflow-hidden select-none">
	<!-- Top Navigation Tabs -->
	<div class="flex items-center justify-between border-b border-border bg-secondary/30 px-3 py-2 shrink-0">
		<div class="flex items-center gap-1.5">
			<button
				class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-all flex items-center gap-1.5 {activeTab === 'review'
					? 'bg-foreground text-background font-bold shadow-xs'
					: 'text-muted-foreground hover:text-foreground'}"
				on:click={() => (activeTab = 'review')}
			>
				<Icon name="search" size={13} />
				<span>Game Review</span>
			</button>

			<button
				class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-all flex items-center gap-1.5 {activeTab === 'overview'
					? 'bg-foreground text-background font-bold shadow-xs'
					: 'text-muted-foreground hover:text-foreground'}"
				on:click={() => (activeTab = 'overview')}
			>
				<Icon name="award" size={13} />
				<span>Overview</span>
			</button>
		</div>

		<div class="flex items-center gap-1">
			{#if reviewResult?.pgn || pgn}
				<button
					class="px-2.5 py-1 rounded-lg text-[11px] font-semibold text-foreground bg-secondary hover:bg-muted transition flex items-center gap-1 border border-border shadow-xs active:translate-y-px"
					on:click={copyPgn}
					title="Copy Annotated PGN to Clipboard"
				>
					<Icon name={copiedPgn ? 'check' : 'download'} size={12} />
					<span>{copiedPgn ? 'Copied' : 'PGN'}</span>
				</button>
				<button
					class="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-secondary transition"
					on:click={downloadPgn}
					title="Download .pgn file"
				>
					<Icon name="download" size={14} />
				</button>
			{/if}

			{#if !isAnalyzing}
				<button
					class="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-secondary transition"
					on:click={startReview}
					title="Re-run Engine Review"
				>
					<Icon name="rotate-cw" size={14} />
				</button>
			{/if}
		</div>
	</div>

	<!-- Body Content -->
	{#if isAnalyzing}
		<!-- Smooth Multi-Step Accurate Progress Bar -->
		<div class="flex-1 flex flex-col items-center justify-center p-6 gap-3 text-center">
			<div class="w-12 h-12 rounded-2xl bg-secondary border border-border flex items-center justify-center text-foreground animate-pulse shadow-xs">
				<Icon name="search" size={22} />
			</div>
			<div>
				<h3 class="text-sm font-bold text-foreground">Analyzing Game Moves</h3>
				<p class="text-xs text-muted-foreground mt-1 max-w-[280px] min-h-[32px] flex items-center justify-center">
					{progressText}
				</p>
			</div>
			<!-- Progress bar container -->
			<div class="w-full max-w-[280px] h-2 bg-secondary rounded-full overflow-hidden border border-border mt-1">
				<div
					class="h-full bg-foreground transition-all duration-300 ease-out"
					style="width: {progressPercent}%"
				></div>
			</div>
			<span class="text-xs font-mono font-bold text-foreground tabular-nums">{progressPercent}%</span>
		</div>
	{:else if !reviewResult}
		<!-- No Review Yet -->
		<div class="flex-1 flex flex-col items-center justify-center p-6 gap-3 text-center">
			<div class="w-12 h-12 rounded-2xl bg-secondary border border-border flex items-center justify-center text-muted-foreground">
				<Icon name="search" size={22} />
			</div>
			<h3 class="text-sm font-bold text-foreground">No Review Available</h3>
			<Button
				variant="primary"
				size="sm"
				className="text-xs font-bold uppercase tracking-wider"
				on:click={startReview}
			>
				<Icon name="search" size={14} />
				<span>Start Game Review</span>
			</Button>
		</div>
	{:else if activeTab === 'review'}
		<!-- MOVE-BY-MOVE REVIEW TAB: Non-nested, single-scrollbar, full-height ledger -->
		<div class="flex-1 flex flex-col min-h-0 overflow-hidden p-2.5 sm:p-3 gap-2.5">
			<!-- Header: Accuracy Comparison & Opening -->
			<div class="flex items-center justify-between px-2.5 py-1.5 bg-secondary/30 border border-border rounded-lg text-xs shrink-0">
				<div class="flex items-center gap-3">
					<div class="flex items-center gap-1.5">
						<span class="w-2.5 h-2.5 rounded-full bg-foreground border border-border"></span>
						<span class="font-bold text-foreground text-xs">{whitePlayerName}</span>
						<span class="font-mono text-[11px] font-bold text-foreground bg-secondary px-1.5 py-0.2 rounded border border-border">
							{reviewResult.whiteAccuracy}%
						</span>
					</div>
					<span class="text-muted-foreground text-xs font-mono">•</span>
					<div class="flex items-center gap-1.5">
						<span class="w-2.5 h-2.5 rounded-full bg-background border border-border"></span>
						<span class="font-bold text-foreground text-xs">{blackPlayerName}</span>
						<span class="font-mono text-[11px] font-bold text-foreground bg-secondary px-1.5 py-0.2 rounded border border-border">
							{reviewResult.blackAccuracy}%
						</span>
					</div>
				</div>
				{#if reviewResult.opening}
					<span class="text-[10px] font-mono text-muted-foreground hidden sm:inline truncate max-w-[150px]">
						{reviewResult.opening.eco} {reviewResult.opening.name}
					</span>
				{/if}
			</div>

			<!-- Current Move Coach Card -->
			{#if currentReviewedMove}
				<div class="bg-secondary/30 border border-border rounded-xl p-2.5 sm:p-3 shadow-xs flex flex-col gap-2 shrink-0 select-none">
					<!-- Row 1: Header (Badge + Move + Accuracy & Eval) -->
					<div class="flex items-center justify-between gap-2">
						<div class="flex items-center gap-2 min-w-0">
							<ReviewBadge classification={currentReviewedMove.classification} size="md" />
							<div class="flex items-center gap-1.5 truncate">
								<span class="text-xs font-bold text-foreground capitalize">
									{currentReviewedMove.classification.replace('_', ' ')}
								</span>
								<span class="text-[11px] font-mono font-bold text-muted-foreground">
									{currentReviewedMove.moveNumber}{currentReviewedMove.color === 'white' ? '.' : '...'} {currentReviewedMove.san}
								</span>
							</div>
						</div>
						<div class="flex items-center gap-1.5 shrink-0">
							<span class="px-2 py-0.5 rounded-md bg-secondary border border-border text-[11px] font-mono font-bold text-foreground tabular-nums">
								{formatScore(currentReviewedMove.evalAfter)}
							</span>
							<span class="px-2 py-0.5 rounded-md bg-secondary border border-border text-[11px] font-mono font-bold text-foreground tabular-nums">
								{currentReviewedMove.accuracy}%
							</span>
						</div>
					</div>

					<!-- Row 2: Coach Explanation -->
					<p class="text-xs text-foreground/90 leading-tight font-normal bg-card border border-border/80 px-2.5 py-1.5 rounded-lg">
						{currentReviewedMove.explanation}
					</p>

					<!-- Row 3: Best Move Comparison -->
					{#if currentReviewedMove.bestMoveSan && currentReviewedMove.bestMoveSan !== currentReviewedMove.san}
						<div class="flex items-center justify-between text-[11px] bg-secondary/60 border border-border px-2.5 py-1 rounded-lg">
							<div class="flex items-center gap-1.5 truncate">
								<ReviewBadge classification="best" size="sm" />
								<span class="text-muted-foreground font-medium">Best was:</span>
								<span class="font-mono font-bold text-foreground">{currentReviewedMove.bestMoveSan}</span>
							</div>
							<span class="font-mono text-[10px] text-muted-foreground tabular-nums">
								{formatScore(currentReviewedMove.evalBefore)}
							</span>
						</div>
					{/if}
				</div>
			{:else}
				<!-- Starting Position Card -->
				<div class="bg-secondary/30 border border-border rounded-xl p-2.5 flex items-center justify-between gap-2 shrink-0">
					<div class="flex items-center gap-2">
						<div class="w-6 h-6 rounded-full bg-secondary border border-border flex items-center justify-center text-foreground shrink-0">
							<Icon name="search" size={12} />
						</div>
						<span class="text-xs font-bold text-foreground">Starting Position</span>
					</div>
					<button
						class="font-bold text-foreground hover:underline flex items-center gap-1 text-[11px] cursor-pointer"
						on:click={() => jumpToPly(1)}
					>
						<span>Start Review</span>
						<Icon name="chevron-right" size={12} />
					</button>
				</div>
			{/if}

			<!-- Move Ledger (Takes all remaining vertical space cleanly!) -->
			<div class="bg-card border border-border rounded-xl overflow-hidden shadow-xs flex flex-col flex-1 min-h-0">
				<!-- Ledger Column Headers -->
				<div class="grid grid-cols-[38px_1fr_1fr] items-center px-3 py-1.5 bg-secondary/50 border-b border-border text-[10px] font-bold text-muted-foreground uppercase tracking-wider shrink-0">
					<span class="text-center font-mono">#</span>
					<div class="flex items-center gap-1.5 pl-1.5">
						<span class="w-2 h-2 rounded-full bg-foreground border border-border"></span>
						<span class="truncate">{whitePlayerName}</span>
					</div>
					<div class="flex items-center gap-1.5 pl-1.5">
						<span class="w-2 h-2 rounded-full bg-background border border-border"></span>
						<span class="truncate">{blackPlayerName}</span>
					</div>
				</div>

				<!-- Scrollable Paired Move Rows -->
				<div bind:this={moveContainer} class="flex-1 overflow-y-auto p-1 divide-y divide-border/60 font-mono text-xs">
					{#each pairedReviewedMoves as pair}
						{@const black = pair.black}
						<div class="grid grid-cols-[38px_1fr_1fr] items-center py-0.5 px-1 rounded-lg hover:bg-muted/40 transition">
							<!-- Turn Number -->
							<span class="text-center text-muted-foreground font-semibold text-[11px] tabular-nums">{pair.turnNumber}.</span>
							
							<!-- White Move -->
							<button
								data-active={currentPly === pair.white.ply}
								class="flex items-center gap-1.5 px-2 py-1 rounded-md text-left transition {currentPly === pair.white.ply
									? 'bg-foreground text-background font-bold shadow-xs'
									: 'text-foreground hover:bg-muted'}"
								on:click={() => jumpToPly(pair.white.ply)}
							>
								<ReviewBadge classification={pair.white.classification} size="sm" />
								<span class="font-bold">{pair.white.san}</span>
								<span class="ml-auto text-[10px] opacity-75 font-normal tabular-nums">
									{pair.white.accuracy}%
								</span>
							</button>

							<!-- Black Move -->
							{#if black}
								<button
									data-active={currentPly === black.ply}
									class="flex items-center gap-1.5 px-2 py-1 rounded-md text-left transition {currentPly === black.ply
										? 'bg-foreground text-background font-bold shadow-xs'
										: 'text-foreground hover:bg-muted'}"
									on:click={() => jumpToPly(black.ply)}
								>
									<ReviewBadge classification={black.classification} size="sm" />
									<span class="font-bold">{black.san}</span>
									<span class="ml-auto text-[10px] opacity-75 font-normal tabular-nums">
										{black.accuracy}%
									</span>
								</button>
							{:else}
								<span></span>
							{/if}
						</div>
					{/each}
				</div>
			</div>

			<!-- Bottom Quick Actions: Key Moves & Auto-Play -->
			<div class="flex items-center justify-between gap-2 p-1.5 bg-secondary/30 border border-border rounded-xl shrink-0">
				<button
					class="flex-1 py-1 px-2 rounded-lg bg-card hover:bg-muted text-foreground text-[11px] font-semibold flex items-center justify-center gap-1 transition border border-border active:translate-y-px"
					on:click={jumpToPrevKeyMove}
					title="Previous Turning Point"
				>
					<Icon name="chevron-left" size={12} />
					<span>Prev Key</span>
				</button>

				<button
					class="px-3 py-1 rounded-lg text-xs font-bold uppercase tracking-wider flex items-center justify-center gap-1.5 transition active:translate-y-px {isAutoPlaying
						? 'bg-foreground text-background shadow-xs'
						: 'bg-card hover:bg-muted text-foreground border border-border'}"
					on:click={toggleAutoPlay}
				>
					<Icon name={isAutoPlaying ? 'clock' : 'play'} size={12} />
					<span>{isAutoPlaying ? 'Pause' : 'Auto'}</span>
				</button>

				<button
					class="flex-1 py-1 px-2 rounded-lg bg-card hover:bg-muted text-foreground text-[11px] font-semibold flex items-center justify-center gap-1 transition border border-border active:translate-y-px"
					on:click={jumpToNextKeyMove}
					title="Next Turning Point"
				>
					<span>Next Key</span>
					<Icon name="chevron-right" size={12} />
				</button>
			</div>
		</div>
	{:else}
		<!-- OVERVIEW TAB -->
		<div class="flex-1 overflow-y-auto p-3.5 flex flex-col gap-3.5">
				<!-- Accuracy Comparison Cards -->
				<div class="grid grid-cols-2 gap-2.5">
					<!-- White Player -->
					<div class="bg-secondary/40 border border-border rounded-xl p-3 flex flex-col items-center text-center shadow-xs">
						<div class="flex items-center gap-1.5 text-xs font-bold text-foreground truncate max-w-full mb-1">
							<span class="w-2.5 h-2.5 rounded-full bg-foreground border border-border"></span>
							<span class="truncate">{whitePlayerName}</span>
						</div>
						<span class="text-2xl font-black text-foreground font-mono tracking-tight tabular-nums">
							{reviewResult.whiteAccuracy}%
						</span>
						<span class="text-xs font-mono font-bold text-foreground mt-0.5 tabular-nums">
							~{getEstimatedElo('white', reviewResult.whiteAccuracy)} Elo
						</span>
						<span class="text-[9px] text-muted-foreground uppercase tracking-wider font-semibold">Calibrated Platform Elo</span>
					</div>

					<!-- Black Player -->
					<div class="bg-secondary/40 border border-border rounded-xl p-3 flex flex-col items-center text-center shadow-xs">
						<div class="flex items-center gap-1.5 text-xs font-bold text-foreground truncate max-w-full mb-1">
							<span class="w-2.5 h-2.5 rounded-full bg-background border border-border"></span>
							<span class="truncate">{blackPlayerName}</span>
						</div>
						<span class="text-2xl font-black text-foreground font-mono tracking-tight tabular-nums">
							{reviewResult.blackAccuracy}%
						</span>
						<span class="text-xs font-mono font-bold text-foreground mt-0.5 tabular-nums">
							~{getEstimatedElo('black', reviewResult.blackAccuracy)} Elo
						</span>
						<span class="text-[9px] text-muted-foreground uppercase tracking-wider font-semibold">Calibrated Platform Elo</span>
					</div>
				</div>

				<!-- Game Phase Accuracies: Opening, Middlegame, Endgame -->
				{#if reviewResult.whitePhases || reviewResult.blackPhases}
					<div class="bg-secondary/40 border border-border rounded-xl p-3 shadow-xs flex flex-col gap-2">
						<div class="flex items-center justify-between text-[11px] font-bold text-foreground pb-1 border-b border-border">
							<span>Phase Accuracies</span>
							<span class="text-[10px] font-mono text-muted-foreground">Opening • Middlegame • Endgame</span>
						</div>

						<div class="grid grid-cols-3 gap-2 text-center">
							<!-- Opening Phase -->
							<div class="bg-card border border-border rounded-lg p-2 flex flex-col items-center">
								<span class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Opening</span>
								<div class="flex items-center justify-center gap-1.5 mt-1">
									<span class="text-xs font-mono font-bold text-foreground tabular-nums">
										{reviewResult.whitePhases?.opening ?? reviewResult.whiteAccuracy}%
									</span>
									<span class="text-muted-foreground">/</span>
									<span class="text-xs font-mono font-bold text-muted-foreground tabular-nums">
										{reviewResult.blackPhases?.opening ?? reviewResult.blackAccuracy}%
									</span>
								</div>
								<span class="text-[9px] text-muted-foreground mt-0.5 font-mono">W / B</span>
							</div>

							<!-- Middlegame Phase -->
							<div class="bg-card border border-border rounded-lg p-2 flex flex-col items-center">
								<span class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Middlegame</span>
								<div class="flex items-center justify-center gap-1.5 mt-1">
									<span class="text-xs font-mono font-bold text-foreground tabular-nums">
										{reviewResult.whitePhases?.middlegame ?? reviewResult.whiteAccuracy}%
									</span>
									<span class="text-muted-foreground">/</span>
									<span class="text-xs font-mono font-bold text-muted-foreground tabular-nums">
										{reviewResult.blackPhases?.middlegame ?? reviewResult.blackAccuracy}%
									</span>
								</div>
								<span class="text-[9px] text-muted-foreground mt-0.5 font-mono">W / B</span>
							</div>

							<!-- Endgame Phase -->
							<div class="bg-card border border-border rounded-lg p-2 flex flex-col items-center">
								<span class="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Endgame</span>
								<div class="flex items-center justify-center gap-1.5 mt-1">
									<span class="text-xs font-mono font-bold text-foreground tabular-nums">
										{reviewResult.whitePhases?.endgame ?? reviewResult.whiteAccuracy}%
									</span>
									<span class="text-muted-foreground">/</span>
									<span class="text-xs font-mono font-bold text-muted-foreground tabular-nums">
										{reviewResult.blackPhases?.endgame ?? reviewResult.blackAccuracy}%
									</span>
								</div>
								<span class="text-[9px] text-muted-foreground mt-0.5 font-mono">W / B</span>
							</div>
						</div>
					</div>
				{/if}

				<!-- Opening Recognition Banner -->
				{#if reviewResult.opening}
					<div class="bg-secondary/40 border border-border rounded-xl p-3 flex items-center gap-3 shadow-xs">
						<div class="w-8 h-8 rounded-lg bg-secondary border border-border flex items-center justify-center text-foreground shrink-0">
							<Icon name="puzzle" size={15} />
						</div>
						<div class="flex flex-col truncate">
							<div class="flex items-center gap-2">
								<span class="text-[10px] font-mono font-bold px-1.5 py-0.5 rounded bg-card text-foreground border border-border">
									{reviewResult.opening.eco}
								</span>
								<span class="text-xs font-bold text-foreground truncate">
									{reviewResult.opening.name}
								</span>
							</div>
							{#if reviewResult.opening.variation}
								<span class="text-[11px] text-muted-foreground truncate">
									{reviewResult.opening.variation}
								</span>
							{/if}
						</div>
					</div>
				{/if}

				<!-- Classification Breakdown Matrix -->
				<div class="bg-secondary/40 border border-border rounded-xl p-3 shadow-xs flex flex-col gap-1">
					<div class="flex items-center justify-between text-[10px] font-mono font-bold text-muted-foreground px-2 pb-1 border-b border-border">
						<span>White</span>
						<span>Classification</span>
						<span>Black</span>
					</div>

					{#each classificationOrder as type}
						{@const wCount = reviewResult.stats.white[type] || 0}
						{@const bCount = reviewResult.stats.black[type] || 0}
						{#if wCount > 0 || bCount > 0 || ['brilliant', 'great', 'best', 'inaccuracy', 'mistake', 'blunder'].includes(type)}
							<div class="flex items-center justify-between px-2 py-1 rounded-lg hover:bg-muted/40 transition text-xs">
								<!-- White Count -->
								<span class="font-mono font-bold {wCount > 0 ? 'text-foreground' : 'text-muted-foreground'} w-6 text-left tabular-nums">
									{wCount}
								</span>

								<!-- Classification Badge & Label -->
								<div class="flex items-center gap-1.5">
									<ReviewBadge classification={type} size="sm" />
									<span class="font-medium text-foreground capitalize text-xs">
										{type.replace('_', ' ')}
									</span>
								</div>

								<!-- Black Count -->
								<span class="font-mono font-bold {bCount > 0 ? 'text-foreground' : 'text-muted-foreground'} w-6 text-right tabular-nums">
									{bCount}
								</span>
							</div>
						{/if}
					{/each}
				</div>

				<!-- Advantage Graph (Centipawn curve) -->
				<div class="bg-secondary/40 border border-border rounded-xl p-3 shadow-xs flex flex-col gap-2">
					<div class="flex items-center justify-between text-xs font-bold text-foreground">
						<span>Advantage Graph</span>
						<span class="text-[10px] font-mono text-muted-foreground">White top / Black bottom</span>
					</div>

					<!-- SVG Chart -->
					<div class="relative w-full h-24 bg-card rounded-lg border border-border overflow-hidden p-1">
						<!-- Zero Center Line -->
						<div class="absolute inset-x-0 top-1/2 -translate-y-1/2 border-b border-border border-dashed"></div>

						{#if reviewResult.evalGraph.length > 0}
							{@const maxPly = reviewResult.evalGraph.length}
							<svg class="w-full h-full text-foreground" preserveAspectRatio="none" viewBox="0 0 100 100">
								<!-- Fill Area -->
								<polygon
									points={`0,50 ${reviewResult.evalGraph
										.map((pt, idx) => {
											const x = (idx / (maxPly - 1 || 1)) * 100;
											const clamped = Math.max(-600, Math.min(600, pt.scoreCp));
											const y = 50 - (clamped / 600) * 45;
											return `${x},${y}`;
										})
										.join(' ')} 100,50`}
									class="fill-foreground/10"
								/>
								<!-- Line Curve -->
								<polyline
									points={reviewResult.evalGraph
										.map((pt, idx) => {
											const x = (idx / (maxPly - 1 || 1)) * 100;
											const clamped = Math.max(-600, Math.min(600, pt.scoreCp));
											const y = 50 - (clamped / 600) * 45;
											return `${x},${y}`;
										})
										.join(' ')}
									fill="none"
									stroke="currentColor"
									stroke-width="1.8"
									stroke-linecap="round"
									stroke-linejoin="round"
								/>
							</svg>
						{/if}
					</div>
				</div>
			</div>
		{/if}
	</div>
