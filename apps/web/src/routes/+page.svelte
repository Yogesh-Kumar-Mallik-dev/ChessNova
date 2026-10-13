<script lang="ts">
	import { authStore } from '$lib/stores/auth';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Spotlight from '$lib/components/magicui/Spotlight.svelte';
	import BorderBeam from '$lib/components/magicui/BorderBeam.svelte';
	import ShimmerButton from '$lib/components/magicui/ShimmerButton.svelte';
	import BentoGrid from '$lib/components/magicui/BentoGrid.svelte';
	import BentoCard from '$lib/components/magicui/BentoCard.svelte';
	import { PIECE_SVGS } from '$lib/chess/pieces';
</script>

<div class="relative flex-1 flex flex-col justify-center max-w-7xl mx-auto w-full px-4 sm:px-6 py-8 sm:py-12 lg:py-20 overflow-hidden">
	<!-- Ambient Animated Spotlight -->
	<Spotlight class="-top-36 left-0 md:left-60 md:-top-20" fill="#38bdf8" />

	<!-- Main Hero Grid -->
	<div class="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-14 items-center relative z-10">
		<!-- Left: Punchy, clean consumer copy -->
		<div class="lg:col-span-7 flex flex-col items-center lg:items-start text-center lg:text-left">
			<Badge variant="default" className="mb-6 bg-slate-900/90 border-slate-700/80 px-3.5 py-1 text-slate-300">
				<Icon name="sparkles" size={13} className="text-sky-400" />
				<span>Compete • Analyze • Master</span>
			</Badge>

			<h1 class="text-4xl sm:text-6xl lg:text-7xl font-black text-white tracking-tight leading-[1.05] mb-6">
				Play chess <br class="hidden sm:inline" />
				<span class="bg-gradient-to-r from-sky-400 via-indigo-300 to-purple-400 bg-clip-text text-transparent">
					unfiltered & fast.
				</span>
			</h1>

			<p class="text-base sm:text-lg text-slate-300/90 max-w-xl mb-8 leading-relaxed font-normal">
				Instant rated matchmaking across all time controls, tiered tactical puzzles, humanized bot sparring with natural delays, and deep post-game analysis.
			</p>

			<!-- Action Buttons with Magic UI ShimmerButton -->
			<div class="flex flex-col sm:flex-row items-center gap-3.5 w-full sm:w-auto mb-10">
				<ShimmerButton href="/play/online" shimmerColor="#38bdf8" class="w-full sm:w-auto text-sm font-bold shadow-xl shadow-sky-500/10">
					<Icon name="zap" size={18} className="text-sky-400" />
					<span>Play Online</span>
				</ShimmerButton>

				<a href="/puzzles" class="w-full sm:w-auto">
					<Button variant="outline" size="lg" className="w-full sm:w-auto border-slate-700/80 hover:bg-slate-800/80">
						<Icon name="puzzle" size={16} className="text-emerald-400" />
						<span>Puzzles Lab</span>
					</Button>
				</a>

				<a href="/play" class="w-full sm:w-auto">
					<Button variant="secondary" size="lg" className="w-full sm:w-auto">
						<Icon name="swords" size={16} className="text-indigo-400" />
						<span>Play Bot</span>
					</Button>
				</a>
			</div>

			<!-- Live Platform Stats -->
			<div class="grid grid-cols-3 gap-6 pt-6 border-t border-slate-800/80 w-full max-w-lg text-left">
				<div>
					<div class="text-xl sm:text-2xl font-black font-mono text-white tracking-tight">400–2800</div>
					<div class="text-[11px] text-slate-400 uppercase tracking-wider font-semibold mt-0.5">Separate Elo</div>
				</div>
				<div>
					<div class="text-xl sm:text-2xl font-black font-mono text-sky-400 tracking-tight">Zero Lag</div>
					<div class="text-[11px] text-slate-400 uppercase tracking-wider font-semibold mt-0.5">Fluid Clocks</div>
				</div>
				<div>
					<div class="text-xl sm:text-2xl font-black font-mono text-emerald-400 tracking-tight">1,000+</div>
					<div class="text-[11px] text-slate-400 uppercase tracking-wider font-semibold mt-0.5">Curated Puzzles</div>
				</div>
			</div>
		</div>

		<!-- Right: Showcase Board Card with BorderBeam -->
		<div class="lg:col-span-5 flex justify-center w-full">
			<div class="relative w-full max-w-[420px] aspect-square rounded-2xl p-3 bg-slate-900/70 backdrop-blur-xl border border-slate-800 shadow-2xl overflow-hidden group">
				<BorderBeam size={220} duration={10} colorFrom="#38bdf8" colorTo="#a855f7" />

				<!-- Miniature Board Matrix -->
				<div class="grid grid-cols-8 grid-rows-8 w-full h-full rounded-xl overflow-hidden border border-slate-800/80 shadow-inner">
					{#each Array(64) as _, i}
						{@const r = Math.floor(i / 8)}
						{@const f = i % 8}
						{@const isLight = (f + r) % 2 === 0}
						<div class="{isLight ? 'bg-[#cdd6e0]' : 'bg-[#2b3648]'} w-full h-full flex items-center justify-center select-none transition-colors">
							{#if i === 4}
								<div class="w-4/5 h-4/5">{@html PIECE_SVGS['k']}</div>
							{:else if i === 60}
								<div class="w-4/5 h-4/5">{@html PIECE_SVGS['K']}</div>
							{:else if i === 28}
								<div class="w-4/5 h-4/5 animate-pulse">{@html PIECE_SVGS['Q']}</div>
							{:else if i === 35}
								<div class="w-4/5 h-4/5">{@html PIECE_SVGS['n']}</div>
							{/if}
						</div>
					{/each}
				</div>

				<!-- Floating Glass Status Pill -->
				<div class="absolute bottom-5 inset-x-5 p-3.5 rounded-xl bg-slate-950/85 backdrop-blur-md border border-slate-700/80 shadow-2xl flex items-center justify-between">
					<div class="flex items-center gap-3">
						<div class="w-9 h-9 rounded-lg bg-sky-500/20 border border-sky-500/40 flex items-center justify-center text-sky-400">
							<Icon name="crown" size={18} />
						</div>
						<div>
							<div class="font-bold text-white text-xs sm:text-sm">Ready to play</div>
							<div class="text-[11px] text-slate-400">Join players worldwide</div>
						</div>
					</div>
					<Badge variant="success">LIVE</Badge>
				</div>
			</div>
		</div>
	</div>

	<!-- Bento Feature Matrix -->
	<div class="mt-16 sm:mt-24 relative z-10">
		<div class="mb-8 text-center sm:text-left">
			<h2 class="text-2xl sm:text-3xl font-black text-white tracking-tight">Everything you need to compete and grow</h2>
			<p class="text-sm text-slate-400 mt-1.5 font-normal">Built for smooth games, precise tactical training, and master-level reviews.</p>
		</div>

		<BentoGrid>
			<BentoCard
				name="Real-Time Matchmaking"
				icon="zap"
				description="Bullet, Blitz, Rapid, and Classical pools with instant fair matchmaking against opponents at your exact skill rating."
				href="/play/online"
				cta="Enter Queue"
				colSpan="md:col-span-1"
			/>

			<BentoCard
				name="Curated Tactics Lab"
				icon="puzzle"
				description="Progressive tactical puzzles organized into Mate, Fork, Pin, Skewer, and tier levels. Includes separate puzzle rating and daily challenges."
				href="/puzzles"
				cta="Solve Tactics"
				colSpan="md:col-span-1"
			/>

			<BentoCard
				name="Post-Game Analysis"
				icon="search"
				description="Full game accuracy scores, move classifications, brilliant move detections, evaluation graphs, and complete PGN export."
				href="/analysis"
				cta="Open Board"
				colSpan="md:col-span-1"
			/>
		</BentoGrid>
	</div>
</div>
