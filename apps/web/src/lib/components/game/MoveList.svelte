<script lang="ts">
	import { afterUpdate, createEventDispatcher } from 'svelte';
	import type { GameMoveRecord } from '$lib/stores/game';

	export let moves: GameMoveRecord[] = [];
	export let activePly: number = -1;
	export let whiteName = 'White';
	export let blackName = 'Black';

	const dispatch = createEventDispatcher<{ selectPly: number }>();
	let container: HTMLDivElement;

	$: turns = (() => {
		const result: { turnNumber: number; white: GameMoveRecord; black?: GameMoveRecord }[] = [];
		for (let i = 0; i < moves.length; i += 2) {
			result.push({
				turnNumber: Math.floor(i / 2) + 1,
				white: moves[i],
				black: moves[i + 1]
			});
		}
		return result;
	})();

	afterUpdate(() => {
		if (container) {
			const activeEl = container.querySelector('[data-active="true"]');
			if (activeEl) {
				activeEl.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
			} else {
				container.scrollTop = container.scrollHeight;
			}
		}
	});
</script>

<div class="w-full h-full flex flex-col select-none overflow-hidden">
	<!-- Ledger Column Headers -->
	<div class="grid grid-cols-[38px_1fr_1fr] items-center px-2 py-1.5 bg-slate-950/60 border-b border-slate-800 text-[11px] font-bold text-slate-400 uppercase tracking-wider shrink-0">
		<span class="text-center font-mono">#</span>
		<div class="flex items-center gap-1.5 pl-1.5">
			<span class="w-2 h-2 rounded-full bg-white border border-slate-400"></span>
			<span class="truncate">{whiteName}</span>
		</div>
		<div class="flex items-center gap-1.5 pl-1.5">
			<span class="w-2 h-2 rounded-full bg-slate-900 border border-slate-600"></span>
			<span class="truncate">{blackName}</span>
		</div>
	</div>

	<!-- Moves List Container -->
	<div bind:this={container} class="flex-1 overflow-y-auto px-1 py-1 text-sm font-sans">
		{#if turns.length === 0}
			<div class="h-full flex flex-col items-center justify-center text-xs text-slate-500 italic gap-1 py-8">
				<span>No moves played</span>
				<span class="text-[10px] text-slate-600">Move pieces on the board to record</span>
			</div>
		{:else}
			<div class="flex flex-col gap-0.5 font-mono">
				{#each turns as turn}
					{@const whitePly = (turn.turnNumber - 1) * 2 + 1}
					{@const blackPly = (turn.turnNumber - 1) * 2 + 2}
					<div class="grid grid-cols-[38px_1fr_1fr] items-center py-0.5 px-1 rounded-lg hover:bg-slate-800/40 text-xs transition-colors">
						<span class="text-slate-500 font-mono text-[11px] font-semibold text-center">{turn.turnNumber}.</span>
						
						<!-- White move -->
						<button
							data-active={activePly === whitePly}
							class="text-left font-mono font-semibold px-2 py-1 rounded-md transition {activePly === whitePly
								? 'bg-emerald-500/20 text-emerald-300 font-bold border border-emerald-500/40 shadow-xs ring-1 ring-emerald-500/30'
								: 'text-slate-200 hover:bg-slate-800'}"
							on:click={() => dispatch('selectPly', whitePly)}
						>
							{turn.white.san}
						</button>

						<!-- Black move -->
						{#if turn.black}
							<button
								data-active={activePly === blackPly}
								class="text-left font-mono font-semibold px-2 py-1 rounded-md transition {activePly === blackPly
									? 'bg-emerald-500/20 text-emerald-300 font-bold border border-emerald-500/40 shadow-xs ring-1 ring-emerald-500/30'
									: 'text-slate-200 hover:bg-slate-800'}"
								on:click={() => dispatch('selectPly', blackPly)}
							>
								{turn.black.san}
							</button>
						{:else}
							<span></span>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>
