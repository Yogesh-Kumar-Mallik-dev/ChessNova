import { describe, it, expect } from 'vitest';
import {
	parseSquare,
	squareName,
	fenToBoard,
	boardToFen,
	getLegalMovesForSquare,
	getCheckedKingSquare,
	isPromotionMove,
	executeMoveClient,
	detectMoveCategory,
	detectMoveSoundType,
	INITIAL_FEN
} from './engine';

describe('Chess Client Edge Cases & Breaking Points', () => {
	describe('Square Parsing and Formatting Bounds', () => {
		it('handles boundary squares accurately', () => {
			expect(parseSquare('a1')).toEqual({ file: 0, rank: 0 });
			expect(parseSquare('h1')).toEqual({ file: 7, rank: 0 });
			expect(parseSquare('a8')).toEqual({ file: 0, rank: 7 });
			expect(parseSquare('h8')).toEqual({ file: 7, rank: 7 });
		});

		it('rejects invalid or out-of-bound square strings gracefully', () => {
			expect(parseSquare('')).toBeNull();
			expect(parseSquare('a')).toBeNull();
			expect(parseSquare('e44')).toBeNull();
			expect(parseSquare('i4')).toBeNull();
			expect(parseSquare('a0')).toBeNull();
			expect(parseSquare('a9')).toBeNull();
			expect(parseSquare('Z1')).toBeNull();
			expect(parseSquare('??')).toBeNull();
		});

		it('formats square names across the full 64-square grid', () => {
			for (let file = 0; file < 8; file++) {
				for (let rank = 0; rank < 8; rank++) {
					const name = squareName(file, rank);
					expect(name).toHaveLength(2);
					const parsed = parseSquare(name);
					expect(parsed).toEqual({ file, rank });
				}
			}
		});
	});

	describe('FEN to Board & Roundtrip Conversion', () => {
		it('preserves board structure across starting position', () => {
			const board = fenToBoard(INITIAL_FEN);
			expect(board).toHaveLength(8);
			board.forEach((rank) => expect(rank).toHaveLength(8));

			// Verify pawns on rank 2 and 7
			for (let f = 0; f < 8; f++) {
				expect(board[1][f]).toEqual({ type: 'p', color: 'white' });
				expect(board[6][f]).toEqual({ type: 'p', color: 'black' });
			}

			// Empty squares in ranks 3, 4, 5, 6 (0-indexed 2, 3, 4, 5)
			for (let r = 2; r <= 5; r++) {
				for (let f = 0; f < 8; f++) {
					expect(board[r][f]).toBeNull();
				}
			}
		});

		it('roundtrips sparse endgame boards', () => {
			const sparseFen = '8/8/8/8/8/8/8/4K2k w - - 0 1';
			const board = fenToBoard(sparseFen);
			expect(board[0][4]).toEqual({ type: 'k', color: 'white' });
			expect(board[0][7]).toEqual({ type: 'k', color: 'black' });

			const generatedFen = boardToFen(board, 'white');
			expect(generatedFen).toContain('8/8/8/8/8/8/8/4K2k w');
		});
	});

	describe('Legal Moves Generation & FEN Tolerances', () => {
		it('returns empty array on malformed FEN or non-existent square', () => {
			expect(getLegalMovesForSquare('malformed fen string', 'e2')).toEqual([]);
			expect(getLegalMovesForSquare(INITIAL_FEN, 'z9')).toEqual([]);
			expect(getLegalMovesForSquare(INITIAL_FEN, 'e4')).toEqual([]); // empty square has 0 moves
		});

		it('detects en passant and castling flags in legal targets', () => {
			// Position where White can castle Kingside and Queenside
			const castleFen = 'r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1';
			const kingMoves = getLegalMovesForSquare(castleFen, 'e1');
			const castleTargets = kingMoves.filter((m) => m.isCastle);
			expect(castleTargets).toHaveLength(2);
			expect(castleTargets.map((m) => m.to)).toEqual(expect.arrayContaining(['g1', 'c1']));

			// En passant position: White pawn on e5, Black pawn on d5 with d6 en passant square
			const epFen = 'rnbqkbnr/ppp1pppp/8/3pP3/8/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2';
			const pawnMoves = getLegalMovesForSquare(epFen, 'e5');
			const epMove = pawnMoves.find((m) => m.to === 'd6');
			expect(epMove).toBeDefined();
			expect(epMove?.isEnPassant).toBe(true);
			expect(epMove?.isCapture).toBe(true);
		});
	});

	describe('Check and Checkmate Detection', () => {
		it('identifies checked king correctly', () => {
			expect(getCheckedKingSquare(INITIAL_FEN)).toBeNull();

			// Fool's mate checkmate position: White king on e1 is checked by Qh4
			const foolsMateFen = 'rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3';
			expect(getCheckedKingSquare(foolsMateFen)).toBe('e1');

			// Black king checked: e8 checked by Qh5
			const blackCheckFen = 'rnbqkbnr/ppppp2p/5p2/6pQ/4P3/8/PPPP1PPP/RNB1KBNR b KQkq - 1 3';
			expect(getCheckedKingSquare(blackCheckFen)).toBe('e8');
		});
	});

	describe('Promotion Move Detection & Execution', () => {
		it('correctly detects pawn promotion requirements', () => {
			const whitePromoReady = '8/4P3/8/8/8/8/8/4K2k w - - 0 1';
			expect(isPromotionMove(whitePromoReady, 'e7', 'e8')).toBe(true);
			// Non-promotion move for white pawn
			expect(isPromotionMove(whitePromoReady, 'e7', 'e6')).toBe(false);

			const blackPromoReady = '4K2k/8/8/8/8/8/4p3/8 b - - 0 1';
			expect(isPromotionMove(blackPromoReady, 'e2', 'e1')).toBe(true);

			// Piece other than pawn moving to back rank
			const rookFen = '8/4R3/8/8/8/8/8/4K2k w - - 0 1';
			expect(isPromotionMove(rookFen, 'e7', 'e8')).toBe(false);
		});

		it('executes underpromotions (Knight, Rook, Bishop)', () => {
			const promoFen = '8/4P3/8/8/8/8/8/4K2k w - - 0 1';

			// Underpromote to Knight
			const knightRes = executeMoveClient(promoFen, 'e7', 'e8', 'n');
			expect(knightRes?.valid).toBe(true);
			expect(knightRes?.san).toBe('e8=N');
			expect(knightRes?.newFen).toContain('4N3/8/8/8/8/8/8/4K2k b');

			// Underpromote to Rook
			const rookRes = executeMoveClient(promoFen, 'e7', 'e8', 'r');
			expect(rookRes?.valid).toBe(true);
			expect(rookRes?.san).toBe('e8=R');

			// Underpromote to Bishop
			const bishopRes = executeMoveClient(promoFen, 'e7', 'e8', 'b');
			expect(bishopRes?.valid).toBe(true);
			expect(bishopRes?.san).toBe('e8=B');
		});
	});

	describe('Pinned Piece and Illegal Move Rejections', () => {
		it('rejects moves by absolute pinned pieces exposing the king', () => {
			// Absolute pin: White king e1, White bishop e2, Black rook e7, Black king h8
			const pinFen = '7k/4r3/8/8/8/8/4B3/4K3 w - - 0 1';
			// Bishop trying to move off the pin line to c4
			const res = executeMoveClient(pinFen, 'e2', 'c4');
			expect(res).toBeNull();

			// King moving out of pin to f1 is legal:
			const kingMove = executeMoveClient(pinFen, 'e1', 'f1');
			expect(kingMove?.valid).toBe(true);
		});

		it('rejects moving pieces of the wrong color on current turn', () => {
			// White's turn, trying to move Black pawn
			const res = executeMoveClient(INITIAL_FEN, 'e7', 'e5');
			expect(res).toBeNull();
		});

		it('rejects moving from empty squares or offboard', () => {
			expect(executeMoveClient(INITIAL_FEN, 'e4', 'e5')).toBeNull();
			expect(executeMoveClient(INITIAL_FEN, 'z9', 'e4')).toBeNull();
		});
	});

	describe('Sound & Move Category Detection Breakdown', () => {
		it('detects sound types for various move patterns', () => {
			const checkSound = detectMoveSoundType(
				'rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3',
				'd8',
				'h4'
			);
			expect(checkSound.isCheck).toBe(true);

			const normalSound = detectMoveSoundType(INITIAL_FEN, 'e2', 'e4');
			expect(normalSound.isCheck).toBe(false);
			expect(normalSound.isCastle).toBe(false);
		});

		it('categorizes checkmate, promotion, castle, and en passant moves', () => {
			expect(detectMoveCategory(INITIAL_FEN, 'e2', 'e4', undefined, 'Qh5#')).toBe('checkmate');
			expect(detectMoveCategory(INITIAL_FEN, 'e7', 'e8', undefined, 'e8=Q')).toBe('promote');
			expect(detectMoveCategory(INITIAL_FEN, 'e1', 'g1', undefined, 'O-O')).toBe('castle');
			expect(detectMoveCategory(INITIAL_FEN, 'e1', 'c1', undefined, 'O-O-O')).toBe('castle');
			expect(detectMoveCategory(INITIAL_FEN, 'e4', 'd5', undefined, 'exd5')).toBe('capture');
		});
	});
});
