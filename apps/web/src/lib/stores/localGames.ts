// ==============================================================================
// ChessNova Local Game Archive & History Store
// Automatically preserves completed games in localStorage for non-signed (and signed)
// users, enabling full Stockfish Game Reviews, PGN exports, and local game history
// inspired by Chess.com and Lichess.
// ==============================================================================

export interface LocalGameRecord {
	id: string;
	date: string;               // ISO date string
	timestamp: number;
	event: string;              // e.g. "Pass & Play", "Live Match", "Imported"
	white: string;
	black: string;
	result: string;             // "1-0", "0-1", "1/2-1/2", "*"
	outcome: string;            // "checkmate", "resignation", "stalemate", etc.
	winner?: string;            // "white", "black", "draw"
	timeControl?: string;       // e.g. "5+0", "10+0", "-"
	movesCount: number;
	pgn: string;
	startFen?: string;
}

const STORAGE_KEY = 'chessnova_local_games_v1';
const MAX_LOCAL_GAMES = 50;

/**
 * Retrieve all locally saved games from localStorage (newest first).
 */
export function getLocalGames(): LocalGameRecord[] {
	if (typeof window === 'undefined') return [];
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return [];
		const parsed = JSON.parse(raw);
		if (Array.isArray(parsed)) {
			return parsed;
		}
	} catch (e) {
		console.warn('Failed to parse local games archive:', e);
	}
	return [];
}

/**
 * Save a completed game to localStorage.
 * If a game with the same ID already exists, it is updated; otherwise prepend.
 */
export function saveLocalGame(record: Omit<LocalGameRecord, 'id' | 'timestamp'> & { id?: string; timestamp?: number }): LocalGameRecord {
	if (typeof window === 'undefined') {
		return {
			...record,
			id: record.id || `game_${Date.now()}`,
			timestamp: record.timestamp || Date.now()
		};
	}

	const existing = getLocalGames();
	const finalId = record.id || `local_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`;
	const finalTimestamp = record.timestamp || Date.now();

	const newRecord: LocalGameRecord = {
		...record,
		id: finalId,
		timestamp: finalTimestamp
	};

	// Filter out any existing game with the same ID
	const filtered = existing.filter((g) => g.id !== finalId);
	// Place newest game at the top, limit to MAX_LOCAL_GAMES
	const updated = [newRecord, ...filtered].slice(0, MAX_LOCAL_GAMES);

	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
	} catch (e) {
		console.error('Failed to save game to localStorage:', e);
	}

	return newRecord;
}

/**
 * Delete a specific game from local archive by ID.
 */
export function deleteLocalGame(id: string): void {
	if (typeof window === 'undefined') return;
	try {
		const existing = getLocalGames();
		const updated = existing.filter((g) => g.id !== id);
		localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
	} catch (e) {
		console.error('Failed to delete game from local archive:', e);
	}
}

/**
 * Clear all locally saved games.
 */
export function clearAllLocalGames(): void {
	if (typeof window === 'undefined') return;
	try {
		localStorage.removeItem(STORAGE_KEY);
	} catch (e) {
		console.error('Failed to clear local games archive:', e);
	}
}

/**
 * Stage a game into sessionStorage and URL for instant Game Review launch.
 */
export function stageGameForReview(pgn: string, white = 'White', black = 'Black'): void {
	if (typeof window === 'undefined') return;
	try {
		sessionStorage.setItem('review_pgn', pgn.trim());
		sessionStorage.setItem(
			'review_players',
			JSON.stringify({
				white,
				black
			})
		);
	} catch (e) {
		console.warn('Failed to stage game for review:', e);
	}
}
