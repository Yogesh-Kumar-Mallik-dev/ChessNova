<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		isTauri,
		minimizeWindow,
		toggleMaximizeWindow,
		closeWindow,
		isWindowMaximized
	} from '$lib/utils/desktop';
	import Icon from '$lib/components/icons/Icon.svelte';

	let inDesktop = false;
	let maximized = false;
	let unlistenResize: (() => void) | null = null;

	onMount(async () => {
		inDesktop = isTauri();
		if (inDesktop) {
			maximized = await isWindowMaximized();
			try {
				const { getCurrentWindow } = await import('@tauri-apps/api/window');
				unlistenResize = await getCurrentWindow().onResized(async () => {
					maximized = await isWindowMaximized();
				});
			} catch (e) {
				console.warn('Could not bind window resize listener:', e);
			}
		}
	});

	onDestroy(() => {
		if (unlistenResize) {
			unlistenResize();
		}
	});

	async function handleMinimize() {
		await minimizeWindow();
	}

	async function handleToggleMaximize() {
		maximized = await toggleMaximizeWindow();
	}

	async function handleClose() {
		await closeWindow();
	}
</script>

{#if inDesktop}
	<aside
		class="h-8.5 bg-card border-b border-border flex items-center justify-between select-none z-50 sticky top-0 px-2.5 text-xs text-foreground transition-colors"
		data-tauri-drag-region
		on:dblclick={handleToggleMaximize}
		aria-label="Desktop Window Titlebar"
	>
		<!-- Left: Branding & Draggable Title -->
		<div class="flex items-center gap-2 pointer-events-none" data-tauri-drag-region>
			<div class="w-5 h-5 rounded border border-border bg-foreground text-background flex items-center justify-center p-0.5 shadow-xs overflow-hidden">
				<img src="/logo.png" alt="ChessNova" class="w-full h-full object-contain invert dark:invert-0" />
			</div>
			<div class="flex items-center gap-1.5 font-bold tracking-tight text-foreground text-[11px]">
				<span>ChessNova</span>
				<span class="px-1.5 py-0.2 text-[9px] font-medium bg-secondary text-muted-foreground border border-border rounded uppercase tracking-wider">
					Desktop
				</span>
			</div>
		</div>

		<!-- Center: Draggable Spacer -->
		<div class="flex-1 h-full mx-4 cursor-default flex items-center justify-center text-slate-500 text-[11px]" data-tauri-drag-region>
			<span class="opacity-40 font-mono tracking-widest text-[10px] pointer-events-none">···</span>
		</div>

		<!-- Right: Window Action Controls -->
		<div class="flex items-center gap-0.5 pointer-events-auto">
			<!-- Minimize -->
			<button
				type="button"
				class="w-8 h-6 flex items-center justify-center rounded hover:bg-slate-800 text-slate-400 hover:text-slate-100 transition-colors"
				title="Minimize"
				aria-label="Minimize Window"
				on:click|stopPropagation={handleMinimize}
			>
				<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 12H4" />
				</svg>
			</button>

			<!-- Maximize / Restore -->
			<button
				type="button"
				class="w-8 h-6 flex items-center justify-center rounded hover:bg-slate-800 text-slate-400 hover:text-slate-100 transition-colors"
				title={maximized ? 'Restore' : 'Maximize'}
				aria-label={maximized ? 'Restore Window' : 'Maximize Window'}
				on:click|stopPropagation={handleToggleMaximize}
			>
				{#if maximized}
					<!-- Restore Icon -->
					<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<rect x="6" y="8" width="14" height="12" rx="1.5" stroke-width="2" />
						<path d="M4 16V5.5A1.5 1.5 0 015.5 4H16" stroke-width="2" stroke-linecap="round" />
					</svg>
				{:else}
					<!-- Maximize Icon -->
					<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<rect x="4" y="4" width="16" height="16" rx="2" stroke-width="2" />
					</svg>
				{/if}
			</button>

			<!-- Close Window -->
			<button
				type="button"
				class="w-8 h-6 flex items-center justify-center rounded hover:bg-red-600 text-slate-400 hover:text-white transition-colors"
				title="Close"
				aria-label="Close Window"
				on:click|stopPropagation={handleClose}
			>
				<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>
	</aside>
{/if}
