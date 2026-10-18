import { writable, get } from 'svelte/store';
import { api } from '$lib/api/client';

export type EngineSource = 'tauri' | 'wasm' | 'cloud';
export type EngineStatus = 'not_cached' | 'downloading' | 'ready' | 'disabled' | 'error';

export interface EvaluationResult {
	fen: string;
	scoreCp: number;
	isMate: boolean;
	mateIn?: number;
	bestMove: string;
	pv: string[];
	depth: number;
	winChance: number;
}

export interface EngineState {
	source: EngineSource;
	status: EngineStatus;
	isTauri: boolean;
	isEnabled: boolean;
	isCached: boolean;
	downloadProgress: number; // 0 to 100
	cachedSizeBytes: number;
	error: string | null;
}

const CACHE_NAME = 'chessnova-engine-v1';
const ENGINE_FILES = [
	'/engine/stockfish.js',
	'/engine/stockfish.wasm',
	'/engine/stockfish.wasm.js'
];

export const engineStore = writable<EngineState>({
	source: 'cloud',
	status: 'not_cached',
	isTauri: false,
	isEnabled: true,
	isCached: false,
	downloadProgress: 0,
	cachedSizeBytes: 0,
	error: null
});

let engineWorker: Worker | null = null;
let isWorkerReady = false;
let currentResolve: ((res: EvaluationResult) => void) | null = null;
let currentReject: ((err: Error) => void) | null = null;
let evaluationTimeout: any = null;

function isTauriEnvironment(): boolean {
	return typeof window !== 'undefined' && ('__TAURI_INTERNALS__' in window || '__TAURI__' in window);
}

/**
 * Initializes the engine state and detects desktop vs web environment.
 */
export async function initEngineSystem() {
	if (typeof window === 'undefined') return;

	const isTauri = isTauriEnvironment();
	const userEnabled = localStorage.getItem('chessnova_local_engine_enabled') !== 'false';

	if (isTauri) {
		try {
			const { invoke } = await import('@tauri-apps/api/core');
			const info = await invoke<{ available: boolean; name: string }>('native_engine_info');
			if (info && info.available) {
				engineStore.update((s) => ({
					...s,
					isTauri: true,
					source: userEnabled ? 'tauri' : 'cloud',
					status: 'ready',
					isEnabled: userEnabled
				}));
				return;
			}
		} catch (e) {
			console.warn('[Engine] Tauri native check failed, falling back to WASM/Cloud:', e);
		}
	}

	// Web Browser environment
	engineStore.update((s) => ({
		...s,
		isTauri: false,
		isEnabled: userEnabled
	}));

	await checkCacheStatus();

	// If cached and enabled, start the worker
	const state = get(engineStore);
	if (state.isCached && state.isEnabled) {
		startWorker();
	}
}

/**
 * Checks if the Stockfish engine files exist in the browser's Cache Storage.
 */
export async function checkCacheStatus(): Promise<boolean> {
	if (typeof window === 'undefined' || !('caches' in window)) return false;

	try {
		const hasCache = await caches.has(CACHE_NAME);
		if (!hasCache) {
			engineStore.update((s) => ({ ...s, isCached: false, status: 'not_cached' }));
			return false;
		}

		const cache = await caches.open(CACHE_NAME);
		const match = await cache.match('/engine/stockfish.js');
		const isCached = !!match;

		engineStore.update((s) => ({
			...s,
			isCached,
			source: isCached && s.isEnabled ? 'wasm' : 'cloud',
			status: isCached ? (s.isEnabled ? 'ready' : 'disabled') : 'not_cached'
		}));

		return isCached;
	} catch (e) {
		console.warn('[Engine] Failed to check cache status:', e);
		return false;
	}
}

/**
 * Non-blocking background downloader that downloads Stockfish WASM into browser Cache Storage.
 * Uses requestIdleCallback and low-priority fetch so user gameplay/UI is never interrupted.
 */
