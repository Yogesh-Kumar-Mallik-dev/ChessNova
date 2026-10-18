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

<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4 animate-in fade-in duration-150">
	<div class="relative bg-card border border-border rounded-xl p-6 max-w-[340px] w-full shadow-2xl flex flex-col items-center text-center">
		<!-- Close 'X' button to dismiss modal and view board -->
		<button
			class="absolute top-3 right-3 w-7 h-7 rounded-lg text-muted-foreground hover:text-foreground hover:bg-secondary flex items-center justify-center transition"
			on:click={() => dispatch('close')}
			title="Close and view board"
		>
			<Icon name="x" size={15} />
		</button>

		<!-- Winner Crown or Draw Emblem -->
		<div class="w-13 h-13 rounded-xl flex items-center justify-center mb-3 shadow-xs {isDraw ? 'bg-secondary text-foreground' : 'bg-amber-400/15 text-amber-400 border border-amber-400/30'}">
			{#if isDraw}
				<Icon name="handshake" size={26} />
			{:else}
				<Icon name="crown" size={26} className="text-amber-400" />
			{/if}
		</div>

		<!-- Result Title: "White Won" / "You Won!" / "Draw" -->
		<h2 class="text-xl font-bold text-foreground tracking-tight">
			{title}
		</h2>

		<!-- Subtitle: "by checkmate" -->
		<p class="text-xs font-normal text-muted-foreground mt-0.5 mb-2">
			{subtitle}
		</p>

		<!-- Score pill & Rating Diff -->
		<div class="inline-flex items-center gap-2 mb-5">
			<div class="inline-flex items-center gap-1.5 px-3 py-0.5 rounded-md bg-secondary border border-border text-[11px] font-mono font-medium text-foreground">
				<span>{result}</span>
			</div>
			{#if ratingDiff !== null && ratingDiff !== undefined}
				<div
					class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-md text-[11px] font-mono font-bold border {ratingDiff > 0
						? 'bg-emerald-950/30 border-emerald-800/40 text-emerald-400'
						: ratingDiff < 0
							? 'bg-red-950/30 border-red-800/40 text-red-400'
							: 'bg-secondary border-border text-muted-foreground'}"
				>
					<span>{ratingDiff > 0 ? `+${ratingDiff}` : `${ratingDiff}`}</span>
				</div>
			{/if}
		</div>

		<!-- Action Buttons -->
		<div class="flex flex-col gap-2 w-full">
			<!-- Primary Game Review Button -->
			<button
				class="w-full py-2.5 px-4 bg-foreground hover:opacity-90 active:scale-[0.99] text-background font-semibold text-xs rounded-lg transition-all flex items-center justify-center gap-2 shadow-xs cursor-pointer"
				on:click={() => dispatch('analyze')}
			>
				<Icon name="search" size={14} />
				<span>Game Review</span>
			</button>

			<!-- Secondary Row: Rematch & New Match -->
			<div class="grid grid-cols-2 gap-2 w-full">
				<button
					class="py-2 px-3 bg-secondary hover:bg-accent text-foreground font-medium text-xs rounded-lg border border-border transition flex items-center justify-center gap-1.5 cursor-pointer"
					on:click={() => dispatch('rematch')}
				>
					<Icon name="rotate-ccw" size={13} />
					<span>Rematch</span>
				</button>
				<button
					class="py-2 px-3 bg-secondary hover:bg-accent text-foreground font-medium text-xs rounded-lg border border-border transition flex items-center justify-center gap-1.5 cursor-pointer"
					on:click={() => dispatch('newGame')}
				>
					<Icon name="zap" size={13} />
					<span>New Match</span>
				</button>
			</div>

			<button
				class="w-full py-1 text-muted-foreground hover:text-foreground text-[11px] font-mono uppercase tracking-widest transition mt-1 cursor-pointer"
				on:click={() => dispatch('close')}
			>
				Review Board
			</button>
		</div>
	</div>
</div>
