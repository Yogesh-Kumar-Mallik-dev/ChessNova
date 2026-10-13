<script lang="ts">
	import { cn } from '$lib/utils';

	type Variant = 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link' | 'glow' | 'primary' | 'emerald';
	type Size = 'default' | 'sm' | 'lg' | 'icon';

	export let variant: Variant = 'default';
	export let size: Size = 'default';
	export let className: string = '';
	export let disabled: boolean = false;
	export let type: 'button' | 'submit' | 'reset' = 'button';

	const variantStyles: Record<Variant, string> = {
		// Clean high-contrast solid button (modern shadcn style)
		default:
			'bg-white text-slate-950 hover:bg-slate-100 font-semibold shadow-sm border border-transparent active:translate-y-px',
		// The iconic, tactile chess green action button (Chess.com style)
		primary:
			'bg-[#81b64c] hover:bg-[#77a942] active:bg-[#689437] text-white font-bold shadow-sm border border-[#6ea03c]/60 active:translate-y-px',
		emerald:
			'bg-[#81b64c] hover:bg-[#77a942] active:bg-[#689437] text-white font-bold shadow-sm border border-[#6ea03c]/60 active:translate-y-px',
		glow:
			'bg-[#81b64c] hover:bg-[#77a942] active:bg-[#689437] text-white font-bold shadow-sm border border-[#6ea03c]/60 active:translate-y-px',
		// Secondary dark slate button with subtle border
		secondary:
			'bg-slate-800/90 hover:bg-slate-700/90 text-slate-100 font-semibold border border-slate-700/60 shadow-xs active:translate-y-px',
		// Sophisticated outline button
		outline:
			'border border-slate-700/80 bg-slate-900/50 hover:bg-slate-800 hover:text-white text-slate-200 font-semibold shadow-xs active:translate-y-px',
		// Ghost button for lightweight actions
		ghost:
			'text-slate-400 hover:text-slate-100 hover:bg-slate-800/60 font-medium active:translate-y-px',
		// Crisp destructive button
		destructive:
			'bg-rose-600 hover:bg-rose-500 text-white font-semibold shadow-sm border border-rose-500/50 active:translate-y-px',
		// Clean text link
		link:
			'text-sky-400 underline-offset-4 hover:underline p-0 h-auto font-medium'
	};

	const sizeStyles: Record<Size, string> = {
		default: 'h-9 sm:h-10 px-4 py-2 text-xs sm:text-sm rounded-xl',
		sm: 'h-8 px-3 text-xs rounded-lg',
		lg: 'h-11 sm:h-12 px-6 text-sm sm:text-base rounded-xl font-bold',
		icon: 'h-9 w-9 p-0 rounded-xl flex items-center justify-center'
	};
</script>

<button
	{type}
	{disabled}
	class={cn(
		'inline-flex items-center justify-center gap-2 whitespace-nowrap transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950 disabled:pointer-events-none disabled:opacity-40 select-none cursor-pointer tracking-tight',
		variantStyles[variant],
		sizeStyles[size],
		className
	)}
	on:click
	{...$$restProps}
>
	<slot />
</button>
