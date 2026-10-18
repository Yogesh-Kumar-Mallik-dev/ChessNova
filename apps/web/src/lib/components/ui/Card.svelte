<script lang="ts">
	import { cn } from '$lib/utils';

	export let className: string = '';
	export let interactive: boolean = false;
</script>

<div
	class={cn(
		'rounded-xl border border-border bg-card text-card-foreground shadow-xs transition-colors',
		interactive && 'hover:border-border/80 hover:bg-accent/30 active:scale-[0.995] cursor-pointer',
		className
	)}
	on:click
	{...$$restProps}
>
	<slot />
</div>
