<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	interface Props {
		class?: string;
		children?: Snippet;
	}

	let { class: className = '', children }: Props = $props();
</script>

<div class={cn('grid w-full grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4.5', className)}>
	{@render children?.()}
</div>
