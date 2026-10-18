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
	$: isWhiteAdvantage = scoreCp >= 0;
</script>

<div
	class="relative w-4 xs:w-5 sm:w-6 h-full bg-neutral-900 rounded-md overflow-hidden border border-border flex flex-col justify-end shadow-xs select-none"
	title="Advantage: {formattedScore}"
>
	<!-- Black Portion (Top if White orientation) -->
	<div class="absolute inset-0 bg-neutral-900"></div>

	<!-- White Portion (Bottom if White orientation) -->
	<div
		class="w-full bg-neutral-100 transition-all duration-300 ease-out"
		style="height: {bottomPercent}%;"
	></div>

	<!-- Score Badge Label -->
	<div
		class="absolute inset-x-0 flex items-center justify-center transition-all duration-300 pointer-events-none"
		style={bottomPercent > 50 ? 'bottom: 6px;' : 'top: 6px;'}
	>
		<span
			class="px-1 py-0.5 rounded text-[9px] sm:text-[10px] font-mono font-bold leading-none {bottomPercent > 50 ? 'text-neutral-900 bg-neutral-100/90' : 'text-neutral-100 bg-neutral-900/90'}"
		>
			{formattedScore}
		</span>
	</div>
</div>
