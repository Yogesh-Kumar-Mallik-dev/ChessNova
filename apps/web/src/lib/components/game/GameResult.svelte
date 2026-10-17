<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Icon from '$lib/components/icons/Icon.svelte';

	export let result: string;
	export let outcome: string;
	export let winner: string;
	export let playerColor: string;
	export let ratingDiff: number | null = null;

	const dispatch = createEventDispatcher<{
		newGame: void;
		rematch: void;
		analyze: void;
		close: void;
	}>();

	$: didWin = (winner === 'white' && playerColor === 'white') || (winner === 'black' && playerColor === 'black');
	$: isDraw = result === '1/2-1/2';

	$: title = isDraw ? 'Draw' : didWin ? 'You Won!' : `${winner === 'white' ? 'White' : 'Black'} Won`;
	$: subtitle = (() => {
		switch (outcome) {
			case 'checkmate':
				return 'by checkmate';
			case 'resignation':
				return 'by resignation';
			case 'timeout':
				return 'by timeout';
			case 'stalemate':
				return 'by stalemate';
			case 'threefold_repetition':
				return 'by repetition';
			case 'fifty_moves':
				return 'by 50-move rule';
			case 'insufficient_material':
				return 'by insufficient material';
			case 'draw_agreed':
				return 'by agreement';
			default:
				return 'game concluded';
		}
	})();
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4 animate-in fade-in duration-200">
	<div class="relative bg-slate-900/95 border border-slate-800 rounded-2xl p-6 max-w-[340px] w-full shadow-2xl flex flex-col items-center text-center">
		<!-- Close 'X' button to dismiss modal and view board -->
		<button
			class="absolute top-3 right-3 w-7 h-7 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 flex items-center justify-center transition"
			on:click={() => dispatch('close')}
			title="Close and view board"
		>
			<Icon name="x" size={16} />
		</button>

		<!-- Winner Crown or Draw Emblem -->
		<div class="w-14 h-14 rounded-2xl flex items-center justify-center mb-3 shadow-lg {isDraw ? 'bg-slate-800 text-slate-300' : 'bg-amber-400/15 text-amber-400 ring-1 ring-amber-400/30'}">
			{#if isDraw}
				<Icon name="handshake" size={28} />
			{:else}
				<Icon name="crown" size={30} className="text-amber-400" />
			{/if}
		</div>

		<!-- Result Title: "White Won" / "You Won!" / "Draw" -->
		<h2 class="text-2xl font-black text-white tracking-tight">
			{title}
		</h2>

		<!-- Subtitle: "by checkmate" -->
		<p class="text-xs font-medium text-slate-400 mt-0.5 mb-2">
			{subtitle}
		</p>

		<!-- Score pill & Rating Diff -->
		<div class="inline-flex items-center gap-2 mb-5">
			<div class="inline-flex items-center gap-1.5 px-3 py-0.5 rounded-full bg-slate-950/80 border border-slate-800 text-[11px] font-mono font-bold text-slate-300">
				<span>{result}</span>
			</div>
			{#if ratingDiff !== null && ratingDiff !== undefined}
				<div
					class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-mono font-extrabold border {ratingDiff > 0
						? 'bg-emerald-500/15 border-emerald-500/30 text-emerald-400'
						: ratingDiff < 0
							? 'bg-rose-500/15 border-rose-500/30 text-rose-400'
							: 'bg-slate-800 border-slate-700 text-slate-400'}"
				>
					<span>{ratingDiff > 0 ? `+${ratingDiff}` : `${ratingDiff}`}</span>
				</div>
			{/if}
		</div>

		<!-- Action Buttons -->
		<div class="flex flex-col gap-2.5 w-full">
			<!-- Primary Game Review Button -->
			<button
				class="w-full py-3 px-4 bg-sky-500 hover:bg-sky-400 active:bg-sky-600 text-white font-extrabold text-xs uppercase tracking-wide rounded-xl transition-all flex items-center justify-center gap-2 shadow-lg shadow-sky-500/20"
				on:click={() => dispatch('analyze')}
			>
				<Icon name="search" size={16} />
				<span>Game Review</span>
			</button>

			<!-- Secondary Row: Rematch & New Match -->
			<div class="grid grid-cols-2 gap-2 w-full">
				<button
					class="py-2.5 px-3 bg-slate-800 hover:bg-slate-700 active:bg-slate-900 text-slate-200 font-bold text-xs rounded-xl border border-slate-700/60 transition-all flex items-center justify-center gap-1.5"
					on:click={() => dispatch('rematch')}
				>
					<Icon name="rotate-ccw" size={14} />
					<span>Rematch</span>
				</button>
				<button
					class="py-2.5 px-3 bg-slate-800 hover:bg-slate-700 active:bg-slate-900 text-slate-200 font-bold text-xs rounded-xl border border-slate-700/60 transition-all flex items-center justify-center gap-1.5"
					on:click={() => dispatch('newGame')}
				>
					<Icon name="zap" size={14} />
					<span>New Match</span>
				</button>
			</div>

			<button
				class="w-full py-1 text-slate-500 hover:text-slate-300 text-[11px] font-mono uppercase tracking-widest transition mt-1"
				on:click={() => dispatch('close')}
			>
				Review Board
			</button>
		</div>
	</div>
</div>
