<script lang="ts">
	import type { Snippet, ComponentProps } from 'svelte';
	import { cn } from '$lib/utils';
	import Icon from '$lib/components/icons/Icon.svelte';

	interface Props {
		name: string;
		class?: string;
		icon?: ComponentProps<typeof Icon>['name'];
		description: string;
		href: string;
		cta?: string;
		colSpan?: string;
		background?: Snippet;
		headerSlot?: Snippet;
	}

	let {
		name,
		class: className = '',
		icon,
		description,
		href,
		cta = 'Explore',
		colSpan = '',
		background,
		headerSlot
	}: Props = $props();
</script>

<div
	class={cn(
		'group relative flex flex-col justify-between overflow-hidden rounded-2xl',
		'bg-slate-900/40 border border-slate-800/80 p-6 shadow-xl backdrop-blur-sm',
		'transition-all duration-300 hover:border-slate-700 hover:shadow-2xl hover:shadow-sky-950/20 hover:-translate-y-0.5',
		colSpan,
		className
	)}
>
	<!-- Ambient top-edge highlight -->
	<div class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-sky-400/25 to-transparent"></div>

	<!-- Background graphic/snippet if any -->
	{#if background}
		<div class="pointer-events-none absolute inset-0 z-0 overflow-hidden opacity-30 transition-opacity duration-300 group-hover:opacity-45">
			{@render background()}
		</div>
	{/if}

	<div class="relative z-10">
		<div class="flex items-center justify-between mb-4">
			{#if icon}
				<div class="w-11 h-11 rounded-xl bg-slate-800/80 border border-slate-700/80 flex items-center justify-center text-sky-400 shadow-md group-hover:scale-105 group-hover:border-sky-500/40 group-hover:text-sky-300 transition-all">
					<Icon name={icon} size={20} />
				</div>
			{/if}
			{#if headerSlot}
				{@render headerSlot()}
			{/if}
		</div>

		<h3 class="text-lg font-bold text-white tracking-tight group-hover:text-sky-200 transition-colors">
			{name}
		</h3>

		<p class="text-xs sm:text-sm text-slate-400 mt-2 leading-relaxed font-normal">
			{description}
		</p>
	</div>

	<div class="relative z-10 mt-6 pt-4 border-t border-slate-800/60 flex items-center justify-between">
		<a
			{href}
			class="inline-flex items-center gap-1.5 text-xs font-semibold text-sky-400 group-hover:text-sky-300 transition-colors"
		>
			<span>{cta}</span>
			<Icon name="chevron-right" size={14} className="group-hover:translate-x-1 transition-transform" />
		</a>
	</div>
</div>
