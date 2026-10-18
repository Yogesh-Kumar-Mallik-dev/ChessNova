<script lang="ts">
	import ChessBoard from '$lib/components/chess/ChessBoard.svelte';
	import MoveList from '$lib/components/game/MoveList.svelte';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import { api } from '$lib/api/client';
	import { INITIAL_FEN, executeMoveClient, type Color, type PieceType } from '$lib/chess/engine';
	import { soundEffects } from '$lib/audio/sounds';
	import { goto } from '$app/navigation';
	import { Chess } from 'chess.js';
	import { authStore } from '$lib/stores/auth';
	import UserBadge from '$lib/components/ui/UserBadge.svelte';
	import Modal from '$lib/components/ui/Modal.svelte';
	import { saveLocalGame, stageGameForReview } from '$lib/stores/localGames';
	import { generateBotMove } from '$lib/engine/localEngine';

	// Game Modes
	type GameMode = 'pass_and_play' | 'vs_computer';
	let gameMode: GameMode = 'vs_computer';
	let showMobileMoves = false;

	interface BotPersona {
		level: number;
		name: string;
		elo: number;
		desc: string;
		quote: string;
		icon: 'user' | 'zap' | 'shield' | 'flame' | 'crown';
		accentColor: string;
		badgeBg: string;
	}

	const botPersonas: BotPersona[] = [
		{
			level: 1,
			name: 'Martin',
			elo: 400,
			desc: 'Novice Club Learner',
			quote: "I'm still learning piece movements. Let's have a fun match!",
			icon: 'user',
			accentColor: 'text-foreground',
			badgeBg: 'bg-secondary border-border'
		},
		{
			level: 2,
			name: 'Elena',
			elo: 800,
			desc: 'Casual Club Enthusiast',
			quote: 'I love open games and aggressive quick development!',
			icon: 'zap',
			accentColor: 'text-foreground',
			badgeBg: 'bg-secondary border-border'
		},
		{
			level: 3,
			name: 'Viktor',
			elo: 1200,
			desc: 'Solid Tactical Defender',
			quote: 'Patient pawn structure and solid counter-play win games.',
			icon: 'shield',
			accentColor: 'text-foreground',
			badgeBg: 'bg-secondary border-border'
		},
		{
			level: 4,
			name: 'Aria',
			elo: 1600,
			desc: 'Positional Tactician',
			quote: 'Watch your unprotected pieces—I capitalize on any slip.',
			icon: 'flame',
			accentColor: 'text-foreground',
			badgeBg: 'bg-secondary border-border'
		},
		{
			level: 5,
			name: 'Nova Master',
			elo: 2000,
			desc: 'Grandmaster Calculation',
			quote: 'Deep strategic calculation with ruthless endgame execution.',
			icon: 'crown',
			accentColor: 'text-foreground',
			badgeBg: 'bg-secondary border-border'
		}
	];

	let botLevel = 2; // Default Elena (800)
	let playerColor: Color = 'white';
	let isBotThinking = false;

	$: activeBot = botPersonas.find((b) => b.level === botLevel) || botPersonas[1];

	let fen = INITIAL_FEN;
	let turn: Color = 'white';
	let orientation: Color = 'white';
	let moves: Array<{ from: string; to: string; san: string; fen: string }> = [];
	let lastMove: { from: string; to: string; san?: string; isCapture?: boolean; isPromotion?: boolean } | null = null;
	$: isGameOver = new Chess(fen).isGameOver();

	$: orientation = gameMode === 'vs_computer' ? playerColor : orientation;

	function getGamePgn(resultStr = '*'): string {
		const c = new Chess();
		for (const m of moves) {
			try {
				c.move(m.san || { from: m.from, to: m.to });
			} catch (_) {}
		}
		const whiteName = gameMode === 'vs_computer' ? (playerColor === 'white' ? 'Player' : `${activeBot.name} (${activeBot.elo})`) : 'White';
		const blackName = gameMode === 'vs_computer' ? (playerColor === 'black' ? 'Player' : `${activeBot.name} (${activeBot.elo})`) : 'Black';
		c.header(
			'Event', gameMode === 'vs_computer' ? `VS ${activeBot.name}` : 'Local Match',
			'Site', 'ChessNova',
			'Date', new Date().toISOString().slice(0, 10).replace(/-/g, '.'),
			'White', whiteName,
			'Black', blackName,
			'Result', resultStr
		);
		return c.pgn();
	}

	async function triggerBotMoveIfNeeded(currentFen: string, currentTurn: Color) {
		if (gameMode !== 'vs_computer' || currentTurn === playerColor || isBotThinking) return;

		const chk = new Chess(currentFen);
		if (chk.isGameOver()) return;

		isBotThinking = true;

		try {
			const res = await generateBotMove(currentFen, botLevel);

			if (res && res.valid) {
				const cap = res.isCapture || res.san?.includes('x') || false;
				const promo = res.isPromotion || res.san?.includes('=') || false;
				lastMove = { from: res.from, to: res.to, san: res.san, isCapture: cap, isPromotion: promo };
				moves = [...moves, { from: res.from, to: res.to, san: res.san, fen: res.newFen }];
				fen = res.newFen;
				turn = res.turn as Color;

				checkGameCompletion(res.newFen);
			}
		} catch (err) {
			console.error('Failed to trigger bot move:', err);
		} finally {
			isBotThinking = false;
		}
	}

	function checkGameCompletion(nextFen: string) {
		const chk = new Chess(nextFen);
		if (chk.isGameOver()) {
			let resultStr = '1/2-1/2';
			let outcomeStr = 'draw';
			let winnerStr = 'draw';

			if (chk.isCheckmate()) {
				outcomeStr = 'checkmate';
				if (chk.turn() === 'b') {
					resultStr = '1-0';
					winnerStr = 'white';
				} else {
					resultStr = '0-1';
					winnerStr = 'black';
				}
			} else if (chk.isStalemate()) {
				outcomeStr = 'stalemate';
			} else if (chk.isInsufficientMaterial()) {
				outcomeStr = 'insufficient_material';
			} else if (chk.isThreefoldRepetition()) {
				outcomeStr = 'threefold_repetition';
			}

			const fullPgn = getGamePgn(resultStr);
			const whiteName = gameMode === 'vs_computer' ? (playerColor === 'white' ? 'Player' : `${activeBot.name} (${activeBot.elo})`) : 'White';
			const blackName = gameMode === 'vs_computer' ? (playerColor === 'black' ? 'Player' : `${activeBot.name} (${activeBot.elo})`) : 'Black';

			saveLocalGame({
				date: new Date().toISOString(),
				event: gameMode === 'vs_computer' ? `VS ${activeBot.name}` : 'Local Match',
				white: whiteName,
				black: blackName,
				result: resultStr,
				outcome: outcomeStr,
				winner: winnerStr,
				timeControl: gameMode === 'vs_computer' ? 'Bot Match' : 'Pass & Play',
				movesCount: moves.length,
				pgn: fullPgn,
				startFen: INITIAL_FEN
			});
		}
	}

	async function handleMove(event: CustomEvent<{ from: string; to: string; promotion?: PieceType; isCapture?: boolean; isPromotion?: boolean }>) {
		if (gameMode === 'vs_computer' && (turn !== playerColor || isBotThinking)) {
			return;
		}

		const { from, to, promotion, isCapture, isPromotion } = event.detail;

		// 1. Snapshot prior state for instant rollback if server rejects
		const prevFen = fen;
		const prevTurn = turn;
		const prevMoves = [...moves];
		const prevLastMove = lastMove;

		// 2. Immediate optimistic client-side move execution (0ms visual latency)
		const localRes = executeMoveClient(prevFen, from, to, promotion);
		if (!localRes || !localRes.valid) {
			return;
		}

		const cap = isCapture || localRes.isCapture;
		const promo = isPromotion || localRes.isPromotion;
		const moveSan = localRes.san;
		const nextFen = localRes.newFen;
		const nextTurn = localRes.turn;

		// 3. Immediately update UI state
		lastMove = { from, to, san: moveSan, isCapture: cap, isPromotion: promo };
		moves = [...moves, { from, to, san: moveSan, fen: nextFen }];
		fen = nextFen;
		turn = nextTurn;

		checkGameCompletion(nextFen);

		if (gameMode === 'vs_computer' && turn !== playerColor) {
			triggerBotMoveIfNeeded(nextFen, turn);
		}

		// 4. Asynchronous authoritative background verification
		try {
			const serverRes = await api.chess.move(prevFen, from, to, promotion);
			if (!serverRes || !serverRes.valid) {
				console.warn('Server rejected move, rolling back optimistic update');
				fen = prevFen;
				turn = prevTurn;
				moves = prevMoves;
				lastMove = prevLastMove;
				soundEffects.playIllegal();
			} else if (serverRes.newFen && serverRes.newFen !== fen && moves.length === prevMoves.length + 1) {
				// Reconcile authoritative FEN
				fen = serverRes.newFen;
			}
		} catch (e) {
			// In offline / local play or network jitter, local state remains fully playable
			console.warn('Server move sync warning:', e);
		}
	}

	function handleAnalyze() {
		const chk = new Chess(fen);
		let resultStr = '*';
		let outcomeStr = 'In Progress';
		let winnerStr: string | undefined = undefined;

		if (chk.isGameOver()) {
			resultStr = '1/2-1/2';
			outcomeStr = 'draw';
			winnerStr = 'draw';
			if (chk.isCheckmate()) {
				outcomeStr = 'checkmate';
				if (chk.turn() === 'b') {
					resultStr = '1-0';
					winnerStr = 'white';
				} else {
					resultStr = '0-1';
					winnerStr = 'black';
				}
			} else if (chk.isStalemate()) {
				outcomeStr = 'stalemate';
			} else if (chk.isInsufficientMaterial()) {
				outcomeStr = 'insufficient_material';
			} else if (chk.isThreefoldRepetition()) {
				outcomeStr = 'threefold_repetition';
			}
		}

		const pgn = getGamePgn(resultStr);
		const whiteName = gameMode === 'vs_computer' ? (playerColor === 'white' ? 'Player' : `${activeBot.name} (${activeBot.elo})`) : 'White';
		const blackName = gameMode === 'vs_computer' ? (playerColor === 'black' ? 'Player' : `${activeBot.name} (${activeBot.elo})`) : 'Black';

		if (moves.length >= 2) {
			saveLocalGame({
				date: new Date().toISOString(),
				event: gameMode === 'vs_computer' ? `VS ${activeBot.name}` : 'Local Match',
				white: whiteName,
				black: blackName,
				result: resultStr,
				outcome: outcomeStr,
				winner: winnerStr,
				timeControl: gameMode === 'vs_computer' ? 'Bot Match' : 'Pass & Play',
				movesCount: moves.length,
				pgn,
				startFen: INITIAL_FEN
			});
			stageGameForReview(pgn, whiteName, blackName, true);
		}
		goto('/analysis?review=1');
	}

	function resetLocalGame() {
		if (moves.length >= 2) {
			const pgn = getGamePgn('*');
			const whiteName = gameMode === 'vs_computer' ? (playerColor === 'white' ? 'Player' : `${activeBot.name} (${activeBot.elo})`) : 'White';
			const blackName = gameMode === 'vs_computer' ? (playerColor === 'black' ? 'Player' : `${activeBot.name} (${activeBot.elo})`) : 'Black';

			saveLocalGame({
				date: new Date().toISOString(),
				event: gameMode === 'vs_computer' ? `VS ${activeBot.name}` : 'Local Match',
				white: whiteName,
				black: blackName,
				result: '*',
				outcome: 'reset',
				timeControl: gameMode === 'vs_computer' ? 'Bot Match' : 'Pass & Play',
				movesCount: moves.length,
				pgn,
				startFen: INITIAL_FEN
			});
		}

		fen = INITIAL_FEN;
		turn = 'white';
		moves = [];
		lastMove = null;
		isBotThinking = false;
		soundEffects.playGameStart();

		if (gameMode === 'vs_computer' && playerColor === 'black') {
			setTimeout(() => triggerBotMoveIfNeeded(INITIAL_FEN, 'white'), 400);
		}
	}

	function setGameMode(mode: GameMode) {
		gameMode = mode;
		resetLocalGame();
	}

	function setPlayerColor(color: Color) {
		playerColor = color;
		resetLocalGame();
	}

	function setBotLevel(lvl: number) {
		botLevel = lvl;
		resetLocalGame();
	}
