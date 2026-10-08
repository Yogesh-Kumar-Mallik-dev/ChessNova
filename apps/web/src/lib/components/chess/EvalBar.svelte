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
		// Sigmoid curve similar to Lichess/Chess.com advantage percentage
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
	class="relative w-6 sm:w-7 h-full bg-[#1b1a18] rounded-xl overflow-hidden border border-[#3d3b37] flex flex-col justify-end shadow-md select-none"
	title="Advantage: {formattedScore}"
>
	<!-- Black Portion (Top if White orientation) -->
	<div class="absolute inset-0 bg-[#2b2926]"></div>

	<!-- White Portion (Bottom if White orientation) -->
	<div
		class="w-full bg-[#ffffff] transition-all duration-300 ease-out"
		style="height: {bottomPercent}%;"
	></div>

	<!-- Score Badge Label -->
	<div
		class="absolute inset-x-0 flex items-center justify-center transition-all duration-300 pointer-events-none"
		style={bottomPercent > 50 ? 'bottom: 8px;' : 'top: 8px;'}
	>
		<span
			class="px-1 py-0.5 rounded text-[10px] font-mono font-black leading-none {bottomPercent > 50 ? 'text-black' : 'text-white'}"
		>
			{formattedScore}
		</span>
	</div>
</div>
