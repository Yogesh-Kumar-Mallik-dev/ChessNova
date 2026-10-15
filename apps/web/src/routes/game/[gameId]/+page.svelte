<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto, beforeNavigate } from '$app/navigation';
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
	import UserBadge from '$lib/components/ui/UserBadge.svelte';
	import Modal from '$lib/components/ui/Modal.svelte';
	import { INITIAL_FEN, type Color, type PieceType } from '$lib/chess/engine';
	import { soundEffects } from '$lib/audio/sounds';
	import { saveLocalGame, stageGameForReview } from '$lib/stores/localGames';

	let gameId = $page.params.gameId || '';

	let socket: GameSocket | null = null;
	let showResultModal = true;
	let showResignModal = false;
	let showOfferDrawModal = false;
	let showLeaveModal = false;
	let pendingNavigation: (() => void) | null = null;
	let allowNavigation = false;
	let flipBoard = false;

	// Determine current user's authenticated seat (check ID, username, and query params)
	$: myPlayerColor = (() => {
		const colorParam = $page.url.searchParams.get('color');
		if (!$authStore.user) {
			if (colorParam === 'black' || colorParam === 'white') return colorParam as Color;
			return null;
		}
		const uid = $authStore.user.id;
		const uname = $authStore.user.username;

		const isWhite = !!($gameStore.whitePlayer?.userId && $gameStore.whitePlayer.userId === uid) ||
			!!(uname && $gameStore.whitePlayer?.username && $gameStore.whitePlayer.username.toLowerCase() === uname.toLowerCase());

		const isBlack = !!($gameStore.blackPlayer?.userId && $gameStore.blackPlayer.userId === uid) ||
			!!(uname && $gameStore.blackPlayer?.username && $gameStore.blackPlayer.username.toLowerCase() === uname.toLowerCase());

		if (isWhite && isBlack) {
			// In self-play / same account testing across tabs, respect ?color query parameter!
			if (colorParam === 'black' || colorParam === 'white') return colorParam as Color;
			return 'white' as Color;
		}
		if (isWhite) return 'white' as Color;
		if (isBlack) return 'black' as Color;

		// Fallback to URL parameter if assigned via matchmaking or direct match
		if (colorParam === 'black' || colorParam === 'white') return colorParam as Color;
		return null;
	})();

	$: isPlayer = myPlayerColor !== null;
	$: isSpectator = !isPlayer;

	// Visual board orientation (defaults to player's color, or 'white' for spectators)
	$: defaultOrientation = (myPlayerColor || ($page.url.searchParams.get('color') as Color) || 'white') as Color;
	$: currentOrientation = (flipBoard
		? (defaultOrientation === 'white' ? 'black' : 'white')
		: defaultOrientation) as Color;

	$: topColor = (currentOrientation === 'white' ? 'black' : 'white') as Color;
	$: bottomColor = currentOrientation;

	$: topPlayer = currentOrientation === 'white' ? $gameStore.blackPlayer : $gameStore.whitePlayer;
	$: bottomPlayer = currentOrientation === 'white' ? $gameStore.whitePlayer : $gameStore.blackPlayer;

	$: topTimeMs = currentOrientation === 'white' ? $gameStore.blackTime : $gameStore.whiteTime;
	$: bottomTimeMs = currentOrientation === 'white' ? $gameStore.whiteTime : $gameStore.blackTime;

	$: isTopTurn = $gameStore.status === 'playing' && $gameStore.turn === topColor;
	$: isBottomTurn = $gameStore.status === 'playing' && $gameStore.turn === bottomColor;
	$: isMyTurn = isPlayer && $gameStore.status === 'playing' && $gameStore.turn === myPlayerColor;

	let copiedInvite = false;
	let opponentCountdown = 60;
	let countdownInterval: any = null;

	// Track opponent disconnection countdown
	$: if (isPlayer && !$gameStore.opponentConnected && $gameStore.status === 'playing') {
		if (!countdownInterval) {
			opponentCountdown = 60;
			countdownInterval = setInterval(() => {
				if (opponentCountdown > 0) {
					opponentCountdown--;
				} else {
					clearInterval(countdownInterval);
					countdownInterval = null;
				}
			}, 1000);
		}
	} else {
		if (countdownInterval) {
			clearInterval(countdownInterval);
			countdownInterval = null;
		}
		opponentCountdown = 60;
	}

	// Warn before leaving active game
	beforeNavigate((nav) => {
		if (allowNavigation) return;
		if (isPlayer && $gameStore.status === 'playing') {
			nav.cancel();
			if (nav.to) {
				const dest = nav.to.url.pathname + nav.to.url.search;
				pendingNavigation = () => {
					allowNavigation = true;
					goto(dest);
				};
			}
			showLeaveModal = true;
		}
	});

	function confirmLeaveMatch() {
		showLeaveModal = false;
		if (pendingNavigation) {
			const action = pendingNavigation;
			pendingNavigation = null;
			action();
		}
	}

	function handleBeforeUnload(e: BeforeUnloadEvent) {
		if (isPlayer && $gameStore.status === 'playing') {
			e.preventDefault();
			e.returnValue = '';
			return '';
		}
	}

	function copyInviteLink() {
		if (typeof window !== 'undefined') {
			navigator.clipboard.writeText(window.location.href);
			copiedInvite = true;
			setTimeout(() => {
				copiedInvite = false;
			}, 2500);
		}
	}

	// Rollback snapshot for optimistic moves
	let rollbackSnapshot: {
		fen: string;
		turn: Color;
		moves: any[];
		lastMove: any;
		isCheck: boolean;
	} | null = null;

	async function getResolvedUser(): Promise<any> {
		if ($authStore.user) return $authStore.user;
		if (!$authStore.loading) return null;
		return new Promise((resolve) => {
			const unsub = authStore.subscribe((state) => {
				if (!state.loading) {
					unsub();
					resolve(state.user);
				}
			});
		});
	}

	onMount(async () => {
		resetGameStore();

		if (typeof window !== 'undefined') {
			window.addEventListener('beforeunload', handleBeforeUnload);
		}

		try {
			const currentUser = await getResolvedUser();
			let info = await api.games.get(gameId);

			// If current user is logged in, not white, and black seat is open/guest, claim the black seat!
			if (
				currentUser &&
				info.white?.userId !== currentUser.id &&
				(info.black?.userId === 'guest-opponent' || !info.black?.userId)
			) {
				try {
					const joined = await api.games.join(gameId);
					if (joined) {
						if (joined.white) info.white = joined.white;
						if (joined.black) info.black = joined.black;
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

		// Handle server move error with rollback
		socket.onError((errMsg) => {
			if (rollbackSnapshot) {
				console.warn('Server rejected move, rolling back:', errMsg);
				gameStore.update((s) => ({
					...s,
					fen: rollbackSnapshot!.fen,
					turn: rollbackSnapshot!.turn,
					moves: rollbackSnapshot!.moves,
					lastMove: rollbackSnapshot!.lastMove,
					isCheck: rollbackSnapshot!.isCheck
				}));
				rollbackSnapshot = null;
				soundEffects.playIllegal();
			}
		});

		socket.connect();
	});

	onDestroy(() => {
		if (typeof window !== 'undefined') {
			window.removeEventListener('beforeunload', handleBeforeUnload);
		}
		if (countdownInterval) {
			clearInterval(countdownInterval);
			countdownInterval = null;
		}
		if (socket) {
			socket.disconnect();
			socket = null;
		}
	});

	function handleMove(event: CustomEvent<{ from: string; to: string; promotion?: PieceType }>) {
		const { from, to, promotion } = event.detail;

		// Only active players on their turn can submit moves
		if (!isPlayer || !isMyTurn || $gameStore.status !== 'playing') {
			return;
		}

		// Optimistic UI update: apply locally using chess.js immediately
		try {
			const c = new Chess($gameStore.fen);
			const moveRes = c.move({
				from,
				to,
				promotion: promotion || undefined
			});

			if (moveRes) {
				rollbackSnapshot = {
					fen: $gameStore.fen,
					turn: $gameStore.turn,
					moves: [...$gameStore.moves],
					lastMove: $gameStore.lastMove,
					isCheck: $gameStore.isCheck
				};

				const nextFen = c.fen();
				const nextTurn = c.turn() === 'w' ? 'white' : 'black';
				const isCap = moveRes.flags.includes('c') || moveRes.flags.includes('e');
				const isPromo = moveRes.flags.includes('p');

				gameStore.update((s) => ({
					...s,
					fen: nextFen,
					turn: nextTurn,
					lastMove: {
						from,
						to,
						san: moveRes.san,
						isCapture: isCap,
						isPromotion: isPromo
					},
					moves: [
						...s.moves,
						{
							from,
							to,
							san: moveRes.san,
							fen: nextFen
						}
					],
					isCheck: c.inCheck()
				}));

				if (socket) {
					socket.sendMove(from, to, promotion);
				}
				if (!socket || !socket.isConnected()) {
					api.games.makeMove(gameId, { from, to, promotion }).catch((e) => {
						console.warn('HTTP fallback move submission failed:', e);
					});
				}
				return;
			}
		} catch (e) {
			console.warn('Optimistic move execution fallback to server:', e);
		}

		if (socket && socket.isConnected()) {
			socket.sendMove(from, to, promotion);
		} else {
			api.games.makeMove(gameId, { from, to, promotion }).catch((e) => {
				console.warn('HTTP move submission failed:', e);
			});
		}
	}

	function handleResign() {
		if (!isPlayer || $gameStore.status !== 'playing') return;
		showResignModal = true;
	}

	async function confirmResign() {
		showResignModal = false;
		socket?.resign();
		try {
			await api.games.resign(gameId);
		} catch (_) {}
	}

	function handleOfferDraw() {
		if (!isPlayer || $gameStore.status !== 'playing') return;
		showOfferDrawModal = true;
	}

	async function confirmOfferDraw() {
		showOfferDrawModal = false;
		socket?.offerDraw();
		try {
			await api.games.offerDraw(gameId);
		} catch (_) {}
	}

	async function handleAcceptDraw() {
		if (!isPlayer || $gameStore.status !== 'playing') return;
		socket?.acceptDraw();
		try {
			await api.games.acceptDraw(gameId);
		} catch (_) {}
	}

	async function handleDeclineDraw() {
		if (!isPlayer || $gameStore.status !== 'playing') return;
		socket?.declineDraw();
		try {
			await api.games.declineDraw(gameId);
		} catch (_) {}
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

<div class="flex-1 flex flex-col justify-center max-w-7xl mx-auto w-full p-2 sm:px-4 sm:py-2.5 lg:px-6 lg:py-2.5">
	<div class="grid grid-cols-1 lg:grid-cols-12 gap-3 lg:gap-6 items-center justify-center my-auto">
		
		<!-- Board Column -->
		<div class="lg:col-span-7 xl:col-span-8 flex flex-col items-center justify-center">
			
			<div class="w-full max-w-[min(92vw,calc(100dvh-170px),570px)] flex flex-col items-center">
				<!-- Opponent Disconnection Countdown Banner -->
				{#if isPlayer && !$gameStore.opponentConnected && $gameStore.status === 'playing'}
					<div class="w-full mb-2 p-2.5 bg-rose-950/80 border border-rose-500/50 rounded-xl flex items-center justify-between text-xs text-rose-200 animate-pulse shadow-lg">
						<div class="flex items-center gap-2 font-bold">
							<Icon name="wifi-off" size={15} className="text-rose-400" />
							<span>Opponent disconnected</span>
						</div>
						<div class="font-mono text-[11px] font-extrabold bg-rose-900/80 px-2 py-0.5 rounded border border-rose-500/40">
							Claiming win in {opponentCountdown}s
						</div>
					</div>
				{/if}

				<!-- Top Player Bar (Opponent) -->
				<div class="w-full flex items-center justify-between py-1 px-1 text-sm font-semibold select-none mb-1">
					<div class="flex items-center gap-2">
						{#if topPlayer?.username && topPlayer.username !== 'guest-opponent' && topPlayer.username !== 'Opponent'}
							<UserBadge
								username={topPlayer.username}
								avatar={topPlayer.avatar}
								rating={topPlayer.rating || 400}
								ratingCategory="blitz"
								size="sm"
								target="_blank"
							/>
						{:else}
							<div class="flex items-center gap-2 text-white font-bold leading-tight">
								<div class="w-7 h-7 rounded-lg bg-slate-800/90 border border-slate-700/80 flex items-center justify-center text-xs font-mono font-bold text-slate-300 shadow-sm">
									<Icon name="user" size={13} />
								</div>
								<span class="truncate max-w-[140px] sm:max-w-none">{topPlayer?.username || 'Opponent'}</span>
								<span class="text-[11px] font-mono font-semibold px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700/60">
									{topPlayer?.rating || 400}
								</span>
							</div>
						{/if}
						<div class="ml-1">
							<CapturedPieces fen={$gameStore.fen} forColor={topColor} />
						</div>
					</div>

					<GameClock timeMs={topTimeMs} isActive={isTopTurn} />
				</div>

				<!-- Chess Board -->
				<ChessBoard
					fen={$gameStore.fen}
					orientation={currentOrientation}
					turn={$gameStore.turn}
					playerColor={isPlayer ? myPlayerColor : null}
					selectedSquare={$gameStore.selectedSquare}
					legalMoves={$gameStore.legalMoves}
					lastMove={$gameStore.lastMove}
					isCheck={$gameStore.isCheck}
					interactive={isPlayer && $gameStore.status === 'playing'}
					gameOver={$gameStore.status === 'finished'}
					gameResult={$gameStore.result || ''}
					gameOutcome={$gameStore.outcome || ''}
					gameWinner={$gameStore.winner || ''}
					disableGameOverModal={true}
					boardSizeClass="w-full"
					on:move={handleMove}
				/>

				<!-- Bottom Player Bar (Self) -->
				<div class="w-full flex items-center justify-between py-1 px-1 text-sm font-semibold select-none mt-1">
					<div class="flex items-center gap-2">
						{#if bottomPlayer?.username && bottomPlayer.username !== 'guest-opponent'}
							<UserBadge
								username={bottomPlayer.username}
								avatar={bottomPlayer.avatar}
								rating={bottomPlayer.rating || 400}
								ratingCategory="blitz"
								size="sm"
								target="_blank"
							/>
						{:else}
							<div class="flex items-center gap-2 text-white font-bold leading-tight">
								<div class="w-7 h-7 rounded-lg bg-sky-500/15 border border-sky-500/40 flex items-center justify-center text-xs font-mono font-bold text-sky-400 shadow-sm">
									<Icon name="crown" size={13} />
								</div>
								<span class="truncate max-w-[140px] sm:max-w-none">{bottomPlayer?.username || (isPlayer ? 'You' : 'White')}</span>
								<span class="text-[11px] font-mono font-semibold px-1.5 py-0.5 rounded bg-sky-500/10 text-sky-400 border border-sky-500/30">
									{bottomPlayer?.rating || 400}
								</span>
							</div>
						{/if}
						<div class="ml-1">
							<CapturedPieces fen={$gameStore.fen} forColor={bottomColor} />
						</div>
					</div>

					<GameClock timeMs={bottomTimeMs} isActive={isBottomTurn} />
				</div>
			</div>

		</div>

		<!-- Right Panel: HUD, Move Ledger & Actions -->
		<div class="lg:col-span-5 xl:col-span-4 w-full flex flex-col h-[420px] lg:h-[min(calc(100dvh-100px),620px)] bg-gradient-to-b from-[#121824] to-[#0c1017] border border-slate-800 rounded-2xl overflow-hidden shadow-2xl">
			
			<!-- HUD Header -->
			<div class="p-3 border-b border-slate-800 bg-[#090d15]/80 flex items-center justify-between">
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
						class="px-3 py-1 bg-[#81b64c] hover:bg-[#72a441] active:bg-[#639037] text-white font-bold rounded-lg text-xs transition shadow-xs active:translate-y-px"
						on:click={copyInviteLink}
					>
						{copiedInvite ? 'Copied Link!' : 'Copy Link'}
					</button>
				</div>
			{/if}

			<!-- Draw Offer Banner -->
			{#if isPlayer && $gameStore.drawOfferBy && $gameStore.drawOfferBy !== myPlayerColor}
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
				<MoveList
					moves={$gameStore.moves}
					whiteName={$gameStore.whitePlayer?.username || 'White'}
					blackName={$gameStore.blackPlayer?.username || 'Black'}
				/>
			</div>

			<!-- Action Controls -->
			<div class="p-3.5 border-t border-slate-800 bg-[#090d15]/80 flex gap-2">
				{#if $gameStore.status === 'playing'}
					{#if isPlayer}
						<button
							class="flex-1 py-2.5 bg-slate-800/80 hover:bg-slate-750 text-slate-300 font-bold text-xs rounded-xl transition border border-slate-700/70 flex items-center justify-center gap-1.5 disabled:opacity-50 disabled:cursor-not-allowed"
							on:click={handleOfferDraw}
							disabled={$gameStore.drawOfferBy === myPlayerColor}
						>
							<Icon name="handshake" size={14} className="text-slate-400" />
							<span>{$gameStore.drawOfferBy === myPlayerColor ? 'Draw Offered' : 'Offer Draw'}</span>
						</button>

						<button
							class="flex-1 py-2.5 bg-slate-800/80 hover:bg-rose-950/60 hover:text-rose-300 hover:border-rose-700/60 text-slate-300 font-bold text-xs rounded-xl transition border border-slate-700/70 flex items-center justify-center gap-1.5"
							on:click={handleResign}
						>
							<Icon name="flag" size={14} className="text-rose-400" />
							<span>Resign</span>
						</button>
					{:else}
						<div class="flex-1 py-2.5 bg-slate-800/60 text-slate-400 text-center text-xs font-semibold rounded-xl border border-slate-800 flex items-center justify-center gap-2">
							<Icon name="user" size={14} className="text-sky-400" />
							<span>Spectating Match</span>
						</div>
					{/if}
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
		playerColor={myPlayerColor || 'white'}
		on:newGame={handleNewGame}
		on:analyze={handleAnalyze}
		on:close={() => (showResultModal = false)}
	/>
{/if}

<!-- Custom Resign Confirmation Modal -->
<Modal
	isOpen={showResignModal}
	title="Concede & Resign?"
	description="Are you sure you want to resign this match? This will immediately conclude the game as a loss."
	icon="flag"
	iconBg="bg-rose-500/15 ring-1 ring-rose-500/30"
	iconColor="text-rose-400"
	confirmText="Yes, Resign"
	confirmVariant="danger"
	cancelText="Keep Playing"
	on:confirm={confirmResign}
	on:cancel={() => (showResignModal = false)}
	on:close={() => (showResignModal = false)}
/>

<!-- Custom Leave Match Warning Modal -->
<Modal
	isOpen={showLeaveModal}
	title="Leave Active Match?"
	description="You have a live online match in progress! If you leave now, you will have 60 seconds to reconnect before forfeiting by abandonment."
	icon="timer"
	iconBg="bg-amber-500/15 ring-1 ring-amber-500/30"
	iconColor="text-amber-400"
	confirmText="Leave Anyway"
	confirmVariant="warning"
	cancelText="Stay & Play"
	on:confirm={confirmLeaveMatch}
	on:cancel={() => {
		showLeaveModal = false;
		pendingNavigation = null;
	}}
	on:close={() => {
		showLeaveModal = false;
		pendingNavigation = null;
	}}
/>

<!-- Custom Draw Offer Modal (Received) -->
<Modal
	isOpen={isPlayer && !!$gameStore.drawOfferBy && $gameStore.drawOfferBy !== myPlayerColor}
	title="Draw Offered"
	description="Your opponent is proposing a mutual draw. Do you wish to accept and share the points?"
	icon="handshake"
	iconBg="bg-amber-500/15 ring-1 ring-amber-500/30"
	iconColor="text-amber-400"
	confirmText="Accept Draw"
	confirmVariant="primary"
	cancelText="Decline"
	on:confirm={handleAcceptDraw}
	on:cancel={handleDeclineDraw}
	on:close={handleDeclineDraw}
/>

<!-- Custom Offer Draw Confirmation Modal (Initiating) -->
<Modal
	isOpen={showOfferDrawModal}
	title="Propose a Draw?"
	description="Are you sure you want to offer a mutual draw to your opponent?"
	icon="handshake"
	iconBg="bg-amber-500/15 ring-1 ring-amber-500/30"
	iconColor="text-amber-400"
	confirmText="Offer Draw"
	confirmVariant="warning"
	cancelText="Keep Playing"
	on:confirm={confirmOfferDraw}
	on:cancel={() => (showOfferDrawModal = false)}
	on:close={() => (showOfferDrawModal = false)}
/>