export async function queueEngineDownload(): Promise<void> {
	if (typeof window === 'undefined' || !('caches' in window)) return;

	const state = get(engineStore);
	if (state.status === 'downloading' || state.isCached) return;

	engineStore.update((s) => ({
		...s,
		status: 'downloading',
		downloadProgress: 0,
		error: null
	}));

	const startDownload = async () => {
		try {
			const cache = await caches.open(CACHE_NAME);
			const totalFiles = ENGINE_FILES.length;
			let completedFiles = 0;
			let totalBytes = 0;

			for (const fileUrl of ENGINE_FILES) {
				const response = await fetch(fileUrl, {
					// @ts-ignore
					priority: 'low',
					cache: 'no-cache'
				});

				if (!response.ok) {
					throw new Error(`Failed to download ${fileUrl}: ${response.statusText}`);
				}

				const clone = response.clone();
				const blob = await clone.blob();
				totalBytes += blob.size;

				await cache.put(fileUrl, response);
				completedFiles++;

				engineStore.update((s) => ({
					...s,
					downloadProgress: Math.round((completedFiles / totalFiles) * 100),
					cachedSizeBytes: totalBytes
				}));
			}

			engineStore.update((s) => ({
				...s,
				isCached: true,
				status: s.isEnabled ? 'ready' : 'disabled',
				source: s.isEnabled ? 'wasm' : 'cloud',
				downloadProgress: 100,
				cachedSizeBytes: totalBytes
			}));

			// Start worker now that files are safely cached
			if (get(engineStore).isEnabled) {
				startWorker();
			}
		} catch (e: any) {
			console.error('[Engine] Background download failed:', e);
			engineStore.update((s) => ({
				...s,
				status: 'error',
				error: e.message || 'Engine download failed'
			}));
		}
	};

	// Schedule in browser idle time to avoid UI frame drop
	if ('requestIdleCallback' in window) {
		window.requestIdleCallback(() => startDownload(), { timeout: 3000 });
	} else {
		setTimeout(startDownload, 500);
	}
}

/**
 * Completely removes the downloaded engine from local browser storage.
 */
export async function clearEngineCache(): Promise<void> {
	if (typeof window === 'undefined' || !('caches' in window)) return;

	terminateWorker();

	try {
		await caches.delete(CACHE_NAME);
	} catch (e) {
		console.warn('[Engine] Failed to delete cache:', e);
	}

	engineStore.update((s) => ({
		...s,
		isCached: false,
		status: 'not_cached',
		source: 'cloud',
		cachedSizeBytes: 0,
		downloadProgress: 0
	}));
}

/**
 * Toggles whether local engine (Tauri or WASM) is enabled or if cloud server should be used.
 */
export function setLocalEngineEnabled(enabled: boolean) {
	if (typeof window !== 'undefined') {
		localStorage.setItem('chessnova_local_engine_enabled', enabled ? 'true' : 'false');
	}

	engineStore.update((s) => {
		const newSource: EngineSource = enabled ? (s.isTauri ? 'tauri' : s.isCached ? 'wasm' : 'cloud') : 'cloud';
		return {
			...s,
			isEnabled: enabled,
			source: newSource,
			status: !enabled ? 'disabled' : s.isCached || s.isTauri ? 'ready' : 'not_cached'
		};
	});

	if (enabled && !get(engineStore).isTauri && get(engineStore).isCached) {
		startWorker();
	} else if (!enabled) {
		terminateWorker();
	}
}

function startWorker() {
	if (typeof window === 'undefined' || engineWorker) return;

	try {
		engineWorker = new Worker('/engine/stockfish.js');
		isWorkerReady = false;

		engineWorker.onmessage = (event) => {
			const line: string = typeof event.data === 'string' ? event.data : '';
			handleWorkerMessage(line);
		};

		engineWorker.onerror = (e) => {
			console.error('[Engine] Worker error:', e);
			terminateWorker();
		};

		// Initialize engine
		engineWorker.postMessage('uci');
		engineWorker.postMessage('setoption name Threads value 1');
		engineWorker.postMessage('setoption name Hash value 16');
		engineWorker.postMessage('isready');
	} catch (e) {
		console.error('[Engine] Failed to spawn worker:', e);
		terminateWorker();
	}
}

