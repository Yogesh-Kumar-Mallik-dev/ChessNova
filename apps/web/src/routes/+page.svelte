<script lang="ts">
	import { Chess } from 'chess.js';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Spotlight from '$lib/components/magicui/Spotlight.svelte';
	import BorderBeam from '$lib/components/magicui/BorderBeam.svelte';
	import ShimmerButton from '$lib/components/magicui/ShimmerButton.svelte';
	import BentoGrid from '$lib/components/magicui/BentoGrid.svelte';
	import BentoCard from '$lib/components/magicui/BentoCard.svelte';
	import ChessBoard from '$lib/components/chess/ChessBoard.svelte';
	import { INITIAL_FEN, type Color, type PieceType } from '$lib/chess/engine';
	import { soundEffects } from '$lib/audio/sounds';

	// Interactive hero board state
	let heroChess = new Chess();
	let heroFen = INITIAL_FEN;
	let heroTurn: Color = 'white';
	let heroLastMove: { from: string; to: string; san?: string; isCapture?: boolean; isPromotion?: boolean } | null = null;
	let heroMovesCount = 0;

	function handleHeroMove(event: CustomEvent<{ from: string; to: string; promotion?: PieceType }>) {
		const { from, to, promotion } = event.detail;
		try {
			const m = heroChess.move({ from, to, promotion: promotion || 'q' });
			if (m) {
				heroFen = heroChess.fen();
				heroTurn = heroChess.turn() === 'w' ? 'white' : 'black';
				heroLastMove = { from, to, san: m.san, isCapture: m.captured !== undefined };
				heroMovesCount++;

				// If played, auto-respond with a simple reply after 350ms to make the board feel alive!
				if (!heroChess.isGameOver()) {
					setTimeout(() => {
						const legalMoves = heroChess.moves({ verbose: true });
						if (legalMoves.length > 0) {
							// Prefer a capture or check if available, else random move
							const capture = legalMoves.find((lm) => lm.captured) || legalMoves[Math.floor(Math.random() * legalMoves.length)];
							heroChess.move(capture);
							heroFen = heroChess.fen();
							heroTurn = heroChess.turn() === 'w' ? 'white' : 'black';
							heroLastMove = { from: capture.from, to: capture.to, san: capture.san, isCapture: !!capture.captured };
							heroMovesCount++;
						}
					}, 380);
				}
			}
		} catch (e) {
			console.warn('Hero move invalid:', e);
		}
	}

	function resetHeroBoard() {
		heroChess = new Chess();
		heroFen = INITIAL_FEN;
		heroTurn = 'white';
		heroLastMove = null;
		heroMovesCount = 0;
		soundEffects.playGameStart();
	}
</script>

<div class="relative flex-1 flex flex-col justify-center max-w-7xl mx-auto w-full px-4 sm:px-6 py-8 sm:py-12 lg:py-16 overflow-hidden">
	<!-- Ambient Directional Spotlight -->
	<Spotlight class="-top-40 left-0 md:left-60 md:-top-20" fill="#38bdf8" />

	<!-- Main Hero Grid -->
	<div class="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-14 items-center relative z-10">
		<!-- Left: Punchy Consumer Copy -->
		<div class="lg:col-span-7 flex flex-col items-center lg:items-start text-center lg:text-left">
			<Badge variant="default" className="mb-6 bg-slate-900/90 border-slate-700/80 px-3.5 py-1 text-slate-300">
				<Icon name="sparkles" size={13} className="text-sky-400" />
				<span>Modern Online Chess</span>
			</Badge>

			<h1 class="text-4xl sm:text-6xl lg:text-7xl font-black text-white tracking-tight leading-[1.05] mb-6">
				Play chess <br class="hidden sm:inline" />
				<span class="bg-gradient-to-r from-sky-400 via-indigo-300 to-purple-400 bg-clip-text text-transparent">
					smooth & distraction-free.
				</span>
			</h1>

			<p class="text-base sm:text-lg text-slate-300/90 max-w-xl mb-8 leading-relaxed font-normal">
				Instant matchmaking across all time controls, tiered tactical puzzles, natural bot sparring, and deep post-game analysis without the noise.
			</p>

			<!-- Action Buttons with ShimmerButton -->
			<div class="flex flex-col sm:flex-row items-center gap-3.5 w-full sm:w-auto mb-10">
				<ShimmerButton href="/play/online" shimmerColor="#38bdf8" class="w-full sm:w-auto text-sm font-bold shadow-xl shadow-sky-500/10">
					<Icon name="zap" size={18} className="text-sky-400" />
					<span>Play Online</span>
				</ShimmerButton>

				<a href="/puzzles" class="w-full sm:w-auto">
					<Button variant="outline" size="lg" className="w-full sm:w-auto border-slate-700/80 hover:bg-slate-800/80">
						<Icon name="puzzle" size={16} className="text-emerald-400" />
						<span>Puzzles</span>
					</Button>
				</a>

				<a href="/play" class="w-full sm:w-auto">
					<Button variant="secondary" size="lg" className="w-full sm:w-auto">
						<Icon name="swords" size={16} className="text-indigo-400" />
						<span>Practice Bot</span>
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
					<div class="text-[11px] text-slate-400 uppercase tracking-wider font-semibold mt-0.5">Instant Clocks</div>
				</div>
				<div>
					<div class="text-xl sm:text-2xl font-black font-mono text-emerald-400 tracking-tight">1,000+</div>
					<div class="text-[11px] text-slate-400 uppercase tracking-wider font-semibold mt-0.5">Curated Puzzles</div>
				</div>
			</div>
		</div>

		<!-- Right: Real Interactive Showcase Board with BorderBeam -->
		<div class="lg:col-span-5 flex flex-col items-center justify-center w-full">
			<div class="relative w-full max-w-[420px] aspect-square rounded-2xl p-2.5 bg-slate-900/80 backdrop-blur-xl border border-slate-800 shadow-2xl overflow-hidden group">
				<BorderBeam size={220} duration={10} colorFrom="#38bdf8" colorTo="#a855f7" />

				<!-- Real Interactive Board -->
				<ChessBoard
					fen={heroFen}
					orientation="white"
					turn={heroTurn}
					lastMove={heroLastMove}
					interactive={true}
					boardSizeClass="w-full"
					disableGameOverModal={true}
					on:move={handleHeroMove}
				/>
			</div>

			<!-- Interactive Board Status Pill & Reset -->
			<div class="w-full max-w-[420px] mt-3 flex items-center justify-between px-2 text-xs text-slate-400">
				<div class="flex items-center gap-2">
					<span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
					<span class="font-medium">
						{heroMovesCount === 0 ? 'Try making a move on the board!' : `Move ${heroMovesCount} • ${heroTurn} to play`}
					</span>
				</div>
				{#if heroMovesCount > 0}
					<button
						class="text-[11px] text-sky-400 hover:text-sky-300 font-semibold flex items-center gap-1 transition"
						on:click={resetHeroBoard}
					>
						<Icon name="rotate-ccw" size={12} />
						<span>Reset</span>
					</button>
				{/if}
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
				description="Full game accuracy scores, move classifications, key moment detections, evaluation graphs, and simple reviews with advanced tools on demand."
				href="/analysis"
				cta="Open Board"
				colSpan="md:col-span-1"
			/>
		</BentoGrid>
	</div>
</div>
