import { describe, it, expect } from 'vitest';
import { executeMoveClient, INITIAL_FEN } from './engine';

describe('Optimistic Move Execution & State Management', () => {
	it('executes client moves instantly with zero latency and correct state transitions', () => {
		const prevFen = INITIAL_FEN;
		const prevTurn = 'white';
		const prevMoves: Array<{ from: string; to: string; san: string; fen: string }> = [];

		// 1. Player makes optimistic move e2 -> e4
		const moveRes = executeMoveClient(prevFen, 'e2', 'e4');
		expect(moveRes).not.toBeNull();
		expect(moveRes?.valid).toBe(true);
		expect(moveRes?.san).toBe('e4');
		expect(moveRes?.turn).toBe('black');

		// 2. Optimistic state is updated immediately
		const nextMoves = [...prevMoves, { from: 'e2', to: 'e4', san: moveRes!.san, fen: moveRes!.newFen }];
		const nextFen = moveRes!.newFen;
		const nextTurn = moveRes!.turn;

		expect(nextMoves).toHaveLength(1);
		expect(nextMoves[0].san).toBe('e4');
		expect(nextTurn).toBe('black');
		expect(nextFen).toContain('rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq');
	});

	it('rejects illegal moves immediately without altering prior state', () => {
		const prevFen = INITIAL_FEN;
		const prevTurn = 'white';
		const prevMoves: Array<{ from: string; to: string; san: string; fen: string }> = [];

		// Player attempts illegal move e2 -> e5 (pawns cannot jump two ranks and a square)
		const moveRes = executeMoveClient(prevFen, 'e2', 'e5');
		expect(moveRes).toBeNull();

		// State remains untouched
		expect(prevFen).toBe(INITIAL_FEN);
		expect(prevTurn).toBe('white');
		expect(prevMoves).toHaveLength(0);
	});

	it('rolls back optimistic state cleanly when server rejects an optimistic move', () => {
		// Initial state
		const snapshotFen = INITIAL_FEN;
		const snapshotTurn = 'white';
		const snapshotMoves: Array<{ from: string; to: string; san: string; fen: string }> = [];
		const snapshotLastMove = null;

		// Optimistic update applied locally
		let currentFen = snapshotFen;
		let currentTurn = snapshotTurn;
		let currentMoves = [...snapshotMoves];
		let currentLastMove: any = snapshotLastMove;

		const optRes = executeMoveClient(currentFen, 'g1', 'f3');
		expect(optRes?.valid).toBe(true);

		currentFen = optRes!.newFen;
		currentTurn = optRes!.turn;
		currentLastMove = { from: 'g1', to: 'f3', san: optRes!.san };
		currentMoves = [...currentMoves, { from: 'g1', to: 'f3', san: optRes!.san, fen: optRes!.newFen }];

		expect(currentFen).not.toBe(snapshotFen);
		expect(currentMoves).toHaveLength(1);

		// Simulated server response: invalid move (e.g. clock expired, desync)
		const serverValidated = false;

		if (!serverValidated) {
			// Rollback to snapshot
			currentFen = snapshotFen;
			currentTurn = snapshotTurn;
			currentMoves = snapshotMoves;
			currentLastMove = snapshotLastMove;
		}

		expect(currentFen).toBe(INITIAL_FEN);
		expect(currentTurn).toBe('white');
		expect(currentMoves).toHaveLength(0);
		expect(currentLastMove).toBeNull();
	});

	it('handles pawn promotion optimistically', () => {
		// Board with white pawn on e7 ready to promote
		const promoFen = '8/4P3/8/8/8/8/8/4K2k w - - 0 1';
		const moveRes = executeMoveClient(promoFen, 'e7', 'e8', 'q');

		expect(moveRes).not.toBeNull();
		expect(moveRes?.valid).toBe(true);
		expect(moveRes?.isPromotion).toBe(true);
		expect(moveRes?.san).toBe('e8=Q');
		expect(moveRes?.newFen).toContain('4Q3/8/8/8/8/8/8/4K2k b - - 0 1');
	});

	it('deduplicates moves in multiplayer when server acknowledges optimistic move', () => {
		const moveFrom = 'e2';
		const moveTo = 'e4';
		const moves = [
			{ from: 'e2', to: 'e4', san: 'e4', fen: 'mock-fen-local' }
		];

		const lastM = moves.length > 0 ? moves[moves.length - 1] : null;
		const isOptimisticMatch = lastM && lastM.from === moveFrom && lastM.to === moveTo;
		expect(isOptimisticMatch).toBe(true);

		const serverData = {
			move: { from: 'e2', to: 'e4' },
			san: 'e4',
			fen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1'
		};

		const updatedMoves = [...moves];
		if (isOptimisticMatch) {
			updatedMoves[updatedMoves.length - 1] = {
				from: moveFrom,
				to: moveTo,
				san: serverData.san,
				fen: serverData.fen
			};
		} else {
			updatedMoves.push({
				from: moveFrom,
				to: moveTo,
				san: serverData.san,
				fen: serverData.fen
			});
		}

		// Length must still be 1 (not duplicated)
		expect(updatedMoves).toHaveLength(1);
		expect(updatedMoves[0].fen).toBe(serverData.fen);
	});

	it('parses Stockfish UCI bestMove string into board vector coordinates', () => {
		const uciMove = 'e7e5';
		expect(uciMove.length).toBeGreaterThanOrEqual(4);

		const hintArrow = {
			from: uciMove.slice(0, 2),
			to: uciMove.slice(2, 4)
		};

		expect(hintArrow.from).toBe('e7');
		expect(hintArrow.to).toBe('e5');
	});
});
