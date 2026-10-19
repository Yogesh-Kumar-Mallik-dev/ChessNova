// ==============================================================================
// ChessPasant Local Game Archive & History Store
// Automatically preserves completed games in localStorage for non-signed (and signed)
// users, enabling full Game Reviews, PGN exports, and local game history.
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

const STORAGE_KEY = 'chesspasant_local_games_v1';
const LEGACY_STORAGE_KEY = 'chessnova_local_games_v1';
const MAX_LOCAL_GAMES = 50;

/**
 * Retrieve all locally saved games from localStorage (newest first).
 */
export function getLocalGames(): LocalGameRecord[] {
	if (typeof window === 'undefined') return [];
	try {
		let raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) {
			raw = localStorage.getItem(LEGACY_STORAGE_KEY);
			if (raw) {
				localStorage.setItem(STORAGE_KEY, raw);
			}
		}
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
export function stageGameForReview(pgn: string, white = 'White', black = 'Black', isLocal = false): void {
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
		sessionStorage.setItem('review_is_local', isLocal ? 'true' : 'false');
	} catch (e) {
		console.warn('Failed to stage game for review:', e);
	}
}

/**
 * Retrieve staged game review data from sessionStorage.
 */
export function getStagedGameForReview(): { pgn: string; white: string; black: string } | null {
	if (typeof window === 'undefined') return null;
	try {
		const pgn = sessionStorage.getItem('review_pgn');
		if (!pgn) return null;
		const raw = sessionStorage.getItem('review_players');
		let players = { white: 'White', black: 'Black' };
		if (raw) {
			players = JSON.parse(raw);
		}
		return { pgn, white: players.white, black: players.black };
	} catch {
		return null;
	}
}
