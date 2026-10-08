export type Color = 'white' | 'black';
export type PieceType = 'pawn' | 'knight' | 'bishop' | 'rook' | 'queen' | 'king';
export type GameStatus = 'waiting' | 'playing' | 'finished' | 'aborted';

export const MIN_RATING = 100;
export const DEFAULT_RATING = 400;

export interface TimeControl {
	initial: number;
	increment: number;
}

export interface PlayerInfo {
	userId: string;
	username: string;
	rating: number;
}

export interface GameEvent {
	type: string;
	gameId?: string;
	from?: string;
	to?: string;
	san?: string;
	fen?: string;
	turn?: Color;
	whiteTime?: number;
	blackTime?: number;
	status?: GameStatus;
	result?: string;
	outcome?: string;
	isCheck?: boolean;
}
