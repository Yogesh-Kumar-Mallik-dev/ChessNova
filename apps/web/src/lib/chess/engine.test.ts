import { describe, it, expect } from 'vitest';
import {
	parseSquare,
	squareName,
	fenToBoard,
	detectMoveCategory,
	executeMoveClient,
	INITIAL_FEN
} from './engine';

describe('Chess Client Engine Utilities', () => {
	it('correctly parses and formats square coordinates', () => {
		expect(parseSquare('a1')).toEqual({ file: 0, rank: 0 });
		expect(parseSquare('e4')).toEqual({ file: 4, rank: 3 });
		expect(parseSquare('h8')).toEqual({ file: 7, rank: 7 });
		expect(parseSquare('z9')).toBeNull();
		expect(parseSquare('invalid')).toBeNull();

		expect(squareName(0, 0)).toBe('a1');
		expect(squareName(4, 3)).toBe('e4');
		expect(squareName(7, 7)).toBe('h8');
	});

	it('converts starting FEN to 8x8 matrix correctly', () => {
		const board = fenToBoard(INITIAL_FEN);
		expect(board.length).toBe(8);
		expect(board[0].length).toBe(8);

		// Rank 1 (index 0 in 0-indexed chess coordinates): a1 = white rook, e1 = white king
		expect(board[0][0]).toEqual({ type: 'r', color: 'white' });
		expect(board[0][4]).toEqual({ type: 'k', color: 'white' });

		// Rank 8 (index 7): a8 = black rook, e8 = black king
		expect(board[7][0]).toEqual({ type: 'r', color: 'black' });
		expect(board[7][4]).toEqual({ type: 'k', color: 'black' });

		// Empty square e4 (rank index 3, col index 4)
		expect(board[3][4]).toBeNull();
	});

	it('executes client moves accurately and updates turn', () => {
		const res = executeMoveClient(INITIAL_FEN, 'e2', 'e4');
		expect(res).not.toBeNull();
		expect(res?.valid).toBe(true);
		expect(res?.san).toBe('e4');
		expect(res?.turn).toBe('black');

		// Black responds with e5
		const res2 = executeMoveClient(res!.newFen, 'e7', 'e5');
		expect(res2?.valid).toBe(true);
		expect(res2?.san).toBe('e5');
		expect(res2?.turn).toBe('white');
	});

	it('identifies capture, check, and castle categories from SAN', () => {
		expect(detectMoveCategory('fen', 'e4', 'd5', undefined, 'exd5')).toBe('capture');
		expect(detectMoveCategory('fen', 'e1', 'g1', undefined, 'O-O')).toBe('castle');
		expect(detectMoveCategory('fen', 'c4', 'f7', undefined, 'Bxf7+')).toBe('check');
		expect(detectMoveCategory('fen', 'd1', 'h5', undefined, 'Qh5#')).toBe('checkmate');
		expect(detectMoveCategory('fen', 'e2', 'e4', undefined, 'e4')).toBe('normal');
	});
});
