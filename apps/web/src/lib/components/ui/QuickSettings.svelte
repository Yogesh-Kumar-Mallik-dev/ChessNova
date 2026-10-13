<script lang="ts">
	import Icon from '$lib/components/icons/Icon.svelte';
	import {
		themeStore,
		appThemeStore,
		volumeStore,
		soundStore,
		type BoardTheme,
		type AppTheme
	} from '$lib/stores/preferences';

	let isOpen = false;

	const boardThemes: { id: BoardTheme; name: string; light: string; dark: string }[] = [
		{ id: 'emerald', name: 'Green', light: '#eeeed2', dark: '#769656' },
		{ id: 'wood', name: 'Wood', light: '#f0d9b5', dark: '#b58863' },
		{ id: 'ocean', name: 'Ocean', light: '#dbeafe', dark: '#467499' },
		{ id: 'slate', name: 'Slate', light: '#cdd6e0', dark: '#475569' },
		{ id: 'charcoal', name: 'Charcoal', light: '#cbd5e1', dark: '#262626' },
		{ id: 'coral', name: 'Coral', light: '#f5e6e8', dark: '#b36675' }
	];

	const appThemes: { id: AppTheme; name: string; bg: string; border: string; accent: string }[] = [
		{ id: 'dark-slate', name: 'Dark Slate', bg: '#080d1a', border: '#1e293b', accent: '#38bdf8' },
		{ id: 'chess-dark', name: 'Chess Classic', bg: '#161512', border: '#2b2925', accent: '#81b64c' },
		{ id: 'oled-black', name: 'OLED Black', bg: '#050505', border: '#1f1f1f', accent: '#22c55e' },
		{ id: 'emerald-studio', name: 'Emerald Studio', bg: '#05140e', border: '#123023', accent: '#10b981' }
	];

	function selectBoardTheme(t: BoardTheme) {
		themeStore.set(t);
	}

	function selectAppTheme(t: AppTheme) {
		appThemeStore.set(t);
	}

	function handleVolumeChange(e: Event) {
		const target = e.target as HTMLInputElement;
		const val = parseFloat(target.value);
		volumeStore.set(val);
		if (val > 0 && !$soundStore) {
			soundStore.set(true);
		}
	}

	function toggleMute() {
		soundStore.toggle();
	}

	function close() {
		isOpen = false;
	}
</script>

<div class="relative">
	<button
		type="button"
		class="p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-850 border border-transparent hover:border-slate-700/60 transition flex items-center justify-center {isOpen ? 'bg-slate-800 text-emerald-400' : ''}"
		on:click={() => (isOpen = !isOpen)}
		title="Theme & Sound Preferences"
		aria-label="Settings"
	>
		<Icon name="settings" size={17} />
	</button>

	{#if isOpen}
		<!-- Backdrop for closing -->
		<!-- svelte-ignore a11y-click-events-have-key-events -->
		<!-- svelte-ignore a11y-no-static-element-interactions -->
		<div class="fixed inset-0 z-40" on:click={close}></div>

		<!-- Dropdown Panel -->
		<div class="absolute right-0 mt-2 w-80 bg-slate-900 border border-slate-700/90 rounded-2xl p-4 shadow-2xl z-50 animate-in fade-in zoom-in-95 duration-150 max-h-[85vh] overflow-y-auto">
			<div class="flex items-center justify-between pb-3 border-b border-slate-800">
				<div class="flex items-center gap-2">
					<Icon name="settings" size={15} className="text-emerald-400" />
					<h3 class="text-xs font-bold uppercase tracking-wider text-slate-200">Customization</h3>
				</div>
				<button class="text-slate-400 hover:text-white text-xs p-1" on:click={close}>
					<Icon name="x" size={14} />
				</button>
			</div>

			<!-- Board Theme Selection (Like Chess.com) -->
			<div class="mt-3">
				<div class="flex items-center justify-between mb-2">
					<span class="text-[11px] font-bold uppercase tracking-wider text-slate-400">
						Board Colors
					</span>
					<span class="text-[10px] text-slate-500 font-mono capitalize">
						{$themeStore}
					</span>
				</div>
				<div class="grid grid-cols-3 gap-2">
					{#each boardThemes as t}
						<button
							type="button"
							class="flex flex-col items-center gap-1.5 p-2 rounded-xl border text-center transition { $themeStore === t.id ? 'border-emerald-500 bg-emerald-500/10 text-white font-bold ring-1 ring-emerald-500/40' : 'border-slate-800 bg-slate-950/60 text-slate-300 hover:border-slate-700' }"
							on:click={() => selectBoardTheme(t.id)}
						>
							<div class="w-8 h-8 rounded-lg overflow-hidden grid grid-cols-2 shrink-0 border border-slate-700/80 shadow-xs">
								<div style="background-color: {t.light}"></div>
								<div style="background-color: {t.dark}"></div>
								<div style="background-color: {t.dark}"></div>
								<div style="background-color: {t.light}"></div>
							</div>
							<span class="text-[11px] font-medium leading-tight">{t.name}</span>
						</button>
					{/each}
				</div>
			</div>

			<!-- App Color Scheme Selection -->
			<div class="mt-4 pt-3 border-t border-slate-800">
				<div class="flex items-center justify-between mb-2">
					<span class="text-[11px] font-bold uppercase tracking-wider text-slate-400">
						App Color Scheme
					</span>
				</div>
				<div class="grid grid-cols-2 gap-2">
					{#each appThemes as at}
						<button
							type="button"
							class="flex items-center gap-2 p-2 rounded-xl border text-left transition { $appThemeStore === at.id ? 'border-emerald-500 bg-emerald-500/10 text-white font-bold ring-1 ring-emerald-500/40' : 'border-slate-800 bg-slate-950/60 text-slate-300 hover:border-slate-700' }"
							on:click={() => selectAppTheme(at.id)}
						>
							<div
								class="w-5 h-5 rounded-md border shrink-0 flex items-center justify-center shadow-xs"
								style="background-color: {at.bg}; border-color: {at.border};"
							>
								<div class="w-2 h-2 rounded-full" style="background-color: {at.accent};"></div>
							</div>
							<span class="text-xs truncate">{at.name}</span>
						</button>
					{/each}
				</div>
			</div>

			<!-- Audio Controls -->
			<div class="mt-4 pt-3 border-t border-slate-800">
				<div class="flex items-center justify-between mb-2">
					<span class="text-[11px] font-bold uppercase tracking-wider text-slate-400">Master Sound</span>
					<button
						type="button"
						class="text-xs flex items-center gap-1.5 px-2 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 transition"
						on:click={toggleMute}
					>
						<Icon name={$soundStore ? 'volume-2' : 'volume-x'} size={14} className={$soundStore ? 'text-emerald-400' : 'text-rose-400'} />
						<span>{$soundStore ? 'Mute' : 'Unmute'}</span>
					</button>
				</div>

				<div class="flex items-center gap-2">
					<input
						type="range"
						min="0"
						max="1"
						step="0.05"
						value={$soundStore ? $volumeStore : 0}
						disabled={!$soundStore}
						on:input={handleVolumeChange}
						class="w-full accent-emerald-500 bg-slate-800 h-1.5 rounded-lg cursor-pointer disabled:opacity-40"
					/>
					<span class="text-[11px] font-mono text-slate-400 w-8 text-right">
						{Math.round(($soundStore ? $volumeStore : 0) * 100)}%
					</span>
				</div>
			</div>
		</div>
	{/if}
</div>
