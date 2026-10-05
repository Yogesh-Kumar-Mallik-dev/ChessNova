<script lang="ts">
	import { afterUpdate, createEventDispatcher } from 'svelte';
	import type { GameMoveRecord } from '$lib/stores/game';

	export let moves: GameMoveRecord[] = [];
	export let activePly: number = -1;

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
			container.scrollTop = container.scrollHeight;
		}
	});
</script>

<div bind:this={container} class="w-full h-full overflow-y-auto px-2 py-1 text-sm font-sans select-none">
	{#if turns.length === 0}
		<div class="h-full flex flex-col items-center justify-center text-xs text-slate-500 italic gap-1">
			<span>No moves recorded</span>
			<span class="text-[10px] text-slate-600">Game ledger will record moves here</span>
		</div>
	{:else}
		<div class="flex flex-col gap-0.5">
			{#each turns as turn}
				{@const whitePly = (turn.turnNumber - 1) * 2 + 1}
				{@const blackPly = (turn.turnNumber - 1) * 2 + 2}
				<div class="grid grid-cols-[36px_1fr_1fr] items-center py-1 px-1.5 rounded-lg hover:bg-slate-800/50 text-xs sm:text-sm transition-colors">
					<span class="text-slate-500 font-mono text-[11px] font-semibold">{turn.turnNumber}.</span>
					
					<!-- White move -->
					<button
						class="text-left font-mono font-medium px-2 py-1 rounded-md transition {activePly === whitePly ? 'bg-sky-500/20 text-sky-300 font-bold border border-sky-500/40 shadow-sm' : 'text-slate-200 hover:bg-slate-700/60'}"
						on:click={() => dispatch('selectPly', whitePly)}
					>
						{turn.white.san}
					</button>

					<!-- Black move -->
					{#if turn.black}
						<button
							class="text-left font-mono font-medium px-2 py-1 rounded-md transition {activePly === blackPly ? 'bg-sky-500/20 text-sky-300 font-bold border border-sky-500/40 shadow-sm' : 'text-slate-200 hover:bg-slate-700/60'}"
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
