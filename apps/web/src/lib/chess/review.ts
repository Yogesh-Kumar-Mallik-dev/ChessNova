// ==============================================================================
// ChessNova Game Review Analysis Engine - Client Types & Authoritative API
// All business logic, Stockfish evaluation, CAPS 2 calculation, and ECO theory
// are authoritatively handled by the Go backend server (apps/server/internal/review)
// ==============================================================================

import { api } from '$lib/api/client';

export type MoveClassification =
	| 'brilliant'   // !!
	| 'great'       // !
	| 'best'        // ★
	| 'excellent'   // ✓
	| 'good'        // ✓
	| 'book'        // 📖
	| 'inaccuracy'  // ?!
	| 'mistake'     // ?
	| 'miss'        // ⨉
	| 'blunder'     // ??
	| 'forced';     // □

export interface ReviewedMove {
	ply: number;
	moveNumber: number;
	color: 'white' | 'black';
	from: string;
	to: string;
	san: string;
	fenBefore: string;
	fenAfter: string;
	evalBefore: number;      // Centipawns from White's perspective
	evalAfter: number;       // Centipawns from White's perspective
	winChanceBefore: number; // 0 to 100
	winChanceAfter: number;  // 0 to 100
	accuracy: number;        // 0 to 100%
	classification: MoveClassification;
	bestMoveSan: string;
	bestMoveUci: string;
	explanation: string;
}

export interface GameReviewResult {
	whiteAccuracy: number;
	blackAccuracy: number;
	opening: {
		eco: string;
		name: string;
		variation?: string;
	} | null;
	moves: ReviewedMove[];
	stats: {
		white: Record<MoveClassification, number>;
		black: Record<MoveClassification, number>;
	};
	evalGraph: Array<{
		ply: number;
		scoreCp: number;
		san: string;
	}>;
	pgn?: string;
	headers?: Record<string, string>;
	white?: string;
	black?: string;
	result?: string;
}

/**
 * Executes authoritative Game Review via the Go backend Stockfish service.
 */
export async function performGameReview(
	input:
		| string
		| { pgn?: string; moves?: Array<{ from: string; to: string; san: string; fen?: string }> }
		| Array<{ from: string; to: string; san: string; fen?: string }>
): Promise<GameReviewResult> {
	return (await api.analysis.review(input)) as GameReviewResult;
}
