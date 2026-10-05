<script lang="ts">
	import { authStore } from '$lib/stores/auth';
	import Icon from '$lib/components/icons/Icon.svelte';
	import { PIECE_SVGS } from '$lib/chess/pieces';
</script>

<div class="flex-1 flex flex-col justify-center max-w-7xl mx-auto w-full px-4 sm:px-6 py-8 lg:py-16">
	<!-- Top Radial Nova Aura -->
	<div class="absolute top-14 inset-x-0 h-96 bg-gradient-to-b from-sky-500/10 via-indigo-500/5 to-transparent pointer-events-none -z-10 blur-3xl"></div>

	<div class="grid grid-cols-1 lg:grid-cols-12 gap-12 items-center">
		<!-- Left: Interactive Hero Graphic -->
		<div class="lg:col-span-7 flex flex-col items-center lg:items-start text-center lg:text-left">
			<div class="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-sky-500/10 border border-sky-500/30 text-xs font-mono text-sky-400 font-semibold mb-6">
				<Icon name="sparkles" size={14} />
				<span>PURE GO RULES ENGINE • ZERO CLIENT DESYNC</span>
			</div>

			<h1 class="text-4xl sm:text-6xl font-black text-white tracking-tight leading-[1.1] mb-6">
				Master the Board in the <br class="hidden sm:inline" />
				<span class="bg-gradient-to-r from-sky-400 via-indigo-400 to-purple-400 bg-clip-text text-transparent">
					Cosmic Arena
				</span>
			</h1>

			<p class="text-base sm:text-lg text-slate-400 max-w-xl mb-8 leading-relaxed">
				ChessNova is a modern, high-precision chess platform powered by a pure Go rules engine, server-authoritative state, real-time WebSockets, and rated matchmaking.
			</p>

			<!-- Action Buttons -->
			<div class="flex flex-col sm:flex-row items-center gap-4 w-full sm:w-auto mb-10">
				<a
					href="/play/online"
					class="w-full sm:w-auto px-8 py-4 rounded-xl bg-gradient-to-r from-sky-500 to-indigo-600 hover:from-sky-400 hover:to-indigo-500 text-white font-black text-base shadow-[0_4px_25px_rgba(56,189,248,0.35)] hover:shadow-[0_6px_35px_rgba(56,189,248,0.5)] transition-all transform hover:-translate-y-0.5 active:scale-95 flex items-center justify-center gap-2.5"
				>
					<Icon name="zap" size={18} />
					<span>Play Online Now</span>
				</a>

				<a
					href="/play"
					class="w-full sm:w-auto px-6 py-4 rounded-xl bg-slate-900/80 hover:bg-slate-800 text-slate-200 font-bold text-sm border border-slate-700/80 transition-all flex items-center justify-center gap-2"
				>
					<Icon name="swords" size={16} className="text-slate-400" />
					<span>Local Pass & Play</span>
				</a>
			</div>

			<!-- Specs Grid -->
			<div class="grid grid-cols-3 gap-6 pt-6 border-t border-slate-800/80 w-full max-w-lg">
				<div>
					<div class="text-xl sm:text-2xl font-black font-mono text-white">100%</div>
					<div class="text-xs text-slate-500 uppercase tracking-wider font-semibold mt-0.5">FIDE Rules Pure Go</div>
				</div>
				<div>
					<div class="text-xl sm:text-2xl font-black font-mono text-sky-400">&lt;50ms</div>
					<div class="text-xs text-slate-500 uppercase tracking-wider font-semibold mt-0.5">WebSocket Latency</div>
				</div>
				<div>
					<div class="text-xl sm:text-2xl font-black font-mono text-indigo-400">Elo</div>
					<div class="text-xs text-slate-500 uppercase tracking-wider font-semibold mt-0.5">Authoritative Rating</div>
				</div>
			</div>
		</div>

		<!-- Right: Decorative Cyber Board Preview -->
		<div class="lg:col-span-5 flex justify-center">
			<div class="relative w-full max-w-[420px] aspect-square rounded-2xl overflow-hidden shadow-[0_25px_60px_rgba(0,0,0,0.85)] border-4 border-[#182133] ring-1 ring-sky-500/20 group">
				<!-- Miniature Nova Board Matrix -->
				<div class="grid grid-cols-8 grid-rows-8 w-full h-full bg-[#2b3648]">
					{#each Array(64) as _, i}
						{@const r = Math.floor(i / 8)}
						{@const f = i % 8}
						{@const isLight = (f + r) % 2 === 0}
						<div class="{isLight ? 'bg-[#cdd6e0]' : 'bg-[#2b3648]'} w-full h-full flex items-center justify-center">
							{#if i === 4}
								<div class="w-4/5 h-4/5">{@html PIECE_SVGS['k']}</div>
							{:else if i === 60}
								<div class="w-4/5 h-4/5">{@html PIECE_SVGS['K']}</div>
							{:else if i === 28}
								<div class="w-4/5 h-4/5">{@html PIECE_SVGS['Q']}</div>
							{:else if i === 35}
								<div class="w-4/5 h-4/5">{@html PIECE_SVGS['n']}</div>
							{/if}
						</div>
					{/each}
				</div>

				<!-- Glass Overlay Badge -->
				<div class="absolute bottom-4 inset-x-4 p-4 rounded-xl bg-slate-900/85 backdrop-blur-md border border-slate-700/80 shadow-lg flex items-center justify-between">
					<div class="flex items-center gap-3">
						<div class="w-10 h-10 rounded-lg bg-sky-500/20 border border-sky-500/40 flex items-center justify-center text-sky-400">
							<Icon name="crown" size={20} />
						</div>
						<div>
							<div class="font-bold text-white text-sm">Grandmaster Arena</div>
							<div class="text-[11px] text-slate-400">Live Server-Checked Games</div>
						</div>
					</div>
					<span class="px-2.5 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 text-xs font-mono font-bold">
						ONLINE
					</span>
				</div>
			</div>
		</div>
	</div>

	<!-- Mode Cards Section -->
	<div class="grid grid-cols-1 md:grid-cols-3 gap-6 mt-16 lg:mt-24">
		<a
			href="/play/online"
			class="p-6 rounded-2xl bg-gradient-to-b from-[#111723] to-[#0c1017] border border-slate-800 hover:border-sky-500/50 transition-all duration-300 group hover:shadow-[0_8px_30px_rgba(56,189,248,0.15)] flex flex-col justify-between"
		>
			<div>
				<div class="w-12 h-12 rounded-xl bg-sky-500/10 border border-sky-500/30 flex items-center justify-center text-sky-400 mb-4 group-hover:scale-110 transition-transform">
					<Icon name="zap" size={24} />
				</div>
				<h3 class="text-lg font-bold text-white group-hover:text-sky-300 transition-colors">Ranked Matchmaking</h3>
				<p class="text-xs text-slate-400 mt-2 leading-relaxed">
					Instant pairings with expanding rating tolerance. Choose from Bullet, Blitz, Rapid, or Classical time controls.
				</p>
			</div>
			<div class="flex items-center gap-1.5 text-xs font-bold text-sky-400 mt-6 group-hover:translate-x-1 transition-transform">
				<span>Queue for match</span>
				<Icon name="chevron-right" size={14} />
			</div>
		</a>

		<a
			href="/puzzles"
			class="p-6 rounded-2xl bg-gradient-to-b from-[#111723] to-[#0c1017] border border-slate-800 hover:border-emerald-500/50 transition-all duration-300 group hover:shadow-[0_8px_30px_rgba(16,185,129,0.15)] flex flex-col justify-between"
		>
			<div>
				<div class="w-12 h-12 rounded-xl bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-400 mb-4 group-hover:scale-110 transition-transform">
					<Icon name="puzzle" size={24} />
				</div>
				<h3 class="text-lg font-bold text-white group-hover:text-emerald-300 transition-colors">Tactics & Puzzles</h3>
				<p class="text-xs text-slate-400 mt-2 leading-relaxed">
					Sharpen your pattern recognition. Solve forks, pins, skewers, and back-rank checkmates with immediate move validation.
				</p>
			</div>
			<div class="flex items-center gap-1.5 text-xs font-bold text-emerald-400 mt-6 group-hover:translate-x-1 transition-transform">
				<span>Solve tactical puzzles</span>
				<Icon name="chevron-right" size={14} />
			</div>
		</a>

		<a
			href="/analysis"
			class="p-6 rounded-2xl bg-gradient-to-b from-[#111723] to-[#0c1017] border border-slate-800 hover:border-indigo-500/50 transition-all duration-300 group hover:shadow-[0_8px_30px_rgba(99,102,241,0.15)] flex flex-col justify-between"
		>
			<div>
				<div class="w-12 h-12 rounded-xl bg-indigo-500/10 border border-indigo-500/30 flex items-center justify-center text-indigo-400 mb-4 group-hover:scale-110 transition-transform">
					<Icon name="search" size={24} />
				</div>
				<h3 class="text-lg font-bold text-white group-hover:text-indigo-300 transition-colors">Deep Analysis</h3>
				<p class="text-xs text-slate-400 mt-2 leading-relaxed">
					Import FEN strings or PGN game records. Step through moves, test alternate branches, and review finished games.
				</p>
			</div>
			<div class="flex items-center gap-1.5 text-xs font-bold text-indigo-400 mt-6 group-hover:translate-x-1 transition-transform">
				<span>Open analysis board</span>
				<Icon name="chevron-right" size={14} />
			</div>
		</a>
	</div>
</div>