</script>

<div class="flex-1 max-w-7xl mx-auto w-full px-2 py-1.5 sm:px-4 sm:py-2 lg:px-6 flex flex-col justify-center min-h-0 lg:h-[calc(100dvh-4rem)] lg:max-h-[calc(100dvh-4rem)] overflow-x-hidden overflow-y-auto lg:overflow-hidden">
	<!-- Slim Header & Mode Toolbar -->
	<div class="mb-1.5 flex flex-wrap items-center justify-between gap-2 bg-card border border-border rounded-xl px-2.5 py-1.5 shrink-0 shadow-xs">
		<div class="flex items-center gap-2">
			<!-- Mode Switcher -->
			<div class="inline-flex p-0.5 rounded-lg bg-secondary border border-border text-xs">
				<button
					class="px-2.5 py-1 rounded font-semibold transition cursor-pointer {gameMode === 'vs_computer' ? 'bg-foreground text-background font-bold shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
					on:click={() => setGameMode('vs_computer')}
				>
					Play Bot
				</button>
				<button
					class="px-2.5 py-1 rounded font-semibold transition cursor-pointer {gameMode === 'pass_and_play' ? 'bg-foreground text-background font-bold shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
					on:click={() => setGameMode('pass_and_play')}
				>
					Pass & Play
				</button>
			</div>

			<!-- Bot Selector Chips (Desktop) & Select (Mobile) -->
			{#if gameMode === 'vs_computer'}
				<div class="hidden sm:flex items-center gap-1 bg-secondary/60 p-0.5 rounded-lg border border-border">
					{#each botPersonas as b}
						{@const isSelected = b.level === botLevel}
						<button
							class="px-2 py-0.5 rounded text-[11px] font-semibold transition flex items-center gap-1 {isSelected
								? 'bg-foreground text-background font-bold shadow-xs'
								: 'text-muted-foreground hover:text-foreground'}"
							on:click={() => setBotLevel(b.level)}
							title={`${b.name} (${b.elo} Elo) - ${b.desc}`}
						>
							<Icon name={b.icon} size={11} className={isSelected ? 'text-background' : 'text-muted-foreground'} />
							<span>{b.name}</span>
							<span class="text-[9px] font-mono opacity-80 tabular-nums">({b.elo})</span>
						</button>
					{/each}
				</div>

				<!-- Compact Mobile Bot Dropdown -->
				<div class="sm:hidden flex items-center">
					<select
						class="bg-secondary text-foreground text-xs font-semibold py-1 px-2 rounded-lg border border-border focus:outline-none focus:ring-1 focus:ring-ring"
						value={botLevel}
						on:change={(e) => setBotLevel(Number(e.currentTarget.value))}
					>
						{#each botPersonas as b}
							<option value={b.level}>{b.name} ({b.elo} Elo)</option>
						{/each}
					</select>
				</div>
			{/if}
		</div>

		<!-- Right: Color choice & quick controls -->
		<div class="flex items-center gap-1.5">
			{#if gameMode === 'vs_computer'}
				<div class="inline-flex p-0.5 rounded-lg bg-secondary border border-border text-xs">
					<button
						class="px-2 py-0.5 rounded text-[11px] font-semibold transition {playerColor === 'white' ? 'bg-foreground text-background font-bold' : 'text-muted-foreground hover:text-foreground'}"
						on:click={() => setPlayerColor('white')}
					>
						White
					</button>
					<button
						class="px-2 py-0.5 rounded text-[11px] font-semibold transition {playerColor === 'black' ? 'bg-foreground text-background font-bold' : 'text-muted-foreground hover:text-foreground'}"
						on:click={() => setPlayerColor('black')}
					>
						Black
					</button>
				</div>
			{/if}

			<button
				class="p-1.5 bg-secondary hover:bg-muted text-muted-foreground hover:text-foreground rounded-lg transition border border-border flex items-center justify-center"
				on:click={() => (orientation = orientation === 'white' ? 'black' : 'white')}
				title="Flip Board"
			>
				<Icon name="rotate-cw" size={13} />
			</button>

			<button
				class="p-1.5 bg-secondary hover:bg-muted text-muted-foreground hover:text-foreground rounded-lg transition border border-border flex items-center justify-center"
				on:click={resetLocalGame}
				title="Reset Match"
			>
				<Icon name="rotate-ccw" size={13} />
			</button>
		</div>
	</div>

	<!-- Main Board & Side Panel Grid -->
	<div class="grid grid-cols-1 lg:grid-cols-12 gap-3 lg:gap-6 items-center justify-center my-auto min-h-0">
		<!-- Chess Board Column (Hero centerpiece) -->
		<div class="lg:col-span-7 xl:col-span-8 flex flex-col items-center justify-center min-h-0 w-full">
			<div class="w-full max-w-[min(100%,calc(100dvh-200px),580px)] flex flex-col items-center">
				
				<!-- Opponent Indicator Bar -->
				<div class="w-full flex items-center justify-between py-1 px-1 text-xs select-none mb-1">
					<div class="flex items-center gap-2">
						{#if gameMode === 'vs_computer'}
							<UserBadge
								username={activeBot.name}
								rating={activeBot.elo}
								ratingCategory="rapid"
								title="BOT"
								size="sm"
								interactive={false}
							/>
						{:else}
							<div class="flex items-center gap-2 text-foreground font-bold leading-tight">
								<div class="w-6 h-6 rounded-lg bg-secondary border border-border flex items-center justify-center text-muted-foreground">
									<Icon name="user" size={13} />
								</div>
								<span>Opponent</span>
							</div>
						{/if}
					</div>

					{#if isBotThinking}
						<div class="flex items-center gap-1.5 text-[11px] font-mono text-foreground animate-pulse font-bold">
							<span class="w-2 h-2 rounded-full bg-foreground"></span>
							<span>Thinking...</span>
						</div>
					{:else}
						<span class="text-[11px] text-muted-foreground font-mono">
							{turn === (gameMode === 'vs_computer' && playerColor === 'white' ? 'black' : 'white') ? 'Their turn' : 'Waiting'}
						</span>
					{/if}
				</div>

				<!-- Board -->
				<ChessBoard
					{fen}
					{orientation}
					{turn}
					{lastMove}
					interactive={!(gameMode === 'vs_computer' && (turn !== playerColor || isBotThinking))}
					boardSizeClass="w-full"
					on:move={handleMove}
					on:newGame={resetLocalGame}
					on:rematch={resetLocalGame}
					on:analyze={handleAnalyze}
				/>

				<!-- Self Indicator Bar -->
				<div class="w-full flex items-center justify-between py-1 px-1 text-xs select-none mt-1">
					<div class="flex items-center gap-2">
						{#if $authStore.user}
							<UserBadge
								username={$authStore.user.username}
								avatar={$authStore.user.avatar}
								rating={$authStore.ratings?.blitz || 400}
								ratingCategory="blitz"
								size="sm"
								target="_blank"
							/>
						{:else}
							<div class="flex items-center gap-2 text-foreground font-bold leading-tight">
								<div class="w-6 h-6 rounded-lg bg-secondary border border-border text-foreground flex items-center justify-center">
									<Icon name="crown" size={13} />
								</div>
								<span>You</span>
							</div>
						{/if}
					</div>

					<!-- Turn Indicator -->
					<div class="flex items-center gap-1.5">
						<span class="text-[11px] font-mono {turn === playerColor ? 'text-foreground font-bold' : 'text-muted-foreground'}">
							{turn === playerColor ? 'Your turn' : 'Waiting...'}
						</span>
					</div>
				</div>

				<!-- Mobile Action Bar (< lg) -->
				<div class="lg:hidden w-full flex items-center justify-between gap-1.5 mt-2 px-2 py-1.5 bg-card border border-border rounded-xl shadow-xs">
					<button
						class="flex-1 py-1 px-2 bg-secondary hover:bg-muted text-foreground font-semibold text-xs rounded-lg transition border border-border flex items-center justify-center gap-1.5 cursor-pointer"
						on:click={() => (showMobileMoves = true)}
					>
						<Icon name="play" size={12} className="text-muted-foreground" />
						<span>Moves ({moves.length})</span>
					</button>

					<button
						class="py-1 px-2.5 bg-secondary hover:bg-muted text-foreground font-semibold text-xs rounded-lg transition border border-border flex items-center justify-center gap-1 cursor-pointer"
						on:click={() => (orientation = orientation === 'white' ? 'black' : 'white')}
						title="Flip Board"
					>
						<Icon name="rotate-cw" size={13} />
					</button>

					<button
						class="py-1 px-2.5 bg-secondary hover:bg-muted text-foreground font-semibold text-xs rounded-lg transition border border-border flex items-center justify-center gap-1 cursor-pointer"
						on:click={resetLocalGame}
						title="New Match"
					>
						<Icon name="rotate-ccw" size={13} />
					</button>

					{#if isGameOver}
						<button
							class="flex-1 py-1 px-2 bg-foreground text-background font-bold text-xs uppercase tracking-wider rounded-lg transition flex items-center justify-center gap-1.5 shadow-xs cursor-pointer"
							on:click={handleAnalyze}
						>
							<Icon name="search" size={12} />
							<span>Review</span>
						</button>
					{/if}
				</div>
			</div>
		</div>

		<!-- Sidebar: Bot Personality Card & Move List (Desktop) -->
		<div class="hidden lg:flex lg:col-span-5 xl:col-span-4 h-[min(calc(100dvh-110px),640px)] bg-card border border-border rounded-xl flex-col overflow-hidden shadow-lg">
			{#if gameMode === 'vs_computer'}
				<!-- Bot Card Header -->
				<div class="p-4 border-b border-border bg-secondary/30 flex items-center justify-between">
					<div class="flex items-center gap-3">
						<div class="w-10 h-10 rounded-lg flex items-center justify-center bg-secondary border border-border text-foreground shadow-xs">
							<Icon name={activeBot.icon} size={20} />
						</div>
						<div>
							<div class="flex items-center gap-2">
								<h3 class="text-sm font-black text-foreground">{activeBot.name}</h3>
								<span class="text-[10px] font-mono font-bold px-1.5 py-0.5 rounded bg-secondary text-foreground border border-border tabular-nums">
									{activeBot.elo} Elo
								</span>
							</div>
							<p class="text-[11px] text-muted-foreground">{activeBot.desc}</p>
						</div>
					</div>
				</div>

				<div class="p-3 bg-secondary/15 border-b border-border text-xs italic text-muted-foreground flex items-center gap-2">
					<Icon name="sparkles" size={13} className="text-muted-foreground shrink-0" />
					<span class="truncate">"{activeBot.quote}"</span>
				</div>
			{/if}

			<!-- Move Ledger Header -->
			<div class="p-3 border-b border-border bg-secondary/40 text-xs font-bold text-foreground flex justify-between items-center">
				<div class="flex items-center gap-2">
					<span class="tracking-wide uppercase text-[10px] text-muted-foreground">Move History ({moves.length})</span>
				</div>
				<div class="flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-secondary border border-border">
					<span class="w-2 h-2 rounded-full {turn === 'white' ? 'bg-foreground' : 'bg-muted border border-border'}"></span>
					<span class="text-[10px] font-mono font-bold text-foreground uppercase">{turn} to play</span>
				</div>
			</div>

			<div class="flex-1 overflow-hidden p-2">
				<MoveList
					{moves}
					whiteName={playerColor === 'white' ? 'You' : activeBot.name}
					blackName={playerColor === 'black' ? 'You' : activeBot.name}
				/>
			</div>

			<div class="p-3 border-t border-border bg-secondary/20 flex items-center gap-2">
				<Button
					variant="outline"
					size="sm"
					className="flex-1"
					on:click={resetLocalGame}
				>
					<Icon name="rotate-ccw" size={13} />
					<span>New Match</span>
				</Button>

				<Button
					variant="primary"
					size="sm"
					className="flex-1"
					on:click={handleAnalyze}
				>
					<Icon name="search" size={13} />
					<span>Analyze Game</span>
				</Button>
			</div>
		</div>
	</div>
</div>

<!-- Mobile Move History Modal -->
<Modal
	isOpen={showMobileMoves}
	title="Move History ({moves.length} moves)"
	on:close={() => (showMobileMoves = false)}
>
	<div class="flex flex-col gap-3">
		{#if gameMode === 'vs_computer'}
			<div class="flex items-center gap-2.5 p-2.5 rounded-lg bg-secondary/50 border border-border">
				<div class="w-8 h-8 rounded-lg bg-secondary border border-border flex items-center justify-center text-foreground shrink-0">
					<Icon name={activeBot.icon} size={16} />
				</div>
				<div class="min-w-0 flex-1">
					<div class="flex items-center gap-1.5">
						<span class="text-xs font-bold text-foreground truncate">{activeBot.name}</span>
						<span class="text-[10px] font-mono px-1 py-0.2 rounded bg-card border border-border text-foreground font-semibold">{activeBot.elo}</span>
					</div>
					<p class="text-[11px] text-muted-foreground truncate">"{activeBot.quote}"</p>
				</div>
			</div>
		{/if}

		<div class="h-64 overflow-hidden rounded-lg border border-border bg-card">
			<MoveList
				{moves}
				whiteName={playerColor === 'white' ? 'You' : activeBot.name}
				blackName={playerColor === 'black' ? 'You' : activeBot.name}
			/>
		</div>
	</div>
</Modal>
