<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	interface Props {
		shimmerColor?: string;
		shimmerSize?: string;
		borderRadius?: string;
		shimmerDuration?: string;
		background?: string;
		class?: string;
		disabled?: boolean;
		type?: 'button' | 'submit' | 'reset';
		href?: string;
		onclick?: (e: MouseEvent) => void;
		children?: Snippet;
	}

	let {
		shimmerColor = '#38bdf8',
		shimmerSize = '0.1em',
		borderRadius = '0.75rem',
		shimmerDuration = '2.5s',
		background = 'rgba(15, 23, 42, 0.95)',
		class: className = '',
		disabled = false,
		type = 'button',
		href,
		onclick,
		children
	}: Props = $props();
</script>

{#if href}
	<a
		{href}
		style="
			--spread: 90deg;
			--shimmer-color: {shimmerColor};
			--radius: {borderRadius};
			--speed: {shimmerDuration};
			--cut: {shimmerSize};
			--bg: {background};
		"
		class={cn(
			'group relative z-0 flex cursor-pointer items-center justify-center overflow-hidden whitespace-nowrap px-6 py-3 text-sm font-semibold text-white',
			'[background:var(--bg)] [border-radius:var(--radius)] border border-slate-700/60 shadow-lg shadow-sky-950/20',
			'transform-gpu transition-all duration-300 ease-in-out hover:scale-[1.02] active:scale-[0.98] hover:shadow-sky-500/20 hover:border-sky-500/50',
			className
		)}
	>
		<!-- Spark container -->
		<div
			class="-z-30 blur-[2px] absolute inset-0 overflow-visible [container-type:size]"
		>
			<!-- Spark element -->
			<div
				class="absolute inset-0 h-[100cqh] animate-shimmer-slide [aspect-ratio:1] [border-radius:0] [mask:none]"
			>
				<!-- Spark rotate -->
				<div
					class="animate-spin-around absolute -inset-full w-auto rotate-0 [background:conic-gradient(from_calc(270deg-(var(--spread)*0.5)),transparent_0,var(--shimmer-color)_var(--spread),transparent_var(--spread))] [translate:0_0]"
				></div>
			</div>
		</div>

		<!-- Children container -->
		<span class="relative z-10 flex items-center gap-2">
			{@render children?.()}
		</span>

		<!-- Highlight overlay -->
		<div
			class="pointer-events-none absolute inset-0 rounded-[inherit] shadow-[inset_0_1px_1px_rgba(255,255,255,0.15)]"
		></div>
	</a>
{:else}
	<button
		{type}
		{disabled}
		{onclick}
		style="
			--spread: 90deg;
			--shimmer-color: {shimmerColor};
			--radius: {borderRadius};
			--speed: {shimmerDuration};
			--cut: {shimmerSize};
			--bg: {background};
		"
		class={cn(
			'group relative z-0 flex cursor-pointer items-center justify-center overflow-hidden whitespace-nowrap px-6 py-3 text-sm font-semibold text-white',
			'[background:var(--bg)] [border-radius:var(--radius)] border border-slate-700/60 shadow-lg shadow-sky-950/20',
			'transform-gpu transition-all duration-300 ease-in-out hover:scale-[1.02] active:scale-[0.98] hover:shadow-sky-500/20 hover:border-sky-500/50 disabled:opacity-50 disabled:pointer-events-none',
			className
		)}
	>
		<!-- Spark container -->
		<div
			class="-z-30 blur-[2px] absolute inset-0 overflow-visible [container-type:size]"
		>
			<!-- Spark element -->
			<div
				class="absolute inset-0 h-[100cqh] animate-shimmer-slide [aspect-ratio:1] [border-radius:0] [mask:none]"
			>
				<!-- Spark rotate -->
				<div
					class="animate-spin-around absolute -inset-full w-auto rotate-0 [background:conic-gradient(from_calc(270deg-(var(--spread)*0.5)),transparent_0,var(--shimmer-color)_var(--spread),transparent_var(--spread))] [translate:0_0]"
				></div>
			</div>
		</div>

		<!-- Children container -->
		<span class="relative z-10 flex items-center gap-2">
			{@render children?.()}
		</span>

		<!-- Highlight overlay -->
		<div
			class="pointer-events-none absolute inset-0 rounded-[inherit] shadow-[inset_0_1px_1px_rgba(255,255,255,0.15)]"
		></div>
	</button>
{/if}
