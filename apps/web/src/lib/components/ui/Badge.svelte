<script lang="ts">
	import { cn } from '$lib/utils';

	type Variant = 'default' | 'secondary' | 'outline' | 'success' | 'warning' | 'destructive';

	export let variant: Variant = 'default';
	export let className: string = '';

	const variantStyles: Record<Variant, string> = {
		default: 'bg-secondary text-foreground border-border',
		secondary: 'bg-muted text-muted-foreground border-border',
		outline: 'text-foreground border-border bg-transparent',
		success: 'bg-emerald-950/30 text-emerald-400 border-emerald-800/40',
		warning: 'bg-amber-950/30 text-amber-400 border-amber-800/40',
		destructive: 'bg-red-950/30 text-red-400 border-red-800/40'
	};
</script>

<div
	class={cn(
		'inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-[11px] font-medium transition-colors select-none',
		variantStyles[variant],
		className
	)}
>
	<slot />
</div>
