<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
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
	let userRating = 1200;
	let ratingDiff: number | null = null;
	let solvedCount = 0;

	async function loadRandomPuzzle() {
		loading = true;
		statusMessage = 'Find the best move in this position.';
		statusType = 'info';
		solutionIndex = 0;
		lastMove = null;
		ratingDiff = null;
		try {
			const p = await api.puzzles.getRandom();
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

	function handleMove(event: CustomEvent<{ from: string; to: string; promotion?: PieceType }>) {
		if (!currentPuzzle || statusType === 'success') return;
		const { from, to, promotion } = event.detail;
		const uci = `${from}${to}${promotion ? promotion.toLowerCase() : ''}`;

		const expected = currentPuzzle.solution[solutionIndex];

		if (expected && expected.startsWith(uci)) {
			const res = executeMoveClient(currentFen, from, to, promotion);
			if (res) {
				currentFen = res.newFen;
				lastMove = { from, to, san: res.san, isCapture: res.isCapture, isPromotion: res.isPromotion };
				turn = res.turn;
			}
			solutionIndex++;
			if (solutionIndex >= currentPuzzle.solution.length) {
				statusMessage = 'Brilliant! Puzzle completed successfully.';
				statusType = 'success';
				api.puzzles.solve(currentPuzzle.id, true).then((res) => {
					userRating = res.newRating;
					ratingDiff = res.diff;
					solvedCount++;
				}).catch(() => {});
			} else {
				statusMessage = 'Best move found! Continue the tactical sequence...';
				statusType = 'info';
				const replyUci = currentPuzzle.solution[solutionIndex];
				setTimeout(() => {
					const replyFrom = replyUci.slice(0, 2);
					const replyTo = replyUci.slice(2, 4);
					const replyPromo = replyUci.slice(4) || undefined;
					const replyRes = executeMoveClient(currentFen, replyFrom, replyTo, replyPromo);
					if (replyRes) {
						currentFen = replyRes.newFen;
						lastMove = { from: replyFrom, to: replyTo, san: replyRes.san, isCapture: replyRes.isCapture, isPromotion: replyRes.isPromotion };
						turn = replyRes.turn;
						solutionIndex++;
					}
				}, 350);
			}
		} else {
			statusMessage = 'Not the optimal move. Try looking deeper!';
			statusType = 'error';
			api.puzzles.solve(currentPuzzle.id, false).then((res) => {
				userRating = res.newRating;
				ratingDiff = res.diff;
			}).catch(() => {});
		}
	}

	onMount(() => {
		loadRandomPuzzle();
	});
</script>

<div class="flex-1 max-w-6xl mx-auto w-full p-4 sm:p-6 flex flex-col justify-center">
	<!-- Header -->
	<div class="mb-6 flex items-center justify-between">
		<div>
			<div class="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-cyan-500/10 border border-cyan-500/20 text-cyan-400 text-xs font-semibold mb-2">
				<Icon name="puzzle" size={14} />
				<span>Tactics Lab</span>
			</div>
			<h1 class="text-2xl sm:text-3xl font-extrabold text-white tracking-tight">Tactical Puzzles</h1>
			<p class="text-xs sm:text-sm text-slate-400 mt-1">Train calculation and sharp pattern recognition</p>
		</div>

		<button
			class="inline-flex items-center gap-2 px-4 py-2 bg-slate-900/80 hover:bg-slate-800 border border-slate-700/60 rounded-xl text-xs font-medium text-slate-200 transition shadow-sm"
			on:click={() => (orientation = orientation === 'white' ? 'black' : 'white')}
		>
			<Icon name="rotate-cw" size={14} className="text-slate-400" />
			<span>Flip Board</span>
		</button>
	</div>

	<div class="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start justify-center">
		<!-- Left: Puzzle Board -->
		<div class="lg:col-span-8 flex justify-center">
			{#if loading}
				<div class="w-full max-w-[560px] aspect-square bg-slate-900/60 border border-slate-800/80 rounded-2xl flex flex-col items-center justify-center text-sm text-slate-400 gap-3 shadow-2xl backdrop-blur-xl animate-pulse">
					<div class="w-8 h-8 border-2 border-sky-500/30 border-t-sky-400 rounded-full animate-spin"></div>
					<span class="text-xs font-mono text-slate-400">Loading tactical position...</span>
				</div>
			{:else}
				<ChessBoard
					fen={currentFen}
					{orientation}
					{turn}
					{lastMove}
					on:move={handleMove}
				/>
			{/if}
		</div>

		<!-- Right: Puzzle Controls & Feedback -->
		<div class="lg:col-span-4 bg-slate-900/80 border border-slate-800/80 backdrop-blur-xl rounded-2xl p-6 shadow-2xl flex flex-col gap-5">
			<div class="flex items-center justify-between pb-4 border-b border-slate-800/80">
				<div>
					<span class="text-[10px] uppercase font-bold text-sky-400 tracking-wider">Tactics Rating</span>
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
						<div class="text-lg font-mono font-extrabold text-cyan-400">
							{currentPuzzle.rating}
						</div>
						<div class="text-[10px] text-slate-500 capitalize">{turn} to move</div>
					</div>
				{/if}
			</div>

			{#if currentPuzzle && currentPuzzle.themes?.length}
				<div>
					<span class="text-[10px] uppercase font-bold text-slate-500 tracking-wider block mb-2">Tactical Themes</span>
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

			<!-- Actions -->
			<div class="flex flex-col gap-2 pt-2">
				<button
					class="w-full py-3 px-4 bg-gradient-to-r from-sky-500 to-indigo-600 hover:from-sky-400 hover:to-indigo-500 text-white font-semibold text-sm rounded-xl transition duration-150 shadow-lg shadow-sky-500/20 flex items-center justify-center gap-2"
					on:click={loadRandomPuzzle}
				>
					<span>Next Puzzle</span>
					<Icon name="arrow-right" size={16} />
				</button>

				{#if statusType === 'error'}
					<button
						class="w-full py-2.5 px-4 bg-slate-800/80 hover:bg-slate-700/80 border border-slate-700/60 text-slate-200 font-medium text-xs rounded-xl transition flex items-center justify-center gap-2"
						on:click={() => {
							if (currentPuzzle) {
								currentFen = currentPuzzle.fen;
								solutionIndex = 0;
								statusMessage = 'Try again! Find the optimal sequence.';
								statusType = 'info';
							}
						}}
					>
						<Icon name="rotate-ccw" size={14} />
						<span>Reset Position</span>
					</button>
				{/if}
			</div>
		</div>
	</div>
</div>
