<script lang="ts">
	export let category: 'normal' | 'capture' | 'castle' | 'en_passant' = 'normal';
	export let isCapture = false;

	$: resolvedType = isCapture && category === 'normal' ? 'capture' : category;
</script>

{#if resolvedType === 'en_passant'}
	<!-- EN-PASSANT: Refined dashed ring -->
	<div class="absolute inset-0 z-20 flex items-center justify-center pointer-events-none scale-in">
		<div class="w-[84%] h-[84%] rounded-full border-2 border-dashed border-foreground/40 flex items-center justify-center">
			<div class="w-2 h-2 rounded-full bg-foreground/30"></div>
		</div>
	</div>
{:else if resolvedType === 'castle'}
	<!-- CASTLE: Distinct double-ring indicating piece swap -->
	<div class="absolute inset-0 z-20 flex items-center justify-center pointer-events-none scale-in">
		<div class="w-[82%] h-[82%] rounded-full border-2 border-foreground/35 flex items-center justify-center">
			<div class="w-3 h-3 rounded-full border border-foreground/40"></div>
		</div>
	</div>
{:else if resolvedType === 'capture'}
	<!-- CAPTURE: High-contrast perimeter framing ring around the piece -->
	<div class="absolute inset-0 z-20 flex items-center justify-center pointer-events-none scale-in">
		<div class="w-[84%] h-[84%] rounded-full border-[3.5px] border-foreground/30"></div>
	</div>
{:else}
	<!-- NORMAL MOVE: Centered translucent reticle dot -->
	<div class="absolute inset-0 z-20 flex items-center justify-center pointer-events-none scale-in">
		<div class="w-3 h-3 sm:w-3.5 sm:h-3.5 rounded-full bg-foreground/25"></div>
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
