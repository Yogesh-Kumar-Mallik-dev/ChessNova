<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Icon from '$lib/components/icons/Icon.svelte';
	import {
		engineStore,
		queueEngineDownload,
		clearEngineCache,
		setLocalEngineEnabled
	} from '$lib/engine/localEngine';

	export let open = false;

	const dispatch = createEventDispatcher();

	function close() {
		dispatch('close');
	}

	function handleToggle() {
		setLocalEngineEnabled(!$engineStore.isEnabled);
	}

	async function handleDownload() {
		await queueEngineDownload();
	}

	async function handleClear() {
		if (confirm('Remove downloaded Stockfish files from browser cache? Cloud engine will be used instead.')) {
			await clearEngineCache();
		}
	}
</script>

{#if open}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-xs">
		<!-- Backdrop click to close -->
		<div class="fixed inset-0" on:click={close} on:keydown={(e) => e.key === 'Escape' && close()} role="button" tabindex="0"></div>

		<!-- Modal Dialog -->
		<div
			class="relative w-full max-w-md bg-slate-900 border border-slate-700/80 rounded-2xl p-5 sm:p-6 shadow-2xl text-slate-100 z-10 flex flex-col gap-4"
			role="dialog"
			aria-modal="true"
		>
			<!-- Header -->
			<div class="flex items-center justify-between border-b border-slate-800 pb-3">
				<div class="flex items-center gap-2.5">
					<div class="w-8 h-8 rounded-xl bg-emerald-500/15 border border-emerald-500/30 flex items-center justify-center text-emerald-400">
						<Icon name="zap" size={16} />
					</div>
					<div>
						<h2 class="text-sm sm:text-base font-bold text-white">Stockfish Engine Settings</h2>
						<p class="text-[11px] text-slate-400">Local compute & background caching</p>
					</div>
				</div>

				<button
					class="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition"
					on:click={close}
					title="Close"
				>
					<Icon name="x" size={16} />
				</button>
			</div>

			<!-- Active Engine Badge & Info -->
			<div class="p-3.5 rounded-xl bg-slate-800/60 border border-slate-700/50 flex flex-col gap-2">
				<div class="flex items-center justify-between">
					<span class="text-xs text-slate-400 font-medium">Active Evaluation Engine:</span>
					{#if $engineStore.source === 'tauri'}
						<span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-semibold bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
							<Icon name="zap" size={10} /> Desktop Native (0ms)
						</span>
					{:else if $engineStore.source === 'wasm'}
						<span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-semibold bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
							<Icon name="zap" size={10} /> Local Browser WASM (0ms)
						</span>
					{:else}
						<span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-semibold bg-sky-500/20 text-sky-400 border border-sky-500/30">
							<Icon name="sparkles" size={10} /> Cloud Cluster
						</span>
					{/if}
				</div>

				<p class="text-xs text-slate-300 leading-relaxed">
					{#if $engineStore.source === 'tauri'}
						Running natively on your computer via desktop IPC. Zero network latency and offline capable.
					{:else if $engineStore.source === 'wasm'}
						Running client-side inside a background Web Worker. Evaluations happen in your browser with zero server load.
					{:else}
						Positions are evaluated by the remote ChessNova Stockfish server cluster.
					{/if}
				</p>
			</div>

			<!-- Local Engine Toggle -->
			<div class="flex items-center justify-between p-3 rounded-xl bg-slate-800/40 border border-slate-800">
				<div class="flex flex-col">
					<span class="text-xs font-semibold text-white">Enable Local Engine</span>
					<span class="text-[11px] text-slate-400">Prefers local computation whenever available</span>
				</div>

				<button
					class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-hidden cursor-pointer {$engineStore.isEnabled
						? 'bg-emerald-500'
						: 'bg-slate-700'}"
					on:click={handleToggle}
					role="switch"
					aria-checked={$engineStore.isEnabled}
				>
					<span
						class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform {$engineStore.isEnabled
							? 'translate-x-6'
							: 'translate-x-1'}"
					></span>
				</button>
			</div>

			<!-- Browser Storage & Background Download (For Web) -->
			{#if !$engineStore.isTauri}
				<div class="flex flex-col gap-2.5 p-3.5 rounded-xl bg-slate-800/40 border border-slate-800">
					<div class="flex items-center justify-between">
						<span class="text-xs font-semibold text-white">Offline Browser Cache</span>
						<span class="text-[11px] text-slate-400">Isolated Origin Storage</span>
					</div>

					{#if $engineStore.isCached}
						<div class="flex items-center justify-between pt-1">
							<div class="flex items-center gap-1.5 text-xs text-emerald-400 font-medium">
								<Icon name="check" size={13} />
								<span>Stored locally (~655 KB)</span>
							</div>

							<button
								class="px-2.5 py-1 text-[11px] font-semibold text-rose-300 hover:text-rose-100 hover:bg-rose-500/20 border border-rose-500/30 rounded-lg transition"
								on:click={handleClear}
							>
								Remove Cache
							</button>
						</div>
					{:else if $engineStore.status === 'downloading'}
						<div class="flex flex-col gap-1.5 pt-1">
							<div class="flex justify-between text-xs text-slate-300 font-medium">
								<span>Downloading engine in background...</span>
								<span>{$engineStore.downloadProgress}%</span>
							</div>
							<div class="w-full h-1.5 bg-slate-700 rounded-full overflow-hidden">
								<div
									class="h-full bg-emerald-500 transition-all duration-300"
									style="width: {$engineStore.downloadProgress}%"
								></div>
							</div>
							<span class="text-[10px] text-slate-400">Non-blocking: will not interrupt gameplay or moves.</span>
						</div>
					{:else}
						<div class="flex items-center justify-between pt-1">
							<span class="text-xs text-slate-400">Not downloaded yet</span>

							<button
								class="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-semibold text-emerald-300 bg-emerald-500/15 hover:bg-emerald-500/25 border border-emerald-500/30 rounded-xl transition cursor-pointer"
								on:click={handleDownload}
							>
								<Icon name="download" size={12} />
								<span>Download (~655 KB)</span>
							</button>
						</div>
					{/if}
				</div>
			{/if}

			<!-- Non-intrusive safety note -->
			<div class="text-[11px] text-slate-400 leading-normal flex items-start gap-1.5">
				<Icon name="shield" size={13} className="text-slate-500 shrink-0 mt-0.5" />
				<span>
					The local engine runs exclusively in a sandboxed Web Worker. It never transmits board positions over the network or uses CPU during multiplayer matches.
				</span>
			</div>
		</div>
	</div>
{/if}
