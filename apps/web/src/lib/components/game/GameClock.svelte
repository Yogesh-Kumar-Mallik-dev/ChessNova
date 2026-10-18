<script lang="ts">
	import { onDestroy } from 'svelte';
	import Icon from '$lib/components/icons/Icon.svelte';

	export let timeMs: number;
	export let isActive: boolean;

	let displayTimeMs = timeMs;
	let interval: any = null;

	$: {
		displayTimeMs = timeMs;
	}

	$: {
		if (isActive) {
			startTimer();
		} else {
			stopTimer();
		}
	}

	function startTimer() {
		stopTimer();
		interval = setInterval(() => {
			if (displayTimeMs > 100) {
				displayTimeMs -= 100;
			} else {
				displayTimeMs = 0;
				stopTimer();
			}
		}, 100);
	}

	function stopTimer() {
		if (interval) {
			clearInterval(interval);
			interval = null;
		}
	}

	onDestroy(() => {
		stopTimer();
	});

	function formatTime(ms: number): string {
		if (ms <= 0) return '0:00.0';
		const totalSeconds = ms / 1000;
		const mins = Math.floor(totalSeconds / 60);
		const secs = Math.floor(totalSeconds % 60);

		if (totalSeconds < 15) {
			const tenths = Math.floor((ms % 1000) / 100);
			return `${mins}:${secs.toString().padStart(2, '0')}.${tenths}`;
		}

		return `${mins}:${secs.toString().padStart(2, '0')}`;
	}

	$: isLow = displayTimeMs < 20000;
</script>

<div
	class="px-3 py-1 sm:px-4 sm:py-1.5 rounded-lg font-mono text-base sm:text-xl font-bold tracking-tight transition-colors duration-100 flex items-center gap-2 select-none tabular-nums
	{isActive
		? isLow
			? 'bg-red-950/70 text-red-300 border-2 border-red-500 shadow-xs animate-pulse'
			: 'bg-foreground text-background shadow-xs font-extrabold'
		: 'bg-secondary text-muted-foreground border border-border'}"
>
	<Icon
		name="timer"
		size={14}
		className={isActive ? (isLow ? 'text-red-400' : 'text-background') : 'text-muted-foreground/60'}
	/>
	<span>{formatTime(displayTimeMs)}</span>
</div>