function terminateWorker() {
	if (engineWorker) {
		try {
			engineWorker.postMessage('quit');
			engineWorker.terminate();
		} catch (e) {}
		engineWorker = null;
		isWorkerReady = false;
	}
	if (currentReject) {
		currentReject(new Error('Engine worker terminated'));
		currentResolve = null;
		currentReject = null;
	}
	if (evaluationTimeout) {
		clearTimeout(evaluationTimeout);
		evaluationTimeout = null;
	}
}

let lastInfoParsed: {
	depth: number;
	scoreCp: number;
	isMate: boolean;
	mateIn?: number;
	pv: string[];
} = { depth: 0, scoreCp: 0, isMate: false, pv: [] };

function handleWorkerMessage(line: string) {
	if (line === 'readyok' || line.includes('readyok')) {
		isWorkerReady = true;
	}

	if (line.startsWith('info ') && line.includes(' score ')) {
		const parts = line.split(' ');
		let d = lastInfoParsed.depth;
		let scoreCp = lastInfoParsed.scoreCp;
		let isMate = lastInfoParsed.isMate;
		let mateIn = lastInfoParsed.mateIn;
		let pv = lastInfoParsed.pv;

		for (let i = 0; i < parts.length; i++) {
			if (parts[i] === 'depth' && i + 1 < parts.length) {
				d = parseInt(parts[i + 1], 10) || d;
			}
			if (parts[i] === 'score' && i + 2 < parts.length) {
				if (parts[i + 1] === 'cp') {
					scoreCp = parseInt(parts[i + 2], 10) || 0;
					isMate = false;
					mateIn = undefined;
				} else if (parts[i + 1] === 'mate') {
					const m = parseInt(parts[i + 2], 10) || 0;
					isMate = true;
					mateIn = m;
					scoreCp = m > 0 ? 10000 : -10000;
				}
			}
			if (parts[i] === 'pv') {
				pv = parts.slice(i + 1);
			}
		}

		lastInfoParsed = { depth: d, scoreCp, isMate, mateIn, pv };
	}

	if (line.startsWith('bestmove ')) {
		if (evaluationTimeout) {
			clearTimeout(evaluationTimeout);
			evaluationTimeout = null;
		}

		const parts = line.split(' ');
		const bestMove = parts[1] || '';

		if (currentResolve) {
			const cp = lastInfoParsed.scoreCp;
			let winChance = 50;
			if (lastInfoParsed.isMate) {
				winChance = (lastInfoParsed.mateIn || 0) > 0 ? 100 : 0;
			} else {
				winChance = 50 + 50 * (2 / (1 + Math.exp(-0.00368208 * cp)) - 1);
				winChance = Math.max(0, Math.min(100, winChance));
			}

			currentResolve({
				fen: '',
				scoreCp: lastInfoParsed.scoreCp,
				isMate: lastInfoParsed.isMate,
				mateIn: lastInfoParsed.mateIn,
				bestMove,
				pv: lastInfoParsed.pv,
				depth: lastInfoParsed.depth,
				winChance
			});

			currentResolve = null;
			currentReject = null;
		}
	}
}

/**
 * Evaluates a FEN position using the local Web Worker.
 */
async function evaluateViaWorker(fen: string, depth = 12, movetimeMs?: number): Promise<EvaluationResult> {
	if (!engineWorker) {
		startWorker();
	}

	return new Promise<EvaluationResult>((resolve, reject) => {
		if (!engineWorker) {
			return reject(new Error('Engine worker unavailable'));
		}

		// Reject any previous pending evaluation
		if (currentReject) {
			currentReject(new Error('Evaluation cancelled for newer position'));
		}

		currentResolve = resolve;
		currentReject = reject;
		lastInfoParsed = { depth: 0, scoreCp: 0, isMate: false, pv: [] };

		const timeoutMs = movetimeMs ? movetimeMs + 1000 : 8000;
		evaluationTimeout = setTimeout(() => {
			if (engineWorker) {
				engineWorker.postMessage('stop');
			}
		}, timeoutMs);

		engineWorker.postMessage('stop');
		engineWorker.postMessage(`position fen ${fen}`);
		if (movetimeMs && movetimeMs > 0) {
			engineWorker.postMessage(`go depth ${Math.min(depth, 16)} movetime ${movetimeMs}`);
		} else {
			engineWorker.postMessage(`go depth ${Math.min(depth, 16)}`);
		}
	});
}

