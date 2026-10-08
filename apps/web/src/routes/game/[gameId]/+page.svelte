<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { Chess } from 'chess.js';
	import { api } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth';
	import { gameStore, resetGameStore } from '$lib/stores/game';
	import { GameSocket } from '$lib/websocket/gameSocket';
	import ChessBoard from '$lib/components/chess/ChessBoard.svelte';
	import GameClock from '$lib/components/game/GameClock.svelte';
	import MoveList from '$lib/components/game/MoveList.svelte';
	import CapturedPieces from '$lib/components/game/CapturedPieces.svelte';
	import GameResult from '$lib/components/game/GameResult.svelte';
	import Icon from '$lib/components/icons/Icon.svelte';
	import { INITIAL_FEN, type Color, type PieceType } from '$lib/chess/engine';
	import { saveLocalGame, stageGameForReview } from '$lib/stores/localGames';

	let gameId = $page.params.gameId || '';

	let socket: GameSocket | null = null;
	let showResultModal = true;
	let flipBoard = false;

	let currentOrientation: Color = 'white';
	let opponentColor: Color = 'black';
	let userColor: Color = 'white';

	$: currentOrientation = flipBoard
		? ($gameStore.orientation === 'white' ? 'black' : 'white')
		: ($authStore.user && $gameStore.blackPlayer?.userId === $authStore.user.id ? 'black' : 'white');

	$: opponentColor = currentOrientation === 'white' ? 'black' : 'white';
	$: userColor = currentOrientation;

	$: topPlayer = currentOrientation === 'white' ? $gameStore.blackPlayer : $gameStore.whitePlayer;
	$: bottomPlayer = currentOrientation === 'white' ? $gameStore.whitePlayer : $gameStore.blackPlayer;

	$: topTimeMs = currentOrientation === 'white' ? $gameStore.blackTime : $gameStore.whiteTime;
	$: bottomTimeMs = currentOrientation === 'white' ? $gameStore.whiteTime : $gameStore.blackTime;

	$: isTopTurn = $gameStore.status === 'playing' && $gameStore.turn === opponentColor;
	$: isBottomTurn = $gameStore.status === 'playing' && $gameStore.turn === userColor;

	let copiedInvite = false;

	function copyInviteLink() {
		if (typeof window !== 'undefined') {
			navigator.clipboard.writeText(window.location.href);
			copiedInvite = true;
			setTimeout(() => {
				copiedInvite = false;
			}, 2500);
		}
	}

	onMount(async () => {
		resetGameStore();

		try {
			const info = await api.games.get(gameId);

			// If current user is not white and black is guest, claim the challenge seat!
			if (
				$authStore.user &&
				info.white?.userId !== $authStore.user.id &&
				(info.black?.userId === 'guest-opponent' || !info.black?.userId)
			) {
				try {
					const joined = await api.games.join(gameId);
					if (joined && joined.black) {
						info.black = joined.black;
					}
				} catch (e) {
					console.warn('Could not join match:', e);
				}
			}

			gameStore.update((s) => ({
				...s,
				gameId: info.id,
				fen: info.fen || info.initialFen,
				turn: info.turn || 'white',
				whitePlayer: info.white,
				blackPlayer: info.black,
				timeControl: info.timeControl,
				whiteTime: info.whiteTime || (info.timeControl?.initial * 1000) || 300000,
				blackTime: info.blackTime || (info.timeControl?.initial * 1000) || 300000,
				status: info.status,
				result: info.result,
				outcome: info.outcome,
				moves: (info.moves || []).map((m: any) => ({
					from: m.from,
					to: m.to,
					san: m.san,
					fen: m.fen
				}))
			}));
		} catch (e) {
			console.error('Failed to load initial game:', e);
		}

		const token = typeof window !== 'undefined' ? localStorage.getItem('access_token') : null;
		socket = new GameSocket(gameId, token);
		socket.connect();
	});

	onDestroy(() => {
		if (socket) {
			socket.disconnect();
			socket = null;
		}
	});

	function handleMove(event: CustomEvent<{ from: string; to: string; promotion?: PieceType }>) {
		const { from, to, promotion } = event.detail;
		if (socket) {
			socket.sendMove(from, to, promotion);
		}
	}

	function handleResign() {
		if (confirm('Concede match and resign?')) {
			socket?.resign();
		}
	}

	function handleOfferDraw() {
		socket?.offerDraw();
	}

	function handleAcceptDraw() {
		socket?.acceptDraw();
	}

	function handleDeclineDraw() {
		socket?.declineDraw();
	}

	function handleNewGame() {
		goto('/play/online');
	}

	function handleAnalyze() {
		if (typeof window !== 'undefined' && $gameStore.moves.length > 0) {
			const c = new Chess();
			for (const m of $gameStore.moves) {
				try {
					c.move(m.san || { from: m.from, to: m.to });
				} catch (_) {}
			}
			const whiteName = $gameStore.whitePlayer?.username || 'White';
			const blackName = $gameStore.blackPlayer?.username || 'Black';
			const resStr = $gameStore.result || '*';

			c.header(
				'Event', 'Live Match',
				'Site', 'ChessNova',
				'Date', new Date().toISOString().slice(0, 10).replace(/-/g, '.'),
				'White', whiteName,
				'Black', blackName,
				'Result', resStr
			);
			const pgn = c.pgn();
			stageGameForReview(pgn, whiteName, blackName);
			sessionStorage.setItem('review_moves', JSON.stringify($gameStore.moves));

			// Save to local games archive so guests and users can review anytime
			saveLocalGame({
				id: gameId,
				date: new Date().toISOString(),
				event: 'Live Match',
				white: whiteName,
				black: blackName,
				result: resStr,
				outcome: $gameStore.outcome || 'finished',
				winner: $gameStore.winner,
				movesCount: $gameStore.moves.length,
				pgn,
				startFen: INITIAL_FEN
			});

			goto(`/analysis?gameId=${gameId}&review=1`);
			return;
		}
		goto(`/analysis?gameId=${gameId}&review=1`);
	}

