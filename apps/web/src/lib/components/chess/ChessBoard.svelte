<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import { Chess } from 'chess.js';
	import { api } from '$lib/api/client';
	import {
		fenToBoard,
		squareName,
		getLegalMovesForSquare,
		getCheckedKingSquare,
		isPromotionMove,
		detectMoveCategory,
		type Color,
		type Piece,
		type PieceType,
		type LegalTarget
	} from '$lib/chess/engine';
	import { playMoveSoundByCategory, soundEffects } from '$lib/audio/sounds';
	import ChessSquare from './ChessSquare.svelte';
	import ChessPiece from './ChessPiece.svelte';
	import PromotionDialog from './PromotionDialog.svelte';
	import Icon from '$lib/components/icons/Icon.svelte';

	import type { MoveClassification } from '$lib/chess/review';

	export let fen: string;
	export let orientation: Color = 'white';
	export let turn: Color = 'white';
	export let selectedSquare: string | null = null;
	export let legalMoves: string[] = [];
	export let lastMove: { from: string; to: string; san?: string; isCapture?: boolean; isPromotion?: boolean } | null = null;
	export let isCheck: boolean = false;
	export let interactive = true;
	export let showSoundToggle = false;
	export let gameOver = false;
	export let gameResult = '';
	export let gameOutcome = '';
	export let gameWinner = '';
	export let disableGameOverModal = false;
	export let mode: 'game' | 'puzzle' | 'analysis' = 'game';
	export let hintSquare: string | null = null;
	export let hintArrow: { from: string; to: string } | null = null;
	export let reviewMove: {
		from: string;
		to: string;
		classification: MoveClassification;
		bestMoveUci?: string;
	} | null = null;
	import { themeStore, type BoardTheme } from '$lib/stores/preferences';

	export let theme: BoardTheme = $themeStore;
	export let boardSizeClass: string = 'max-w-[min(90vw,calc(100dvh-180px),580px)]';

	onMount(() => {
		const unsub = themeStore.subscribe((val) => {
			if (val) theme = val;
		});
		return () => unsub();
	});

	const dispatch = createEventDispatcher<{
		move: { from: string; to: string; promotion?: PieceType; isCapture?: boolean; isPromotion?: boolean };
		select: string;
		newGame: void;
		rematch: void;
		analyze: void;
	}>();

	let draggedFrom: string | null = null;
	let pendingPromotion: { from: string; to: string; isCapture?: boolean } | null = null;
	let soundMuted = false;
	let showGameOverModal = true;
	let losingKingReason: 'checkmate' | 'resignation' | 'timeout' = 'checkmate';

	// Animation state for multi-piece sliding (e.g. King + Rook in Castling)
	let prevLastMoveKey = '';
	let animatingMap = new Map<string, { dx: number; dy: number }>();
	let isSliding = false;

	$: board = fenToBoard(fen);

	let serverLegalTargets: LegalTarget[] = [];
	let lastReqKey = '';
	$: if (selectedSquare && fen) {
		const key = `${fen}:${selectedSquare}`;
		if (key !== lastReqKey) {
			lastReqKey = key;
			api.chess.legalMoves(fen, selectedSquare).then((res) => {
				serverLegalTargets = res.targets.map((t) => ({
					from: selectedSquare!,
					to: t.to,
					category: (t.category as any) || (t.isCapture ? 'capture' : 'normal'),
					isCapture: t.isCapture,
					isEnPassant: t.category === 'en_passant',
					isCastle: t.category === 'castle',
					isPromotion: false,
					san: t.san
				}));
			}).catch(() => {
				serverLegalTargets = getLegalMovesForSquare(fen, selectedSquare!);
			});
		}
	} else {
		serverLegalTargets = [];
		lastReqKey = '';
	}

	// Compute legal moves authoritatively from backend server
	$: legalTargets = selectedSquare ? (serverLegalTargets.length > 0 ? serverLegalTargets : getLegalMovesForSquare(fen, selectedSquare)) : [];
	$: legalTargetMap = new Map<string, LegalTarget>(legalTargets.map((t) => [t.to, t]));
	$: legalTargetSquares = new Set([...legalTargets.map((t) => t.to), ...legalMoves]);
	$: legalCaptureSquares = new Set(legalTargets.filter((t) => t.isCapture).map((t) => t.to));

	// Engine state evaluation for game termination & check detection
	$: engineState = (() => {
		try {
			const c = new Chess(fen);
			return {
				isCheckmate: c.isCheckmate(),
				isDraw: c.isDraw(),
				inCheck: c.inCheck()
			};
		} catch {
			return { isCheckmate: false, isDraw: false, inCheck: false };
		}
	})();

	$: kingInCheckSquare = getCheckedKingSquare(fen) || (isCheck || engineState.inCheck ? getCheckedKingSquare(fen) : null);

	$: isCheckmate = gameOver ? gameOutcome === 'checkmate' : engineState.isCheckmate;
	$: isDraw = gameOver
		? gameResult === '1/2-1/2' || ['stalemate', 'threefold_repetition', 'fifty_moves', 'insufficient_material', 'draw_agreed'].includes(gameOutcome)
		: engineState.isDraw;
	$: isGameOver = mode === 'game' ? (gameOver || isCheckmate || isDraw) : false;

	$: winnerColor = mode === 'game' ? (gameOver ? gameWinner : (isCheckmate ? (turn === 'white' ? 'black' : 'white') : '')) : '';

	$: losingKingSquare = (() => {
		if (mode !== 'game') return null;
		if (isCheckmate) return getCheckedKingSquare(fen);
		if (gameOver && winnerColor && winnerColor !== 'draw') {
			const loserColor = winnerColor === 'white' ? 'black' : 'white';
			for (let r = 0; r < 8; r++) {
				for (let f = 0; f < 8; f++) {
					const p = board[r][f];
					if (p && p.type === 'k' && p.color === loserColor) {
						return squareName(f, r);
					}
				}
			}
		}
		return null;
	})();

	$: winningKingSquare = (() => {
		if (mode !== 'game') return null;
		if (!winnerColor || winnerColor === 'draw') return null;
		for (let r = 0; r < 8; r++) {
			for (let f = 0; f < 8; f++) {
				const p = board[r][f];
				if (p && p.type === 'k' && p.color === winnerColor) {
					return squareName(f, r);
				}
			}
		}
		return null;
	})();

	$: losingKingReason = (() => {
		if (gameOutcome === 'resignation') return 'resignation';
		if (gameOutcome === 'timeout') return 'timeout';
		return 'checkmate';
	})();

	$: modalTitle = isDraw ? 'Draw' : winnerColor === 'white' ? 'White Won' : 'Black Won';
	$: modalSubtitle = (() => {
		if (isCheckmate) return 'by checkmate';
		if (gameOutcome === 'resignation') return 'by resignation';
		if (gameOutcome === 'timeout') return 'by timeout';
		if (gameOutcome === 'stalemate') return 'by stalemate';
		if (gameOutcome === 'insufficient_material') return 'by insufficient material';
		if (gameOutcome === 'threefold_repetition') return 'by repetition';
		if (gameOutcome === 'fifty_moves') return 'by 50-move rule';
		if (isDraw) return 'by agreement';
		return 'game concluded';
	})();

	let prevFen = fen;

	// Trigger animation & sound on lastMove change
	$: if (lastMove) {
		const key = `${lastMove.from}-${lastMove.to}-${fen}`;
		if (key !== prevLastMoveKey) {
			prevLastMoveKey = key;
			triggerMoveEffects(lastMove.from, lastMove.to, prevFen, lastMove.san, lastMove.isCapture, lastMove.isPromotion);
			prevFen = fen;
		}
	} else {
		prevFen = fen;
	}

	let prevGameOver = false;
	$: if (isGameOver && !prevGameOver) {
		prevGameOver = true;
		if (!isCheckmate && mode === 'game') {
			soundEffects.playGameEnd();
		}
	} else if (!isGameOver) {
		prevGameOver = false;
	}

	function getSlideDelta(fromSq: string, toSq: string) {
		const fromF = fromSq.charCodeAt(0) - 'a'.charCodeAt(0);
		const fromR = parseInt(fromSq[1], 10) - 1;
		const toF = toSq.charCodeAt(0) - 'a'.charCodeAt(0);
		const toR = parseInt(toSq[1], 10) - 1;

		let dx = (fromF - toF) * 100;
		let dy = (toR - fromR) * 100;

		if (orientation === 'black') {
			dx = -dx;
			dy = -dy;
		}

		return { dx, dy };
	}

	function triggerMoveEffects(
		fromSq: string,
		toSq: string,
		fenBefore?: string,
		san?: string,
		explicitCapture?: boolean,
		explicitPromotion?: boolean
	) {
		// 1. Detect iconic move category: normal, capture, castle, en_passant, promote, check, or checkmate
		let category = detectMoveCategory(fen, fromSq, toSq, fenBefore, san, explicitPromotion);
		if (explicitPromotion && category !== 'checkmate') {
			category = 'promote';
		} else if (explicitCapture && (category === 'normal' || category === 'castle')) {
			category = 'capture';
		}
		if (mode === 'puzzle' && category === 'checkmate') {
			soundEffects.playCheck();
		} else {
			playMoveSoundByCategory(category);
		}

		// 2. Calculate primary slide animation delta
		const primaryDelta = getSlideDelta(fromSq, toSq);
		const newMap = new Map<string, { dx: number; dy: number }>();
		newMap.set(toSq, primaryDelta);

		// 3. If castling, simultaneously slide the accompanying Rook!
		if (category === 'castle') {
			let rookFrom = '';
			let rookTo = '';
			if (fromSq === 'e1' && toSq === 'g1') { rookFrom = 'h1'; rookTo = 'f1'; }
			else if (fromSq === 'e1' && toSq === 'c1') { rookFrom = 'a1'; rookTo = 'd1'; }
			else if (fromSq === 'e8' && toSq === 'g8') { rookFrom = 'h8'; rookTo = 'f8'; }
			else if (fromSq === 'e8' && toSq === 'c8') { rookFrom = 'a8'; rookTo = 'd8'; }

			if (rookFrom && rookTo) {
				newMap.set(rookTo, getSlideDelta(rookFrom, rookTo));
			}
		}

		animatingMap = newMap;
		isSliding = false;

		requestAnimationFrame(() => {
			requestAnimationFrame(() => {
				isSliding = true;
				animatingMap = new Map(Array.from(newMap.keys()).map((k) => [k, { dx: 0, dy: 0 }]));
			});
		});

		setTimeout(() => {
			animatingMap = new Map();
			isSliding = false;
		}, 200);
	}

	function toggleAudio() {
		soundMuted = !soundEffects.toggleSound();
	}

	function handleSquareClick(sq: string) {
		if (!interactive) return;

		// If a piece is already selected and user clicked another square
		if (selectedSquare) {
			if (selectedSquare === sq) {
				selectedSquare = null;
				return;
			}

			// Check if clicked user's own piece -> switch selection to that piece
			const [f, r] = [sq.charCodeAt(0) - 'a'.charCodeAt(0), parseInt(sq[1], 10) - 1];
			const targetPiece = board[r][f];
			if (targetPiece && targetPiece.color === turn) {
				selectedSquare = sq;
				dispatch('select', sq);
				return;
			}

			// If it's a legal move target, execute the move!
			if (legalTargetSquares.has(sq)) {
				const isCap = legalCaptureSquares.has(sq);
				if (isPromotionMove(fen, selectedSquare, sq)) {
					pendingPromotion = { from: selectedSquare, to: sq, isCapture: isCap };
					return;
				}

				executeMove(selectedSquare, sq, undefined, isCap);
				selectedSquare = null;
				return;
			}

			// Clicked an illegal/empty square -> deselect
			selectedSquare = null;
			return;
		}

		// No piece selected yet: select if it's the current player's piece
		const [f, r] = [sq.charCodeAt(0) - 'a'.charCodeAt(0), parseInt(sq[1], 10) - 1];
		const piece = board[r][f];
		if (piece && piece.color === turn) {
			selectedSquare = sq;
			dispatch('select', sq);
		}
	}

	function handleDragStart(e: DragEvent, sq: string) {
		if (!interactive) return;
		const [f, r] = [sq.charCodeAt(0) - 'a'.charCodeAt(0), parseInt(sq[1], 10) - 1];
		const piece = board[r][f];
		if (!piece || piece.color !== turn) {
			e.preventDefault();
			return;
		}

		draggedFrom = sq;
		selectedSquare = sq;
		if (e.dataTransfer) {
			e.dataTransfer.setData('text/plain', sq);
			e.dataTransfer.effectAllowed = 'move';
		}
		dispatch('select', sq);
	}

	function handleDragEnd() {
		draggedFrom = null;
	}

	function handleDrop(e: CustomEvent<DragEvent>, targetSq: string) {
		if (!interactive) return;
		const sourceSq = e.detail.dataTransfer?.getData('text/plain') || draggedFrom || selectedSquare;
		if (!sourceSq || sourceSq === targetSq) {
			draggedFrom = null;
			return;
		}

		// Check if target is a legal move
		const legalForSource = legalTargets.length > 0 && selectedSquare === sourceSq ? legalTargets : getLegalMovesForSquare(fen, sourceSq);
		const targetMove = legalForSource.find((m) => m.to === targetSq);

		if (targetMove) {
			const isCap = targetMove.isCapture;
			if (isPromotionMove(fen, sourceSq, targetSq)) {
				pendingPromotion = { from: sourceSq, to: targetSq, isCapture: isCap };
				draggedFrom = null;
				return;
			}
			executeMove(sourceSq, targetSq, undefined, isCap);
		}

		draggedFrom = null;
		selectedSquare = null;
	}

	function executeMove(from: string, to: string, promo?: PieceType, isCapture?: boolean, isPromotion?: boolean) {
		dispatch('move', { from, to, promotion: promo, isCapture, isPromotion });
	}

	function handlePromotionSelect(event: CustomEvent<PieceType>) {
		if (pendingPromotion) {
			executeMove(pendingPromotion.from, pendingPromotion.to, event.detail, pendingPromotion.isCapture, true);
			pendingPromotion = null;
			selectedSquare = null;
		}
	}

	function squareToPercent(sq: string, isWhiteOriented: boolean): { x: number; y: number } {
		const f = sq.charCodeAt(0) - 'a'.charCodeAt(0);
		const r = parseInt(sq[1], 10) - 1;
		const x = isWhiteOriented ? (f + 0.5) * 12.5 : (7 - f + 0.5) * 12.5;
		const y = isWhiteOriented ? (7 - r + 0.5) * 12.5 : (r + 0.5) * 12.5;
		return { x, y };
	}

	// Calculate 8x8 squares in order depending on orientation
	$: ranks = orientation === 'white' ? [7, 6, 5, 4, 3, 2, 1, 0] : [0, 1, 2, 3, 4, 5, 6, 7];
	$: files = orientation === 'white' ? [0, 1, 2, 3, 4, 5, 6, 7] : [7, 6, 5, 4, 3, 2, 1, 0];
	$: effectiveDisableGameOverModal = disableGameOverModal || mode !== 'game';