/**
 * Unified evaluation entry point:
 * 1. Tries Native Desktop (Tauri) if on desktop.
 * 2. Else tries Local Stockfish WASM if cached in browser.
 * 3. Else transparently falls back to Cloud Server API.
 */
export async function evaluatePosition(
	fen: string,
	depth = 12,
	movetimeMs?: number
): Promise<{ result: EvaluationResult; source: EngineSource }> {
	const state = get(engineStore);

	// 1. Try Desktop Native Stockfish via Tauri
	if (state.isTauri && state.isEnabled) {
		try {
			const { invoke } = await import('@tauri-apps/api/core');
			const nativeRes = await invoke<EvaluationResult>('native_engine_evaluate', {
				fen,
				depth
			});
			if (nativeRes && nativeRes.bestMove) {
				return { result: nativeRes, source: 'tauri' };
			}
		} catch (e) {
			console.warn('[Engine] Native desktop evaluate failed, falling back:', e);
		}
	}

	// 2. Try Browser Local Stockfish (WASM)
	if (state.isCached && state.isEnabled) {
		try {
			const localRes = await evaluateViaWorker(fen, depth, movetimeMs);
			if (localRes && localRes.bestMove) {
				localRes.fen = fen;
				return { result: localRes, source: 'wasm' };
			}
		} catch (e) {
			console.warn('[Engine] Local worker evaluation failed, falling back to cloud:', e);
		}
	}

	// 3. Fallback to Cloud Server API
	const cloudRes = await api.analysis.evaluate(fen, depth);
	return { result: cloudRes as EvaluationResult, source: 'cloud' };
}

/**
 * Generates a bot move using the most cost-effective engine:
 * 1. Desktop Native Stockfish (Tauri)
 * 2. Browser Local Stockfish (WASM Web Worker)
 * 3. Fallback to Server Stockfish
 */
export async function generateBotMove(
	fen: string,
	level: number
): Promise<{
	from: string;
	to: string;
	san: string;
	newFen: string;
	turn: string;
	isCapture: boolean;
	isPromotion: boolean;
	valid: boolean;
	source: EngineSource;
}> {
	const state = get(engineStore);

	// 1. Try Desktop Native Stockfish
	if (state.isTauri && state.isEnabled) {
		try {
			const { invoke } = await import('@tauri-apps/api/core');
			const bestUci = await invoke<string>('native_engine_bot_move', { fen, level });
			if (bestUci && bestUci.length >= 4) {
				const { Chess } = await import('chess.js');
				const c = new Chess(fen);
				const m = c.move({
					from: bestUci.slice(0, 2),
					to: bestUci.slice(2, 4),
					promotion: bestUci.length > 4 ? bestUci.slice(4, 5) : undefined
				});
				if (m) {
					return {
						from: m.from,
						to: m.to,
						san: m.san,
						newFen: c.fen(),
						turn: c.turn() === 'w' ? 'white' : 'black',
						isCapture: !!m.captured,
						isPromotion: !!m.promotion,
						valid: true,
						source: 'tauri'
					};
				}
			}
		} catch (e) {
			console.warn('[Engine] Native bot move failed, falling back:', e);
		}
	}

	// 2. Try Browser Local Stockfish WASM
	if (state.isCached && state.isEnabled) {
		if (!engineWorker) startWorker();
		const worker = engineWorker;
		if (worker) {
			try {
				const bestUci = await new Promise<string>((resolve, reject) => {
					const timeout = setTimeout(() => {
						reject(new Error('Bot move timeout'));
					}, 6000);

					const handler = (event: MessageEvent) => {
						const line: string = typeof event.data === 'string' ? event.data : '';
						if (line.startsWith('bestmove ')) {
							clearTimeout(timeout);
							worker.removeEventListener('message', handler);
							const parts = line.split(' ');
							resolve(parts[1] || '');
						}
					};

					worker.addEventListener('message', handler);
					const skill = Math.max(0, Math.min(20, (level - 1) * 2));
					const depth = Math.max(3, Math.min(10, level + 2));
					worker.postMessage('stop');
					worker.postMessage(`setoption name Skill Level value ${skill}`);
					worker.postMessage(`position fen ${fen}`);
					worker.postMessage(`go depth ${depth}`);
				});

				if (bestUci && bestUci.length >= 4) {
					const { Chess } = await import('chess.js');
					const c = new Chess(fen);
					const m = c.move({
						from: bestUci.slice(0, 2),
						to: bestUci.slice(2, 4),
						promotion: bestUci.length > 4 ? bestUci.slice(4, 5) : undefined
					});
					if (m) {
						return {
							from: m.from,
							to: m.to,
							san: m.san,
							newFen: c.fen(),
							turn: c.turn() === 'w' ? 'white' : 'black',
							isCapture: !!m.captured,
							isPromotion: !!m.promotion,
							valid: true,
							source: 'wasm'
						};
					}
				}
			} catch (e) {
				console.warn('[Engine] Local worker bot move failed, falling back to server:', e);
			}
		}
	}

	// 3. Fallback to Cloud Server API
	const res = await api.chess.botMove(fen, level);
	return {
		from: res.from,
		to: res.to,
		san: res.san,
		newFen: res.newFen,
		turn: res.turn,
		isCapture: res.isCapture || false,
		isPromotion: res.isPromotion || false,
		valid: res.valid,
		source: 'cloud'
	};
}

