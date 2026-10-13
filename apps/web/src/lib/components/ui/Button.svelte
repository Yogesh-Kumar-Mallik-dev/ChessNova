<script lang="ts">
	import { cn } from '$lib/utils';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	type Variant = 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link' | 'glow';
	type Size = 'default' | 'sm' | 'lg' | 'icon';

	export let variant: Variant = 'default';
	export let size: Size = 'default';
	export let className: string = '';
	export let disabled: boolean = false;
	export let type: 'button' | 'submit' | 'reset' = 'button';

	const variantStyles: Record<Variant, string> = {
		default: 'bg-sky-500 hover:bg-sky-400 text-white font-semibold shadow-md shadow-sky-500/20 active:scale-[0.98]',
		glow: 'bg-gradient-to-r from-sky-500 to-indigo-600 hover:from-sky-400 hover:to-indigo-500 text-white font-semibold shadow-lg shadow-sky-500/25 active:scale-[0.98]',
		destructive: 'bg-rose-600 hover:bg-rose-500 text-white font-semibold shadow-md shadow-rose-600/20 active:scale-[0.98]',
		outline: 'border border-slate-700/80 bg-slate-900/60 hover:bg-slate-800 text-slate-200 hover:text-white shadow-sm active:scale-[0.98]',
		secondary: 'bg-slate-800 hover:bg-slate-700 text-slate-100 font-medium active:scale-[0.98]',
		ghost: 'hover:bg-slate-800/80 text-slate-300 hover:text-white active:scale-[0.98]',
		link: 'text-sky-400 underline-offset-4 hover:underline p-0 h-auto'
	};

	const sizeStyles: Record<Size, string> = {
		default: 'h-10 px-4 py-2 text-xs sm:text-sm rounded-xl',
		sm: 'h-8 px-3 text-xs rounded-lg',
		lg: 'h-12 px-6 text-sm sm:text-base rounded-2xl font-bold',
		icon: 'h-9 w-9 p-0 rounded-xl flex items-center justify-center'
	};
</script>

<button
	{type}
	{disabled}
	class={cn(
		'inline-flex items-center justify-center gap-2 whitespace-nowrap transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-400 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950 disabled:pointer-events-none disabled:opacity-50 select-none cursor-pointer',
		variantStyles[variant],
		sizeStyles[size],
		className
	)}
	on:click
	{...$$restProps}
>
	<slot />
</button>
