<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Icon from '$lib/components/icons/Icon.svelte';

	export let isOpen = false;
	export let title = '';
	export let description = '';
	export let icon: any = null;
	export let iconBg = '';
	export let iconColor = '';
	export let confirmText = 'Confirm';
	export let confirmVariant: 'danger' | 'primary' | 'warning' | 'default' = 'primary';
	export let cancelText = 'Cancel';
	export let showCancel = true;
	export let closeOnBackdrop = true;

	const dispatch = createEventDispatcher<{
		confirm: void;
		cancel: void;
		close: void;
	}>();

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget && closeOnBackdrop) {
			dispatch('close');
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && isOpen) {
			dispatch('close');
		}
	}

	$: confirmBtnClass = (() => {
		switch (confirmVariant) {
			case 'danger':
				return 'bg-red-950/40 text-red-400 hover:bg-red-900/60 font-semibold border border-red-800/50 shadow-xs';
			case 'warning':
				return 'bg-amber-950/40 text-amber-400 hover:bg-amber-900/60 font-semibold border border-amber-800/50 shadow-xs';
			case 'primary':
			default:
				return 'bg-foreground text-background hover:opacity-90 font-semibold shadow-xs border border-transparent';
		}
	})();
</script>

<svelte:window on:keydown={handleKeydown} />

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4 animate-in fade-in duration-150"
		role="dialog"
		aria-modal="true"
		tabindex="-1"
		on:click={handleBackdropClick}
		on:keydown={(e) => e.key === 'Escape' && dispatch('close')}
	>
		<div class="relative bg-card border border-border rounded-xl p-6 max-w-sm w-full shadow-2xl flex flex-col items-center text-center scale-in-95 duration-150 animate-in">
			<!-- Close button -->
			<button
				class="absolute top-3.5 right-3.5 w-7 h-7 rounded-lg text-muted-foreground hover:text-foreground hover:bg-secondary flex items-center justify-center transition"
				on:click={() => dispatch('close')}
				title="Close"
				aria-label="Close"
			>
				<Icon name="x" size={15} />
			</button>

			<!-- Modal Icon Emblem -->
			{#if icon}
				<div class="w-12 h-12 rounded-xl border border-border flex items-center justify-center mb-3.5 {iconBg || 'bg-secondary'}">
					<Icon name={icon} size={22} className={iconColor || 'text-foreground'} />
				</div>
			{/if}

			<!-- Title -->
			<h2 class="text-lg font-bold text-foreground tracking-tight mb-1.5">
				{title}
			</h2>

			<!-- Description -->
			{#if description}
				<p class="text-xs text-muted-foreground leading-relaxed mb-6 font-normal">
					{description}
				</p>
			{/if}

			<slot />

			<!-- Action Buttons -->
			<div class="flex gap-2.5 w-full mt-2">
				{#if showCancel}
					<button
						class="flex-1 py-2 px-3 bg-secondary hover:bg-accent text-secondary-foreground font-medium text-xs rounded-lg border border-border transition cursor-pointer"
						on:click={() => dispatch('cancel')}
					>
						{cancelText}
					</button>
				{/if}

				<button
					class="flex-1 py-2 px-3 font-semibold text-xs rounded-lg transition cursor-pointer {confirmBtnClass}"
					on:click={() => dispatch('confirm')}
				>
					{confirmText}
				</button>
			</div>
		</div>
	</div>
{/if}