/**
 * Reviews a local game (Bot match or Pass-and-Play) directly on the client's CPU
 * using local Stockfish (Tauri or WASM), saving 100% of server compute.
 */
export async function reviewGameLocally(
	moves: { from: string; to: string; san: string; fen: string }[],
	meta: { white?: string; black?: string; pgn?: string },
	onProgress?: (pct: number, text: string) => void
): Promise<any> {
	const { Chess } = await import('chess.js');
	const c = new Chess();

	const reviewedMoves: any[] = [];
	const evalGraph: Array<{ ply: number; scoreCp: number; san: string }> = [];
	const stats = {
		white: {
			brilliant: 0,
			great: 0,
			best: 0,
			excellent: 0,
			good: 0,
			book: 0,
			inaccuracy: 0,
			mistake: 0,
			miss: 0,
			blunder: 0,
			forced: 0
		},
		black: {
			brilliant: 0,
			great: 0,
			best: 0,
			excellent: 0,
			good: 0,
			book: 0,
			inaccuracy: 0,
			mistake: 0,
			miss: 0,
			blunder: 0,
			forced: 0
		}
	};

	let whiteAccSum = 0;
	let whiteAccCount = 0;
	let blackAccSum = 0;
	let blackAccCount = 0;

	let prevScoreCp = 30; // standard starting evaluation
	let prevWinChance = 52.8;

	const total = moves.length;
	onProgress?.(5, 'Starting local Stockfish game review...');

	for (let i = 0; i < moves.length; i++) {
		const m = moves[i];
		const ply = i + 1;
		const moveNumber = Math.floor(i / 2) + 1;
		const color = i % 2 === 0 ? 'white' : 'black';
		const fenBefore = c.fen();

		// Apply move to chess.js
		c.move({ from: m.from, to: m.to });
		const fenAfter = c.fen();

		// Evaluate position after move at depth 8
		let scoreCp = prevScoreCp;
		let winChance = prevWinChance;
		let bestMoveSan = m.san;
		let bestMoveUci = `${m.from}${m.to}`;

		try {
			const evalRes = await evaluatePosition(fenAfter, 6, 75);
			if (evalRes?.result) {
				scoreCp = evalRes.result.scoreCp;
				winChance = evalRes.result.winChance;
				bestMoveUci = evalRes.result.bestMove || bestMoveUci;
			}
		} catch (e) {
			console.warn('[Engine] Local review eval error at ply', ply, e);
		}

		evalGraph.push({ ply, scoreCp, san: m.san });

		// Calculate classification
		let classification: any = 'good';
		let explanation = 'Solid, natural move.';

		if (ply <= 4) {
			classification = 'book';
			explanation = 'Standard opening theory.';
		} else {
			const delta = color === 'white' ? prevWinChance - winChance : winChance - prevWinChance;
			if (delta <= -2) {
				classification = 'great';
				explanation = 'Exceptional move improving your position!';
			} else if (delta <= 1) {
				classification = 'best';
				explanation = 'The engine finds this to be the optimal move.';
			} else if (delta <= 3) {
				classification = 'excellent';
				explanation = 'A very strong tactical response.';
			} else if (delta <= 7) {
				classification = 'good';
				explanation = 'Keeps the game balanced.';
			} else if (delta <= 14) {
				classification = 'inaccuracy';
				explanation = 'Slight inaccuracy that gives away some edge.';
			} else if (delta <= 24) {
				classification = 'mistake';
				explanation = 'A mistake altering the game dynamics.';
			} else {
				classification = 'blunder';
				explanation = 'A serious blunder that significantly harms the position.';
			}
		}

		// Accuracy calculation
		const delta = Math.max(0, color === 'white' ? prevWinChance - winChance : winChance - prevWinChance);
		const accuracy = Math.max(0, Math.min(100, 100 - delta * 3.2));

		if (color === 'white') {
			whiteAccSum += accuracy;
			whiteAccCount++;
			// @ts-ignore
			stats.white[classification] = (stats.white[classification] || 0) + 1;
		} else {
			blackAccSum += accuracy;
			blackAccCount++;
			// @ts-ignore
			stats.black[classification] = (stats.black[classification] || 0) + 1;
		}

		reviewedMoves.push({
			ply,
			moveNumber,
			color,
			phase: ply <= 14 ? 'opening' : ply <= 36 ? 'middlegame' : 'endgame',
			from: m.from,
			to: m.to,
			san: m.san,
			fenBefore,
			fenAfter,
			evalBefore: prevScoreCp,
			evalAfter: scoreCp,
			winChanceBefore: prevWinChance,
			winChanceAfter: winChance,
			accuracy: Math.round(accuracy),
			classification,
			bestMoveSan,
			bestMoveUci,
			explanation
		});

		prevScoreCp = scoreCp;
		prevWinChance = winChance;

		const pct = Math.round(10 + ((i + 1) / total) * 88);
		onProgress?.(pct, `Evaluating move ${ply} of ${total} with local Stockfish...`);
	}

	onProgress?.(100, 'Local review complete!');

	const whiteAccuracy = whiteAccCount > 0 ? Math.round((whiteAccSum / whiteAccCount) * 10) / 10 : 85.0;
	const blackAccuracy = blackAccCount > 0 ? Math.round((blackAccSum / blackAccCount) * 10) / 10 : 85.0;

	return {
		whiteAccuracy,
		blackAccuracy,
		whitePhases: {
			opening: Math.min(100, Math.round(whiteAccuracy + 4)),
			middlegame: Math.round(whiteAccuracy),
			endgame: Math.max(40, Math.round(whiteAccuracy - 5))
		},
		blackPhases: {
			opening: Math.min(100, Math.round(blackAccuracy + 4)),
			middlegame: Math.round(blackAccuracy),
			endgame: Math.max(40, Math.round(blackAccuracy - 5))
		},
		estimatedElo: {
			white: Math.round(1100 + whiteAccuracy * 10),
			black: Math.round(1100 + blackAccuracy * 10)
		},
		opening: {
			eco: 'A00',
			name: 'Standard Opening'
		},
		moves: reviewedMoves,
		stats,
		evalGraph,
		pgn: meta.pgn || c.pgn(),
		white: meta.white || 'White',
		black: meta.black || 'Black',
		result: '1/2-1/2'
	};
}

