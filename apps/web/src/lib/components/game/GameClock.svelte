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
	class="px-3.5 py-1.5 sm:px-4 sm:py-2 rounded-xl font-mono text-lg sm:text-2xl font-black tracking-tight transition-all duration-200 flex items-center gap-2 select-none backdrop-blur-md
	{isActive
		? isLow
			? 'bg-rose-950/80 text-rose-100 ring-2 ring-rose-500 shadow-[0_0_20px_rgba(244,63,94,0.4)] animate-pulse'
			: 'bg-slate-900/90 text-white ring-2 ring-sky-400 shadow-[0_0_20px_rgba(56,189,248,0.3)]'
		: 'bg-slate-900/50 text-slate-400 border border-slate-800/80'}"
>
	<Icon
		name="timer"
		size={16}
		className={isActive ? (isLow ? 'text-rose-400 animate-spin' : 'text-sky-400') : 'text-slate-600'}
	/>
	<span>{formatTime(displayTimeMs)}</span>
</div>
