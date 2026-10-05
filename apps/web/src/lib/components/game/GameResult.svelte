<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Icon from '$lib/components/icons/Icon.svelte';

	export let result: string;
	export let outcome: string;
	export let winner: string;
	export let playerColor: string;

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

<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/65 backdrop-blur-[2px] p-4 animate-in fade-in duration-200">
	<div class="relative bg-[#262522] border border-[#3d3b37] rounded-2xl p-6 max-w-[340px] w-full shadow-[0_20px_50px_rgba(0,0,0,0.85)] flex flex-col items-center text-center">
		<!-- Close 'X' button to dismiss modal and view board -->
		<button
			class="absolute top-3 right-3 w-7 h-7 rounded-lg text-stone-400 hover:text-white hover:bg-stone-800 flex items-center justify-center transition"
			on:click={() => dispatch('close')}
			title="Close and view board"
		>
			<Icon name="x" size={16} />
		</button>

		<!-- Winner Crown or Draw Emblem -->
		<div class="w-14 h-14 rounded-2xl flex items-center justify-center mb-3 shadow-lg {isDraw ? 'bg-stone-800 text-stone-300' : 'bg-amber-400/15 text-amber-400 ring-1 ring-amber-400/30'}">
			{#if isDraw}
				<Icon name="handshake" size={28} />
			{:else}
				<svg class="w-8 h-8 text-[#ffc83d] fill-current" viewBox="0 0 24 24">
					<path d="M5 19h14v2H5v-2zm14.5-12c-.83 0-1.5.67-1.5 1.5 0 .24.06.47.16.67L15 11l-2.6-4.34c.37-.3.6-.76.6-1.28 0-.89-.72-1.62-1.62-1.62-.89 0-1.62.73-1.62 1.62 0 .52.23.98.6 1.28L7.38 11l-3.16-1.83c.1-.2.16-.43.16-.67 0-.83-.67-1.5-1.5-1.5S1.38 7.67 1.38 8.5c0 .76.57 1.39 1.3 1.48L4.3 17h15.4l1.62-7.02c.73-.09 1.3-.72 1.3-1.48 0-.83-.67-1.5-1.5-1.5z"/>
				</svg>
			{/if}
		</div>

		<!-- Result Title: "White Won" / "You Won!" / "Draw" -->
		<h2 class="text-2xl font-black text-white tracking-tight">
			{title}
		</h2>

		<!-- Subtitle: "by checkmate" -->
		<p class="text-xs font-medium text-stone-400 mt-0.5 mb-2">
			{subtitle}
		</p>

		<!-- Score pill -->
		<div class="inline-flex items-center gap-1.5 px-3 py-0.5 rounded-full bg-[#1e1d1b] border border-[#3d3b37] text-[11px] font-mono font-bold text-stone-300 mb-5">
			<span>{result}</span>
		</div>

		<!-- Action Buttons: Chess.com signature 3D buttons -->
		<div class="flex flex-col gap-2.5 w-full">
			<!-- Primary Green 3D Button: Game Review -->
			<button
				class="w-full py-3 px-4 bg-[#81b64c] hover:bg-[#91c65d] active:bg-[#73a443] text-white font-extrabold text-xs uppercase tracking-wide rounded-xl border-b-4 border-[#5d8336] active:border-b-0 active:translate-y-1 transition-all flex items-center justify-center gap-2 shadow-md"
				on:click={() => dispatch('analyze')}
			>
				<Icon name="search" size={16} />
				<span>Game Review</span>
			</button>

			<!-- Secondary Row: Rematch & New Match -->
			<div class="grid grid-cols-2 gap-2 w-full">
				<button
					class="py-2.5 px-3 bg-[#383633] hover:bg-[#484643] active:bg-[#2e2d2a] text-stone-200 font-bold text-xs rounded-xl border-b-4 border-[#292825] active:border-b-0 active:translate-y-1 transition-all flex items-center justify-center gap-1.5"
					on:click={() => dispatch('rematch')}
				>
					<Icon name="rotate-ccw" size={14} />
					<span>Rematch</span>
				</button>
				<button
					class="py-2.5 px-3 bg-[#383633] hover:bg-[#484643] active:bg-[#2e2d2a] text-stone-200 font-bold text-xs rounded-xl border-b-4 border-[#292825] active:border-b-0 active:translate-y-1 transition-all flex items-center justify-center gap-1.5"
					on:click={() => dispatch('newGame')}
				>
					<Icon name="zap" size={14} />
					<span>New Match</span>
				</button>
			</div>

			<button
				class="w-full py-1 text-stone-500 hover:text-stone-300 text-[11px] font-mono uppercase tracking-widest transition mt-1"
				on:click={() => dispatch('close')}
			>
				Review Board
			</button>
		</div>
	</div>
</div>
