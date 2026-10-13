<script lang="ts">
	import { cn } from '$lib/utils';

	export let className: string = '';
	export let interactive: boolean = false;
</script>

<div
	class={cn(
		'rounded-2xl border border-slate-800/80 bg-slate-900/75 backdrop-blur-xl text-slate-100 shadow-xl transition-all duration-200',
		interactive && 'hover:border-slate-700 hover:bg-slate-900/90 hover:shadow-2xl hover:shadow-sky-500/5 active:scale-[0.99] cursor-pointer',
		className
	)}
	on:click
	{...$$restProps}
>
	<slot />
</div>
