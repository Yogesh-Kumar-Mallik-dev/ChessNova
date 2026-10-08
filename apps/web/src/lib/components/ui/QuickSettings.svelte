<script lang="ts">
	import Icon from '$lib/components/icons/Icon.svelte';
	import { themeStore, volumeStore, soundStore, type BoardTheme } from '$lib/stores/preferences';

	let isOpen = false;

	const themes: { id: BoardTheme; name: string; light: string; dark: string }[] = [
		{ id: 'cyber', name: 'Cyber', light: '#334155', dark: '#1e293b' },
		{ id: 'emerald', name: 'Emerald', light: '#d1fae5', dark: '#059669' },
		{ id: 'wood', name: 'Wood', light: '#fed7aa', dark: '#b45309' },
		{ id: 'ocean', name: 'Ocean', light: '#bae6fd', dark: '#0284c7' }
	];

	function selectTheme(t: BoardTheme) {
		themeStore.set(t);
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
		class="p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800/60 border border-transparent hover:border-slate-700/60 transition flex items-center justify-center {isOpen ? 'bg-slate-800 text-sky-400' : ''}"
		on:click={() => (isOpen = !isOpen)}
		title="Appearance & Audio Settings"
		aria-label="Settings"
	>
		<Icon name="settings" size={18} />
	</button>

	{#if isOpen}
		<!-- Backdrop for closing -->
		<!-- svelte-ignore a11y-click-events-have-key-events -->
		<!-- svelte-ignore a11y-no-static-element-interactions -->
		<div class="fixed inset-0 z-40" on:click={close}></div>

		<!-- Dropdown Panel -->
		<div class="absolute right-0 mt-2 w-72 bg-slate-900 border border-slate-700/90 rounded-2xl p-4 shadow-2xl z-50 animate-in fade-in zoom-in-95 duration-150">
			<div class="flex items-center justify-between pb-3 border-b border-slate-800">
				<div class="flex items-center gap-2">
					<Icon name="settings" size={16} className="text-sky-400" />
					<h3 class="text-xs font-bold uppercase tracking-wider text-slate-200">Board & Audio</h3>
				</div>
				<button class="text-slate-400 hover:text-white text-xs" on:click={close}>
					<Icon name="x" size={14} />
				</button>
			</div>

			<!-- Theme Selection -->
			<div class="mt-3">
				<span class="text-[11px] font-bold uppercase tracking-wider text-slate-400 mb-2 block">
					Board Theme
				</span>
				<div class="grid grid-cols-2 gap-2">
					{#each themes as t}
						<button
							type="button"
							class="flex items-center gap-2 p-2 rounded-xl border text-left transition { $themeStore === t.id ? 'border-sky-500 bg-sky-500/10 text-white font-bold' : 'border-slate-800 bg-slate-800/40 text-slate-300 hover:border-slate-700' }"
							on:click={() => selectTheme(t.id)}
						>
							<div class="w-5 h-5 rounded-md overflow-hidden grid grid-cols-2 shrink-0 border border-slate-700">
								<div style="background-color: {t.light}"></div>
								<div style="background-color: {t.dark}"></div>
								<div style="background-color: {t.dark}"></div>
								<div style="background-color: {t.light}"></div>
							</div>
							<span class="text-xs">{t.name}</span>
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
						class="text-xs flex items-center gap-1.5 px-2 py-1 rounded-lg bg-slate-800 hover:bg-slate-750 text-slate-300 transition"
						on:click={toggleMute}
					>
						<Icon name={$soundStore ? 'volume-2' : 'volume-x'} size={14} className={$soundStore ? 'text-sky-400' : 'text-rose-400'} />
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
						class="w-full accent-sky-400 bg-slate-800 h-1.5 rounded-lg cursor-pointer disabled:opacity-40"
					/>
					<span class="text-[11px] font-mono text-slate-400 w-8 text-right">
						{Math.round(($soundStore ? $volumeStore : 0) * 100)}%
					</span>
				</div>
			</div>
		</div>
	{/if}
</div>
