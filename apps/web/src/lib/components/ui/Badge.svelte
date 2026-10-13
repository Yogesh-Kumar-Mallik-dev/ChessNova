<script lang="ts">
	import { cn } from '$lib/utils';

	type Variant = 'default' | 'secondary' | 'outline' | 'success' | 'warning' | 'destructive';

	export let variant: Variant = 'default';
	export let className: string = '';

	const variantStyles: Record<Variant, string> = {
		default: 'bg-sky-500/10 text-sky-400 border-sky-500/25',
		secondary: 'bg-slate-800 text-slate-300 border-slate-700/60',
		outline: 'text-slate-300 border-slate-700/80 bg-transparent',
		success: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/25',
		warning: 'bg-amber-500/10 text-amber-400 border-amber-500/25',
		destructive: 'bg-rose-500/10 text-rose-400 border-rose-500/25'
	};
</script>

<div
	class={cn(
		'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[11px] font-semibold transition-colors select-none',
		variantStyles[variant],
		className
	)}
>
	<slot />
</div>
