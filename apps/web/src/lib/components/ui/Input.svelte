<script lang="ts">
	import { cn } from '$lib/utils';
	import type { HTMLInputAttributes } from 'svelte/elements';

	export let value: string = '';
	export let type: string = 'text';
	export let placeholder: string = '';
	export let disabled: boolean = false;
	export let required: boolean = false;
	export let className: string = '';
	export let id: string = '';
</script>

<input
	{id}
	{type}
	{placeholder}
	{disabled}
	{required}
	bind:value
	class={cn(
		'flex h-10 w-full rounded-xl border border-slate-800 bg-slate-950/80 px-3.5 py-2 text-sm text-slate-100 placeholder:text-slate-500 shadow-sm transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500/50 focus-visible:border-sky-500 disabled:cursor-not-allowed disabled:opacity-50',
		className
	)}
	on:input
	on:change
	on:keydown
	{...$$restProps}
/>