</script>

<div class="flex-1 flex flex-col justify-center max-w-7xl mx-auto w-full p-2 sm:p-4 lg:p-6">
	<div class="grid grid-cols-1 lg:grid-cols-12 gap-4 lg:gap-8 items-start justify-center">
		
		<!-- Board Column -->
		<div class="lg:col-span-8 flex flex-col items-center justify-center">
			
			<!-- Top Player Bar (Opponent) -->
			<div class="w-full max-w-[min(88vw,590px)] flex items-center justify-between py-2 px-1 text-sm font-semibold select-none">
				<div class="flex items-center gap-3">
					<div class="w-9 h-9 rounded-xl bg-slate-800/90 border border-slate-700/80 flex items-center justify-center text-xs font-mono font-bold text-slate-300 shadow-md">
						<Icon name="user" size={16} />
					</div>
					<div>
						<div class="flex items-center gap-2 text-white font-bold leading-tight">
							<span>{topPlayer?.username || 'Opponent'}</span>
							<span class="text-[11px] font-mono font-semibold px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700/60">
								{topPlayer?.rating || 1200}
							</span>
						</div>
						<div class="mt-1">
							<CapturedPieces fen={$gameStore.fen} forColor={opponentColor} />
						</div>
					</div>
				</div>

				<GameClock timeMs={topTimeMs} isActive={isTopTurn} />
			</div>

			<!-- Chess Board -->
			<ChessBoard
				fen={$gameStore.fen}
				orientation={currentOrientation}
				turn={$gameStore.turn}
				selectedSquare={$gameStore.selectedSquare}
				legalMoves={$gameStore.legalMoves}
				lastMove={$gameStore.lastMove}
				isCheck={$gameStore.isCheck}
				interactive={$gameStore.status === 'playing'}
				gameOver={$gameStore.status === 'finished'}
				gameResult={$gameStore.result || ''}
				gameOutcome={$gameStore.outcome || ''}
				gameWinner={$gameStore.winner || ''}
				disableGameOverModal={true}
				on:move={handleMove}
			/>

			<!-- Bottom Player Bar (Self) -->
			<div class="w-full max-w-[min(88vw,590px)] flex items-center justify-between py-2 px-1 text-sm font-semibold select-none">
				<div class="flex items-center gap-3">
					<div class="w-9 h-9 rounded-xl bg-sky-500/15 border border-sky-500/40 flex items-center justify-center text-xs font-mono font-bold text-sky-400 shadow-md">
						<Icon name="crown" size={16} />
					</div>
					<div>
						<div class="flex items-center gap-2 text-white font-bold leading-tight">
							<span>{bottomPlayer?.username || 'You'}</span>
							<span class="text-[11px] font-mono font-semibold px-1.5 py-0.5 rounded bg-sky-500/10 text-sky-400 border border-sky-500/30">
								{bottomPlayer?.rating || 1200}
							</span>
						</div>
						<div class="mt-1">
							<CapturedPieces fen={$gameStore.fen} forColor={userColor} />
						</div>
					</div>
				</div>

				<GameClock timeMs={bottomTimeMs} isActive={isBottomTurn} />
			</div>

		</div>

		<!-- Right Panel: HUD, Move Ledger & Actions -->
		<div class="lg:col-span-4 w-full flex flex-col h-[400px] lg:h-[610px] bg-gradient-to-b from-[#121824] to-[#0c1017] border border-slate-800 rounded-2xl overflow-hidden shadow-2xl">
			
			<!-- HUD Header -->
			<div class="p-3.5 border-b border-slate-800 bg-[#090d15]/80 flex items-center justify-between">
				<div class="flex items-center gap-2.5 text-xs font-bold">
					<span class="w-2 h-2 rounded-full {$gameStore.connectionStatus === 'connected' ? 'bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]' : 'bg-rose-500 animate-ping'}"></span>
					<span class="font-mono text-slate-300">
						{$gameStore.timeControl ? `${$gameStore.timeControl.initial / 60}+${$gameStore.timeControl.increment}` : '5+0'}
					</span>
					<span class="text-slate-600">•</span>
					<span class="uppercase tracking-wider text-[10px] font-mono font-extrabold {$gameStore.status === 'playing' ? 'text-sky-400' : 'text-slate-400'}">
						{$gameStore.status}
					</span>
				</div>

				<button
					class="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition flex items-center gap-1 text-xs font-medium"
					on:click={() => (flipBoard = !flipBoard)}
					title="Flip Board View"
				>
					<Icon name="rotate-cw" size={14} />
					<span class="text-[11px] font-mono">Flip</span>
				</button>
			</div>

			<!-- Challenge Invite Link Banner -->
			{#if $gameStore.blackPlayer?.userId === 'guest-opponent' && $gameStore.status === 'playing'}
				<div class="p-3 bg-sky-950/40 border-b border-sky-600/30 flex items-center justify-between text-xs animate-in fade-in duration-150">
					<div class="flex items-center gap-2 text-sky-300 font-semibold">
						<Icon name="sparkles" size={15} />
						<span>Share Link to Play</span>
					</div>
					<button
						class="px-2.5 py-1 bg-gradient-to-r from-sky-500 to-indigo-600 hover:from-sky-400 hover:to-indigo-500 text-white font-bold rounded-lg text-xs transition shadow-sm"
						on:click={copyInviteLink}
					>
						{copiedInvite ? 'Copied Link!' : 'Copy Link'}
					</button>
				</div>
			{/if}

			<!-- Draw Offer Banner -->
			{#if $gameStore.drawOfferBy && $gameStore.drawOfferBy !== userColor}
				<div class="p-3 bg-amber-950/50 border-b border-amber-600/40 flex items-center justify-between text-xs animate-in fade-in duration-150">
					<div class="flex items-center gap-2 text-amber-300 font-semibold">
						<Icon name="handshake" size={16} />
						<span>Opponent offers draw</span>
					</div>
					<div class="flex gap-1.5">
						<button
							class="px-2.5 py-1 bg-emerald-600 hover:bg-emerald-500 text-white font-bold rounded-lg text-xs transition shadow-sm"
							on:click={handleAcceptDraw}
						>
							Accept
						</button>
						<button
							class="px-2.5 py-1 bg-slate-800 hover:bg-slate-700 text-slate-300 font-bold rounded-lg text-xs transition"
							on:click={handleDeclineDraw}
						>
							Decline
						</button>
					</div>
				</div>
			{/if}

			<!-- Move History Ledger -->
			<div class="flex-1 overflow-hidden p-1">
				<MoveList moves={$gameStore.moves} />
			</div>

			<!-- Action Controls -->
			<div class="p-3.5 border-t border-slate-800 bg-[#090d15]/80 flex gap-2">
				{#if $gameStore.status === 'playing'}
					<button
						class="flex-1 py-2.5 bg-slate-800/80 hover:bg-slate-750 text-slate-300 font-bold text-xs rounded-xl transition border border-slate-700/70 flex items-center justify-center gap-1.5"
						on:click={handleOfferDraw}
					>
						<Icon name="handshake" size={14} className="text-slate-400" />
						<span>Offer Draw</span>
					</button>

					<button
						class="flex-1 py-2.5 bg-slate-800/80 hover:bg-rose-950/60 hover:text-rose-300 hover:border-rose-700/60 text-slate-300 font-bold text-xs rounded-xl transition border border-slate-700/70 flex items-center justify-center gap-1.5"
						on:click={handleResign}
					>
						<Icon name="flag" size={14} className="text-rose-400" />
						<span>Resign</span>
					</button>
				{:else}
					<div class="flex flex-col gap-2 w-full">
						<button
							class="w-full py-2.5 bg-[#81b64c] hover:bg-[#91c65d] active:bg-[#73a443] text-white font-extrabold text-xs uppercase tracking-wider rounded-xl border-b-4 border-[#5d8336] active:border-b-0 active:translate-y-1 transition shadow-md flex items-center justify-center gap-2"
							on:click={handleAnalyze}
						>
							<Icon name="search" size={15} />
							<span>Game Review</span>
						</button>

						<div class="grid grid-cols-2 gap-2 w-full">
							<button
								class="py-2 px-3 bg-[#383633] hover:bg-[#484643] active:bg-[#2e2d2a] text-stone-200 font-bold text-xs rounded-xl border-b-4 border-[#292825] active:border-b-0 active:translate-y-1 transition flex items-center justify-center gap-1.5"
								on:click={() => (showResultModal = true)}
							>
								<span>View Result</span>
							</button>
							<button
								class="py-2 px-3 bg-[#383633] hover:bg-[#484643] active:bg-[#2e2d2a] text-stone-200 font-bold text-xs rounded-xl border-b-4 border-[#292825] active:border-b-0 active:translate-y-1 transition flex items-center justify-center gap-1.5"
								on:click={handleNewGame}
							>
								<Icon name="zap" size={13} />
								<span>New Match</span>
							</button>
						</div>
					</div>
				{/if}
			</div>

		</div>

	</div>
</div>

<!-- End Game Modal -->
{#if $gameStore.status === 'finished' && showResultModal}
	<GameResult
		result={$gameStore.result || '*'}
		outcome={$gameStore.outcome || ''}
		winner={$gameStore.winner || ''}
		playerColor={userColor}
		on:newGame={handleNewGame}
		on:analyze={handleAnalyze}
		on:close={() => (showResultModal = false)}
	/>
{/if}
