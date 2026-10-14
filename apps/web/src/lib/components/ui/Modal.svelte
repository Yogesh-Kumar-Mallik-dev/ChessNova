<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Icon from '$lib/components/icons/Icon.svelte';

	export let isOpen = false;
	export let title = '';
	export let description = '';
	export let icon: any = null;
	export let iconBg = 'bg-slate-800 text-slate-300';
	export let iconColor = 'text-sky-400';
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
				return 'bg-rose-600 hover:bg-rose-500 active:bg-rose-700 text-white shadow-lg shadow-rose-900/30';
			case 'warning':
				return 'bg-amber-600 hover:bg-amber-500 active:bg-amber-700 text-white shadow-lg shadow-amber-900/30';
			case 'primary':
				return 'bg-[#81b64c] hover:bg-[#91c65d] active:bg-[#73a443] text-white shadow-lg shadow-[#81b64c]/20';
			default:
				return 'bg-sky-600 hover:bg-sky-500 active:bg-sky-700 text-white shadow-lg shadow-sky-900/30';
		}
	})();
</script>

<svelte:window on:keydown={handleKeydown} />

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-md p-4 animate-in fade-in duration-200"
		role="dialog"
		aria-modal="true"
		tabindex="-1"
		on:click={handleBackdropClick}
		on:keydown={(e) => e.key === 'Escape' && dispatch('close')}
	>
		<div class="relative bg-gradient-to-b from-[#161f30] to-[#0c121e] border border-slate-700/80 rounded-2xl p-6 max-w-sm w-full shadow-2xl flex flex-col items-center text-center scale-in-95 duration-150 animate-in">
			<!-- Close button -->
			<button
				class="absolute top-3.5 right-3.5 w-7 h-7 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800/80 flex items-center justify-center transition"
				on:click={() => dispatch('close')}
				title="Close"
				aria-label="Close"
			>
				<Icon name="x" size={15} />
			</button>

			<!-- Modal Icon Emblem -->
			{#if icon}
				<div class="w-13 h-13 rounded-2xl flex items-center justify-center mb-3.5 shadow-lg {iconBg}">
					<Icon name={icon} size={26} className={iconColor} />
				</div>
			{/if}

			<!-- Title -->
			<h2 class="text-xl font-black text-white tracking-tight mb-2">
				{title}
			</h2>

			<!-- Description -->
			{#if description}
				<p class="text-xs sm:text-sm text-slate-300/90 leading-relaxed mb-6 font-medium">
					{description}
				</p>
			{/if}

			<slot />

			<!-- Action Buttons -->
			<div class="flex gap-2.5 w-full mt-2">
				{#if showCancel}
					<button
						class="flex-1 py-2.5 px-4 bg-slate-800 hover:bg-slate-750 text-slate-300 font-bold text-xs rounded-xl border border-slate-700 transition cursor-pointer"
						on:click={() => dispatch('cancel')}
					>
						{cancelText}
					</button>
				{/if}

				<button
					class="flex-1 py-2.5 px-4 font-bold text-xs rounded-xl transition cursor-pointer {confirmBtnClass}"
					on:click={() => dispatch('confirm')}
				>
					{confirmText}
				</button>
			</div>
		</div>
	</div>
{/if}
