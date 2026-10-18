<script lang="ts">
	import type { Color } from '$lib/chess/engine';

	export let scoreCp: number = 0;
	export let orientation: Color = 'white';
	export let isMate: boolean = false;
	export let mateIn: number | undefined = undefined;

	// Calculate White's visual percentage (0% to 100%)
	$: whitePercent = (() => {
		if (isMate && mateIn !== undefined) {
			return mateIn > 0 ? 100 : 0;
		}
		// Clamp centipawns between -1000 (-10 pawns) and +1000 (+10 pawns)
		const clamped = Math.max(-1000, Math.min(1000, scoreCp));
		// Sigmoid curve for smooth advantage percentage
		const p = 50 + 50 * (2 / (1 + Math.exp(-0.004 * clamped)) - 1);
		return Math.max(4, Math.min(96, Math.round(p)));
	})();

	$: formattedScore = (() => {
		if (isMate && mateIn !== undefined) {
			return mateIn > 0 ? `M${Math.abs(mateIn)}` : `-M${Math.abs(mateIn)}`;
		}
		const pawns = (scoreCp / 100).toFixed(1);
		if (scoreCp > 0) return `+${pawns}`;
		if (scoreCp < 0) return pawns;
		return '0.0';
	})();

	// Orientation flip: if black, flip perspective
	$: bottomPercent = orientation === 'white' ? whitePercent : 100 - whitePercent;
</script>

<div class="relative flex flex-col items-center h-full">
	<!-- Evaluation Score Label directly above the bar -->
	<div class="absolute -top-6 sm:-top-6.5 left-1/2 -translate-x-1/2 flex items-center justify-center pointer-events-none z-30">
		<span class="px-1.5 py-0.5 rounded-md text-[10px] sm:text-[11px] font-mono font-bold text-foreground bg-card border border-border shadow-xs tabular-nums whitespace-nowrap">
			{formattedScore}
		</span>
	</div>

	<!-- Slim Evaluation Bar -->
	<div
		class="relative w-3.5 sm:w-4 h-full bg-[#18181b] rounded-full overflow-hidden border border-border flex flex-col justify-end shadow-xs select-none shrink-0"
		title="Evaluation: {formattedScore}"
	>
		<!-- Top portion background:
		     If orientation is white: top is Black (dark)
		     If orientation is black: top is White (light) -->
		<div class="absolute inset-0 {orientation === 'white' ? 'bg-[#18181b]' : 'bg-[#f4f4f6]'}"></div>

		<!-- Bottom portion:
		     If orientation is white: bottom is White (light)
		     If orientation is black: bottom is Black (dark) -->
		<div
			class="w-full transition-all duration-300 ease-out {orientation === 'white' ? 'bg-[#f4f4f6]' : 'bg-[#18181b]'}"
			style="height: {bottomPercent}%;"
		></div>

		<!-- Center divider line for 0.0 equality -->
		<div class="absolute inset-x-0 top-1/2 h-px bg-neutral-500/50 pointer-events-none z-10"></div>
	</div>
</div>
