<script lang="ts">
	import { Chess } from 'chess.js';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Card from '$lib/components/ui/Card.svelte';
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

				// Auto-respond with an immediate reply to make the board feel responsive
				if (!heroChess.isGameOver()) {
					setTimeout(() => {
						const legalMoves = heroChess.moves({ verbose: true });
						if (legalMoves.length > 0) {
							const capture = legalMoves.find((lm) => lm.captured) || legalMoves[Math.floor(Math.random() * legalMoves.length)];
							heroChess.move(capture);
							heroFen = heroChess.fen();
							heroTurn = heroChess.turn() === 'w' ? 'white' : 'black';
							heroLastMove = { from: capture.from, to: capture.to, san: capture.san, isCapture: !!capture.captured };
							heroMovesCount++;
						}
					}, 320);
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

<div class="relative flex-1 flex flex-col justify-center max-w-7xl mx-auto w-full px-4 sm:px-6 py-8 sm:py-12 lg:py-16">
	<!-- Main Hero Section -->
	<div class="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-14 items-center">
		<!-- Left: Typography & Value Proposition -->
		<div class="lg:col-span-7 flex flex-col items-center lg:items-start text-center lg:text-left">
			<Badge variant="outline" className="mb-6 px-3 py-1 font-mono text-xs tracking-wider uppercase text-muted-foreground border-border">
				<span class="inline-block w-1.5 h-1.5 rounded-full bg-foreground mr-1.5"></span>
				<span>Competitive Chess Platform</span>
			</Badge>

			<h1 class="text-4xl sm:text-6xl lg:text-7xl font-black text-foreground tracking-tight leading-[1.04] mb-6">
				Play chess.<br />
				<span class="text-muted-foreground font-light tracking-normal">
					Pure. Precise. Focused.
				</span>
			</h1>

			<p class="text-base sm:text-lg text-muted-foreground max-w-xl mb-8 leading-relaxed font-normal">
				Instant low-latency matchmaking across all rating tiers, progressive tactical training, Stockfish engine sparring, and deep post-game analysis without the visual noise.
			</p>

			<!-- Action Buttons -->
			<div class="flex flex-col sm:flex-row items-center gap-3 w-full sm:w-auto mb-10">
				<a href="/play/online" class="w-full sm:w-auto">
					<Button variant="primary" size="lg" className="w-full sm:w-auto text-base">
						<Icon name="zap" size={17} />
						<span>Play Online</span>
					</Button>
				</a>

				<a href="/puzzles" class="w-full sm:w-auto">
					<Button variant="secondary" size="lg" className="w-full sm:w-auto">
						<Icon name="puzzle" size={16} />
						<span>Tactics Lab</span>
					</Button>
				</a>

				<a href="/play" class="w-full sm:w-auto">
					<Button variant="outline" size="lg" className="w-full sm:w-auto">
						<Icon name="swords" size={16} />
						<span>Spar with Bot</span>
					</Button>
				</a>
			</div>

			<!-- Live Platform Stats -->
			<div class="grid grid-cols-3 gap-6 pt-6 border-t border-border w-full max-w-lg text-left">
				<div>
					<div class="text-xl sm:text-2xl font-black font-mono text-foreground tracking-tight">400–2800</div>
					<div class="text-[11px] text-muted-foreground uppercase tracking-wider font-semibold mt-0.5">Rating Pools</div>
				</div>
				<div>
					<div class="text-xl sm:text-2xl font-black font-mono text-foreground tracking-tight">Sub-10ms</div>
					<div class="text-[11px] text-muted-foreground uppercase tracking-wider font-semibold mt-0.5">Clock Sync</div>
				</div>
				<div>
					<div class="text-xl sm:text-2xl font-black font-mono text-foreground tracking-tight">1,000+</div>
					<div class="text-[11px] text-muted-foreground uppercase tracking-wider font-semibold mt-0.5">Curated Puzzles</div>
				</div>
			</div>
		</div>

		<!-- Right: Showcase Interactive Board -->
		<div class="lg:col-span-5 flex flex-col items-center justify-center w-full">
			<div class="w-full max-w-[420px] rounded-2xl p-2.5 bg-card border border-border shadow-2xl overflow-hidden">
				<!-- Board Header Bar -->
				<div class="flex items-center justify-between px-2 pb-2 mb-1 border-b border-border text-xs text-muted-foreground">
					<div class="flex items-center gap-2">
						<span class="w-2 h-2 rounded-full bg-foreground animate-pulse"></span>
						<span class="font-medium text-foreground text-[11px]">
							{heroMovesCount === 0 ? 'Interactive Board — Try a move' : `Move ${heroMovesCount} • ${heroTurn} to play`}
						</span>
					</div>
					{#if heroMovesCount > 0}
						<button
							class="text-[11px] text-muted-foreground hover:text-foreground font-medium flex items-center gap-1 transition"
							on:click={resetHeroBoard}
							title="Reset board"
						>
							<Icon name="rotate-ccw" size={11} />
							<span>Reset</span>
						</button>
					{/if}
				</div>

				<!-- Board Canvas -->
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

			<p class="text-[11px] text-muted-foreground text-center mt-3 font-mono">
				Standard FIDE algebraic rules · Drag or click to move
			</p>
		</div>
	</div>

	<!-- Core Capabilities Grid -->
	<div class="mt-16 sm:mt-24">
		<div class="mb-8 flex flex-col sm:flex-row sm:items-end justify-between gap-4 border-b border-border pb-4">
			<div>
				<h2 class="text-2xl sm:text-3xl font-black text-foreground tracking-tight">Built for serious play</h2>
				<p class="text-sm text-muted-foreground mt-1">Engineered for low-latency competition, deliberate study, and master-level reviews.</p>
			</div>
			<div class="font-mono text-xs text-muted-foreground uppercase tracking-wider">
				Features & Protocols
			</div>
		</div>

		<div class="grid grid-cols-1 md:grid-cols-3 gap-5">
			<!-- Online Matchmaking -->
			<Card className="flex flex-col justify-between p-6 transition-colors hover:border-foreground/30">
				<div>
					<div class="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center text-foreground mb-5 border border-border">
						<Icon name="zap" size={20} />
					</div>
					<h3 class="text-lg font-bold text-foreground mb-2">Real-Time Matchmaking</h3>
					<p class="text-sm text-muted-foreground leading-relaxed">
						Bullet, Blitz, Rapid, and Classical pools with fair Elo pairings, sub-millisecond clock reconciliation, and reconnection safeguards.
					</p>
				</div>
				<div class="pt-6 mt-6 border-t border-border">
					<a href="/play/online">
						<Button variant="ghost" size="sm" className="w-full justify-between px-0 hover:bg-transparent text-foreground hover:text-foreground/80">
							<span class="font-semibold text-xs uppercase tracking-wider">Enter Queue</span>
							<Icon name="arrow-right" size={14} />
						</Button>
					</a>
				</div>
			</Card>

			<!-- Tactics Lab -->
			<Card className="flex flex-col justify-between p-6 transition-colors hover:border-foreground/30">
				<div>
					<div class="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center text-foreground mb-5 border border-border">
						<Icon name="puzzle" size={20} />
					</div>
					<h3 class="text-lg font-bold text-foreground mb-2">Curated Tactics Lab</h3>
					<p class="text-sm text-muted-foreground leading-relaxed">
						Tactical positions categorized by motif (Mate, Fork, Pin, Skewer, Deflection) with separate puzzle ratings and streak tracking.
					</p>
				</div>
				<div class="pt-6 mt-6 border-t border-border">
					<a href="/puzzles">
						<Button variant="ghost" size="sm" className="w-full justify-between px-0 hover:bg-transparent text-foreground hover:text-foreground/80">
							<span class="font-semibold text-xs uppercase tracking-wider">Solve Tactics</span>
							<Icon name="arrow-right" size={14} />
						</Button>
					</a>
				</div>
			</Card>

			<!-- Analysis Board -->
			<Card className="flex flex-col justify-between p-6 transition-colors hover:border-foreground/30">
				<div>
					<div class="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center text-foreground mb-5 border border-border">
						<Icon name="search" size={20} />
					</div>
					<h3 class="text-lg font-bold text-foreground mb-2">Deep Game Analysis</h3>
					<p class="text-sm text-muted-foreground leading-relaxed">
						Stockfish engine evaluation, accuracy percentages, move classifications (Brilliant, Best, Mistake, Blunder), and full PGN export.
					</p>
				</div>
				<div class="pt-6 mt-6 border-t border-border">
					<a href="/analysis">
						<Button variant="ghost" size="sm" className="w-full justify-between px-0 hover:bg-transparent text-foreground hover:text-foreground/80">
							<span class="font-semibold text-xs uppercase tracking-wider">Open Board</span>
							<Icon name="arrow-right" size={14} />
						</Button>
					</a>
				</div>
			</Card>
		</div>
	</div>
</div>
