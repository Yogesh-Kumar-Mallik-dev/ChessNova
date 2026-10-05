<script lang="ts">
	export let category: 'normal' | 'capture' | 'castle' | 'en_passant' = 'normal';
	export let isCapture = false;

	$: resolvedType = isCapture && category === 'normal' ? 'capture' : category;
</script>

{#if resolvedType === 'en_passant'}
	<!-- EN-PASSANT: Distinct amber-rose dashed ring with diagonal phantom indicator -->
	<div class="absolute inset-0 z-20 flex items-center justify-center pointer-events-none scale-in">
		<div class="w-[84%] h-[84%] rounded-full border-2 border-dashed border-amber-400 dark:border-amber-400/90 ring-2 ring-amber-500/40 shadow-[0_0_12px_rgba(251,191,36,0.5)] flex items-center justify-center">
			<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="text-amber-400 animate-pulse">
				<path d="M7 17l10-10M17 7H9M17 7v8" />
			</svg>
		</div>
	</div>
{:else if resolvedType === 'castle'}
	<!-- CASTLE: Cyan-indigo dual energy ring indicating King & Rook swap -->
	<div class="absolute inset-0 z-20 flex items-center justify-center pointer-events-none scale-in">
		<div class="w-[82%] h-[82%] rounded-xl border-2 border-cyan-400 dark:border-cyan-400/80 ring-2 ring-indigo-500/40 shadow-[0_0_12px_rgba(34,211,238,0.45)] flex items-center justify-center">
			<div class="w-2.5 h-2.5 rounded-full bg-cyan-400/80"></div>
		</div>
	</div>
{:else if resolvedType === 'capture'}
	<!-- CAPTURE: High-contrast red/coral circular reticle with 4 corner bracket accents -->
	<div class="absolute inset-0 z-20 flex items-center justify-center pointer-events-none scale-in">
		<div class="relative w-[86%] h-[86%] rounded-full border-4 border-rose-500/80 dark:border-rose-400/90 ring-2 ring-rose-500/50 shadow-[0_0_14px_rgba(244,63,94,0.5)]">
			<!-- Crosshair bracket accents -->
			<span class="absolute -top-1 left-1/2 -translate-x-1/2 w-2 h-1 bg-rose-400 rounded-full"></span>
			<span class="absolute -bottom-1 left-1/2 -translate-x-1/2 w-2 h-1 bg-rose-400 rounded-full"></span>
			<span class="absolute -left-1 top-1/2 -translate-y-1/2 w-1 h-2 bg-rose-400 rounded-full"></span>
			<span class="absolute -right-1 top-1/2 -translate-y-1/2 w-1 h-2 bg-rose-400 rounded-full"></span>
		</div>
	</div>
{:else}
	<!-- NORMAL MOVE: Centered translucent celestial dot with spring pop-in -->
	<div class="absolute inset-0 z-20 flex items-center justify-center pointer-events-none scale-in">
		<div class="w-3.5 h-3.5 sm:w-4 sm:h-4 rounded-full bg-slate-900/30 dark:bg-sky-400/50 ring-1 ring-sky-300/60 shadow-[0_0_8px_rgba(56,189,248,0.45)]"></div>
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
