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

	let prevActivePly: number | null = null;
	afterUpdate(() => {
		if (container && prevActivePly !== activePly) {
			prevActivePly = activePly;
			const activeEl = container.querySelector<HTMLElement>('[data-active="true"]');
			if (activeEl) {
				const containerTop = container.scrollTop;
				const containerHeight = container.clientHeight;
				const elTop = activeEl.offsetTop - container.offsetTop;
				const elHeight = activeEl.offsetHeight;

				if (elTop < containerTop) {
					container.scrollTo({ top: elTop - 8, behavior: 'smooth' });
				} else if (elTop + elHeight > containerTop + containerHeight) {
					container.scrollTo({ top: elTop + elHeight - containerHeight + 8, behavior: 'smooth' });
				}
			} else if (activePly === -1 && container.scrollHeight > container.clientHeight) {
				container.scrollTop = container.scrollHeight;
			}
		}
	});
</script>

<div class="w-full h-full flex flex-col select-none overflow-hidden">
	<!-- Ledger Column Headers -->
	<div class="grid grid-cols-[38px_1fr_1fr] items-center px-2 py-1.5 bg-secondary/50 border-b border-border text-[11px] font-medium text-muted-foreground uppercase tracking-wider shrink-0">
		<span class="text-center font-mono">#</span>
		<div class="flex items-center gap-1.5 pl-1.5">
			<span class="w-2 h-2 rounded-full bg-foreground border border-border"></span>
			<span class="truncate font-medium">{whiteName}</span>
		</div>
		<div class="flex items-center gap-1.5 pl-1.5">
			<span class="w-2 h-2 rounded-full bg-background border border-border"></span>
			<span class="truncate font-medium">{blackName}</span>
		</div>
	</div>

	<!-- Moves List Container -->
	<div bind:this={container} class="flex-1 overflow-y-auto px-1 py-1 text-sm font-sans">
		{#if turns.length === 0}
			<div class="h-full flex flex-col items-center justify-center text-xs text-muted-foreground italic gap-1 py-8">
				<span>No moves recorded</span>
				<span class="text-[10px] text-muted-foreground/60">Move pieces on the board to begin</span>
			</div>
		{:else}
			<div class="flex flex-col gap-0.5 font-mono">
				{#each turns as turn}
					{@const whitePly = (turn.turnNumber - 1) * 2 + 1}
					{@const blackPly = (turn.turnNumber - 1) * 2 + 2}
					<div class="grid grid-cols-[38px_1fr_1fr] items-center py-0.5 px-1 rounded hover:bg-secondary/40 text-xs transition-colors">
						<span class="text-muted-foreground font-mono text-[11px] font-medium text-center">{turn.turnNumber}.</span>
						
						<!-- White move -->
						<button
							data-active={activePly === whitePly}
							class="text-left font-mono font-medium px-2 py-1 rounded transition {activePly === whitePly
								? 'bg-foreground text-background font-bold shadow-2xs'
								: 'text-foreground/90 hover:bg-secondary'}"
							on:click={() => dispatch('selectPly', whitePly)}
						>
							{turn.white.san}
						</button>

						<!-- Black move -->
						{#if turn.black}
							<button
								data-active={activePly === blackPly}
								class="text-left font-mono font-medium px-2 py-1 rounded transition {activePly === blackPly
									? 'bg-foreground text-background font-bold shadow-2xs'
									: 'text-foreground/90 hover:bg-secondary'}"
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
