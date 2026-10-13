<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import Icon from '$lib/components/icons/Icon.svelte';
	import ReviewBadge from './ReviewBadge.svelte';
	import { api } from '$lib/api/client';
	import {
		performGameReview,
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
	let progressText = 'Authoritative Stockfish Server Reviewing Game...';
	let activeTab: 'review' | 'overview' = 'review';
	let isAutoPlaying = false;
	let autoPlayTimer: any = null;
	let copiedPgn = false;

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

	let lastAnalyzedPgn = '';

	$: if (pgn && pgn !== lastAnalyzedPgn && !reviewResult && !isAnalyzing) {
		startReview();
	}

	export async function startReview() {
		if ((!moves || moves.length === 0) && !pgn) return;
		isAnalyzing = true;
		progressPercent = 20;
		progressText = 'Authoritative Stockfish Server Reviewing Game...';
		lastAnalyzedPgn = pgn;


		try {
			progressPercent = 60;
			const payload = pgn ? { pgn, moves } : moves;
			reviewResult = (await api.analysis.review(payload)) as GameReviewResult;
			if (reviewResult?.white) whitePlayerName = reviewResult.white;
			if (reviewResult?.black) blackPlayerName = reviewResult.black;
			if (reviewResult?.pgn) pgn = reviewResult.pgn;
			progressPercent = 100;
		} catch (e: any) {
			console.error('Authoritative server review failed:', e);
			progressText = `Review error: ${e?.message || 'Server error'}`;
		} finally {
			isAnalyzing = false;
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

	function estimateElo(accuracy: number): number {
		if (accuracy >= 98) return 2600;
		if (accuracy >= 95) return 2350;
		if (accuracy >= 90) return 2100;
		if (accuracy >= 85) return 1850;
		if (accuracy >= 80) return 1650;
		if (accuracy >= 75) return 1450;
		if (accuracy >= 70) return 1250;
		if (accuracy >= 60) return 1000;
		return Math.max(500, Math.round(accuracy * 12));
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
			return 'Grandmaster level game! Both sides executed high-caliber positional and tactical ideas.';
		} else if (avg >= 80) {
			return 'Sharp and competitive! Solid strategic awareness with decisive turning points.';
		} else if (avg >= 70) {
			return 'An eventful battle! Notice the tactical moments where the evaluation swung.';
		} else {
			return 'A wild, tactical clash! Stepping through the key inaccuracies will uncover big opportunities.';
		}
	})();

	$: currentReviewedMove =
		reviewResult && currentPly > 0 && currentPly <= reviewResult.moves.length
			? reviewResult.moves[currentPly - 1]
			: null;

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
</script>

<div class="flex flex-col h-full bg-slate-900/90 border border-slate-800 rounded-2xl shadow-2xl overflow-hidden select-none backdrop-blur-xl">
	<!-- Top Navigation Tabs -->
	<div class="flex items-center justify-between border-b border-slate-800 bg-slate-950/80 px-3 py-2 shrink-0">
		<div class="flex items-center gap-1.5">
			<button
				class="px-3.5 py-1.5 rounded-xl text-xs font-extrabold transition-all flex items-center gap-1.5 {activeTab === 'review'
					? 'bg-sky-500 text-white shadow-md shadow-sky-500/20'
					: 'text-slate-400 hover:text-white'}"
				on:click={() => (activeTab = 'review')}
			>
				<Icon name="search" size={14} />
				<span>Coach Review</span>
			</button>

			<button
				class="px-3.5 py-1.5 rounded-xl text-xs font-extrabold transition-all flex items-center gap-1.5 {activeTab === 'overview'
					? 'bg-sky-500 text-white shadow-md shadow-sky-500/20'
					: 'text-slate-400 hover:text-white'}"
				on:click={() => (activeTab = 'overview')}
			>
				<Icon name="award" size={14} />
				<span>Overview</span>
			</button>
		</div>

		<div class="flex items-center gap-1">
			{#if reviewResult?.pgn || pgn}
				<button
					class="px-2 py-1 rounded-lg text-[11px] font-bold text-slate-300 hover:text-white bg-slate-800 hover:bg-slate-700 transition flex items-center gap-1 border border-slate-700/60"
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
	<div class="flex-1 overflow-y-auto p-4 flex flex-col gap-4">
		{#if isAnalyzing}
			<!-- Progress Bar while Analyzing -->
			<div class="flex flex-col items-center justify-center py-12 gap-3 text-center">
				<div class="w-12 h-12 rounded-2xl bg-sky-500/20 border border-sky-500/40 flex items-center justify-center text-sky-400 animate-pulse">
					<Icon name="search" size={24} />
				</div>
				<div>
					<h3 class="text-base font-bold text-white">Analyzing Game Moves</h3>
					<p class="text-xs text-slate-400 mt-1 max-w-[260px]">{progressText}</p>
				</div>
				<!-- Progress bar container -->
				<div class="w-full max-w-[280px] h-2.5 bg-slate-950 rounded-full overflow-hidden border border-slate-800 mt-2">
					<div
						class="h-full bg-gradient-to-r from-sky-500 to-indigo-500 transition-all duration-150"
						style="width: {progressPercent}%"
					></div>
				</div>
				<span class="text-xs font-mono font-bold text-sky-400">{progressPercent}%</span>
			</div>
		{:else if !reviewResult}
			<!-- No Review Yet -->
			<div class="flex flex-col items-center justify-center py-12 gap-3 text-center">
				<div class="w-12 h-12 rounded-2xl bg-slate-800 flex items-center justify-center text-slate-400">
					<Icon name="search" size={24} />
				</div>
				<h3 class="text-sm font-bold text-white">No Review Available</h3>
				<button
					class="px-4 py-2 bg-sky-500 hover:bg-sky-400 active:bg-sky-600 text-white font-extrabold text-xs uppercase tracking-wider rounded-xl shadow-md transition flex items-center gap-1.5"
					on:click={startReview}
				>
					<Icon name="search" size={14} />
					<span>Start Game Review</span>
				</button>
			</div>
		{:else if activeTab === 'review'}
			<!-- COACH / MOVE-BY-MOVE REVIEW TAB -->
			<div class="flex flex-col gap-3.5">
				<!-- Coach Avatar & Summary Speech Bubble -->
				<div class="bg-slate-950/80 border border-slate-800 rounded-2xl p-3 shadow-md flex items-center gap-3">
					<div class="w-10 h-10 rounded-2xl bg-gradient-to-tr from-sky-500 to-indigo-600 border border-sky-400/40 flex items-center justify-center shrink-0 shadow-[0_0_12px_rgba(56,189,248,0.3)]">
						<Icon name="crown" size={18} className="text-white drop-shadow" />
					</div>
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-2 mb-0.5">
							<span class="text-xs font-black text-white">Coach Nova</span>
							<span class="px-1.5 py-0.2 rounded bg-sky-500/20 text-sky-400 text-[10px] font-mono font-bold border border-sky-500/30">Coach</span>
						</div>
						<p class="text-[11px] text-slate-300 leading-snug">
							{coachQuote}
						</p>
					</div>
				</div>

				<!-- Current Move Coach Card -->
				{#if currentReviewedMove}
					<div class="bg-slate-950/80 border border-slate-800 rounded-2xl p-4 shadow-lg flex flex-col gap-3">
						<!-- Header with Badge + Classification -->
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-2">
								<ReviewBadge classification={currentReviewedMove.classification} size="md" />
								<div>
									<h4 class="text-sm font-extrabold text-white capitalize leading-tight">
										{currentReviewedMove.classification.replace('_', ' ')}
									</h4>
									<span class="text-[11px] font-mono font-bold text-slate-400">
										Move {currentReviewedMove.moveNumber}{currentReviewedMove.color === 'white' ? '.' : '...'} {currentReviewedMove.san}
									</span>
								</div>
							</div>

							<!-- Accuracy pill & Evaluation pill -->
							<div class="flex items-center gap-1.5">
								<div class="px-2 py-0.5 rounded-full bg-slate-900 border border-slate-700/60 text-[10px] font-mono font-bold text-slate-300">
									{formatScore(currentReviewedMove.evalAfter)}
								</div>
								<div class="px-2.5 py-1 rounded-full bg-slate-900 border border-slate-700/60 text-[11px] font-mono font-bold text-slate-300">
									{currentReviewedMove.accuracy}% acc
								</div>
							</div>
						</div>

						<!-- Coach Message Explanation -->
						<p class="text-xs text-slate-200 leading-relaxed font-medium bg-slate-900/90 border border-slate-800 p-2.5 rounded-xl">
							{currentReviewedMove.explanation}
						</p>

						<!-- Best Move Comparison (if move was not best) -->
						{#if currentReviewedMove.bestMoveSan && currentReviewedMove.bestMoveSan !== currentReviewedMove.san}
							<div class="flex items-center justify-between text-xs bg-emerald-950/40 border border-emerald-500/30 px-3 py-2 rounded-xl text-slate-300">
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
					<div class="bg-slate-950/80 border border-slate-800 rounded-2xl p-4 text-center">
						<h4 class="text-sm font-bold text-white">Starting Position</h4>
						<p class="text-xs text-slate-400 mt-1">Step forward to review each move with the Coach.</p>
					</div>
				{/if}

				<!-- Interactive Move Navigation Controls -->
				<div class="bg-slate-950/80 border border-slate-800 rounded-2xl p-3 flex flex-col gap-2.5">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-1.5">
							<button
								class="w-8 h-8 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white flex items-center justify-center transition border border-slate-700/60 disabled:opacity-40"
								on:click={() => jumpToPly(0)}
								disabled={currentPly <= 0}
								title="Start of Game"
							>
								<Icon name="chevrons-left" size={15} />
							</button>
							<button
								class="w-8 h-8 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white flex items-center justify-center transition border border-slate-700/60 disabled:opacity-40"
								on:click={handlePrev}
								disabled={currentPly <= 0}
								title="Previous Move (Left Arrow)"
							>
								<Icon name="chevron-left" size={15} />
							</button>
						</div>

						<button
							class="px-4 py-2 rounded-xl text-xs font-extrabold uppercase tracking-wide flex items-center gap-1.5 transition {isAutoPlaying
								? 'bg-amber-600 hover:bg-amber-500 text-white shadow-md'
								: 'bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700/60'}"
							on:click={toggleAutoPlay}
						>
							<Icon name={isAutoPlaying ? 'clock' : 'play'} size={13} />
							<span>{isAutoPlaying ? 'Pause' : 'Auto'}</span>
						</button>

						<div class="flex items-center gap-1.5">
							<button
								class="w-8 h-8 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white flex items-center justify-center transition border border-slate-700/60 disabled:opacity-40"
								on:click={handleNext}
								disabled={currentPly >= moves.length}
								title="Next Move (Right Arrow)"
							>
								<Icon name="chevron-right" size={15} />
							</button>
							<button
								class="w-8 h-8 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white flex items-center justify-center transition border border-slate-700/60 disabled:opacity-40"
								on:click={() => jumpToPly(moves.length)}
								disabled={currentPly >= moves.length}
								title="End of Game"
							>
								<Icon name="chevrons-right" size={15} />
							</button>
						</div>
					</div>

					<!-- Fast Jump To Key Moments -->
					<div class="flex items-center justify-between gap-2 pt-2 border-t border-slate-800">
						<button
							class="flex-1 py-1.5 px-2 rounded-xl bg-slate-900/90 hover:bg-slate-800 text-slate-300 hover:text-white text-[11px] font-bold flex items-center justify-center gap-1.5 transition border border-slate-800"
							on:click={jumpToPrevKeyMove}
							title="Jump to Previous Key Move / Mistake"
						>
							<Icon name="chevron-left" size={13} className="text-amber-400" />
							<span>Prev Key Move</span>
						</button>
						<button
							class="flex-1 py-1.5 px-2 rounded-xl bg-slate-900/90 hover:bg-slate-800 text-slate-300 hover:text-white text-[11px] font-bold flex items-center justify-center gap-1.5 transition border border-slate-800"
							on:click={jumpToNextKeyMove}
							title="Jump to Next Key Move / Mistake"
						>
							<span>Next Key Move</span>
							<Icon name="chevron-right" size={13} className="text-amber-400" />
						</button>
					</div>
				</div>

				<!-- Mini Move Strip -->
				<div class="flex flex-wrap gap-1 max-h-36 overflow-y-auto p-2 bg-slate-950/80 rounded-xl border border-slate-800">
					{#each reviewResult.moves as rm}
						<button
							class="px-2 py-1 rounded-lg text-xs font-mono font-bold flex items-center gap-1 transition {currentPly === rm.ply
								? 'bg-sky-500 text-white shadow'
								: 'bg-slate-900 text-slate-300 hover:bg-slate-800'}"
							on:click={() => jumpToPly(rm.ply)}
						>
							<ReviewBadge classification={rm.classification} size="sm" />
							<span>{rm.san}</span>
						</button>
					{/each}
				</div>
			</div>
		{:else}
			<!-- OVERVIEW TAB -->
			<div class="flex flex-col gap-4">
				<!-- Accuracy Comparison Cards -->
				<div class="grid grid-cols-2 gap-3">
					<!-- White Player -->
					<div class="bg-slate-950/80 border border-slate-800 rounded-2xl p-3 flex flex-col items-center text-center shadow">
						<div class="flex items-center gap-1.5 text-xs font-bold text-slate-300 truncate max-w-full mb-1">
							<span class="w-2.5 h-2.5 rounded-full bg-white border border-slate-400"></span>
							<span class="truncate">{whitePlayerName}</span>
						</div>
						<span class="text-2xl font-black text-white font-mono tracking-tight">
							{reviewResult.whiteAccuracy}%
						</span>
						<span class="text-xs font-mono font-bold text-sky-400 mt-0.5">
							~{estimateElo(reviewResult.whiteAccuracy)} Elo
						</span>
						<span class="text-[9px] text-slate-400 uppercase tracking-widest font-bold">Accuracy & Rating</span>
					</div>

					<!-- Black Player -->
					<div class="bg-slate-950/80 border border-slate-800 rounded-2xl p-3 flex flex-col items-center text-center shadow">
						<div class="flex items-center gap-1.5 text-xs font-bold text-slate-300 truncate max-w-full mb-1">
							<span class="w-2.5 h-2.5 rounded-full bg-slate-900 border border-slate-600"></span>
							<span class="truncate">{blackPlayerName}</span>
						</div>
						<span class="text-2xl font-black text-white font-mono tracking-tight">
							{reviewResult.blackAccuracy}%
						</span>
						<span class="text-xs font-mono font-bold text-sky-400 mt-0.5">
							~{estimateElo(reviewResult.blackAccuracy)} Elo
						</span>
						<span class="text-[9px] text-slate-400 uppercase tracking-widest font-bold">Accuracy & Rating</span>
					</div>
				</div>

				<!-- Opening Recognition Banner -->
				{#if reviewResult.opening}
					<div class="bg-slate-950/80 border border-slate-800 rounded-xl p-3 flex items-center gap-3 shadow">
						<div class="w-8 h-8 rounded-lg bg-sky-500/15 border border-sky-500/30 flex items-center justify-center text-sky-400">
							<Icon name="puzzle" size={16} />
						</div>
						<div class="flex flex-col truncate">
							<div class="flex items-center gap-2">
								<span class="text-[10px] font-mono font-extrabold px-1.5 py-0.5 rounded bg-slate-800 text-sky-400 border border-slate-700/60">
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
				<div class="bg-slate-950/80 border border-slate-800 rounded-2xl p-3 shadow flex flex-col gap-1.5">
					<div class="flex items-center justify-between text-[11px] font-mono font-bold text-slate-400 px-2 pb-1 border-b border-slate-800">
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
									<span class="font-bold text-slate-300 capitalize text-xs">
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
				<div class="bg-slate-950/80 border border-slate-800 rounded-2xl p-3 shadow flex flex-col gap-2">
					<div class="flex items-center justify-between text-xs font-bold text-slate-300">
						<span>Advantage Graph</span>
						<span class="text-[10px] font-mono text-slate-400">White top / Black bottom</span>
					</div>

					<!-- SVG Chart -->
					<div class="relative w-full h-24 bg-slate-950 rounded-xl border border-slate-800 overflow-hidden p-1">
						<!-- Zero Center Line -->
						<div class="absolute inset-x-0 top-1/2 -translate-y-1/2 border-b border-slate-700/50 border-dashed"></div>

						{#if reviewResult.evalGraph.length > 0}
							{@const maxPly = reviewResult.evalGraph.length}
							<svg class="w-full h-full" preserveAspectRatio="none" viewBox="0 0 100 100">
								<!-- Fill Area -->
								<polygon
									points={`0,50 ${reviewResult.evalGraph
										.map((pt, idx) => {
											const x = (idx / (maxPly - 1 || 1)) * 100;
											// Clamp eval between -600 and +600 for chart
											const clamped = Math.max(-600, Math.min(600, pt.scoreCp));
											const y = 50 - (clamped / 600) * 45;
											return `${x},${y}`;
										})
										.join(' ')} 100,50`}
									fill="rgba(56, 189, 248, 0.2)"
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
									stroke="#38bdf8"
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
