// @vitest-environment happy-dom
import { describe, it, expect, beforeEach } from 'vitest';
import {
	getLocalGames,
	saveLocalGame,
	deleteLocalGame,
	clearAllLocalGames,
	stageGameForReview,
	getStagedGameForReview
} from './localGames';

describe('Local Games Archive Store', () => {
	beforeEach(() => {
		clearAllLocalGames();
	});

	it('saves and retrieves local games correctly', () => {
		expect(getLocalGames()).toEqual([]);

		const saved = saveLocalGame({
			date: new Date().toISOString(),
			event: 'Test Match',
			white: 'Player1',
			black: 'Player2',
			result: '1-0',
			outcome: 'checkmate',
			movesCount: 10,
			pgn: '1. e4 e5 2. Qh5 Nc6 3. Bc4 Nf6 4. Qxf7# 1-0'
		});

		expect(saved.id).toBeDefined();
		const list = getLocalGames();
		expect(list.length).toBe(1);
		expect(list[0].white).toBe('Player1');
		expect(list[0].result).toBe('1-0');
	});

	it('deletes games by ID', () => {
		const g1 = saveLocalGame({
			date: new Date().toISOString(),
			event: 'Game 1',
			white: 'A',
			black: 'B',
			result: '1-0',
			outcome: 'resignation',
			movesCount: 20,
			pgn: '1. e4 1-0'
		});

		const g2 = saveLocalGame({
			date: new Date().toISOString(),
			event: 'Game 2',
			white: 'C',
			black: 'D',
			result: '0-1',
			outcome: 'timeout',
			movesCount: 30,
			pgn: '1. d4 0-1'
		});

		expect(getLocalGames().length).toBe(2);
		deleteLocalGame(g1.id);
		const remaining = getLocalGames();
		expect(remaining.length).toBe(1);
		expect(remaining[0].id).toBe(g2.id);
	});

	it('stages and retrieves games for post-game review', () => {
		stageGameForReview('1. e4 e5 1-0', 'WhiteHero', 'BlackHero');
		const staged = getStagedGameForReview();
		expect(staged).not.toBeNull();
		expect(staged?.white).toBe('WhiteHero');
		expect(staged?.black).toBe('BlackHero');
		expect(staged?.pgn).toBe('1. e4 e5 1-0');
	});
});
