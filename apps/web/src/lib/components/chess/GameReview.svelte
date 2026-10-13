<script lang="ts">
	import { afterUpdate, createEventDispatcher, onDestroy, onMount } from 'svelte';
	import Icon from '$lib/components/icons/Icon.svelte';
	import ReviewBadge from './ReviewBadge.svelte';
	import { api } from '$lib/api/client';
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

	afterUpdate(() => {
		if (moveContainer) {
			const activeBtn = moveContainer.querySelector('[data-active="true"]');
			if (activeBtn) {
				activeBtn.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
			}
		}
	});
</script>

<div class="flex flex-col h-full bg-slate-900/90 border border-slate-800 rounded-2xl shadow-2xl overflow-hidden select-none backdrop-blur-xl">
	<!-- Top Navigation Tabs -->
	<div class="flex items-center justify-between border-b border-slate-800 bg-slate-950/80 px-3 py-2 shrink-0">
		<div class="flex items-center gap-1.5">
			<button
				class="px-3 py-1.5 rounded-xl text-xs font-semibold transition-all flex items-center gap-1.5 {activeTab === 'review'
					? 'bg-slate-800 text-white border border-slate-700/80 shadow-xs'
					: 'text-slate-400 hover:text-white'}"
				on:click={() => (activeTab = 'review')}
			>
				<Icon name="search" size={13} className="text-emerald-400" />
				<span>Game Review</span>
			</button>

			<button
				class="px-3 py-1.5 rounded-xl text-xs font-semibold transition-all flex items-center gap-1.5 {activeTab === 'overview'
					? 'bg-slate-800 text-white border border-slate-700/80 shadow-xs'
					: 'text-slate-400 hover:text-white'}"
				on:click={() => (activeTab = 'overview')}
			>
				<Icon name="award" size={13} className="text-amber-400" />
				<span>Overview</span>
			</button>
		</div>

		<div class="flex items-center gap-1">
			{#if reviewResult?.pgn || pgn}
				<button
					class="px-2.5 py-1 rounded-lg text-[11px] font-semibold text-slate-300 hover:text-white bg-slate-800/90 hover:bg-slate-700 transition flex items-center gap-1 border border-slate-700/60 shadow-xs active:translate-y-px"
					on:click={copyPgn}
					title="Copy Annotated PGN to Clipboard"
				>
					<Icon name={copiedPgn ? 'check' : 'download'} size={12} className={copiedPgn ? 'text-emerald-400' : ''} />
					<span>{copiedPgn ? 'Copied' : 'PGN'}</span>
				</button>
				<button
					class="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition"
					on:click={downloadPgn}
					title="Download .pgn file"
				>
					<Icon name="download" size={14} />
				</button>
			{/if}

			{#if !isAnalyzing}
				<button
					class="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition"
					on:click={startReview}
					title="Re-run Engine Review"
				>
					<Icon name="rotate-cw" size={14} />
				</button>
			{/if}
		</div>
	</div>

	<!-- Body Content -->
	<div class="flex-1 overflow-y-auto p-3.5 flex flex-col gap-3.5">
		{#if isAnalyzing}
			<!-- Smooth Multi-Step Accurate Progress Bar -->
			<div class="flex flex-col items-center justify-center py-12 gap-3 text-center">
				<div class="w-12 h-12 rounded-2xl bg-emerald-500/15 border border-emerald-500/30 flex items-center justify-center text-emerald-400 animate-pulse">
					<Icon name="search" size={22} />
				</div>
				<div>
					<h3 class="text-sm font-bold text-white">Analyzing Game Moves</h3>
					<p class="text-xs text-slate-400 mt-1 max-w-[280px] min-h-[32px] flex items-center justify-center">
						{progressText}
					</p>
				</div>
				<!-- Progress bar container -->
				<div class="w-full max-w-[280px] h-2 bg-slate-950 rounded-full overflow-hidden border border-slate-800 mt-1">
					<div
						class="h-full bg-emerald-500 transition-all duration-300 ease-out"
						style="width: {progressPercent}%"
					></div>
				</div>
				<span class="text-xs font-mono font-bold text-emerald-400">{progressPercent}%</span>
			</div>
		{:else if !reviewResult}
			<!-- No Review Yet -->
			<div class="flex flex-col items-center justify-center py-12 gap-3 text-center">
				<div class="w-12 h-12 rounded-2xl bg-slate-800 flex items-center justify-center text-slate-400">
					<Icon name="search" size={22} />
				</div>
				<h3 class="text-sm font-bold text-white">No Review Available</h3>
				<button
					class="px-4 py-2 bg-[#81b64c] hover:bg-[#72a441] active:bg-[#639037] text-white font-bold text-xs uppercase tracking-wider rounded-xl shadow-sm transition flex items-center gap-1.5"
					on:click={startReview}
				>
					<Icon name="search" size={14} />
					<span>Start Game Review</span>
				</button>
			</div>
		{:else if activeTab === 'review'}
			<!-- COACH / MOVE-BY-MOVE REVIEW TAB -->
			<div class="flex flex-col gap-3">
				<!-- Coach Avatar & Summary Speech Bubble -->
				<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-3 shadow-sm flex items-center gap-3">
					<div class="w-9 h-9 rounded-xl bg-slate-850 border border-slate-700/80 flex items-center justify-center shrink-0 text-emerald-400 shadow-xs">
						<Icon name="crown" size={18} />
					</div>
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-1.5 mb-0.5">
							<span class="text-xs font-bold text-white">Coach Nova</span>
							<span class="px-1.5 py-0.2 rounded bg-emerald-500/15 text-emerald-400 text-[9px] font-mono font-bold border border-emerald-500/30">Coach</span>
						</div>
						<p class="text-[11px] text-slate-300 leading-snug">
							{coachQuote}
						</p>
					</div>
				</div>

				<!-- Current Move Coach Card -->
				{#if currentReviewedMove}
					<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-3.5 shadow-sm flex flex-col gap-2.5">
						<!-- Header with Badge + Move Name + Eval -->
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-2">
								<ReviewBadge classification={currentReviewedMove.classification} size="md" />
								<div>
									<div class="flex items-center gap-1.5">
										<h4 class="text-xs font-bold text-white capitalize leading-tight">
											{currentReviewedMove.classification.replace('_', ' ')}
										</h4>
										{#if currentReviewedMove.phase}
											<span class="text-[9px] font-mono px-1.5 py-0.2 rounded bg-slate-800 text-slate-400 border border-slate-700/60 uppercase">
												{currentReviewedMove.phase}
											</span>
										{/if}
									</div>
									<span class="text-[11px] font-mono font-bold text-slate-400">
										Move {currentReviewedMove.moveNumber}{currentReviewedMove.color === 'white' ? '.' : '...'} {currentReviewedMove.san}
									</span>
								</div>
							</div>

							<!-- Accuracy pill & Evaluation pill -->
							<div class="flex items-center gap-1.5">
								<div class="px-2 py-0.5 rounded-md bg-slate-900 border border-slate-800 text-[10px] font-mono font-bold text-slate-300">
									{formatScore(currentReviewedMove.evalAfter)}
								</div>
								<div class="px-2 py-0.5 rounded-md bg-slate-900 border border-slate-800 text-[10px] font-mono font-bold text-emerald-400">
									{currentReviewedMove.accuracy}%
								</div>
							</div>
						</div>

						<!-- Coach Message Explanation -->
						<p class="text-xs text-slate-200 leading-relaxed font-normal bg-slate-900/80 border border-slate-850 p-2.5 rounded-lg">
							{currentReviewedMove.explanation}
						</p>

						<!-- Best Move Comparison (if move was not best) -->
						{#if currentReviewedMove.bestMoveSan && currentReviewedMove.bestMoveSan !== currentReviewedMove.san}
							<div class="flex items-center justify-between text-xs bg-emerald-950/30 border border-emerald-500/30 px-2.5 py-1.5 rounded-lg text-slate-300">
								<div class="flex items-center gap-1.5">
									<ReviewBadge classification="best" size="sm" />
									<span class="font-bold text-emerald-400">Best was:</span>
									<span class="font-mono font-bold text-white">{currentReviewedMove.bestMoveSan}</span>
								</div>
								<span class="text-[10px] text-slate-400 font-mono">
									{formatScore(currentReviewedMove.evalBefore)}
								</span>
							</div>
						{/if}
					</div>
				{:else}
					<!-- Starting position message -->
					<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-3 text-center">
						<h4 class="text-xs font-bold text-white">Starting Position</h4>
						<p class="text-[11px] text-slate-400 mt-0.5">Use the stepper or select any move below to inspect the game.</p>
					</div>
				{/if}

				<!-- Interactive Stepper & Navigation Controls -->
				<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-2.5 flex flex-col gap-2">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-1">
							<button
								class="w-8 h-8 rounded-lg bg-slate-850 hover:bg-slate-800 text-slate-300 hover:text-white flex items-center justify-center transition border border-slate-700/60 disabled:opacity-30 active:translate-y-px"
								on:click={() => jumpToPly(0)}
								disabled={currentPly <= 0}
								title="Start of Game"
							>
								<Icon name="chevrons-left" size={14} />
							</button>
							<button
								class="w-8 h-8 rounded-lg bg-slate-850 hover:bg-slate-800 text-slate-300 hover:text-white flex items-center justify-center transition border border-slate-700/60 disabled:opacity-30 active:translate-y-px"
								on:click={handlePrev}
								disabled={currentPly <= 0}
								title="Previous Move (Left Arrow)"
							>
								<Icon name="chevron-left" size={14} />
							</button>
						</div>

						<button
							class="px-3.5 py-1.5 rounded-lg text-xs font-bold uppercase tracking-wider flex items-center gap-1.5 transition active:translate-y-px {isAutoPlaying
								? 'bg-amber-600 hover:bg-amber-500 text-white shadow-xs'
								: 'bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700/60'}"
							on:click={toggleAutoPlay}
						>
							<Icon name={isAutoPlaying ? 'clock' : 'play'} size={12} />
							<span>{isAutoPlaying ? 'Pause' : 'Auto'}</span>
						</button>

						<div class="flex items-center gap-1">
							<button
								class="w-8 h-8 rounded-lg bg-slate-850 hover:bg-slate-800 text-slate-300 hover:text-white flex items-center justify-center transition border border-slate-700/60 disabled:opacity-30 active:translate-y-px"
								on:click={handleNext}
								disabled={currentPly >= moves.length}
								title="Next Move (Right Arrow)"
							>
								<Icon name="chevron-right" size={14} />
							</button>
							<button
								class="w-8 h-8 rounded-lg bg-slate-850 hover:bg-slate-800 text-slate-300 hover:text-white flex items-center justify-center transition border border-slate-700/60 disabled:opacity-30 active:translate-y-px"
								on:click={() => jumpToPly(moves.length)}
								disabled={currentPly >= moves.length}
								title="End of Game"
							>
								<Icon name="chevrons-right" size={14} />
							</button>
						</div>
					</div>

					<!-- Fast Jump To Key Moments -->
					<div class="grid grid-cols-2 gap-2 pt-1.5 border-t border-slate-855">
						<button
							class="py-1.5 px-2 rounded-lg bg-slate-900 hover:bg-slate-850 text-slate-300 hover:text-white text-[11px] font-semibold flex items-center justify-center gap-1 transition border border-slate-800 active:translate-y-px"
							on:click={jumpToPrevKeyMove}
							title="Jump to Previous Turning Point"
						>
							<Icon name="chevron-left" size={12} className="text-amber-400" />
							<span>Prev Key Move</span>
						</button>
						<button
							class="py-1.5 px-2 rounded-lg bg-slate-900 hover:bg-slate-850 text-slate-300 hover:text-white text-[11px] font-semibold flex items-center justify-center gap-1 transition border border-slate-800 active:translate-y-px"
							on:click={jumpToNextKeyMove}
							title="Jump to Next Turning Point"
						>
							<span>Next Key Move</span>
							<Icon name="chevron-right" size={12} className="text-amber-400" />
						</button>
					</div>
				</div>

				<!-- CLEAR PAIRED MOVE LEDGER: White vs Black Side by Side -->
				<div class="bg-slate-950/90 border border-slate-800 rounded-xl overflow-hidden shadow-sm flex flex-col">
					<!-- Ledger Column Headers -->
					<div class="grid grid-cols-[38px_1fr_1fr] items-center px-3 py-1.5 bg-slate-900 border-b border-slate-800 text-[10px] font-bold text-slate-400 uppercase tracking-wider shrink-0">
						<span class="text-center font-mono">#</span>
						<div class="flex items-center gap-1.5 pl-1.5">
							<span class="w-2 h-2 rounded-full bg-white border border-slate-400"></span>
							<span class="truncate">{whitePlayerName}</span>
						</div>
						<div class="flex items-center gap-1.5 pl-1.5">
							<span class="w-2 h-2 rounded-full bg-slate-900 border border-slate-600"></span>
							<span class="truncate">{blackPlayerName}</span>
						</div>
					</div>

					<!-- Scrollable Paired Move Rows -->
					<div bind:this={moveContainer} class="max-h-48 overflow-y-auto p-1 divide-y divide-slate-850/60 font-mono text-xs">
						{#each pairedReviewedMoves as pair}
							{@const black = pair.black}
							<div class="grid grid-cols-[38px_1fr_1fr] items-center py-0.5 px-1 rounded-lg hover:bg-slate-900/60 transition">
								<!-- Turn Number -->
								<span class="text-center text-slate-500 font-semibold text-[11px]">{pair.turnNumber}.</span>
								
								<!-- White Move -->
								<button
									data-active={currentPly === pair.white.ply}
									class="flex items-center gap-1.5 px-2 py-1 rounded-md text-left transition {currentPly === pair.white.ply
										? 'bg-emerald-500/20 text-emerald-300 font-bold border border-emerald-500/40 ring-1 ring-emerald-500/30 shadow-xs'
										: 'text-slate-200 hover:bg-slate-850'}"
									on:click={() => jumpToPly(pair.white.ply)}
								>
									<ReviewBadge classification={pair.white.classification} size="sm" />
									<span class="font-bold">{pair.white.san}</span>
									<span class="ml-auto text-[10px] text-slate-500 font-normal">
										{pair.white.accuracy}%
									</span>
								</button>

								<!-- Black Move -->
								{#if black}
									<button
										data-active={currentPly === black.ply}
										class="flex items-center gap-1.5 px-2 py-1 rounded-md text-left transition {currentPly === black.ply
											? 'bg-emerald-500/20 text-emerald-300 font-bold border border-emerald-500/40 ring-1 ring-emerald-500/30 shadow-xs'
											: 'text-slate-200 hover:bg-slate-850'}"
										on:click={() => jumpToPly(black.ply)}
									>
										<ReviewBadge classification={black.classification} size="sm" />
										<span class="font-bold">{black.san}</span>
										<span class="ml-auto text-[10px] text-slate-500 font-normal">
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
			</div>
		{:else}
			<!-- OVERVIEW TAB -->
			<div class="flex flex-col gap-3.5">
				<!-- Accuracy Comparison Cards -->
				<div class="grid grid-cols-2 gap-2.5">
					<!-- White Player -->
					<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-3 flex flex-col items-center text-center shadow-xs">
						<div class="flex items-center gap-1.5 text-xs font-bold text-slate-300 truncate max-w-full mb-1">
							<span class="w-2.5 h-2.5 rounded-full bg-white border border-slate-400"></span>
							<span class="truncate">{whitePlayerName}</span>
						</div>
						<span class="text-2xl font-black text-white font-mono tracking-tight">
							{reviewResult.whiteAccuracy}%
						</span>
						<span class="text-xs font-mono font-bold text-emerald-400 mt-0.5">
							~{getEstimatedElo('white', reviewResult.whiteAccuracy)} Elo
						</span>
						<span class="text-[9px] text-slate-500 uppercase tracking-wider font-semibold">Calibrated Platform Elo</span>
					</div>

					<!-- Black Player -->
					<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-3 flex flex-col items-center text-center shadow-xs">
						<div class="flex items-center gap-1.5 text-xs font-bold text-slate-300 truncate max-w-full mb-1">
							<span class="w-2.5 h-2.5 rounded-full bg-slate-900 border border-slate-600"></span>
							<span class="truncate">{blackPlayerName}</span>
						</div>
						<span class="text-2xl font-black text-white font-mono tracking-tight">
							{reviewResult.blackAccuracy}%
						</span>
						<span class="text-xs font-mono font-bold text-emerald-400 mt-0.5">
							~{getEstimatedElo('black', reviewResult.blackAccuracy)} Elo
						</span>
						<span class="text-[9px] text-slate-500 uppercase tracking-wider font-semibold">Calibrated Platform Elo</span>
					</div>
				</div>

				<!-- Game Phase Accuracies: Opening, Middlegame, Endgame -->
				{#if reviewResult.whitePhases || reviewResult.blackPhases}
					<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-3 shadow-xs flex flex-col gap-2">
						<div class="flex items-center justify-between text-[11px] font-bold text-slate-300 pb-1 border-b border-slate-850">
							<span>Phase Accuracies</span>
							<span class="text-[10px] font-mono text-slate-500">Opening • Middlegame • Endgame</span>
						</div>

						<div class="grid grid-cols-3 gap-2 text-center">
							<!-- Opening Phase -->
							<div class="bg-slate-900/80 border border-slate-850 rounded-lg p-2 flex flex-col items-center">
								<span class="text-[10px] font-bold uppercase tracking-wider text-slate-400">Opening</span>
								<div class="flex items-center justify-center gap-1.5 mt-1">
									<span class="text-xs font-mono font-bold text-white">
										{reviewResult.whitePhases?.opening ?? reviewResult.whiteAccuracy}%
									</span>
									<span class="text-slate-600">/</span>
									<span class="text-xs font-mono font-bold text-slate-300">
										{reviewResult.blackPhases?.opening ?? reviewResult.blackAccuracy}%
									</span>
								</div>
								<span class="text-[9px] text-slate-500 mt-0.5 font-mono">W / B</span>
							</div>

							<!-- Middlegame Phase -->
							<div class="bg-slate-900/80 border border-slate-850 rounded-lg p-2 flex flex-col items-center">
								<span class="text-[10px] font-bold uppercase tracking-wider text-slate-400">Middlegame</span>
								<div class="flex items-center justify-center gap-1.5 mt-1">
									<span class="text-xs font-mono font-bold text-white">
										{reviewResult.whitePhases?.middlegame ?? reviewResult.whiteAccuracy}%
									</span>
									<span class="text-slate-600">/</span>
									<span class="text-xs font-mono font-bold text-slate-300">
										{reviewResult.blackPhases?.middlegame ?? reviewResult.blackAccuracy}%
									</span>
								</div>
								<span class="text-[9px] text-slate-500 mt-0.5 font-mono">W / B</span>
							</div>

							<!-- Endgame Phase -->
							<div class="bg-slate-900/80 border border-slate-850 rounded-lg p-2 flex flex-col items-center">
								<span class="text-[10px] font-bold uppercase tracking-wider text-slate-400">Endgame</span>
								<div class="flex items-center justify-center gap-1.5 mt-1">
									<span class="text-xs font-mono font-bold text-white">
										{reviewResult.whitePhases?.endgame ?? reviewResult.whiteAccuracy}%
									</span>
									<span class="text-slate-600">/</span>
									<span class="text-xs font-mono font-bold text-slate-300">
										{reviewResult.blackPhases?.endgame ?? reviewResult.blackAccuracy}%
									</span>
								</div>
								<span class="text-[9px] text-slate-500 mt-0.5 font-mono">W / B</span>
							</div>
						</div>
					</div>
				{/if}

				<!-- Opening Recognition Banner -->
				{#if reviewResult.opening}
					<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-3 flex items-center gap-3 shadow-xs">
						<div class="w-8 h-8 rounded-lg bg-emerald-500/15 border border-emerald-500/30 flex items-center justify-center text-emerald-400 shrink-0">
							<Icon name="puzzle" size={15} />
						</div>
						<div class="flex flex-col truncate">
							<div class="flex items-center gap-2">
								<span class="text-[10px] font-mono font-bold px-1.5 py-0.5 rounded bg-slate-850 text-emerald-400 border border-slate-750">
									{reviewResult.opening.eco}
								</span>
								<span class="text-xs font-bold text-white truncate">
									{reviewResult.opening.name}
								</span>
							</div>
							{#if reviewResult.opening.variation}
								<span class="text-[11px] text-slate-400 truncate">
									{reviewResult.opening.variation}
								</span>
							{/if}
						</div>
					</div>
				{/if}

				<!-- Classification Breakdown Matrix -->
				<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-3 shadow-xs flex flex-col gap-1">
					<div class="flex items-center justify-between text-[10px] font-mono font-bold text-slate-400 px-2 pb-1 border-b border-slate-850">
						<span>White</span>
						<span>Classification</span>
						<span>Black</span>
					</div>

					{#each classificationOrder as type}
						{@const wCount = reviewResult.stats.white[type] || 0}
						{@const bCount = reviewResult.stats.black[type] || 0}
						{#if wCount > 0 || bCount > 0 || ['brilliant', 'great', 'best', 'inaccuracy', 'mistake', 'blunder'].includes(type)}
							<div class="flex items-center justify-between px-2 py-1 rounded-lg hover:bg-slate-900 transition text-xs">
								<!-- White Count -->
								<span class="font-mono font-bold {wCount > 0 ? 'text-white' : 'text-slate-600'} w-6 text-left">
									{wCount}
								</span>

								<!-- Classification Badge & Label -->
								<div class="flex items-center gap-1.5">
									<ReviewBadge classification={type} size="sm" />
									<span class="font-medium text-slate-300 capitalize text-xs">
										{type.replace('_', ' ')}
									</span>
								</div>

								<!-- Black Count -->
								<span class="font-mono font-bold {bCount > 0 ? 'text-white' : 'text-slate-600'} w-6 text-right">
									{bCount}
								</span>
							</div>
						{/if}
					{/each}
				</div>

				<!-- Advantage Graph (Centipawn curve) -->
				<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-3 shadow-xs flex flex-col gap-2">
					<div class="flex items-center justify-between text-xs font-bold text-slate-300">
						<span>Advantage Graph</span>
						<span class="text-[10px] font-mono text-slate-500">White top / Black bottom</span>
					</div>

					<!-- SVG Chart -->
					<div class="relative w-full h-24 bg-slate-950 rounded-lg border border-slate-850 overflow-hidden p-1">
						<!-- Zero Center Line -->
						<div class="absolute inset-x-0 top-1/2 -translate-y-1/2 border-b border-slate-750/50 border-dashed"></div>

						{#if reviewResult.evalGraph.length > 0}
							{@const maxPly = reviewResult.evalGraph.length}
							<svg class="w-full h-full" preserveAspectRatio="none" viewBox="0 0 100 100">
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
									fill="rgba(129, 182, 76, 0.18)"
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
									stroke="#81b64c"
									stroke-width="2"
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
</div>
