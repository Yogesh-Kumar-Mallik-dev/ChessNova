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
		{ id: 'monochrome', name: 'Monochrome', light: '#ffffff', dark: '#2b2c34' },
		{ id: 'slate', name: 'Slate', light: '#dcdde1', dark: '#404552' },
		{ id: 'charcoal', name: 'Charcoal', light: '#e4e4e7', dark: '#27272a' },
		{ id: 'wood', name: 'Wood', light: '#eedbc5', dark: '#9b7653' },
		{ id: 'emerald', name: 'Classic', light: '#eeeed2', dark: '#69884a' },
		{ id: 'ocean', name: 'Ocean', light: '#dbeafe', dark: '#467499' }
	];

	const appThemes: { id: AppTheme; name: string; bg: string; border: string; accent: string }[] = [
		{ id: 'dark', name: 'Studio Dark', bg: '#0f0f12', border: '#2b2b32', accent: '#fafafa' },
		{ id: 'light', name: 'Studio Light', bg: '#f7f7f8', border: '#dedee3', accent: '#17171d' },
		{ id: 'oled', name: 'OLED Black', bg: '#000000', border: '#262626', accent: '#ffffff' }
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
		class="p-2 rounded-lg text-muted-foreground hover:text-foreground hover:bg-secondary border border-transparent hover:border-border transition flex items-center justify-center {isOpen ? 'bg-secondary text-foreground' : ''}"
		on:click={() => (isOpen = !isOpen)}
		title="Theme & Sound Preferences"
		aria-label="Settings"
	>
		<Icon name="settings" size={16} />
	</button>

	{#if isOpen}
		<!-- Backdrop for closing -->
		<!-- svelte-ignore a11y-click-events-have-key-events -->
		<!-- svelte-ignore a11y-no-static-element-interactions -->
		<div class="fixed inset-0 z-40" on:click={close}></div>

		<!-- Dropdown Panel -->
		<div class="absolute right-0 mt-2 w-80 bg-card border border-border rounded-xl p-4 shadow-2xl z-50 animate-in fade-in zoom-in-95 duration-150 max-h-[85vh] overflow-y-auto">
			<div class="flex items-center justify-between pb-3 border-b border-border">
				<div class="flex items-center gap-2">
					<Icon name="settings" size={14} className="text-foreground" />
					<h3 class="text-xs font-bold uppercase tracking-wider text-foreground">Preferences</h3>
				</div>
				<button class="text-muted-foreground hover:text-foreground text-xs p-1 rounded-md" on:click={close}>
					<Icon name="x" size={14} />
				</button>
			</div>

			<!-- Board Theme Selection -->
			<div class="mt-3">
				<div class="flex items-center justify-between mb-2">
					<span class="text-[11px] font-bold uppercase tracking-wider text-muted-foreground">
						Board Theme
					</span>
					<span class="text-[10px] text-muted-foreground font-mono capitalize">
						{$themeStore}
					</span>
				</div>
				<div class="grid grid-cols-3 gap-2">
					{#each boardThemes as t}
						<button
							type="button"
							class="flex flex-col items-center gap-1.5 p-2 rounded-lg border text-center transition { $themeStore === t.id ? 'border-foreground bg-secondary text-foreground font-bold shadow-xs' : 'border-border bg-card/60 text-muted-foreground hover:text-foreground hover:border-foreground/30' }"
							on:click={() => selectBoardTheme(t.id)}
						>
							<div class="w-8 h-8 rounded-md overflow-hidden grid grid-cols-2 shrink-0 border border-border shadow-xs">
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
			<div class="mt-4 pt-3 border-t border-border">
				<div class="flex items-center justify-between mb-2">
					<span class="text-[11px] font-bold uppercase tracking-wider text-muted-foreground">
						Interface Theme
					</span>
				</div>
				<div class="grid grid-cols-3 gap-2">
					{#each appThemes as at}
						<button
							type="button"
							class="flex flex-col items-center gap-1.5 p-2 rounded-lg border text-center transition { ($appThemeStore === at.id || ($appThemeStore === 'dark-slate' && at.id === 'dark') || ($appThemeStore === 'chess-dark' && at.id === 'dark')) ? 'border-foreground bg-secondary text-foreground font-bold shadow-xs' : 'border-border bg-card/60 text-muted-foreground hover:text-foreground hover:border-foreground/30' }"
							on:click={() => selectAppTheme(at.id)}
						>
							<div
								class="w-6 h-6 rounded-md border shrink-0 flex items-center justify-center shadow-xs"
								style="background-color: {at.bg}; border-color: {at.border};"
							>
								<div class="w-2 h-2 rounded-full" style="background-color: {at.accent};"></div>
							</div>
							<span class="text-[11px] font-medium truncate w-full">{at.name}</span>
						</button>
					{/each}
				</div>
			</div>

			<!-- Audio Controls -->
			<div class="mt-4 pt-3 border-t border-border">
				<div class="flex items-center justify-between mb-2">
					<span class="text-[11px] font-bold uppercase tracking-wider text-muted-foreground">Sound</span>
					<button
						type="button"
						class="text-xs flex items-center gap-1.5 px-2 py-1 rounded-md bg-secondary hover:bg-accent text-foreground transition"
						on:click={toggleMute}
					>
						<Icon name={$soundStore ? 'volume-2' : 'volume-x'} size={13} className={$soundStore ? 'text-foreground' : 'text-red-400'} />
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
						class="w-full accent-foreground bg-secondary h-1.5 rounded-lg cursor-pointer disabled:opacity-40"
					/>
					<span class="text-[11px] font-mono text-muted-foreground w-8 text-right">
						{Math.round(($soundStore ? $volumeStore : 0) * 100)}%
					</span>
				</div>
			</div>
		</div>
	{/if}
</div>