</script>

<div class="relative w-full aspect-square {boardSizeClass} rounded-2xl overflow-hidden shadow-[0_25px_60px_rgba(0,0,0,0.85)] border-4 border-slate-800 ring-1 ring-slate-700/60 select-none {mode === 'game' && isCheckmate ? 'ring-2 ring-amber-400/80 animate-[boardImpact_450ms_ease-out]' : ''}">
	{#if showSoundToggle}
		<button
			class="absolute top-2 right-2 z-30 p-2 rounded-xl bg-slate-900/80 hover:bg-slate-800 border border-slate-700/60 text-slate-300 transition shadow-md"
			on:click={toggleAudio}
			title={soundMuted ? 'Unmute Sound Effects' : 'Mute Sound Effects'}
		>
			<Icon name={soundMuted ? 'volume-x' : 'volume-2'} size={15} />
		</button>
	{/if}

	<div class="grid grid-cols-8 grid-rows-8 w-full h-full">
		{#each ranks as r, rIndex}
			{#each files as f, fIndex}
				{@const sq = squareName(f, r)}
				{@const isLight = (f + r) % 2 !== 0}
				{@const piece = board[r][f]}
				{@const isSelected = selectedSquare === sq}
				{@const isLastFrom = lastMove?.from === sq}
				{@const isLastTo = lastMove?.to === sq}
				{@const targetInfo = legalTargetMap.get(sq)}
				{@const isLegal = legalTargetSquares.has(sq)}
				{@const isCapture = legalCaptureSquares.has(sq)}
				{@const legalCategory = targetInfo?.category || 'normal'}
				{@const isCheckSquare = kingInCheckSquare === sq}
				{@const anim = animatingMap.get(sq)}
				{@const showRank = fIndex === 0}
				{@const showFile = rIndex === 7}

				<ChessSquare
					file={f}
					rank={r}
					squareName={sq}
					{isLight}
					{isSelected}
					isLastMoveFrom={isLastFrom}
					isLastMoveTo={isLastTo}
					{isLegal}
					{isCapture}
					isHintSquare={hintSquare === sq}
					{legalCategory}
					isCheck={isCheckSquare && (mode !== 'game' || !isCheckmate)}
					isWinningKing={winningKingSquare === sq}
					isLosingKing={losingKingSquare === sq}
					{losingKingReason}
					isDrawKing={isDraw && piece?.type === 'k'}
					reviewClassification={reviewMove?.to === sq ? reviewMove.classification : null}
					showRankCoord={showRank}
					showFileCoord={showFile}
					{theme}
					on:click={() => handleSquareClick(sq)}
					on:drop={(e) => handleDrop(e, sq)}
				>
					{#if piece}
						<div
							class="w-full h-full"
							style={anim
								? `transform: translate(${anim.dx}%, ${anim.dy}%); transition: ${isSliding ? 'transform 180ms cubic-bezier(0.2, 0, 0.2, 1)' : 'none'}; z-index: 40; will-change: transform;`
								: ''}
						>
							<ChessPiece
								{piece}
								draggable={interactive && piece.color === turn}
								isDragging={draggedFrom === sq}
								on:dragstart={(e) => handleDragStart(e, sq)}
								on:dragend={handleDragEnd}
							/>
						</div>
					{/if}
				</ChessSquare>
			{/each}
		{/each}
	</div>

	<!-- Tactical Hint Arrow Overlay (Luminous cyan tactical vector arrow) -->
	{#if hintArrow && hintArrow.from && hintArrow.to}
		{@const hp1 = squareToPercent(hintArrow.from, orientation === 'white')}
		{@const hp2 = squareToPercent(hintArrow.to, orientation === 'white')}
		<svg class="absolute inset-0 w-full h-full pointer-events-none z-20 overflow-visible">
			<defs>
				<marker id="hint-target-arrow" markerWidth="6" markerHeight="6" refX="4" refY="3" orient="auto">
					<polygon points="0 0, 6 3, 0 6" fill="#06b6d4" />
				</marker>
			</defs>
			<line
				x1="{hp1.x}%"
				y1="{hp1.y}%"
				x2="{hp2.x}%"
				y2="{hp2.y}%"
				stroke="#06b6d4"
				stroke-width="6"
				stroke-linecap="round"
				marker-end="url(#hint-target-arrow)"
				opacity="0.9"
			/>
		</svg>
	{/if}

	<!-- Best Move Arrow Overlay -->
	{#if reviewMove && reviewMove.bestMoveUci && reviewMove.bestMoveUci.length >= 4 && reviewMove.bestMoveUci !== `${reviewMove.from}${reviewMove.to}`}
		{@const bFrom = reviewMove.bestMoveUci.slice(0, 2)}
		{@const bTo = reviewMove.bestMoveUci.slice(2, 4)}
		{@const p1 = squareToPercent(bFrom, orientation === 'white')}
		{@const p2 = squareToPercent(bTo, orientation === 'white')}
		<svg class="absolute inset-0 w-full h-full pointer-events-none z-20 overflow-visible">
			<defs>
				<marker id="review-best-arrow" markerWidth="6" markerHeight="6" refX="4" refY="3" orient="auto">
					<polygon points="0 0, 6 3, 0 6" fill="#38bdf8" />
				</marker>
			</defs>
			<line
				x1="{p1.x}%"
				y1="{p1.y}%"
				x2="{p2.x}%"
				y2="{p2.y}%"
				stroke="#38bdf8"
				stroke-width="6"
				stroke-linecap="round"
				marker-end="url(#review-best-arrow)"
				opacity="0.88"
			/>
		</svg>
	{/if}

	<!-- Game Over Modal -->
	{#if isGameOver && !effectiveDisableGameOverModal && showGameOverModal}
		<div class="absolute inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-3 animate-in fade-in duration-200">
			<div class="relative bg-slate-900/95 border border-slate-800 rounded-2xl p-6 shadow-2xl max-w-[320px] sm:max-w-[340px] w-full flex flex-col items-center text-center">
				<!-- Close 'X' button to dismiss modal and view the board freely -->
				<button
					class="absolute top-3 right-3 w-7 h-7 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 flex items-center justify-center transition"
					on:click={() => (showGameOverModal = false)}
					title="Close and view board"
				>
					<Icon name="x" size={16} />
				</button>

				<!-- Winner Crown or Draw Emblem -->
				<div class="w-12 h-12 rounded-xl flex items-center justify-center mb-3 {isDraw ? 'bg-slate-800 text-slate-300' : 'bg-amber-400/15 text-amber-400 ring-1 ring-amber-400/30'}">
					{#if isDraw}
						<Icon name="handshake" size={24} />
					{:else}
						<Icon name="crown" size={26} className="text-amber-400" />
					{/if}
				</div>

				<!-- Result Title -->
				<h3 class="text-xl font-black text-white tracking-tight">
					{modalTitle}
				</h3>

				<!-- Subtitle -->
				<p class="text-xs text-slate-400 font-medium mt-1 mb-5">
					{modalSubtitle}
				</p>

				<!-- Actions -->
				<div class="flex flex-col gap-2.5 w-full">
					<a
						href="/analysis?fen={encodeURIComponent(fen)}"
						class="w-full py-2.5 px-4 bg-sky-500 hover:bg-sky-400 active:bg-sky-600 text-white font-extrabold text-xs uppercase tracking-wide rounded-xl transition-all flex items-center justify-center gap-2 shadow-lg shadow-sky-500/20"
						on:click={() => dispatch('analyze')}
					>
						<Icon name="search" size={15} />
						<span>Game Review</span>
					</a>

					<div class="grid grid-cols-2 gap-2 w-full">
						<button
							class="py-2 px-3 bg-slate-800 hover:bg-slate-700 text-slate-200 font-bold text-xs rounded-xl border border-slate-700/60 transition-all flex items-center justify-center gap-1.5"
							on:click={() => dispatch('rematch')}
						>
							<Icon name="rotate-ccw" size={13} />
							<span>Rematch</span>
						</button>
						<button
							class="py-2 px-3 bg-slate-800 hover:bg-slate-700 text-slate-200 font-bold text-xs rounded-xl border border-slate-700/60 transition-all flex items-center justify-center gap-1.5"
							on:click={() => dispatch('newGame')}
						>
							<Icon name="zap" size={13} />
							<span>New Game</span>
						</button>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- Minimized Bottom Status Bar -->
	{#if isGameOver && !effectiveDisableGameOverModal && !showGameOverModal}
		<div class="absolute bottom-3 inset-x-3 z-40 bg-slate-900/95 border border-slate-800 shadow-2xl rounded-xl py-2 px-3 flex items-center justify-between text-xs backdrop-blur-md animate-in fade-in">
			<div class="flex items-center gap-2 truncate pr-2">
				{#if !isDraw}
					<div class="w-5 h-5 rounded-full bg-amber-500/20 border border-amber-500/40 flex items-center justify-center shrink-0 text-amber-400">
						<Icon name="crown" size={12} />
					</div>
				{/if}
				<span class="font-bold text-white capitalize truncate">{modalTitle}</span>
				<span class="text-slate-400 font-normal shrink-0">{modalSubtitle}</span>
			</div>
			<div class="flex items-center gap-2 shrink-0">
				<a
					href="/analysis?fen={encodeURIComponent(fen)}"
					class="px-2.5 py-1 bg-sky-500 hover:bg-sky-400 text-white font-bold text-[11px] rounded-lg shadow transition"
				>
					Review
				</a>
				<button
					class="px-2.5 py-1 bg-slate-800 hover:bg-slate-700 text-slate-200 font-bold text-[11px] rounded-lg transition"
					on:click={() => (showGameOverModal = true)}
				>
					Dialog
				</button>
			</div>
		</div>
	{/if}

	<!-- Promotion Dialog -->
	{#if pendingPromotion}
		<PromotionDialog color={turn} on:select={handlePromotionSelect} />
	{/if}
</div>

<style>
	@keyframes boardImpact {
		0% { transform: scale(1); }
		20% { transform: scale(0.985); }
		45% { transform: scale(1.018); }
		75% { transform: scale(0.995); }
		100% { transform: scale(1); }
	}
</style>
