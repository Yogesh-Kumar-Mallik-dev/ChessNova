import { writable } from 'svelte/store';
import { INITIAL_FEN, type Color } from '$lib/chess/engine';

export interface PlayerInfo {
	userId: string;
	username: string;
	rating: number;
}

export interface GameMoveRecord {
	from: string;
	to: string;
	san: string;
	fen: string;
}

export interface GameState {
	gameId: string | null;
	fen: string;
	turn: Color;
	whitePlayer: PlayerInfo | null;
	blackPlayer: PlayerInfo | null;
	whiteTime: number; // in ms
	blackTime: number; // in ms
	status: 'waiting' | 'playing' | 'finished' | 'aborted';
	outcome?: string;
	result?: string;
	winner?: string;
	timeControl?: { initial: number; increment: number };
	selectedSquare: string | null;
	legalMoves: string[];
	lastMove: { from: string; to: string; san?: string; isCapture?: boolean; isPromotion?: boolean } | null;
	isCheck: boolean;
	connectionStatus: 'disconnected' | 'connecting' | 'connected';
	orientation: Color;
	promotionPending: { from: string; to: string } | null;
	drawOfferBy: string | null;
	moves: GameMoveRecord[];
}

const initialGameState: GameState = {
	gameId: null,
	fen: INITIAL_FEN,
	turn: 'white',
	whitePlayer: null,
	blackPlayer: null,
	whiteTime: 300000,
	blackTime: 300000,
	status: 'waiting',
	selectedSquare: null,
	legalMoves: [],
	lastMove: null,
	isCheck: false,
	connectionStatus: 'disconnected',
	orientation: 'white',
	promotionPending: null,
	drawOfferBy: null,
	moves: []
};

export const gameStore = writable<GameState>(initialGameState);

export function resetGameStore() {
	gameStore.set({ ...initialGameState });
}
