<script lang="ts">
	import { authStore } from '$lib/stores/auth';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import { PIECE_SVGS } from '$lib/chess/pieces';
</script>

<div class="flex-1 flex flex-col justify-center max-w-7xl mx-auto w-full px-4 sm:px-6 py-8 lg:py-16">
	<!-- Subtle ambient glow background -->
	<div class="absolute top-12 inset-x-0 h-[450px] bg-gradient-to-b from-sky-500/10 via-indigo-500/5 to-transparent pointer-events-none -z-10 blur-3xl"></div>

	<!-- Main Hero Grid -->
	<div class="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-14 items-center">
		<!-- Left: Punchy, clean consumer copy -->
		<div class="lg:col-span-7 flex flex-col items-center lg:items-start text-center lg:text-left">
			<Badge variant="default" className="mb-6">
				<Icon name="sparkles" size={13} />
				<span>Compete • Analyze • Master</span>
			</Badge>

			<h1 class="text-4xl sm:text-6xl font-black text-white tracking-tight leading-[1.08] mb-6">
				Play chess online <br class="hidden sm:inline" />
				<span class="bg-gradient-to-r from-sky-400 via-indigo-300 to-purple-400 bg-clip-text text-transparent">
					at the highest speed.
				</span>
			</h1>

			<p class="text-base sm:text-lg text-slate-300 max-w-xl mb-8 leading-relaxed font-normal">
				Instant rated matchmaking, certified tactical puzzles, intelligent humanized bot practice, and deep game reviews.
			</p>

			<!-- Action Buttons -->
			<div class="flex flex-col sm:flex-row items-center gap-3.5 w-full sm:w-auto mb-10">
				<a href="/play/online" class="w-full sm:w-auto">
					<Button variant="glow" size="lg" className="w-full sm:w-auto">
						<Icon name="zap" size={18} />
						<span>Play Online</span>
					</Button>
				</a>

				<a href="/play" class="w-full sm:w-auto">
					<Button variant="outline" size="lg" className="w-full sm:w-auto">
						<Icon name="swords" size={16} className="text-slate-400" />
						<span>Pass & Play</span>
					</Button>
				</a>

				<a href="/puzzles" class="w-full sm:w-auto">
					<Button variant="secondary" size="lg" className="w-full sm:w-auto">
						<Icon name="puzzle" size={16} className="text-emerald-400" />
						<span>Puzzles</span>
					</Button>
				</a>
			</div>

			<!-- Live Feature Highlights -->
			<div class="grid grid-cols-3 gap-6 pt-6 border-t border-slate-800/80 w-full max-w-lg text-left">
				<div>
					<div class="text-xl sm:text-2xl font-black font-mono text-white">400–2800</div>
					<div class="text-xs text-slate-400 uppercase tracking-wider font-semibold mt-0.5">Separate Elo</div>
				</div>
				<div>
					<div class="text-xl sm:text-2xl font-black font-mono text-sky-400">100% Free</div>
					<div class="text-xs text-slate-400 uppercase tracking-wider font-semibold mt-0.5">No Paywalls</div>
				</div>
				<div>
					<div class="text-xl sm:text-2xl font-black font-mono text-indigo-400">Lichess</div>
					<div class="text-xs text-slate-400 uppercase tracking-wider font-semibold mt-0.5">Verified Puzzles</div>
				</div>
			</div>
		</div>

		<!-- Right: Interactive Board Showcase -->
		<div class="lg:col-span-5 flex justify-center">
			<Card className="w-full max-w-[420px] aspect-square p-3 shadow-2xl relative group overflow-hidden border-slate-700/60">
				<!-- Miniature Board Matrix -->
				<div class="grid grid-cols-8 grid-rows-8 w-full h-full rounded-xl overflow-hidden border border-slate-800 shadow-inner">
					{#each Array(64) as _, i}
						{@const r = Math.floor(i / 8)}
						{@const f = i % 8}
						{@const isLight = (f + r) % 2 === 0}
						<div class="{isLight ? 'bg-[#cdd6e0]' : 'bg-[#2b3648]'} w-full h-full flex items-center justify-center select-none">
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

				<!-- Floating Glass Status Pill -->
				<div class="absolute bottom-6 inset-x-6 p-3.5 rounded-xl bg-slate-950/85 backdrop-blur-md border border-slate-700/80 shadow-2xl flex items-center justify-between">
					<div class="flex items-center gap-3">
						<div class="w-9 h-9 rounded-lg bg-sky-500/20 border border-sky-500/40 flex items-center justify-center text-sky-400">
							<Icon name="crown" size={18} />
						</div>
						<div>
							<div class="font-bold text-white text-xs sm:text-sm">Ready to play</div>
							<div class="text-[11px] text-slate-400">Join thousands of players</div>
						</div>
					</div>
					<Badge variant="success">LIVE</Badge>
				</div>
			</Card>
		</div>
	</div>

	<!-- Mode Cards Section -->
	<div class="grid grid-cols-1 md:grid-cols-3 gap-5 mt-16 lg:mt-24">
		<a href="/play/online" class="block group">
			<Card interactive className="p-6 h-full flex flex-col justify-between border-slate-800 hover:border-sky-500/50">
				<div>
					<div class="w-12 h-12 rounded-xl bg-sky-500/10 border border-sky-500/30 flex items-center justify-center text-sky-400 mb-4 group-hover:scale-105 transition-transform">
						<Icon name="zap" size={22} />
					</div>
					<h3 class="text-base font-bold text-white group-hover:text-sky-300 transition-colors">Quick Matchmaking</h3>
					<p class="text-xs text-slate-400 mt-2 leading-relaxed">
						Bullet, Blitz, Rapid, and Classical pools with real-time pairing against opponents of equal rating.
					</p>
				</div>
				<div class="flex items-center gap-1.5 text-xs font-semibold text-sky-400 mt-6 group-hover:translate-x-1 transition-transform">
					<span>Join match queue</span>
					<Icon name="chevron-right" size={14} />
				</div>
			</Card>
		</a>

		<a href="/puzzles" class="block group">
			<Card interactive className="p-6 h-full flex flex-col justify-between border-slate-800 hover:border-emerald-500/50">
				<div>
					<div class="w-12 h-12 rounded-xl bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-400 mb-4 group-hover:scale-105 transition-transform">
						<Icon name="puzzle" size={22} />
					</div>
					<h3 class="text-base font-bold text-white group-hover:text-emerald-300 transition-colors">Tactics Laboratory</h3>
					<p class="text-xs text-slate-400 mt-2 leading-relaxed">
						Lichess-verified tactical puzzles categorized by Mate, Fork, Pin, Skewer, and progressive Tiers. Includes daily challenges.
					</p>
				</div>
				<div class="flex items-center gap-1.5 text-xs font-semibold text-emerald-400 mt-6 group-hover:translate-x-1 transition-transform">
					<span>Practice tactics</span>
					<Icon name="chevron-right" size={14} />
				</div>
			</Card>
		</a>

		<a href="/analysis" class="block group">
			<Card interactive className="p-6 h-full flex flex-col justify-between border-slate-800 hover:border-indigo-500/50">
				<div>
					<div class="w-12 h-12 rounded-xl bg-indigo-500/10 border border-indigo-500/30 flex items-center justify-center text-indigo-400 mb-4 group-hover:scale-105 transition-transform">
						<Icon name="search" size={22} />
					</div>
					<h3 class="text-base font-bold text-white group-hover:text-indigo-300 transition-colors">Post-Game Analysis</h3>
					<p class="text-xs text-slate-400 mt-2 leading-relaxed">
						Review every move with full accuracy scoring, move classifications, alternate branch evaluations, and PGN export.
					</p>
				</div>
				<div class="flex items-center gap-1.5 text-xs font-semibold text-indigo-400 mt-6 group-hover:translate-x-1 transition-transform">
					<span>Open analysis board</span>
					<Icon name="chevron-right" size={14} />
				</div>
			</Card>
		</a>
	</div>
</div>
