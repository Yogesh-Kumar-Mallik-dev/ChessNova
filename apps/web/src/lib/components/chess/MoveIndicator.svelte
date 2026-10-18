<script lang="ts">
	export let category: 'normal' | 'capture' | 'castle' | 'en_passant' = 'normal';
	export let isCapture = false;
	export let isLight = false;

	$: resolvedType = isCapture && category === 'normal' ? 'capture' : category;
</script>

{#if resolvedType === 'en_passant'}
	<!-- EN-PASSANT: High-contrast dashed perimeter ring + center dot -->
	<div class="absolute inset-0 z-20 flex items-center justify-center pointer-events-none scale-in">
		<div
			class="w-[84%] h-[84%] rounded-full border-2 sm:border-[2.5px] border-dashed flex items-center justify-center {isLight
				? 'border-neutral-900/60 shadow-[0_0_0_1px_rgba(255,255,255,0.7)]'
				: 'border-white/70 shadow-[0_0_0_1px_rgba(0,0,0,0.5)]'}"
		>
			<div
				class="w-2.5 h-2.5 sm:w-3 sm:h-3 rounded-full {isLight
					? 'bg-neutral-900/60 ring-1 ring-white/50'
					: 'bg-white/80 ring-1 ring-black/40'}"
			></div>
		</div>
	</div>
{:else if resolvedType === 'castle'}
	<!-- CASTLE: Distinct double-concentric ring indicating king/rook swap -->
	<div class="absolute inset-0 z-20 flex items-center justify-center pointer-events-none scale-in">
		<div
			class="w-[80%] h-[80%] rounded-full border-2 sm:border-[2.5px] flex items-center justify-center {isLight
				? 'border-neutral-900/50 shadow-[0_0_0_1px_rgba(255,255,255,0.6)]'
				: 'border-white/60 shadow-[0_0_0_1px_rgba(0,0,0,0.5)]'}"
		>
			<div
				class="w-3.5 h-3.5 sm:w-4 sm:h-4 rounded-full {isLight
					? 'bg-neutral-900/50 ring-1 ring-white/40'
					: 'bg-white/70 ring-1 ring-black/40'}"
			></div>
		</div>
	</div>
{:else if resolvedType === 'capture'}
	<!-- CAPTURE: High-contrast perimeter framing ring with dual-rim outline that pops against any piece and square -->
	<div class="absolute inset-0 z-20 flex items-center justify-center pointer-events-none scale-in">
		<div
			class="w-[84%] h-[84%] rounded-full border-[4.5px] sm:border-[6px] transition-transform duration-100 {isLight
				? 'border-neutral-900/45 shadow-[0_0_0_1.5px_rgba(255,255,255,0.8),inset_0_0_0_1.5px_rgba(255,255,255,0.7)]'
				: 'border-white/60 shadow-[0_0_0_1.5px_rgba(0,0,0,0.6),inset_0_0_0_1.5px_rgba(0,0,0,0.5)]'}"
		></div>
	</div>
{:else}
	<!-- NORMAL MOVE: Sized properly (~28% of square), razor-sharp contrast on both light and dark squares -->
	<div class="absolute inset-0 z-20 flex items-center justify-center pointer-events-none scale-in">
		<div
			class="w-3.5 h-3.5 xs:w-4 xs:h-4 sm:w-5 sm:h-5 rounded-full transition-transform duration-100 {isLight
				? 'bg-neutral-900/35 ring-1 ring-neutral-900/15 shadow-[0_1px_2px_rgba(0,0,0,0.2)]'
				: 'bg-white/55 ring-1 ring-black/35 shadow-[0_1px_3px_rgba(0,0,0,0.45)]'}"
		></div>
	</div>
{/if}

<style>
	.scale-in {
		animation: popIn 120ms cubic-bezier(0.16, 1, 0.3, 1) forwards;
	}

	@keyframes popIn {
		0% {
			transform: scale(0.25);
			opacity: 0;
		}
		100% {
			transform: scale(1);
			opacity: 1;
		}
	}
</style>
