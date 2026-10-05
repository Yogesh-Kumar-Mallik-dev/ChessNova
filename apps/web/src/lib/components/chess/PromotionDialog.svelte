<script lang="ts">
	import { PIECE_SVGS } from '$lib/chess/pieces';
	import type { Color, PieceType } from '$lib/chess/engine';
	import { createEventDispatcher } from 'svelte';

	export let color: Color = 'white';
	const dispatch = createEventDispatcher<{ select: PieceType }>();

	const promoOptions: { type: PieceType; label: string }[] = [
		{ type: 'q', label: 'Queen' },
		{ type: 'r', label: 'Rook' },
		{ type: 'b', label: 'Bishop' },
		{ type: 'n', label: 'Knight' }
	];

	function getPieceSvg(type: PieceType): string {
		const key = color === 'white' ? type.toUpperCase() : type.toLowerCase();
		return PIECE_SVGS[key] || '';
	}
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-md animate-in fade-in duration-200">
	<div class="bg-gradient-to-b from-[#161f30] to-[#0c121e] border border-sky-500/30 p-5 rounded-2xl shadow-[0_0_35px_rgba(56,189,248,0.25)] flex flex-col items-center">
		<h3 class="text-xs font-mono font-bold uppercase tracking-widest text-sky-400 mb-4 flex items-center gap-2">
			<span class="w-1.5 h-1.5 rounded-full bg-sky-400 animate-ping"></span>
			Promote Pawn
		</h3>
		
		<div class="flex items-center gap-3">
			{#each promoOptions as opt}
				<button
					class="w-16 h-16 sm:w-20 sm:h-20 bg-slate-800/80 hover:bg-slate-700/90 border border-slate-700/80 hover:border-sky-400/80 rounded-xl p-2.5 transition-all duration-200 flex flex-col items-center justify-center hover:scale-105 active:scale-95 shadow-lg group hover:shadow-[0_0_15px_rgba(56,189,248,0.3)]"
					on:click={() => dispatch('select', opt.type)}
					title={opt.label}
				>
					<div class="w-full h-full flex items-center justify-center group-hover:scale-110 transition-transform">
						{@html getPieceSvg(opt.type)}
					</div>
				</button>
			{/each}
		</div>
	</div>
</div>
