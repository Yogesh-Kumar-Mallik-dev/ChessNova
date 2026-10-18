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

<div
	class="relative w-7 xs:w-8 sm:w-9 md:w-10 h-full bg-[#18181b] rounded-lg overflow-hidden border border-border flex flex-col justify-end shadow-md select-none shrink-0"
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
	<div class="absolute inset-x-0 top-1/2 h-px bg-neutral-500/40 pointer-events-none z-10"></div>

	<!-- Evaluation Number Label -->
	<div
		class="absolute inset-x-0 flex items-center justify-center transition-all duration-300 pointer-events-none z-20 px-0.5"
		style={bottomPercent > 50 ? 'bottom: 8px;' : 'top: 8px;'}
	>
		<span
			class="px-1.5 py-0.5 rounded text-[11px] sm:text-xs font-mono font-black tabular-nums tracking-tight leading-none shadow-xs border {bottomPercent > 50
				? orientation === 'white'
					? 'text-neutral-950 bg-white/95 border-neutral-950/20'
					: 'text-white bg-neutral-950/95 border-white/20'
				: orientation === 'white'
					? 'text-white bg-neutral-950/95 border-white/20'
					: 'text-neutral-950 bg-white/95 border-neutral-950/20'}"
		>
			{formattedScore}
		</span>
	</div>
</div>
