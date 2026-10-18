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
async function evaluateViaWorker(fen: string, depth = 12): Promise<EvaluationResult> {
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

		evaluationTimeout = setTimeout(() => {
			if (engineWorker) {
				engineWorker.postMessage('stop');
			}
		}, 8000);

		engineWorker.postMessage('stop');
		engineWorker.postMessage(`position fen ${fen}`);
		engineWorker.postMessage(`go depth ${Math.min(depth, 16)}`);
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
	depth = 12
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
			const localRes = await evaluateViaWorker(fen, depth);
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
