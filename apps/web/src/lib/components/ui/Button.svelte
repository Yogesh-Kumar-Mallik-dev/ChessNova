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
		// High-contrast primary action (solid white on dark, solid black on light)
		default:
			'bg-foreground text-background hover:opacity-90 font-semibold shadow-xs border border-transparent active:translate-y-px',
		primary:
			'bg-foreground text-background hover:opacity-90 font-semibold shadow-xs border border-transparent active:translate-y-px',
		emerald:
			'bg-foreground text-background hover:opacity-90 font-semibold shadow-xs border border-transparent active:translate-y-px',
		glow:
			'bg-foreground text-background hover:opacity-90 font-semibold shadow-xs border border-transparent active:translate-y-px',
		// Secondary surface button with subtle border
		secondary:
			'bg-secondary text-secondary-foreground hover:bg-accent font-medium border border-border shadow-xs active:translate-y-px',
		// Refined outline button
		outline:
			'border border-border bg-transparent hover:bg-secondary hover:text-foreground text-foreground font-medium shadow-xs active:translate-y-px',
		// Ghost button for clean toolbar actions
		ghost:
			'text-muted-foreground hover:text-foreground hover:bg-secondary font-medium active:translate-y-px',
		// Restrained destructive button
		destructive:
			'bg-red-950/40 text-red-400 hover:bg-red-900/60 font-semibold border border-red-800/50 active:translate-y-px',
		// Clean text link
		link:
			'text-foreground underline-offset-4 hover:underline p-0 h-auto font-medium'
	};

	const sizeStyles: Record<Size, string> = {
		default: 'h-9 sm:h-10 px-4 py-2 text-xs sm:text-sm rounded-lg',
		sm: 'h-8 px-3 text-xs rounded-md',
		lg: 'h-11 sm:h-12 px-6 text-sm sm:text-base rounded-lg font-semibold',
		icon: 'h-9 w-9 p-0 rounded-lg flex items-center justify-center'
	};
</script>

<button
	{type}
	{disabled}
	class={cn(
		'inline-flex items-center justify-center gap-2 whitespace-nowrap transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-40 select-none cursor-pointer tracking-tight',
		variantStyles[variant],
		sizeStyles[size],
		className
	)}
	on:click
	{...$$restProps}
>
	<slot />
</button>
